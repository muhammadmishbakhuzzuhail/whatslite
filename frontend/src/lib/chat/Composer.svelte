<script>
  import { tick, onDestroy } from "svelte";
  import { get } from "svelte/store";
  import { sendMessage, sendMediaMessage, replyDraft, pushToast, editDraft, editMessage, chats, mediaDraft, theme, setDraft, getDraft } from "../../stores.js";
  import { getGroupInfo, sendLocation, sendPoll, sendContact, sendGif, sendSticker, fetchRemoteMedia, scheduleMessage } from "../../services/data.js";
  import GifPicker from "./GifPicker.svelte";
  import StickerPicker from "./StickerPicker.svelte";
  import { t } from "../i18n.js";
  export let chatId;
  export let group = false;

  let value = "";
  // Draf per-chat: simpan teks chat lama saat pindah, pulihkan draf chat baru.
  let _draftChat = null;
  $: if (chatId !== _draftChat) {
    if (_draftChat != null && !$editDraft) setDraft(_draftChat, value);
    value = $editDraft ? value : getDraft(chatId);
    _draftChat = chatId;
  }
  function saveDraft() { if (!$editDraft) setDraft(chatId, value); }
  // Jadwalkan pesan: pilih waktu → simpan terjadwal (dikirim oleh ticker).
  let schedInput;
  function openSchedule() { if (schedInput) { schedInput.showPicker ? schedInput.showPicker() : schedInput.click(); } }
  function onSchedule(e) {
    const v = e.target.value; e.target.value = "";
    if (!v || !value.trim()) return;
    const at = Math.floor(new Date(v).getTime() / 1000);
    if (at * 1000 < Date.now()) { pushToast($t("schedule_past")); return; }
    scheduleMessage(chatId, value.trim(), at);
    value = ""; setDraft(chatId, "");
    pushToast($t("scheduled_ok"), "ok");
  }
  // Mode sunting: isi composer dgn teks pesan yg disunting.
  let lastEdit = null;
  $: if ($editDraft && $editDraft.id !== lastEdit) { lastEdit = $editDraft.id; value = $editDraft.text; }
  $: if (!$editDraft) lastEdit = null;
  let inputEl;
  // Auto-fokus ke textarea saat mulai balas / sunting (UX: langsung bisa ketik).
  let _replyId = null, _editId = null;
  $: if ($replyDraft && $replyDraft.id !== _replyId) { _replyId = $replyDraft.id; focusInput(); }
  $: if (!$replyDraft) _replyId = null;
  $: if ($editDraft && $editDraft.id !== _editId) { _editId = $editDraft.id; focusInput(); }
  $: if (!$editDraft) _editId = null;
  function focusInput() { tick().then(() => inputEl && inputEl.focus()); }
  $: typing = value.trim().length > 0;
  let emojiOpen = new URLSearchParams(location.search).get("emoji") === "1";
  let fileInput;

  // --- menu lampiran (file / lokasi / polling) ---
  let attachOpen = false;
  function toggleAttach() { attachOpen = !attachOpen; }

  // Quick replies (template balasan tersimpan, lokal).
  let qrOpen = false, quickReplies = [];
  try { quickReplies = JSON.parse(localStorage.getItem("wa-quickreplies") || "[]") || []; } catch (e) {}
  function saveQR(v) { try { localStorage.setItem("wa-quickreplies", JSON.stringify(v)); } catch (e) {} }
  function openQuick() { attachOpen = false; qrOpen = true; }
  function insertQuick(txt) { value = value ? value + " " + txt : txt; qrOpen = false; focusInput(); saveDraft(); }
  function addQuick() { const v = value.trim(); if (!v) return; quickReplies = [v, ...quickReplies.filter((x) => x !== v)].slice(0, 30); saveQR(quickReplies); }
  function delQuick(t) { quickReplies = quickReplies.filter((x) => x !== t); saveQR(quickReplies); }
  function attachFile() { attachOpen = false; pickFile(); }
  // Dokumen: file APA PUN → kirim sebagai dokumen (bukan pratinjau gambar).
  let docInput;
  function attachDocument() { attachOpen = false; docInput && docInput.click(); }
  // Batas ukuran: file di-encode base64 (data-URI) di memori + lewat IPC →
  // file raksasa membekukan renderer WebKitGTK. Tolak > 64MB.
  const MAX_BYTES = 64 * 1024 * 1024;
  function tooBig(f) { if (f.size > MAX_BYTES) { pushToast($t("file_too_big")); return true; } return false; }
  async function onDoc(e) {
    const files = [...(e.target.files || [])];
    e.target.value = "";
    const items = [];
    for (const f of files) {
      if (tooBig(f)) continue;
      items.push({ kind: "document", name: f.name, dataURI: await fileToDataURI(f) });
    }
    if (items.length) mediaDraft.set({ chatId, items }); // → MediaPreviewModal (nama + caption)
  }
  function attachLocation() {
    attachOpen = false;
    if (!navigator.geolocation) { pushToast($t("loc_unavailable")); return; }
    navigator.geolocation.getCurrentPosition(
      (pos) => sendLocation(chatId, pos.coords.latitude, pos.coords.longitude, ""),
      () => pushToast($t("loc_denied")),
      { enableHighAccuracy: true, timeout: 10000 }
    );
  }
  // --- drag-drop & tempel gambar ---
  let dragOver = false;
  // audio/* yang DIPILIH = file (document), bukan PTT voice note (PTT khusus
  // rekaman mic → jalur handleMic). Cegah MP3 terkirim sbg voice memo.
  function kindOfFile(type) { return type.startsWith("video/") ? "video" : type.startsWith("image/") ? "image" : "document"; }
  function fileToDataURI(f) { return new Promise((res) => { const r = new FileReader(); r.onload = () => res(r.result); r.readAsDataURL(f); }); }
  // SEMUA jenis (gambar/video/audio/dokumen) → modal pratinjau dulu (preview +
  // caption), bukan kirim langsung. Modal menampilkan player audio / pdf / ikon.
  async function previewFiles(fileList, viewOnce = false) {
    const items = [];
    for (const f of fileList) {
      if (tooBig(f)) continue;
      const kind = kindOfFile(f.type);
      const dataURI = await fileToDataURI(f);
      items.push({ kind, name: f.name, dataURI });
    }
    if (items.length) mediaDraft.set({ chatId, items, viewOnce });
  }
  function previewFile(f, viewOnce = false) { previewFiles([f], viewOnce); }
  // Media dari web (URL) → unduh sisi Go → pratinjau.
  async function previewUrl(url) {
    pushToast($t("fetching_media"), "ok");
    const dataURI = await fetchRemoteMedia(url);
    if (!dataURI) { pushToast($t("media_fetch_fail")); return; }
    mediaDraft.set({ chatId, items: [{ kind: dataURI.startsWith("data:video") ? "video" : "image", name: "web", dataURI }] });
  }
  function onDrop(e) {
    e.preventDefault(); dragOver = false;
    const files = [...(e.dataTransfer?.files || [])];
    if (files.length) { previewFiles(files); return; }
    const uri = e.dataTransfer?.getData("text/uri-list") || e.dataTransfer?.getData("text/plain") || "";
    const m = uri.match(/https?:\/\/[^\s"]+/);
    if (m) previewUrl(m[0]);
  }
  function onPaste(e) {
    const items = [...(e.clipboardData?.items || [])];
    const files = items.filter((it) => it.type.startsWith("image/") || it.type.startsWith("video/")).map((it) => it.getAsFile()).filter(Boolean);
    if (files.length) { e.preventDefault(); previewFiles(files); return; }
    const text = (e.clipboardData?.getData("text") || "").trim();
    if (/^https?:\/\/\S+\.(png|jpe?g|gif|webp|bmp|mp4|webm|mov)(\?\S*)?$/i.test(text)) { e.preventDefault(); previewUrl(text); }
  }
  // --- GIF (Giphy) ---
  let gifOpen = false;
  function openGif() { attachOpen = false; gifOpen = true; }
  function onGifPick(e) { gifOpen = false; sendGif(chatId, e.detail); }
  // --- Stiker ---
  let stickerOpen = false;
  function openSticker() { attachOpen = false; stickerOpen = true; }
  function onStickerPick(e) { stickerOpen = false; sendSticker(chatId, e.detail); }
  // View-once kini toggle di preview media (MediaPreviewModal), bukan di attach menu.
  // --- kirim kontak ---
  let contactOpen = false, contactQ = "";
  function openContact() { attachOpen = false; contactOpen = true; contactQ = ""; }
  $: contactList = $chats.filter((c) => !c.group && (c.name || "").toLowerCase().includes(contactQ.toLowerCase()));
  function pickContact(c) {
    const num = (c.id || "").split("@")[0].split(":")[0];
    sendContact(chatId, c.name, num);
    contactOpen = false;
  }
  // --- modal polling ---
  let pollOpen = false, pollQ = "", pollOpts = ["", ""], pollMulti = false;
  function openPoll() { attachOpen = false; pollOpen = true; pollQ = ""; pollOpts = ["", ""]; pollMulti = false; }
  function addPollOpt() { if (pollOpts.length < 12) pollOpts = [...pollOpts, ""]; }
  function removePollOpt(i) { if (pollOpts.length > 2) pollOpts = pollOpts.filter((_, j) => j !== i); }
  function submitPoll() {
    const opts = pollOpts.map((o) => o.trim()).filter(Boolean);
    if (!pollQ.trim() || opts.length < 2) return;
    sendPoll(chatId, pollQ.trim(), opts, pollMulti ? opts.length : 1);
    pollOpen = false;
  }

  // --- @mention autocomplete ---
  let members = []; // {jid,name,num}
  let lastLoaded = null;
  $: if (group && chatId && chatId !== lastLoaded) loadMembers(chatId);
  async function loadMembers(id) {
    lastLoaded = id;
    members = [];
    const gi = await getGroupInfo(id);
    if (gi && gi.participants) {
      members = gi.participants.map((p) => ({ jid: p.jid, name: p.name || p.jid.split("@")[0], num: p.jid.split("@")[0] }));
    }
  }
  let picked = []; // {name,num,jid} yang sudah dipilih (utk konversi saat kirim)
  let mOpen = false, mStart = -1, mItems = [];
  function detectMention() {
    if (!inputEl) return;
    const cur = inputEl.selectionStart;
    const upto = value.slice(0, cur);
    const at = upto.lastIndexOf("@");
    if (at < 0 || (at > 0 && !/\s/.test(value[at - 1]))) { mOpen = false; return; }
    const q = upto.slice(at + 1);
    if (q.includes("\n")) { mOpen = false; return; }
    mStart = at;
    const ql = q.toLowerCase();
    const mem = members.filter((m) => m.name.toLowerCase().includes(ql)).slice(0, 8);
    const extra = [{ name: get(t)("mention_all"), special: true }, { name: "Meta AI", special: true }]
      .filter((x) => x.name.toLowerCase().includes(ql));
    mItems = [...mem, ...extra];
    mOpen = mItems.length > 0;
  }
  async function pickMention(item) {
    const cur = inputEl.selectionStart;
    const before = value.slice(0, mStart);
    const after = value.slice(cur);
    const insert = "@" + item.name + " ";
    value = before + insert + after;
    if (!item.special) picked = [...picked, { name: item.name, num: item.num, jid: item.jid }];
    mOpen = false;
    await tick();
    const pos = (before + insert).length;
    inputEl.selectionStart = inputEl.selectionEnd = pos;
    inputEl.focus();
  }

  function pickFile() { fileInput && fileInput.click(); }
  function kindOf(type) {
    if (type.startsWith("image/")) return "image";
    if (type.startsWith("video/")) return "video";
    return "document"; // audio dipilih = file, bukan PTT (PTT hanya dari rekam mic)
  }
  async function onFile(e) {
    const files = [...(e.target.files || [])];
    e.target.value = "";
    if (files.length) await previewFiles(files);
  }

  // Emoji + kata kunci (cari ID/EN) → filter di picker.
  // Emoji picker penuh (set Unicode + kategori + search + skin-tone + recents)
  // via emoji-picker-element. Modul + data di-load LAZY saat pertama dibuka.
  let emojiReady = false;
  async function ensureEmoji() {
    if (emojiReady) return;
    await import("emoji-picker-element");
    emojiReady = true;
  }
  function onEmojiClick(e) { if (e.detail?.unicode) addEmoji(e.detail.unicode); }
  async function toggleEmoji() { if (!emojiOpen) await ensureEmoji(); emojiOpen = !emojiOpen; }

  // --- Autocomplete :shortcode: (mis. :fire: → 🔥) ---
  let emojiDb = null;
  let scOpen = false, scStart = -1, scItems = [];
  async function ensureDb() {
    if (emojiDb) return;
    const m = await import("emoji-picker-element");
    emojiDb = new m.Database();
  }
  async function detectShortcode() {
    if (!inputEl) return;
    const cur = inputEl.selectionStart;
    const upto = value.slice(0, cur);
    const m = upto.match(/(?:^|\s)(:([a-z0-9_+\-]{2,}))$/i);
    if (!m) { scOpen = false; return; }
    scStart = cur - m[1].length;
    await ensureDb();
    try {
      const res = await emojiDb.getEmojiBySearchQuery(m[2]);
      scItems = (res || []).slice(0, 8).map((e) => ({ u: e.unicode || (e.skins && e.skins[0]?.native), name: (e.shortcodes && e.shortcodes[0]) || e.annotation })).filter((x) => x.u);
      scOpen = scItems.length > 0;
    } catch (e) { scOpen = false; }
  }
  async function pickShortcode(item) {
    const cur = inputEl.selectionStart;
    value = value.slice(0, scStart) + item.u + value.slice(cur);
    scOpen = false;
    await tick();
    const pos = scStart + item.u.length;
    inputEl.selectionStart = inputEl.selectionEnd = pos;
    inputEl.focus();
  }

  function send() {
    if (!value.trim()) return;
    // Mode sunting → edit pesan, bukan kirim baru.
    if ($editDraft) {
      editMessage($editDraft.chatId, $editDraft.id, value);
      editDraft.set(null); value = ""; return;
    }
    // Konversi @Nama → @<nomor> + kumpulkan JID utk mention nyata.
    let finalText = value;
    const jids = [];
    for (const p of picked) {
      if (finalText.includes("@" + p.name)) {
        finalText = finalText.replace("@" + p.name, "@" + p.num);
        jids.push(p.jid);
      }
    }
    sendMessage(chatId, finalText, $replyDraft, jids);
    value = ""; picked = []; mOpen = false;
    setDraft(chatId, ""); // teks terkirim → buang draf
    replyDraft.set(null);
  }
  // Bungkus teks terpilih dgn penanda format WhatsApp (Ctrl+B/I/dll).
  function wrapSel(mark) {
    if (!inputEl) return;
    const a = inputEl.selectionStart, b = inputEl.selectionEnd;
    const sel = value.slice(a, b) || "";
    value = value.slice(0, a) + mark + sel + mark + value.slice(b);
    saveDraft();
    tick().then(() => { inputEl.focus(); inputEl.selectionStart = a + mark.length; inputEl.selectionEnd = b + mark.length; });
  }
  function onKey(e) {
    if (scOpen && e.key === "Enter") { e.preventDefault(); pickShortcode(scItems[0]); return; }
    if (scOpen && e.key === "Escape") { scOpen = false; return; }
    if (mOpen && e.key === "Enter") { e.preventDefault(); pickMention(mItems[0]); return; }
    if (mOpen && e.key === "Escape") { mOpen = false; return; }
    if (e.ctrlKey || e.metaKey) {
      const k = e.key.toLowerCase();
      if (k === "b") { e.preventDefault(); wrapSel("*"); return; }   // tebal
      if (k === "i") { e.preventDefault(); wrapSel("_"); return; }   // miring
      if (e.shiftKey && k === "x") { e.preventDefault(); wrapSel("~"); return; } // coret
    }
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); send(); }
  }
  // Auto-grow textarea (multi-baris) sampai batas, lalu scroll.
  function autoGrow(el) {
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 130) + "px";
  }
  // Reset tinggi setelah kirim/clear.
  $: if (inputEl && value === "") inputEl.style.height = "auto";
  function addEmoji(e) { value += e; }

  // --- Rekam voice note (MediaRecorder) ---
  // WhatsApp memutar voice note sebagai ogg/opus. Pilih kontainer ogg/opus bila
  // didukung; mime asli ikut di data-URI agar engine tak salah-label.
  function pickAudioMime() {
    const cands = ["audio/ogg;codecs=opus", "audio/webm;codecs=opus", "audio/ogg", "audio/webm"];
    if (typeof MediaRecorder === "undefined") return "";
    for (const c of cands) if (MediaRecorder.isTypeSupported(c)) return c;
    return "";
  }
  let recording = false, mediaRec = null, chunks = [], recCancel = false;
  let recElapsed = 0, _recTimer = null;
  $: recLabel = `${String(Math.floor(recElapsed / 60)).padStart(2, "0")}:${String(recElapsed % 60).padStart(2, "0")}`;
  function stopRecTimer() { clearInterval(_recTimer); _recTimer = null; }
  function cancelRec() { if (!recording) return; recCancel = true; mediaRec && mediaRec.stop(); }
  // Ganti chat / unmount saat merekam → batalkan: hentikan timer + recorder
  // (onstop melepas track mic), supaya mic tak tetap hidup.
  onDestroy(() => { saveDraft(); if (recording) { recCancel = true; try { mediaRec && mediaRec.stop(); } catch (e) {} } stopRecTimer(); });
  async function handleMic() {
    if (value.trim()) { send(); return; }
    if (recording) { mediaRec && mediaRec.stop(); return; }    // tap lagi = stop & kirim
    if (typeof MediaRecorder === "undefined" || !navigator.mediaDevices?.getUserMedia) {
      pushToast($t("voice_unsupported")); return;
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const mime = pickAudioMime();
      mediaRec = mime ? new MediaRecorder(stream, { mimeType: mime }) : new MediaRecorder(stream);
      chunks = []; recCancel = false;
      mediaRec.ondataavailable = (e) => e.data.size && chunks.push(e.data);
      mediaRec.onstop = async () => {
        stream.getTracks().forEach((t) => t.stop());
        recording = false; stopRecTimer();
        if (recCancel) return;                                 // dibatalkan → tak kirim
        const blob = new Blob(chunks, { type: mediaRec.mimeType || mime || "audio/ogg" });
        if (blob.size < 800) return;                           // terlalu pendek
        const secs = Math.max(1, recElapsed);                  // durasi → AudioMessage.Seconds
        const dataURI = await new Promise((res) => { const r = new FileReader(); r.onload = () => res(r.result); r.readAsDataURL(blob); });
        await sendMediaMessage(chatId, "voice", "", "voice", dataURI, false, secs);
      };
      mediaRec.start();
      recording = true; recElapsed = 0;
      _recTimer = setInterval(() => (recElapsed += 1), 1000);
    } catch (e) {
      pushToast($t("mic_denied"));
    }
  }
</script>

<svelte:window on:paste={onPaste} />

{#if $editDraft}
  <div class="reply-bar">
    <div class="rb-body">
      <div class="rb-name">{$t("edit_message")}</div>
      <div class="rb-text">{$editDraft.text}</div>
    </div>
    <button class="icon-btn" aria-label={$t("cancel")} on:click={() => { editDraft.set(null); value = ""; }}>
      <svg viewBox="0 0 24 24"><path d="M6 6l12 12M18 6L6 18"/></svg>
    </button>
  </div>
{/if}

{#if $replyDraft}
  <div class="reply-bar">
    <div class="rb-body">
      <div class="rb-name">{$replyDraft.name}</div>
      <div class="rb-text">{$replyDraft.text}</div>
    </div>
    <button class="icon-btn" aria-label={$t("cancel")} on:click={() => replyDraft.set(null)}>
      <svg viewBox="0 0 24 24"><path d="M6 6l12 12M18 6L6 18"/></svg>
    </button>
  </div>
{/if}

{#if mOpen}
  <div class="mention-pop">
    {#each mItems as it}
      <button class="mention-item" on:click={() => pickMention(it)}>
        <span class="mi-av" class:special={it.special}>{it.name[0]}</span>
        <span class="mi-name">{it.name}</span>
      </button>
    {/each}
  </div>
{/if}

{#if scOpen}
  <div class="mention-pop sc-pop">
    {#each scItems as it}
      <button class="mention-item" on:click={() => pickShortcode(it)}>
        <span class="sc-emoji">{it.u}</span>
        <span class="mi-name">:{it.name}:</span>
      </button>
    {/each}
  </div>
{/if}

{#if emojiOpen}
  <button class="menu-backdrop" aria-label={$t("close")} on:click={() => (emojiOpen = false)}></button>
  <div class="emoji-panel">
    {#if emojiReady}
      <emoji-picker class={$theme === "dark" ? "dark" : "light"} on:emoji-click={onEmojiClick}></emoji-picker>
    {:else}
      <div class="emoji-none">…</div>
    {/if}
  </div>
{/if}

{#if attachOpen}
  <button class="menu-backdrop" aria-label={$t("close")} on:click={() => (attachOpen = false)}></button>
  <div class="attach-menu">
    <button class="am-item" on:click={attachFile}>
      <svg viewBox="0 0 24 24"><path d="M4 7h3l2-2h6l2 2h3v12H4z"/><circle cx="12" cy="13" r="3.5"/></svg>{$t("attach_media")}
    </button>
    <button class="am-item" on:click={attachLocation}>
      <svg viewBox="0 0 24 24"><path d="M12 21s7-6 7-11a7 7 0 0 0-14 0c0 5 7 11 7 11z"/><circle cx="12" cy="10" r="2.5"/></svg>{$t("attach_location")}
    </button>
    <button class="am-item" on:click={openPoll}>
      <svg viewBox="0 0 24 24"><path d="M5 5h14M5 12h9M5 19h5"/></svg>{$t("attach_poll")}
    </button>
    <button class="am-item" on:click={openGif}>
      <svg viewBox="0 0 24 24"><rect x="3" y="5" width="18" height="14" rx="2"/><path d="M8 9v6M11 9v6h2M16 9h-2v6M16 12h-1"/></svg>GIF
    </button>
    <button class="am-item" on:click={openSticker}>
      <svg viewBox="0 0 24 24"><path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h8l6-6V5a2 2 0 0 0-2-2z"/><path d="M14 21v-4a2 2 0 0 1 2-2h4"/></svg>{$t("attach_sticker")}
    </button>
    <button class="am-item" on:click={openContact}>
      <svg viewBox="0 0 24 24"><circle cx="12" cy="8" r="4"/><path d="M4 21c0-4 4-6 8-6s8 2 8 6"/></svg>{$t("attach_contact")}
    </button>
    <button class="am-item" on:click={openQuick}>
      <svg viewBox="0 0 24 24"><path d="M13 2L3 14h7l-1 8 10-12h-7z"/></svg>{$t("quick_replies")}
    </button>
    <button class="am-item" on:click={attachDocument}>
      <svg viewBox="0 0 24 24"><path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><path d="M14 3v6h6"/></svg>{$t("attach_document")}
    </button>
    <input type="file" multiple bind:this={docInput} on:change={onDoc} hidden />
  </div>
{/if}

{#if contactOpen}
  <div class="nc-modal" on:click|self={() => (contactOpen = false)}>
    <div class="nc-card" style="max-width:420px;max-height:70vh;display:flex;flex-direction:column">
      <h3 style="margin:0 0 12px">{$t("attach_contact")}</h3>
      <input bind:value={contactQ} placeholder={$t("search")}
        style="width:100%;border:1px solid var(--line);border-radius:12px;padding:10px 12px;background:var(--bg2);color:var(--text);font:inherit;margin-bottom:10px" />
      <div style="overflow-y:auto;flex:1">
        {#each contactList as c (c.id)}
          <button class="am-item" style="width:100%" on:click={() => pickContact(c)}>
            <span style="width:34px;height:34px;border-radius:50%;background:{c.color};color:#fff;display:grid;align-items:center;justify-items:center;font-weight:600">{(c.name||"?")[0]}</span>
            {c.name}
          </button>
        {/each}
      </div>
      <div style="display:flex;justify-content:flex-end;margin-top:12px">
        <button class="btn-ghost" on:click={() => (contactOpen = false)}>{$t("cancel")}</button>
      </div>
    </div>
  </div>
{/if}

{#if gifOpen}
  <button class="menu-backdrop" aria-label={$t("close")} on:click={() => (gifOpen = false)}></button>
  <GifPicker on:pick={onGifPick} />
{/if}

{#if stickerOpen}
  <button class="menu-backdrop" aria-label={$t("close")} on:click={() => (stickerOpen = false)}></button>
  <StickerPicker on:pick={onStickerPick} />
{/if}

{#if qrOpen}
  <button class="menu-backdrop" aria-label={$t("close")} on:click={() => (qrOpen = false)}></button>
  <div class="qr-panel">
    <div class="qr-head">{$t("quick_replies")}<button class="qr-add" on:click={addQuick} disabled={!value.trim()}>+ {$t("save")}</button></div>
    <div class="qr-list">
      {#each quickReplies as qr}
        <div class="qr-item"><button class="qr-text" on:click={() => insertQuick(qr)}>{qr}</button><button class="qr-del" on:click={() => delQuick(qr)}>✕</button></div>
      {/each}
      {#if quickReplies.length === 0}<div class="qr-empty">{$t("quick_replies_empty")}</div>{/if}
    </div>
  </div>
{/if}

{#if pollOpen}
  <div class="nc-modal" on:click|self={() => (pollOpen = false)}>
    <div class="nc-card poll-card">
      <h3 class="poll-title">{$t("poll_new")}</h3>
      <label class="poll-lbl">{$t("poll_question")}</label>
      <input class="poll-in" bind:value={pollQ} placeholder={$t("poll_question")} />
      <label class="poll-lbl">{$t("poll_option")}</label>
      <div class="poll-opts">
        {#each pollOpts as _, i}
          <div class="poll-opt-row">
            <input class="poll-in" bind:value={pollOpts[i]} placeholder={`${$t("poll_option")} ${i + 1}`} />
            {#if pollOpts.length > 2}
              <button class="poll-rm" title={$t("delete")} on:click={() => removePollOpt(i)}>✕</button>
            {/if}
          </div>
        {/each}
      </div>
      {#if pollOpts.length < 12}
        <button class="poll-add" on:click={addPollOpt}>
          <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>{$t("poll_add_option")}
        </button>
      {/if}
      <label class="poll-multi">
        <input type="checkbox" bind:checked={pollMulti} />
        <span>{$t("poll_multi")}</span>
      </label>
      <div class="poll-foot">
        <button class="btn-ghost" on:click={() => (pollOpen = false)}>{$t("cancel")}</button>
        <button class="btn-accent" on:click={submitPoll} disabled={!pollQ.trim() || pollOpts.filter((o) => o.trim()).length < 2}>{$t("send")}</button>
      </div>
    </div>
  </div>
{/if}

<footer class="composer {dragOver ? 'dragover' : ''}"
  on:dragover|preventDefault={() => (dragOver = true)}
  on:dragleave={() => (dragOver = false)}
  on:drop={onDrop}>
  <button class="icon-btn" aria-label={$t("emoji")} on:click={toggleEmoji}>
    <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><circle cx="9" cy="10" r="1"/><circle cx="15" cy="10" r="1"/><path d="M8.5 14.5a4 4 0 0 0 7 0"/></svg>
  </button>
  <button class="icon-btn" aria-label={$t("attach")} on:click={toggleAttach}>
    <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
  </button>
  <input type="file" multiple bind:this={fileInput} on:change={onFile} hidden
    accept="image/*,video/*,application/pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.zip" />

  {#if recording}
    <div class="rec-bar">
      <span class="rec-dot"></span>
      <span class="rec-time">{recLabel}</span>
      <span class="rec-hint">{$t("recording")}</span>
      <button class="rec-cancel" on:click={cancelRec} aria-label={$t("cancel")}>
        <svg viewBox="0 0 24 24"><path d="M4 7h16M9 7V5h6v2M6 7l1 13h10l1-13"/></svg>
      </button>
    </div>
  {:else}
    <div class="input">
      <textarea rows="1" placeholder={$t("composer_placeholder")} aria-label={$t("composer_placeholder")}
        bind:this={inputEl} bind:value on:keydown={onKey} on:input={(e) => { detectMention(); detectShortcode(); autoGrow(e.target); saveDraft(); }} on:click={detectMention}></textarea>
    </div>
  {/if}
  {#if typing && !$editDraft}
    <button class="icon-btn" title={$t("schedule_msg")} on:click={openSchedule}>
      <svg viewBox="0 0 24 24"><circle cx="12" cy="13" r="8"/><path d="M12 9v4l3 2M9 2h6"/></svg>
    </button>
    <input type="datetime-local" bind:this={schedInput} on:change={onSchedule} style="position:absolute;width:0;height:0;opacity:0;pointer-events:none" />
  {/if}
  <button class="icon-btn mic {recording ? 'rec' : ''}" aria-label={typing ? $t("send") : recording ? $t("send") : $t("voice_msg")} on:click={handleMic}>
    {#if typing}
      <svg viewBox="0 0 24 24"><path d="M3 11l18-8-8 18-2-7-8-3z"/></svg>
    {:else if recording}
      <svg viewBox="0 0 24 24"><rect x="6" y="6" width="12" height="12" rx="2"/></svg>
    {:else}
      <svg viewBox="0 0 24 24"><rect x="9" y="3" width="6" height="11" rx="3"/><path d="M5 11a7 7 0 0 0 14 0M12 18v3"/></svg>
    {/if}
  </button>
</footer>
