<script>
  import { onMount, createEventDispatcher } from "svelte";
  import { searchGifs, fetchRemoteMedia } from "../../services/data.js";
  import { t } from "../i18n.js";

  // GIF via Go backend (Tenor) → hindari CORS WebKitGTK yang bikin picker kosong.
  const dispatch = createEventDispatcher();

  let q = "";
  let gifs = [];
  let loading = false;
  let busyId = null;
  let _t;

  async function fetchGifs(query) {
    loading = true;
    gifs = await searchGifs(query);
    loading = false;
  }
  onMount(() => fetchGifs(""));
  $: { clearTimeout(_t); const query = q.trim(); _t = setTimeout(() => fetchGifs(query), 300); }

  async function pick(g) {
    if (busyId) return;
    busyId = g.id;
    try {
      // Unduh mp4 lewat backend (no-CORS) → data-URI → kirim sbg GIF.
      const dataURI = await fetchRemoteMedia(g.mp4);
      if (dataURI) dispatch("pick", dataURI);
    } catch (e) {}
    busyId = null;
  }
</script>

<div class="gif-panel">
  <input class="gif-search" placeholder="{$t('search')} GIF" bind:value={q} />
  <div class="gif-cats">
    {#each ["trending","lol","love","sad","wow","ok","thanks","hi","bye","angry","dance","clap"] as c}
      <button class="gif-cat {(/(^|\s)/.test(q) && q.trim().toLowerCase()===c) || (c==='trending'&&!q.trim()) ? 'on' : ''}"
        on:click={() => (q = c === "trending" ? "" : c)}>{c}</button>
    {/each}
  </div>
  <div class="gif-grid">
    {#if loading}
      <div class="gif-empty">…</div>
    {:else}
      {#each gifs as g (g.id)}
        <button class="gif-cell" on:click={() => pick(g)} disabled={busyId === g.id}>
          <img src={g.preview} alt="" loading="lazy" />
          {#if busyId === g.id}<span class="gif-load">…</span>{/if}
        </button>
      {/each}
      {#if gifs.length === 0}<div class="gif-empty">{$t("no_match")}</div>{/if}
    {/if}
  </div>
  <div class="gif-credit">Powered by Tenor</div>
</div>

<style>
  .gif-panel { position:absolute; bottom:68px; left:8px; right:8px; max-width:420px; z-index:40; background:var(--bg); border:1px solid var(--line); border-radius:14px; box-shadow:0 8px 30px rgba(0,0,0,.18); padding:10px; }
  .gif-search { width:100%; border:1px solid var(--line); border-radius:10px; padding:8px 12px; background:var(--bg2); color:var(--text); font:inherit; margin-bottom:8px; }
  /* Masonry 2-kolom ala Discord: rasio natural, tak di-crop, tak bertumpuk. */
  .gif-grid { columns:2; column-gap:6px; max-height:320px; overflow-y:auto; }
  .gif-cell { position:relative; display:block; width:100%; margin:0 0 6px; padding:0; border:0;
    background:var(--bg2); border-radius:8px; overflow:hidden; cursor:pointer; break-inside:avoid; }
  .gif-cell img { width:100%; height:auto; display:block; }
  .gif-load { position:absolute; inset:0; display:grid; align-items:center;justify-items:center; background:rgba(0,0,0,.4); color:#fff; }
  .gif-empty { text-align:center; color:var(--text2); padding:24px; }
  .gif-credit { text-align:center; font-size:10px; color:var(--text2); margin-top:6px; letter-spacing:.5px; }
</style>
