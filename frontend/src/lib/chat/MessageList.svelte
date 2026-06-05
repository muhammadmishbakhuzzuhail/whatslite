<script>
  import Bubble from "./Bubble.svelte";
  import { t } from "../i18n.js";
  import { tick } from "svelte";
  import { loadOlder, jumpMsg } from "../../stores.js";
  export let messages = [];
  export let group = false;
  export let chatId;
  export let peerName = "";

  let box;            // wadah scroll
  let atBottom = true;
  let loadingOlder = false, noMore = false;
  $: chatId, (noMore = false); // reset saat ganti chat

  function onScroll() {
    if (!box) return;
    atBottom = box.scrollHeight - box.scrollTop - box.clientHeight < 60;
    if (box.scrollTop < 120) maybeLoadOlder();
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

  $: items = build(messages, group);
  function dayLabel(ts) {
    if (!ts) return "";
    const d = new Date(ts * 1000), now = new Date();
    const day = (x) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
    const diff = Math.round((day(now) - day(d)) / 86400000);
    if (diff <= 0) return $t("today");
    if (diff === 1) return $t("yesterday");
    return d.toLocaleDateString(undefined, { day: "numeric", month: "long", year: d.getFullYear() === now.getFullYear() ? undefined : "numeric" });
  }

  function build(msgs, grp) {
    let prevKey = null;
    let prevDay = null;
    const out = [];
    msgs.forEach((m, idx) => {
      if (empty(m)) return; // idx tetap index asli (untuk hapus/reaksi)
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

  // Scroll cerdas: HANYA turun otomatis saat (a) buka chat, atau (b) ada pesan
  // BARU di ujung & user sedang di bawah. Re-sync data sama TIDAK menyentak
  // (cegah "scroll terus-menerus" saat sinkronisasi). Kalau user scroll ke atas,
  // tombol kanan-bawah muncul untuk turun manual.
  let curChat = null, lastCount = 0, lastId = "";
  $: handleScroll(chatId, items, box);
  function handleScroll(cid, it, b) {
    if (!b) return;
    const id = it.length ? it[it.length - 1].m.id || "" : "";
    if (cid !== curChat) { // chat berganti → reset + ke bawah
      curChat = cid; lastCount = it.length; lastId = id; atBottom = true;
      tick().then(() => toBottom(false));
      return;
    }
    const grew = it.length > lastCount || id !== lastId;
    lastCount = it.length; lastId = id;
    if (grew && atBottom) tick().then(() => toBottom(false));
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
  <div class="messages" bind:this={box} on:scroll={onScroll}>
    {#each items as it (it.idx)}
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
    <button class="scroll-fab" on:click={() => toBottom()} aria-label={$t("scroll_bottom")}>
      <svg viewBox="0 0 24 24"><path d="M6 9l6 6 6-6"/></svg>
    </button>
  {/if}
</div>
