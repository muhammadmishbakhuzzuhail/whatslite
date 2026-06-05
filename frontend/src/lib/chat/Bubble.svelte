<script>
  import { onMount } from "svelte";
  import Ticks from "../common/Ticks.svelte";
  import Avatar from "../common/Avatar.svelte";
  import { t } from "../i18n.js";
  import { translateMessage } from "../../services/translate.js";
  import { LIVE, senderColorFor, avatarUrl } from "../../services/data.js";
  import { reactMessage, deleteMessage, starMessage, replyDraft, forwardDraft, activeChatId, chats, translateLang, editDraft, pushToast, pinMessageAction, showMessageInfo, lightbox } from "../../stores.js";

  export let msg;
  export let group = false;
  export let firstOfRun = true;
  export let chatId;
  export let idx;
  export let peerName = "";

  $: isMedia = msg.type === "image" || msg.type === "video" || msg.type === "sticker";
  // Semua media (gambar/video/stiker) → bubble TRANSPARAN (tanpa kartu).
  $: bubbleClass = isMedia ? "media sticker-bubble" : msg.type === "voice" ? "voice" : "";
  $: isGroupIn = group && msg.dir === "in";
  $: showSender = isGroupIn && msg.sender && firstOfRun;
  $: senderCol = msg.senderColor || senderColorFor(msg.senderId || msg.sender || "");
  $: caption = msg.caption || (msg.type !== "sticker" ? msg.text : "") || "";

  // Pecah teks → bagian biasa + bagian mention (@<nomor> → @Nama berwarna+klik).
  $: textParts = renderParts(msg.text, msg.mentions);
  function renderParts(text, mentions) {
    if (!text) return [];
    const map = {};
    (mentions || []).forEach((m) => (map[m.num] = m));
    const re = /@(\d{5,})/g;
    const out = [];
    let last = 0, mt;
    while ((mt = re.exec(text)) !== null) {
      if (mt.index > last) out.push({ t: text.slice(last, mt.index) });
      const info = map[mt[1]];
      if (info) out.push({ m: true, name: info.name, jid: info.jid });
      else out.push({ t: mt[0] });
      last = re.lastIndex;
    }
    if (last < text.length) out.push({ t: text.slice(last) });
    return out;
  }
  function openMention(jid) {
    if ($chats.some((c) => c.id === jid)) activeChatId.set(jid);
  }
  $: source = msg.text || msg.caption || "";
  $: canTranslate = msg.type === "text" || ((msg.type === "image" || msg.type === "video") && caption);

  // URL media disajikan asset-server (cache FILE, bukan data-URI memori). Native
  // → /media/<chat>/<id> (browser auto-load + cache); preview/browser → thumb.
  $: mediaUrl = LIVE && msg.id
    ? "/media/" + encodeURIComponent(chatId) + "/" + encodeURIComponent(msg.id)
    : (msg.thumb || "");
  let mediaErr = false;
  let videoPlaying = false;

  let translated = null;
  let busy = false;
  let playing = false;
  let audioEl = null;
  function playVoice() {
    if (!mediaUrl) return;
    if (!audioEl) {
      audioEl = new Audio(mediaUrl);
      audioEl.addEventListener("ended", () => (playing = false));
    }
    if (playing) {
      audioEl.pause();
      playing = false;
    } else {
      audioEl.play().then(() => (playing = true)).catch(() => {});
    }
  }
  async function doTranslate() {
    if (busy) return;
    busy = true;
    translated = await translateMessage(source, $translateLang);
    busy = false;
  }
  onMount(() => {
    const p = new URLSearchParams(location.search);
    if (p.get("tr") === "1" && canTranslate && msg.dir === "in") doTranslate();
    if (p.get("menu") !== null && Number(p.get("menu")) === idx) menuOpen = true; // pratinjau
  });

  // --- context menu & aksi ---
  let menuOpen = false;
  let emojiMore = false;
  const QUICK = ["❤️", "😂", "👍", "😮", "😢", "🙏"];
  const MORE = "😀 😅 😊 😍 🥰 😘 🤔 😎 🥳 😇 🙂 😉 😋 😜 🤩 🥺 😢 😭 😡 😱 😴 🤯 🤗 🙏 👍 👎 👏 🙌 💪 🔥 ✨ 🎉 ❤️ 🧡 💛 💚 💙 💜 🖤 💔 💯 ✅ ❌ ⭐ 👀 🤝 🎁 🍕 ☕".split(" ");
  function react(e) { reactMessage(chatId, idx, e); emojiMore = false; menuOpen = false; }
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
  function openMedia() {
    if (!mediaUrl || mediaErr) return;
    if (msg.type === "image") lightbox.set({ url: mediaUrl, type: "image", caption });
    else if (msg.type === "video") videoPlaying = true; // putar inline
    // stiker: tak ada aksi
  }
  function pin() { pinMessageAction(chatId, idx, !msg.pinned); menuOpen = false; }
  function info() { showMessageInfo(chatId, idx); menuOpen = false; }
</script>

<div class="msg {msg.dir} {isGroupIn ? 'gin' : ''} {firstOfRun ? '' : 'cont'} {msg.reactions ? 'has-react' : ''}" data-mid={msg.id}>
  {#if isGroupIn}
    <div class="msg-avatar">
      {#if firstOfRun && msg.sender}<Avatar name={msg.sender} color={senderCol} photo={avatarUrl(msg.senderId)} tiny={true} />{/if}
    </div>
  {/if}

  <div class="bubble {bubbleClass} {msg.type === 'deleted' ? 'deleted' : ''}">
    {#if msg.type !== "deleted"}
      <button class="msg-menu-btn" aria-label={$t("menu")} on:click={() => (menuOpen = !menuOpen)}>
        <svg viewBox="0 0 24 24"><path d="M7 10l5 5 5-5"/></svg>
      </button>
    {/if}

    {#if showSender}
      <span class="sender" style="color:{senderCol}">{msg.sender}</span>
    {/if}
    {#if msg.type === "deleted"}
      <span class="text deleted-text">
        <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M5.6 5.6l12.8 12.8"/></svg>
        {msg.dir === "out" ? $t("deleted_out") : $t("deleted_in")}
      </span>
    {/if}
    {#if msg.forwarded}
      <div class="forwarded"><svg viewBox="0 0 24 24"><path d="M10 9V5l8 7-8 7v-4c-5 0-8 2-9 5 0-6 3-9 9-9z"/></svg>{$t("forwarded")}</div>
    {/if}
    {#if msg.quote}
      <div class="quote"><div class="quote-name">{msg.quote.name}</div><div class="quote-text">{msg.quote.text}</div></div>
    {/if}

    {#if msg.type === "text"}
      <span class="text">{#each textParts as p}{#if p.m}<span class="mention" role="button" tabindex="0" on:click|stopPropagation={() => openMention(p.jid)} on:keydown={(e) => e.key === "Enter" && openMention(p.jid)}>@{p.name}</span>{:else}{p.t}{/if}{/each}</span>
    {:else if isMedia}
      <div class="media-box {msg.type === 'sticker' ? 'sticker' : ''}"
        role="button" tabindex="0" on:click={openMedia}>
        {#if msg.type === "video" && videoPlaying}
          <video class="media-img" src={mediaUrl} controls autoplay></video>
        {:else if mediaUrl && !mediaErr}
          <img class="media-img" src={mediaUrl} alt="" loading="lazy" on:error={() => (mediaErr = true)} />
        {:else}
          <div class="img-ph">
            <span class="ph-dl"><svg viewBox="0 0 24 24"><path d="M12 4v11M7 11l5 5 5-5M5 20h14"/></svg></span>
            <span class="ph-lbl">{msg.type === "video" ? $t("t_video") : msg.type === "sticker" ? $t("t_sticker") : $t("t_photo")}</span>
          </div>
        {/if}
        {#if msg.type === "video" && !videoPlaying}<span class="play-badge"><svg viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg></span>{/if}
      </div>
      {#if caption}<span class="text caption">{caption}</span>{/if}
    {:else if msg.type === "voice"}
      <button class="play" aria-label="Play" on:click={playVoice}>
        {#if playing}
          <svg viewBox="0 0 24 24"><rect x="6" y="5" width="4" height="14" rx="1"/><rect x="14" y="5" width="4" height="14" rx="1"/></svg>
        {:else}
          <svg viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg>
        {/if}
      </button>
      <div class="wave">{#each Array(18) as _}<span></span>{/each}</div>
      <span class="vtime">{msg.duration || msg.text || ""}</span>
    {/if}

    {#if translated}
      <div class="tr-block"><span class="tr-lbl">{$t("translated")}</span>{translated}</div>
    {/if}

    <span class="meta">
      <span class="time">{msg.time}</span>
      {#if msg.dir === "out"}<Ticks status={msg.status || "sent"} />{/if}
    </span>

    {#if msg.reactions && msg.reactions.length}
      <div class="reactions">{#each msg.reactions as r}<button class="reaction" class:mine={r.mine} on:click={() => r.mine && undoReact(r.emoji)} title={r.mine ? $t("reaction_remove") : ""}>{r.emoji}{#if r.count > 1} {r.count}{/if}</button>{/each}</div>
    {/if}

    {#if menuOpen}
      <div class="msg-menu">
        <div class="react-row">
          {#each QUICK as e}<button class="rx" on:click={() => react(e)}>{e}</button>{/each}
          <button class="rx rx-more" on:click|stopPropagation={() => (emojiMore = !emojiMore)} aria-label={$t("emoji")}>+</button>
        </div>
        {#if emojiMore}
          <div class="rx-grid">{#each MORE as e}<button class="rx" on:click={() => react(e)}>{e}</button>{/each}</div>
        {/if}
        <button class="mi" on:click={reply}>{$t("reply")}</button>
        {#if isGroupIn && msg.senderId}<button class="mi" on:click={replyPrivate}>{$t("reply_private")}</button>{/if}
        {#if source}<button class="mi" on:click={copyText}>{$t("copy")}</button>{/if}
        {#if msg.dir === "out" && msg.type === "text"}<button class="mi" on:click={editMsg}>{$t("edit")}</button>{/if}
        {#if canTranslate}<button class="mi" on:click={menuTranslate}>{translated ? $t("show_original") : $t("translate")}</button>{/if}
        <button class="mi" on:click={forward}>{$t("forward_action")}</button>
        <button class="mi" on:click={star}>{msg.starred ? $t("star") + " ✓" : $t("star")}</button>
        <button class="mi" on:click={pin}>{msg.pinned ? $t("unpin") : $t("pin_msg")}</button>
        {#if msg.dir === "out"}<button class="mi" on:click={info}>{$t("msg_info")}</button>{/if}
        <button class="mi danger" on:click={del}>{$t("delete")}</button>
      </div>
    {/if}
  </div>

  {#if menuOpen}<button class="menu-backdrop" aria-label={$t("close")} on:click={() => (menuOpen = false)}></button>{/if}
</div>
