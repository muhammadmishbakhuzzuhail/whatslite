<script>
  import { onMount, createEventDispatcher } from "svelte";
  import { t } from "../i18n.js";

  // Giphy public beta key (anonim, rate-limited) — tak perlu key pengguna.
  const KEY = "dc6zaTOxFJmzC";
  const dispatch = createEventDispatcher();

  let q = "";
  let gifs = [];
  let loading = false;
  let busyId = null;
  let _t;

  async function fetchGifs(query) {
    loading = true;
    const url = query
      ? `https://api.giphy.com/v1/gifs/search?api_key=${KEY}&limit=24&rating=pg-13&q=${encodeURIComponent(query)}`
      : `https://api.giphy.com/v1/gifs/trending?api_key=${KEY}&limit=24&rating=pg-13`;
    try {
      const r = await fetch(url).then((x) => x.json());
      gifs = (r.data || []).map((g) => ({
        id: g.id,
        preview: g.images?.fixed_width_small?.url || g.images?.preview_gif?.url,
        mp4: g.images?.original_mp4?.url || g.images?.looping?.mp4 || g.images?.fixed_height?.mp4,
      })).filter((g) => g.mp4);
    } catch (e) { gifs = []; }
    loading = false;
  }
  onMount(() => fetchGifs(""));
  $: { clearTimeout(_t); const query = q.trim(); _t = setTimeout(() => fetchGifs(query), 300); }

  async function pick(g) {
    if (busyId) return;
    busyId = g.id;
    try {
      const blob = await fetch(g.mp4).then((r) => r.blob());
      const dataURI = await new Promise((res) => { const fr = new FileReader(); fr.onload = () => res(fr.result); fr.readAsDataURL(blob); });
      dispatch("pick", dataURI);
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
  <div class="gif-credit">Powered by GIPHY</div>
</div>

<style>
  .gif-panel { position:absolute; bottom:68px; left:8px; right:8px; max-width:420px; z-index:40; background:var(--bg); border:1px solid var(--line); border-radius:14px; box-shadow:0 8px 30px rgba(0,0,0,.18); padding:10px; }
  .gif-search { width:100%; border:1px solid var(--line); border-radius:10px; padding:8px 12px; background:var(--bg2); color:var(--text); font:inherit; margin-bottom:8px; }
  .gif-grid { display:grid; grid-template-columns:repeat(3,1fr); gap:6px; max-height:300px; overflow-y:auto; }
  .gif-cell { position:relative; padding:0; border:0; background:var(--bg2); border-radius:8px; overflow:hidden; cursor:pointer; aspect-ratio:1; }
  .gif-cell img { width:100%; height:100%; object-fit:cover; display:block; }
  .gif-load { position:absolute; inset:0; display:grid; place-items:center; background:rgba(0,0,0,.4); color:#fff; }
  .gif-empty { grid-column:1/-1; text-align:center; color:var(--text2); padding:24px; }
  .gif-credit { text-align:center; font-size:10px; color:var(--text2); margin-top:6px; letter-spacing:.5px; }
</style>
