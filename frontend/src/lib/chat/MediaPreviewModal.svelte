<script>
  import { mediaDraft, sendMediaMessage } from "../../stores.js";
  import { t } from "../i18n.js";

  let caption = "";
  let idx = 0;
  let last = null;
  let once = false; // sekali-lihat (view-once): toggle di preview, bukan attach menu
  $: if ($mediaDraft && $mediaDraft !== last) { last = $mediaDraft; caption = ""; idx = 0; once = !!$mediaDraft.viewOnce; }
  $: items = $mediaDraft?.items || [];
  $: cur = items[idx];

  function close() { mediaDraft.set(null); caption = ""; idx = 0; once = false; }
  async function send() {
    const d = $mediaDraft;
    if (!d) return;
    mediaDraft.set(null);
    // Caption ikut gambar yang sedang dilihat (sisanya tanpa caption) — ala WhatsApp.
    for (let i = 0; i < d.items.length; i++) {
      const it = d.items[i];
      await sendMediaMessage(d.chatId, it.kind, i === idx ? caption.trim() : "", it.name || "", it.dataURI, once);
    }
    caption = ""; once = false;
  }
  function onKey(e) {
    if (!$mediaDraft) return;
    if (e.key === "Escape") close();
    else if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); send(); }
  }
</script>

<svelte:window on:keydown={onKey} />

{#if $mediaDraft && cur}
  <div class="mp-overlay" on:click|self={close}>
    <button class="mp-x" title={$t("close")} on:click={close}>✕</button>
    <div class="mp-stage">
      {#if cur.kind === "video"}
        <video class="mp-media" src={cur.dataURI} controls></video>
      {:else}
        <img class="mp-media" src={cur.dataURI} alt="" />
      {/if}
      <!-- Toggle sekali-lihat di atas foto (badge "1") — ala WhatsApp. -->
      <button class="mp-once-badge {once ? 'on' : ''}" title={$t("view_once")} on:click={() => (once = !once)}>
        <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><text x="12" y="16" text-anchor="middle" font-size="11" font-weight="700" fill="currentColor" stroke="none">1</text></svg>
      </button>
    </div>
    {#if items.length > 1}
      <div class="mp-strip">
        {#each items as it, i}
          <button class="mp-thumb {i === idx ? 'on' : ''}" on:click={() => (idx = i)}>
            {#if it.kind === "video"}<video src={it.dataURI} muted></video>{:else}<img src={it.dataURI} alt="" />{/if}
          </button>
        {/each}
      </div>
    {/if}
    <div class="mp-bar">
      <input class="mp-caption" placeholder={$t("add_caption")} bind:value={caption} autofocus />
      <!-- Tombol sekali-lihat di kanan area teks (sebelum tombol kirim). -->
      <button class="mp-once {once ? 'on' : ''}" on:click={() => (once = !once)} title={$t("view_once")}>
        <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><text x="12" y="16" text-anchor="middle" font-size="11" font-weight="700" fill="currentColor" stroke="none">1</text></svg>
      </button>
      <button class="mp-send" on:click={send} title={$t("send")}>
        {#if items.length > 1}<span class="mp-count">{items.length}</span>{/if}
        <svg viewBox="0 0 24 24"><path d="M3 11l18-8-8 18-2-7-8-3z"/></svg>
      </button>
    </div>
  </div>
{/if}

<style>
  .mp-overlay { position:fixed; inset:0; z-index:70; background:rgba(11,20,26,.97); display:flex; flex-direction:column; }
  .mp-x { position:absolute; top:16px; left:18px; background:none; border:0; color:#fff; font-size:22px; cursor:pointer; z-index:2; }
  .mp-stage { flex:1; display:grid; align-items:center;justify-items:center; overflow:hidden; padding:48px 16px 8px; }
  .mp-media { max-width:90vw; max-height:66vh; object-fit:contain; border-radius:8px; }
  .mp-strip { display:flex; gap:8px; justify-content:center; padding:6px 16px; overflow-x:auto; }
  .mp-thumb { width:54px; height:54px; border-radius:8px; overflow:hidden; border:2px solid transparent; padding:0; background:none; cursor:pointer; flex:0 0 auto; }
  .mp-thumb.on { border-color:var(--accent); }
  .mp-thumb img, .mp-thumb video { width:100%; height:100%; object-fit:cover; }
  .mp-bar { display:flex; align-items:center; gap:10px; padding:14px 18px 22px; max-width:760px; width:100%; margin:0 auto; }
  .mp-caption { flex:1; border:0; border-radius:22px; padding:12px 18px; background:var(--bg2,#1f2c34); color:var(--text,#e9edef); font:inherit; outline:none; }
  .mp-send { position:relative; width:48px; height:48px; border-radius:50%; border:0; background:var(--accent); color:#fff; cursor:pointer; display:grid; align-items:center;justify-items:center; flex:0 0 auto; }
  .mp-send svg { width:22px; height:22px; fill:currentColor; }
  .mp-count { position:absolute; top:-4px; right:-4px; background:#fff; color:var(--accent); font-size:11px; font-weight:700; border-radius:9px; min-width:18px; height:18px; display:grid; align-items:center;justify-items:center; }
  .mp-once { width:48px; height:48px; border-radius:50%; border:0; background:rgba(255,255,255,.14); color:#fff; cursor:pointer; display:flex; align-items:center; justify-content:center; flex:0 0 auto; }
  .mp-once svg { width:24px; height:24px; fill:none; stroke:currentColor; stroke-width:2; }
  .mp-once.on { background:var(--accent); color:#fff; }
  .mp-stage { position:relative; }
  .mp-once-badge { position:absolute; top:54px; right:18px; width:40px; height:40px; border-radius:50%; border:0; background:rgba(0,0,0,.5); color:#fff; cursor:pointer; display:flex; align-items:center; justify-content:center; z-index:2; }
  .mp-once-badge svg { width:24px; height:24px; fill:none; stroke:currentColor; stroke-width:2; }
  .mp-once-badge.on { background:var(--accent); }
</style>
