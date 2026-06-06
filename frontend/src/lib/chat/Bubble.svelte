<script>
  import { onMount, onDestroy } from "svelte";
  import Ticks from "../common/Ticks.svelte";
  import Avatar from "../common/Avatar.svelte";
  import { t } from "../i18n.js";
  import { translateMessage } from "../../services/translate.js";
  import { LIVE, senderColorFor, avatarUrl, getLinkPreview, votePoll, getPollVotes, onEvent } from "../../services/data.js";
  import { reactMessage, deleteMessage, starMessage, replyDraft, forwardDraft, activeChatId, chats, translateLang, editDraft, pushToast, pinMessageAction, showMessageInfo, lightbox, selectMode, selectedIdx, enterSelect, toggleSelect, jumpMsg, reactionTarget, openProfile } from "../../stores.js";

  export let msg;
  export let group = false;
  export let firstOfRun = true;
  export let chatId;
  export let idx;
  export let peerName = "";

  $: stickerBubble = msg.type === "sticker" || msg.type === "gif";
  $: isMedia = msg.type === "image" || msg.type === "video" || msg.type === "sticker" || msg.type === "gif";
  // Stiker & GIF → bubble TRANSPARAN (tanpa kartu putih, hanya nama yg ber-pill).
  // Foto/video → KARTU (bg + padding tipis), rasio natural (min/max), caption di bawah.
  $: bubbleClass = msg.type === "sticker"
    ? "media sticker-bubble"
    : msg.type === "gif"
      ? "media sticker-bubble gif-bubble"
      : (msg.type === "image" || msg.type === "video")
        ? "media imgcard"
        : msg.type === "voice" ? "voice"
          : (msg.type === "text" || msg.type === "deleted") ? "txt" : "";
  $: isGroupIn = group && msg.dir === "in";
  $: showSender = isGroupIn && msg.sender && firstOfRun;
  $: senderCol = msg.senderColor || senderColorFor(msg.senderId || msg.sender || "");
  $: caption = msg.caption || (msg.type !== "sticker" ? msg.text : "") || "";

  // Pecah teks → mention (@<nomor>→@Nama) + format WhatsApp (*tebal* _miring_
  // ~coret~ `mono` ```blok```).
  $: textParts = renderParts(msg.text, msg.mentions);
  // Tokenisasi format pada satu potongan teks polos.
  function fmt(s) {
    const out = [];
    // URL (group 1) dulu → tautan biru klik-able; lalu format WhatsApp.
    const RE = /(https?:\/\/[^\s]+|www\.[^\s]+)|\|\|([\s\S]+?)\|\||```([\s\S]+?)```|`([^`\n]+?)`|\*([^*\n]+?)\*|_([^_\n]+?)_|~([^~\n]+?)~/;
    while (s) {
      const m = s.match(RE);
      if (!m) { out.push({ t: s }); break; }
      if (m.index > 0) out.push({ t: s.slice(0, m.index) });
      if (m[1] != null) {
        // buang tanda baca akhir yg ikut ke-match (mis. "url).")
        let url = m[1], tail = "";
        const tm = url.match(/[.,!?;:)\]]+$/);
        if (tm) { tail = tm[0]; url = url.slice(0, -tail.length); }
        out.push({ t: url, link: true });
        if (tail) out.push({ t: tail });
      } else if (m[2] != null) out.push({ t: m[2], sp: true });
      else if (m[3] != null || m[4] != null) out.push({ t: m[3] ?? m[4], code: true });
      else if (m[5] != null) out.push({ t: m[5], b: true });
      else if (m[6] != null) out.push({ t: m[6], i: true });
      else if (m[7] != null) out.push({ t: m[7], s: true });
      s = s.slice(m.index + m[0].length);
    }
    return out;
  }
  // Buka tautan di browser sistem (Wails) — bukan di dalam webview app.
  function openURL(u) {
    const href = u.startsWith("http") ? u : "https://" + u;
    if (typeof window !== "undefined" && window.runtime && window.runtime.BrowserOpenURL) window.runtime.BrowserOpenURL(href);
    else window.open(href, "_blank", "noreferrer");
  }
  let revealed = {};
  function renderParts(text, mentions) {
    if (!text) return [];
    const map = {};
    (mentions || []).forEach((m) => (map[m.num] = m));
    const re = /@(\d{5,})/g;
    const out = [];
    let last = 0, mt;
    while ((mt = re.exec(text)) !== null) {
      if (mt.index > last) out.push(...fmt(text.slice(last, mt.index)));
      const info = map[mt[1]];
      if (info) out.push({ m: true, name: info.name, jid: info.jid });
      else out.push(...fmt(mt[0]));
      last = re.lastIndex;
    }
    if (last < text.length) out.push(...fmt(text.slice(last)));
    return out;
  }
  // Klik mention → buka panel profil kontak (foto/nomor/about + Pesan/Simpan).
  function openMention(jid) {
    if (jid) openProfile(jid);
  }
  // Hostname aman utk link-preview (URL rusak/kosong tak boleh throw saat render).
  function hostOf(u) { try { return new URL(u).hostname; } catch (e) { return u || ""; } }
  $: source = msg.text || msg.caption || "";
  // "Baca selengkapnya": klem pesan panjang ke N baris (CSS line-clamp), tombol
  // toggle. everLong dijaga agar tombol "lebih sedikit" tetap muncul saat dibuka.
  let expanded = false, overflowing = false, everLong = false;
  $: if (overflowing) everLong = true;
  function clampCheck(node) {
    const check = () => { if (!expanded) overflowing = node.scrollHeight - node.clientHeight > 6; };
    requestAnimationFrame(check);
    return { update: check };
  }
  $: canTranslate = msg.type === "text" || ((msg.type === "image" || msg.type === "video") && caption);

  // URL media disajikan asset-server (cache FILE, bukan data-URI memori). Native
  // → /media/<chat>/<id> (browser auto-load + cache); preview/browser → thumb.
  $: mediaUrl = LIVE && msg.id
    ? "/media/" + encodeURIComponent(chatId) + "/" + encodeURIComponent(msg.id)
    : (msg.thumb || "");
  let mediaErr = false;
  let imgDead = false; // /media + thumb dua-duanya gagal → tampilkan placeholder
  let videoPlaying = false;
  // Sumber gambar: /media (kualitas penuh) → bila gagal (mis. media terkirim
  // belum punya proto tersimpan) jatuh ke thumb data-URI. Cegah gambar kosong.
  $: imgSrc = mediaErr ? (msg.thumb || "") : (mediaUrl || msg.thumb || "");

  let translated = null;
  let busy = false;
  let playing = false;
  let audioEl = null;
  let vProgress = 0; // 0..1
  let vRate = 1;
  function ensureAudio() {
    if (audioEl || !mediaUrl) return;
    audioEl = new Audio(mediaUrl);
    audioEl.playbackRate = vRate;
    audioEl.addEventListener("ended", () => { playing = false; vProgress = 0; });
    audioEl.addEventListener("timeupdate", () => {
      if (audioEl.duration) vProgress = audioEl.currentTime / audioEl.duration;
    });
  }
  function playVoice() {
    if (!mediaUrl) return;
    ensureAudio();
    if (playing) { audioEl.pause(); playing = false; }
    else audioEl.play().then(() => (playing = true)).catch(() => {});
  }
  function seekVoice(e) {
    ensureAudio();
    if (!audioEl || !audioEl.duration) return;
    const r = e.currentTarget.getBoundingClientRect();
    audioEl.currentTime = ((e.clientX - r.left) / r.width) * audioEl.duration;
  }
  function cycleRate() {
    vRate = vRate >= 2 ? 1 : vRate + 0.5;
    if (audioEl) audioEl.playbackRate = vRate;
  }
  async function doTranslate() {
    if (busy) return;
    busy = true;
    translated = await translateMessage(source, $translateLang);
    busy = false;
  }
  // Pratinjau tautan: ambil URL pertama di teks → fetch OG (sisi Go).
  let linkPrev = null;
  $: firstUrl = (msg.type === "text" && msg.text) ? (msg.text.match(/https?:\/\/[^\s]+/) || [])[0] : null;
  let lpDone = null;
  $: if (LIVE && firstUrl && firstUrl !== lpDone) { lpDone = firstUrl; getLinkPreview(firstUrl).then((p) => (linkPrev = p)); }

  let offPoll = null;
  onMount(() => {
    const p = new URLSearchParams(location.search);
    if (p.get("tr") === "1" && canTranslate && msg.dir === "in") doTranslate();
    if (p.get("menu") !== null && Number(p.get("menu")) === idx) menuOpen = true; // pratinjau
    if (msg.type === "poll") {
      loadPollVotes();
      offPoll = onEvent("wa:poll", (id) => { if (id === msg.id) loadPollVotes(); });
    }
  });
  // Bersihkan saat bubble dihancurkan (mis. ganti chat / prune): lepas listener
  // poll (cegah tumpuk) + hentikan audio voice yg sedang main.
  onDestroy(() => { offPoll && offPoll(); if (audioEl) { audioEl.pause(); audioEl = null; } });

  // --- context menu & aksi ---
  let menuOpen = false;
  let menuUp = false; // buka ke ATAS bila pesan dekat bawah viewport
  function toggleMenu(e) {
    if (!menuOpen) {
      const r = e.currentTarget.getBoundingClientRect();
      menuUp = r.bottom > window.innerHeight * 0.55;
    }
    menuOpen = !menuOpen;
  }
  const QUICK = ["❤️", "😂", "👍", "😮", "😢", "🙏"];
  function react(e) { reactMessage(chatId, idx, e); menuOpen = false; }
  // Buka emoji-picker PENUH, di-anchor ke tombol (kanan) — bukan grid kecil.
  function openReact(e) {
    const r = e.currentTarget.getBoundingClientRect();
    reactionTarget.set({ chatId, idx, x: r.right, y: r.top });
    menuOpen = false;
  }
  function reply() {
    const name = msg.sender || (msg.dir === "out" ? $t("you") : peerName);
    replyDraft.set({ name, text: source || "📎", id: msg.id, senderId: msg.senderId });
    menuOpen = false;
  }
  function del() { deleteMessage(chatId, idx, true); menuOpen = false; } // hapus utk semua → placeholder
  function undoReact(emoji) { reactMessage(chatId, idx, emoji); } // klik reaksi sendiri → lepas (toggle)
  function star() { starMessage(chatId, idx, !msg.starred); menuOpen = false; }
  function forward() { forwardDraft.set({ chat: chatId, idx }); menuOpen = false; }
  function copyText() { if (source) navigator.clipboard?.writeText(source).then(() => pushToast($t("copied"), "ok")); menuOpen = false; }
  function editMsg() { editDraft.set({ chatId, id: msg.id, text: source }); menuOpen = false; }
  function replyPrivate() { activeChatId.set(msg.senderId); replyDraft.set({ name: msg.sender, text: source, id: msg.id, senderId: msg.senderId }); menuOpen = false; }
  function menuTranslate() {
    if (translated) translated = null;
    else doTranslate();
    menuOpen = false;
  }
  // Lokasi: thumb = "lat,lng".
  $: mapUrl = msg.type === "location" && msg.thumb
    ? `https://staticmap.openstreetmap.de/staticmap.php?center=${msg.thumb}&zoom=15&size=300x150&markers=${msg.thumb},red-pushpin`
    : "";
  function openMap() {
    if (!msg.thumb) return;
    const [lat, lng] = msg.thumb.split(",");
    window.open(`https://www.openstreetmap.org/?mlat=${lat}&mlon=${lng}#map=16/${lat}/${lng}`, "_blank");
  }
  // Polling: thumb = JSON array opsi.
  $: pollOptions = msg.type === "poll" && msg.thumb ? (() => { try { return JSON.parse(msg.thumb); } catch (e) { return []; } })() : [];
  let pollVoted = null;
  let pollCounts = {};
  let pollTotal = 0;
  async function loadPollVotes() {
    if (msg.type !== "poll" || !LIVE) return;
    const r = await getPollVotes(msg.id);
    pollCounts = r.counts || {};
    pollTotal = r.total || 0;
  }
  function vote(opt) {
    if (pollVoted) return;
    pollVoted = opt;
    votePoll(chatId, msg.senderId || "", msg.id, [opt]);
    setTimeout(loadPollVotes, 400);
  }
  function openMedia() {
    if (msg.type === "image") { if (imgSrc) lightbox.set({ url: imgSrc, type: "image", caption }); }
    else if (msg.type === "video") videoPlaying = true; // putar inline
    // stiker: tak ada aksi
  }
  function pin() { pinMessageAction(chatId, idx, !msg.pinned); menuOpen = false; }
  function info() { showMessageInfo(chatId, idx); menuOpen = false; }
  function selectThis() { enterSelect(idx); menuOpen = false; }
  function onRowClick() { if ($selectMode) toggleSelect(idx); }
  $: isSelected = $selectMode && $selectedIdx.includes(idx);
</script>

<div class="msg {msg.dir} {isGroupIn ? 'gin' : ''} {firstOfRun ? '' : 'cont'} {msg.reactions ? 'has-react' : ''} {$selectMode ? 'selmode' : ''} {isSelected ? 'sel' : ''}" data-mid={msg.id} data-ts={msg.ts}
  on:click={onRowClick} role={$selectMode ? "button" : undefined} tabindex={$selectMode ? 0 : undefined}>
  {#if $selectMode}
    <span class="sel-check {isSelected ? 'on' : ''}">{isSelected ? "✓" : ""}</span>
  {/if}
  {#if isGroupIn}
    <div class="msg-avatar">
      {#if firstOfRun && msg.sender}<Avatar name={msg.sender} color={senderCol} photo={avatarUrl(msg.senderId)} tiny={true} />{/if}
    </div>
  {/if}

  <div class="bubble {bubbleClass} {msg.type === 'deleted' ? 'deleted' : ''}"
    class:withtr={!!translated}
    class:hascap={(msg.type === 'image' || msg.type === 'video') && caption}
    class:nohead={(msg.type === 'image' || msg.type === 'video') && !(showSender || msg.forwarded || msg.quote)}>
    {#if msg.type !== "deleted" && !$selectMode}
      <div class="msg-actions">
        <button title="👍" on:click={() => react('👍')}>👍</button>
        <button title={$t("reaction_remove")} on:click={openReact}>
          <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><circle cx="9" cy="10" r="1.2"/><circle cx="15" cy="10" r="1.2"/><path d="M8.5 14.5a4 4 0 0 0 7 0"/></svg>
        </button>
        <button title={$t("reply")} on:click={reply}>
          <svg viewBox="0 0 24 24"><path d="M10 9V5l-7 7 7 7v-4c5 0 8 1 10 4-1-6-4-9-10-10z"/></svg>
        </button>
        <button class="ma-more" title={$t("menu")} on:click={toggleMenu}>
          <svg viewBox="0 0 24 24"><circle cx="12" cy="5" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="12" cy="19" r="1.6"/></svg>
        </button>
      </div>
    {/if}

    {#if showSender || (msg.type === "deleted") || msg.forwarded || msg.quote}
      <div class="head" class:sticker-head={stickerBubble}>
        {#if showSender}
          <span class="sender" style="color:{senderCol}" role="button" tabindex="0"
            on:click={() => msg.senderId && openProfile(msg.senderId)}
            on:keydown={(e) => e.key === "Enter" && msg.senderId && openProfile(msg.senderId)}>
            {msg.sender}{#if !msg.senderSaved && msg.senderPhone}<span class="sender-phone">{msg.senderPhone}</span>{/if}
          </span>
        {/if}
        {#if msg.type === "deleted"}
          <span class="text deleted-text">
            <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M5.6 5.6l12.8 12.8"/></svg>
            {msg.dir === "out" ? $t("deleted_out") : $t("deleted_in")}<span class="t-spacer" class:out={msg.dir === 'out'} aria-hidden="true">{msg.time}</span>
          </span>
        {/if}
        {#if msg.forwarded}
          <div class="forwarded"><svg viewBox="0 0 24 24"><path d="M10 9V5l8 7-8 7v-4c-5 0-8 2-9 5 0-6 3-9 9-9z"/></svg>{$t("forwarded")}</div>
        {/if}
        {#if msg.quote}
          <div class="quote" class:jumpable={msg.quote.id} role={msg.quote.id ? "button" : undefined} tabindex={msg.quote.id ? 0 : undefined}
            on:click={() => msg.quote.id && jumpMsg.set(msg.quote.id)} on:keydown={(e) => e.key === "Enter" && msg.quote.id && jumpMsg.set(msg.quote.id)}>
            <div class="quote-name">{msg.quote.name}</div><div class="quote-text">{msg.quote.text}</div>
          </div>
        {/if}
      </div>
    {/if}

    {#if msg.type === "text"}
      {#if linkPrev}
        <a class="link-prev" href={linkPrev.url} target="_blank" rel="noreferrer">
          {#if linkPrev.image}<img class="lp-img" src={linkPrev.image} alt="" on:error={(e) => (e.target.style.display = 'none')} />{/if}
          <span class="lp-body">
            {#if linkPrev.title}<span class="lp-title">{linkPrev.title}</span>{/if}
            {#if linkPrev.desc}<span class="lp-desc">{linkPrev.desc}</span>{/if}
            <span class="lp-host">{hostOf(linkPrev.url)}</span>
          </span>
        </a>
      {/if}
      <span class="text" dir="auto" class:clamp={!expanded} use:clampCheck>{#each textParts as p, i}{#if p.m}<span class="mention" role="button" tabindex="0" on:click|stopPropagation={() => openMention(p.jid)} on:keydown={(e) => e.key === "Enter" && openMention(p.jid)}>@{p.name}</span>{:else if p.sp}<span class="spoiler {revealed[i] ? 'on' : ''}" role="button" tabindex="0" on:click|stopPropagation={() => (revealed[i] = true)} on:keydown={(e) => e.key === "Enter" && (revealed[i] = true)}>{p.t}</span>{:else if p.code}<code class="md-code">{p.t}</code>{:else if p.b}<strong>{p.t}</strong>{:else if p.i}<em>{p.t}</em>{:else if p.s}<s>{p.t}</s>{:else if p.link}<a class="msg-link" href={p.t} on:click|stopPropagation|preventDefault={() => openURL(p.t)}>{p.t}</a>{:else}{p.t}{/if}{/each}{#if msg.edited}<span class="edited-tag">{$t("edited_tag")}</span>{/if}<span class="t-spacer" class:out={msg.dir === 'out'} aria-hidden="true">{msg.time}</span></span>{#if everLong}<button class="read-more" on:click|stopPropagation={() => (expanded = !expanded)}>{expanded ? $t("read_less") : $t("read_more")}</button>{/if}
    {:else if isMedia}
      <div class="media-box {msg.type === 'sticker' ? 'sticker' : 'card'}"
        role="button" tabindex="0" on:click={openMedia}
        on:keydown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), openMedia())}>
        {#if msg.type === "gif"}
          <video class="media-img" src={mediaUrl} autoplay loop muted playsinline on:error={() => { if (!mediaErr) mediaErr = true; }}></video>
        {:else if msg.type === "video" && videoPlaying}
          <video class="media-img" src={mediaUrl} controls autoplay></video>
        {:else if imgSrc && !imgDead}
          <img class="media-img" src={imgSrc} alt="" loading="lazy" on:error={() => { if (!mediaErr) mediaErr = true; else imgDead = true; }} />
        {:else}
          <div class="img-ph">
            <span class="ph-dl"><svg viewBox="0 0 24 24"><path d="M12 4v11M7 11l5 5 5-5M5 20h14"/></svg></span>
            <span class="ph-lbl">{msg.type === "video" ? $t("t_video") : msg.type === "sticker" ? $t("t_sticker") : $t("t_photo")}</span>
          </div>
        {/if}
        {#if msg.type === "video" && !videoPlaying}<span class="play-badge"><svg viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg></span>{/if}
      </div>
      {#if caption}<span class="text caption" dir="auto">{caption}</span>{/if}
    {:else if msg.type === "voice"}
      <button class="play" aria-label="Play" on:click={playVoice}>
        {#if playing}
          <svg viewBox="0 0 24 24"><rect x="6" y="5" width="4" height="14" rx="1"/><rect x="14" y="5" width="4" height="14" rx="1"/></svg>
        {:else}
          <svg viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg>
        {/if}
      </button>
      <div class="wave" role="slider" aria-label="seek" tabindex="0" on:click={seekVoice} style="--vp:{vProgress}">
        {#each Array(18) as _, i}<span class:on={i / 18 <= vProgress}></span>{/each}
      </div>
      <span class="vtime">{msg.duration || msg.text || ""}</span>
      {#if playing || vProgress > 0}<button class="vrate" on:click={cycleRate}>{vRate}×</button>{/if}
    {:else if msg.type === "location"}
      <button class="loc-card" on:click={openMap}>
        <img class="loc-map" src={mapUrl} alt="" on:error={(e) => (e.target.style.display = 'none')} />
        <span class="loc-lbl"><svg viewBox="0 0 24 24"><path d="M12 21s7-6 7-11a7 7 0 0 0-14 0c0 5 7 11 7 11z"/><circle cx="12" cy="10" r="2.5"/></svg>{msg.text || "📍 Lokasi"}</span>
      </button>
    {:else if msg.type === "document"}
      <a class="doc-card" href={mediaUrl} download={msg.text || "dokumen"}>
        <span class="doc-ico"><svg viewBox="0 0 24 24"><path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5"/></svg></span>
        <span class="doc-name">{msg.text || "Dokumen"}</span>
        <span class="doc-dl"><svg viewBox="0 0 24 24"><path d="M12 4v11M7 11l5 5 5-5M5 20h14"/></svg></span>
      </a>
    {:else if msg.type === "poll"}
      <div class="poll-card">
        <div class="poll-q"><svg viewBox="0 0 24 24"><path d="M5 5h14M5 12h9M5 19h5"/></svg>{msg.text}</div>
        {#each pollOptions as opt}
          <button class="poll-opt {pollVoted === opt ? 'voted' : ''}" disabled={!!pollVoted} on:click={() => vote(opt)}>
            <span class="poll-bar" style="width:{pollTotal ? Math.round((pollCounts[opt] || 0) / pollTotal * 100) : 0}%"></span>
            <span class="poll-radio">{pollVoted === opt ? "●" : ""}</span>
            <span class="poll-opt-txt">{opt}</span>
            <span class="poll-cnt">{pollCounts[opt] || 0}</span>
          </button>
        {/each}
        <div class="poll-note">{pollVoted ? $t("poll_voted") + " · " : ""}{pollTotal} {$t("poll_votes_n")}</div>
      </div>
    {:else if msg.type === "contact"}
      <div class="ctc-card">
        <span class="ctc-av">{(msg.text || "?").replace(/^👤\s*/, "")[0] || "?"}</span>
        <span class="ctc-info">
          <span class="ctc-name">{(msg.text || "").replace(/^👤\s*/, "")}</span>
          {#if msg.thumb}<span class="ctc-num">{msg.thumb}</span>{/if}
        </span>
        {#if msg.thumb}<button class="ctc-btn" on:click={() => navigator.clipboard?.writeText(msg.thumb).then(() => pushToast($t("copied"), "ok"))}>{$t("copy")}</button>{/if}
      </div>
    {/if}

    {#if translated}
      <div class="tr-block" dir="auto"><span class="tr-lbl">{$t("translated")}</span>{translated}</div>
    {/if}

    <span class="meta">
      <span class="time">{msg.time}</span>
      {#if msg.dir === "out"}<Ticks status={msg.status || "sent"} />{/if}
    </span>

    {#if msg.reactions && msg.reactions.length}
      <div class="reactions">{#each msg.reactions as r}<button class="reaction" class:mine={r.mine} on:click={() => r.mine && undoReact(r.emoji)} title={r.mine ? $t("reaction_remove") : ""}>{r.emoji}{#if r.count > 1} {r.count}{/if}</button>{/each}</div>
    {/if}

    {#if menuOpen}
      <div class="msg-menu {menuUp ? 'up' : ''}">
        <div class="react-row">
          {#each QUICK as e}<button class="rx" on:click={() => react(e)}>{e}</button>{/each}
          <button class="rx rx-more" on:click|stopPropagation={openReact} aria-label={$t("emoji")}>+</button>
        </div>
        <button class="mi" on:click={reply}>{$t("reply")}</button>
        {#if isGroupIn && msg.senderId}<button class="mi" on:click={replyPrivate}>{$t("reply_private")}</button>{/if}
        {#if source}<button class="mi" on:click={copyText}>{$t("copy")}</button>{/if}
        {#if msg.dir === "out" && msg.type === "text"}<button class="mi" on:click={editMsg}>{$t("edit")}</button>{/if}
        {#if canTranslate}<button class="mi" on:click={menuTranslate}>{translated ? $t("show_original") : $t("translate")}</button>{/if}
        <button class="mi" on:click={forward}>{$t("forward_action")}</button>
        <button class="mi" on:click={star}>{msg.starred ? $t("star") + " ✓" : $t("star")}</button>
        <button class="mi" on:click={pin}>{msg.pinned ? $t("unpin") : $t("pin_msg")}</button>
        {#if msg.dir === "out"}<button class="mi" on:click={info}>{$t("msg_info")}</button>{/if}
        <button class="mi" on:click={selectThis}>{$t("select_messages")}</button>
        <button class="mi danger" on:click={del}>{$t("delete")}</button>
      </div>
    {/if}
  </div>

  {#if menuOpen}<button class="menu-backdrop" aria-label={$t("close")} on:click={() => (menuOpen = false)}></button>{/if}
</div>
