<script>
  import { tick } from "svelte";
  import { get } from "svelte/store";
  import { sendMessage, sendMediaMessage, replyDraft, pushToast, editDraft, editMessage } from "../../stores.js";
  import { getGroupInfo, sendLocation, sendPoll } from "../../services/data.js";
  import { t } from "../i18n.js";
  export let chatId;
  export let group = false;

  let value = "";
  // Mode sunting: isi composer dgn teks pesan yg disunting.
  let lastEdit = null;
  $: if ($editDraft && $editDraft.id !== lastEdit) { lastEdit = $editDraft.id; value = $editDraft.text; }
  $: if (!$editDraft) lastEdit = null;
  let inputEl;
  $: typing = value.trim().length > 0;
  let emojiOpen = new URLSearchParams(location.search).get("emoji") === "1";
  let fileInput;

  // --- menu lampiran (file / lokasi / polling) ---
  let attachOpen = false;
  function toggleAttach() { attachOpen = !attachOpen; }
  function attachFile() { attachOpen = false; pickFile(); }
  function attachLocation() {
    attachOpen = false;
    if (!navigator.geolocation) { pushToast($t("loc_unavailable")); return; }
    navigator.geolocation.getCurrentPosition(
      (pos) => sendLocation(chatId, pos.coords.latitude, pos.coords.longitude, ""),
      () => pushToast($t("loc_denied")),
      { enableHighAccuracy: true, timeout: 10000 }
    );
  }
  // --- modal polling ---
  let pollOpen = false, pollQ = "", pollOpts = ["", ""];
  function openPoll() { attachOpen = false; pollOpen = true; pollQ = ""; pollOpts = ["", ""]; }
  function addPollOpt() { if (pollOpts.length < 12) pollOpts = [...pollOpts, ""]; }
  function submitPoll() {
    const opts = pollOpts.map((o) => o.trim()).filter(Boolean);
    if (!pollQ.trim() || opts.length < 2) return;
    sendPoll(chatId, pollQ.trim(), opts, 1);
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
    if (type.startsWith("audio/")) return "voice";
    return "document";
  }
  async function onFile(e) {
    const file = e.target.files && e.target.files[0];
    if (!file) return;
    const dataURI = await new Promise((res) => {
      const r = new FileReader();
      r.onload = () => res(r.result);
      r.readAsDataURL(file);
    });
    await sendMediaMessage(chatId, kindOf(file.type), "", file.name, dataURI);
    e.target.value = "";
  }

  const EMOJIS = "😀 😂 🥰 😍 😎 🤔 😅 😭 😡 👍 👎 🙏 👏 🙌 💪 🔥 ✨ 🎉 ❤️ 💔 😴 🤝 👀 🤣 😊 😘 😢 🤯 🥳 😱 💯 ✅".split(" ");

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
    replyDraft.set(null);
  }
  function onKey(e) {
    if (mOpen && e.key === "Enter") { e.preventDefault(); pickMention(mItems[0]); return; }
    if (mOpen && e.key === "Escape") { mOpen = false; return; }
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); send(); }
  }
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
  let recording = false, mediaRec = null, chunks = [];
  async function handleMic() {
    if (value.trim()) { send(); return; }
    if (recording) { mediaRec && mediaRec.stop(); return; }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const mime = pickAudioMime();
      mediaRec = mime ? new MediaRecorder(stream, { mimeType: mime }) : new MediaRecorder(stream);
      chunks = [];
      mediaRec.ondataavailable = (e) => e.data.size && chunks.push(e.data);
      mediaRec.onstop = async () => {
        stream.getTracks().forEach((t) => t.stop());
        recording = false;
        const blob = new Blob(chunks, { type: mediaRec.mimeType || mime || "audio/ogg" });
        if (blob.size < 800) return; // terlalu pendek
        const dataURI = await new Promise((res) => { const r = new FileReader(); r.onload = () => res(r.result); r.readAsDataURL(blob); });
        await sendMediaMessage(chatId, "voice", "", "voice", dataURI);
      };
      mediaRec.start();
      recording = true;
    } catch (e) {
      pushToast($t("mic_denied"));
    }
  }
</script>

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

{#if emojiOpen}
  <div class="emoji-panel">
    {#each EMOJIS as e}<button class="emoji" on:click={() => addEmoji(e)}>{e}</button>{/each}
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
  </div>
{/if}

{#if pollOpen}
  <div class="nc-modal" on:click|self={() => (pollOpen = false)}>
    <div class="nc-card" style="max-width:420px">
      <h3 style="margin:0 0 12px">{$t("poll_new")}</h3>
      <input bind:value={pollQ} placeholder={$t("poll_question")}
        style="width:100%;border:1px solid var(--line);border-radius:12px;padding:11px 12px;background:var(--bg2);color:var(--text);font:inherit;margin-bottom:10px" />
      {#each pollOpts as _, i}
        <input bind:value={pollOpts[i]} placeholder={`${$t("poll_option")} ${i + 1}`}
          style="width:100%;border:1px solid var(--line);border-radius:12px;padding:10px 12px;background:var(--bg2);color:var(--text);font:inherit;margin-bottom:8px" />
      {/each}
      <button class="btn-ghost" on:click={addPollOpt}>+ {$t("poll_add_option")}</button>
      <div style="display:flex;gap:10px;justify-content:flex-end;margin-top:14px">
        <button class="btn-ghost" on:click={() => (pollOpen = false)}>{$t("cancel")}</button>
        <button class="btn-accent" on:click={submitPoll} disabled={!pollQ.trim() || pollOpts.filter((o) => o.trim()).length < 2}>{$t("send")}</button>
      </div>
    </div>
  </div>
{/if}

<footer class="composer">
  <button class="icon-btn" aria-label={$t("emoji")} on:click={() => (emojiOpen = !emojiOpen)}>
    <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><circle cx="9" cy="10" r="1"/><circle cx="15" cy="10" r="1"/><path d="M8.5 14.5a4 4 0 0 0 7 0"/></svg>
  </button>
  <button class="icon-btn" aria-label={$t("attach")} on:click={toggleAttach}>
    <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
  </button>
  <input type="file" bind:this={fileInput} on:change={onFile} hidden
    accept="image/*,video/*,application/pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.zip" />

  <div class="input">
    <input type="text" placeholder={$t("composer_placeholder")} aria-label={$t("composer_placeholder")}
      bind:this={inputEl} bind:value on:keydown={onKey} on:input={detectMention} on:click={detectMention} />
  </div>
  <button class="icon-btn mic {recording ? 'rec' : ''}" aria-label={typing ? $t("send") : $t("voice_msg")} on:click={handleMic}>
    {#if typing}
      <svg viewBox="0 0 24 24"><path d="M3 11l18-8-8 18-2-7-8-3z"/></svg>
    {:else if recording}
      <svg viewBox="0 0 24 24"><rect x="6" y="6" width="12" height="12" rx="2"/></svg>
    {:else}
      <svg viewBox="0 0 24 24"><rect x="9" y="3" width="6" height="11" rx="3"/><path d="M5 11a7 7 0 0 0 14 0M12 18v3"/></svg>
    {/if}
  </button>
</footer>
