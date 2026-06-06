<script>
  import { createEventDispatcher } from "svelte";
  import { searchGifs, fetchRemoteMedia } from "../../services/data.js";
  import { getHistory, addHistory, removeHistory, clearHistory, suggest } from "../searchHistory.js";
  import { t } from "../i18n.js";

  // GIF via Go backend → hindari CORS WebKitGTK yang bikin picker kosong.
  const dispatch = createEventDispatcher();

  let q = "";
  let hist = getHistory("gif");
  let acOpen = true; // tampilkan autocomplete (tutup setelah pilih)
  $: sugg = acOpen ? suggest(q, hist) : [];
  function commit(term) { q = term; acOpen = false; hist = addHistory("gif", term); } // → reaktif fetch
  let gifs = [];
  let loading = false;
  let loadingMore = false;
  let next = "";
  let busyId = null;
  let _t, _q = null, grid;

  // Muat halaman pertama (reset) untuk query. Reaktif thd q (debounce).
  async function fetchGifs(query) {
    loading = true; next = ""; _q = query;
    const p = await searchGifs(query, "");
    gifs = p.items; next = p.next || "";
    loading = false;
  }
  // Halaman berikutnya (infinite scroll) — append, jaga kursor.
  async function more() {
    if (loadingMore || !next) return;
    loadingMore = true;
    const p = await searchGifs(_q ?? q.trim(), next);
    const seen = new Set(gifs.map((g) => g.id));
    gifs = [...gifs, ...p.items.filter((g) => !seen.has(g.id))];
    next = p.next || "";
    loadingMore = false;
  }
  function onScroll() {
    if (grid && grid.scrollHeight - grid.scrollTop - grid.clientHeight < 240) more();
  }
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
  <div class="pk-searchbox">
    <input class="gif-search" placeholder="{$t('search')} GIF" bind:value={q}
      on:input={() => (acOpen = true)}
      on:keydown={(e) => e.key === "Enter" && q.trim() && commit(q.trim())} />
    {#if q.trim() && sugg.length}
      <div class="ac-pop">
        {#each sugg as s}<button class="ac-item" on:click={() => commit(s)}>{s}</button>{/each}
      </div>
    {/if}
  </div>
  {#if !q.trim() && hist.length}
    <div class="hist-row">
      {#each hist as h}
        <button class="hist-chip" on:click={() => commit(h)}>{h}<span class="hx" role="button" tabindex="0" on:click|stopPropagation={() => (hist = removeHistory("gif", h))} on:keydown={(e) => e.key === "Enter" && (hist = removeHistory("gif", h))}>×</span></button>
      {/each}
      <button class="hist-clear" on:click={() => (hist = clearHistory("gif"))}>{$t("clear")}</button>
    </div>
  {/if}
  <div class="gif-cats">
    {#each ["trending","lol","love","sad","wow","ok","thanks","hi","bye","angry","dance","clap"] as c}
      <button class="gif-cat {(/(^|\s)/.test(q) && q.trim().toLowerCase()===c) || (c==='trending'&&!q.trim()) ? 'on' : ''}"
        on:click={() => { acOpen = false; q = c === "trending" ? "" : c; }}>{c}</button>
    {/each}
  </div>
  <div class="gif-grid" bind:this={grid} on:scroll={onScroll}>
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
      {#if loadingMore}<div class="gif-empty">…</div>{/if}
    {/if}
  </div>
  <div class="gif-credit">Powered by Tenor</div>
</div>

<style>
  .gif-panel { position:absolute; bottom:68px; left:8px; right:8px; max-width:520px; z-index:40; background:var(--bg); border:1px solid var(--line); border-radius:14px; box-shadow:0 8px 30px rgba(0,0,0,.18); padding:10px; }
  .gif-search { width:100%; border:1px solid var(--line); border-radius:10px; padding:8px 12px; background:var(--bg2); color:var(--text); font:inherit; margin-bottom:8px; }
  /* Grid 2-kolom (Y-only). CSS `columns` lama bikin kolom meluber HORIZONTAL
     saat tinggi dibatasi → X-scroll + scrollHeight kacau (infinite-scroll mati).
     Grid → scroll vertikal saja + deteksi scroll andal → muat banyak GIF. */
  .gif-grid { display:grid; grid-template-columns:repeat(3,1fr); gap:6px; align-content:start;
    max-height:360px; overflow-y:auto; overflow-x:hidden; }
  .gif-cell { position:relative; display:block; width:100%; margin:0; padding:0; border:0;
    background:var(--bg2); border-radius:8px; overflow:hidden; cursor:pointer; }
  .gif-cell img { width:100%; height:100%; object-fit:cover; display:block; aspect-ratio:1; }
  .gif-load { position:absolute; inset:0; display:grid; align-items:center;justify-items:center; background:rgba(0,0,0,.4); color:#fff; }
  .gif-empty { grid-column:1/-1; text-align:center; color:var(--text2); padding:24px; }
  .gif-credit { text-align:center; font-size:10px; color:var(--text2); margin-top:6px; letter-spacing:.5px; }
</style>
