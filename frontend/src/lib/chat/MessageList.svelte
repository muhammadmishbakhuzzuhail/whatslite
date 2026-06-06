<script>
  import Bubble from "./Bubble.svelte";
  import { t } from "../i18n.js";
  import { tick, beforeUpdate, afterUpdate } from "svelte";
  import { loadOlder, jumpMsg } from "../../stores.js";
  export let messages = [];
  export let group = false;
  export let chatId;
  export let peerName = "";
  export let firstUnreadId = null; // id pesan pertama belum dibaca (pembatas)
  export let unreadCount = 0;      // jumlah belum dibaca (label pembatas)

  let box;            // wadah scroll
  let atBottom = true;
  let loadingOlder = false, noMore = false;
  let newCount = 0;          // pesan baru selagi scroll ke atas (badge FAB)
  let floatDate = "";        // tanggal mengambang saat scroll
  let floatVisible = false;
  let _floatTimer;
  $: chatId, (noMore = false, newCount = 0, _scrolledUnread = null); // reset saat ganti chat

  // Gulir ke pembatas "belum dibaca" begitu ia muncul (dihitung sedikit setelah
  // pesan termuat) — sekali per chat.
  let _scrolledUnread = null;
  $: if (firstUnreadId && firstUnreadId !== _scrolledUnread && box && items) {
    _scrolledUnread = firstUnreadId;
    tick().then(() => { const d = box && box.querySelector(".unread-divider"); if (d) d.scrollIntoView({ block: "center" }); });
  }

  // === Scroll behavior (pola kanonik beforeUpdate/afterUpdate) ===
  // follow=true → "menempel" ke bawah (ikut pesan baru). User scroll-up → false.
  let follow = true;
  let snapNearBottom = true;     // snapshot SEBELUM DOM update
  let prevChat = null, prevLastId = null, prevLen = 0;

  function nearBottom() {
    return box ? box.scrollHeight - box.scrollTop - box.clientHeight < 80 : true;
  }
  function pin() { if (box) box.scrollTop = box.scrollHeight; }

  function onScroll() {
    if (!box) return;
    follow = nearBottom();            // user di bawah → ikut; scroll-up → lepas
    atBottom = follow;
    if (follow) newCount = 0;
    if (box.scrollTop < 120) maybeLoadOlder();
    updateFloatDate();
  }

  beforeUpdate(() => { snapNearBottom = nearBottom(); });

  afterUpdate(() => {
    if (!box) return;
    const last = messages.length ? messages[messages.length - 1] : null;
    const lastId = last ? last.id || "" : "";
    const lastMine = last ? last.dir === "out" : false;

    if (chatId !== prevChat) {            // (1) buka chat → ke bawah
      prevChat = chatId; prevLastId = lastId; prevLen = messages.length;
      follow = true; atBottom = true; newCount = 0;
      pin(); requestAnimationFrame(pin); setTimeout(pin, 120); setTimeout(pin, 350);
      return;
    }
    const isNew = lastId !== prevLastId;  // pesan BARU di ujung (bukan reaksi/edit/prepend)
    const added = Math.max(0, messages.length - prevLen);
    prevLastId = lastId; prevLen = messages.length;
    if (!isNew) return;                   // (7) reaksi/edit/receipt → diam

    if (lastMine) {                       // (4) kirim sendiri → SELALU ke bawah
      follow = true; atBottom = true; newCount = 0; pin();
    } else if (snapNearBottom) {          // (2) di bawah + pesan baru → ikut
      atBottom = true; newCount = 0; pin();
    } else {                              // (3) scroll-up + pesan baru → diam + badge
      newCount += added || 1;
    }
  });
  // Tanggal pesan teratas yang terlihat → pil mengambang (Telegram-style).
  function updateFloatDate() {
    if (!box) return;
    const top = box.getBoundingClientRect().top;
    const bubbles = box.querySelectorAll("[data-ts]");
    for (const el of bubbles) {
      if (el.getBoundingClientRect().bottom > top + 8) {
        const ts = +el.getAttribute("data-ts");
        if (ts) floatDate = dayLabel(ts);
        break;
      }
    }
    floatVisible = true;
    clearTimeout(_floatTimer);
    _floatTimer = setTimeout(() => (floatVisible = false), 1400);
  }
  // Scroll ke atas → muat riwayat lebih lama, jaga posisi (tak melompat).
  async function maybeLoadOlder() {
    if (loadingOlder || noMore || !box) return;
    loadingOlder = true;
    const prevH = box.scrollHeight, prevT = box.scrollTop;
    const n = await loadOlder(chatId);
    if (n === 0) noMore = true;
    await tick();
    box.scrollTop = box.scrollHeight - prevH + prevT;
    loadingOlder = false;
  }
  function toBottom(smooth = true) {
    if (!box) return;
    box.scrollTo({ top: box.scrollHeight, behavior: smooth ? "smooth" : "auto" });
  }

  // Buang pesan teks kosong (tak ada isi & bukan media) → tak ada bubble kosong.
  function empty(m) {
    if (m.type === "deleted") return false; // pesan dihapus → tetap tampil (placeholder)
    const media = m.type === "image" || m.type === "video" || m.type === "sticker" || m.type === "voice";
    return !media && m.type !== "day" && m.type !== "system" && m.type !== "unread" && !(m.text && m.text.trim()) && !m.thumb;
  }

  $: items = build(messages, group, firstUnreadId, unreadCount);
  function dayLabel(ts) {
    if (!ts) return "";
    const d = new Date(ts * 1000), now = new Date();
    const day = (x) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
    const diff = Math.round((day(now) - day(d)) / 86400000);
    if (diff <= 0) return $t("today");
    if (diff === 1) return $t("yesterday");
    return d.toLocaleDateString(undefined, { day: "numeric", month: "long", year: d.getFullYear() === now.getFullYear() ? undefined : "numeric" });
  }

  function build(msgs, grp, unreadId, unreadN) {
    let prevKey = null;
    let prevDay = null;
    const out = [];
    msgs.forEach((m, idx) => {
      if (empty(m)) return; // idx tetap index asli (untuk hapus/reaksi)
      // Pembatas "belum dibaca" tepat sebelum pesan pertama yg belum dibaca.
      if (unreadId && m.id === unreadId) {
        out.push({ m: { type: "unread", count: unreadN }, idx: "unread", firstOfRun: false });
        prevKey = null; // putus runtun
      }
      // Sisipkan pemisah hari saat tanggal berganti (abaikan chip/sistem bawaan).
      if (m.type !== "day" && m.type !== "system" && m.type !== "unread" && m.ts) {
        const d = new Date(m.ts * 1000);
        const key = `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;
        if (key !== prevDay) {
          prevDay = key;
          out.push({ m: { type: "day", label: dayLabel(m.ts) }, idx: "day-" + idx, firstOfRun: false });
          prevKey = null;
        }
      }
      const isBubble = m.type === "text" || m.type === "image" || m.type === "video" || m.type === "sticker" || m.type === "voice";
      let firstOfRun = true;
      if (isBubble) {
        // Kelompokkan pesan beruntun: per arah, + per pengirim utk grup-masuk.
        const key = m.dir + "|" + (grp && m.dir === "in" ? m.sender || m.senderId || "" : "");
        firstOfRun = key !== prevKey;
        prevKey = key;
      } else {
        prevKey = null; // chip hari/sistem memutus runtun
      }
      out.push({ m, idx, firstOfRun });
    });
    return out;
  }

  // Lompat + highlight ke pesan (dari hasil pencarian). Best-effort: hanya bila
  // pesannya termuat (dalam ~200 terakhir); kalau tidak, biarkan di posisi bawah.
  $: if ($jumpMsg && box && items) tick().then(() => flashTo($jumpMsg));
  function flashTo(mid) {
    const el = box && box.querySelector(`[data-mid="${(window.CSS && CSS.escape) ? CSS.escape(mid) : mid}"]`);
    if (el) {
      el.scrollIntoView({ block: "center" });
      el.classList.add("flash");
      setTimeout(() => el.classList.remove("flash"), 1500);
    }
    jumpMsg.set(null);
  }
</script>

<div class="msg-wrap">
  {#if floatDate}
    <div class="float-date {floatVisible ? 'on' : ''}"><span>{floatDate}</span></div>
  {/if}
  <div class="messages" bind:this={box} on:scroll={onScroll}>
    {#each items as it (it.m.id || it.idx)}
      {#if it.m.type === "day"}
        <div class="day-chip"><span>{it.m.label || $t("today")}</span></div>
      {:else if it.m.type === "system"}
        <div class="system-msg">
          <svg class="lock" viewBox="0 0 24 24"><rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/></svg>
          {$t("enc_notice")}
        </div>
      {:else if it.m.type === "unread"}
        <div class="unread-divider"><span>{$t("unread_count", { n: it.m.count })}</span></div>
      {:else}
        <Bubble msg={it.m} {group} {chatId} {peerName} idx={it.idx} firstOfRun={it.firstOfRun} />
      {/if}
    {/each}
  </div>
  {#if !atBottom}
    <button class="scroll-fab" on:click={() => { follow = true; atBottom = true; newCount = 0; toBottom(); }} aria-label={$t("scroll_bottom")}>
      {#if newCount > 0}<span class="fab-badge">{newCount > 99 ? "99+" : newCount}</span>{/if}
      <svg viewBox="0 0 24 24"><path d="M6 9l6 6 6-6"/></svg>
    </button>
  {/if}
</div>
