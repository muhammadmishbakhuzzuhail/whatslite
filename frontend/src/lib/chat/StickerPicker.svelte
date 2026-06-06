<script>
  import { createEventDispatcher } from "svelte";
  import { pushToast } from "../../stores.js";
  import { searchStickers, fetchRemoteMedia } from "../../services/data.js";
  import { getHistory, addHistory, removeHistory, clearHistory, suggest } from "../searchHistory.js";
  import { t } from "../i18n.js";

  const dispatch = createEventDispatcher();
  const REC_KEY = "wa-sticker-recents";
  const PACK_KEY = "wa-sticker-pack";
  let recents = [];
  let pack = [];
  try { recents = JSON.parse(localStorage.getItem(REC_KEY) || "[]"); } catch (e) {}
  try { pack = JSON.parse(localStorage.getItem(PACK_KEY) || "[]"); } catch (e) {}
  function saveRecents() { try { localStorage.setItem(REC_KEY, JSON.stringify(recents.slice(0, 24))); } catch (e) {} }
  function savePack() { try { localStorage.setItem(PACK_KEY, JSON.stringify(pack.slice(0, 100))); } catch (e) {} }

  // --- Stiker online (transparan) — paginated infinite scroll + history/autocomplete ---
  let online = [], onlineQ = "", onlineLoading = false, onlineMore = false, onlineNext = "", onlineBusy = null, _ot, _oq = null, onlineGrid;
  let sHist = getHistory("sticker");
  let sAcOpen = true;
  $: sSugg = sAcOpen ? suggest(onlineQ, sHist) : [];
  function sCommit(term) { onlineQ = term; sAcOpen = false; sHist = addHistory("sticker", term); }
  async function fetchOnline(query) {
    onlineLoading = true; onlineNext = ""; _oq = query;
    const p = await searchStickers(query, "");
    online = p.items; onlineNext = p.next || "";
    onlineLoading = false;
  }
  async function moreOnline() {
    if (onlineMore || !onlineNext) return;
    onlineMore = true;
    const p = await searchStickers(_oq ?? onlineQ.trim(), onlineNext);
    const seen = new Set(online.map((s) => s.id));
    online = [...online, ...p.items.filter((s) => !seen.has(s.id))];
    onlineNext = p.next || "";
    onlineMore = false;
  }
  function onOnlineScroll() {
    if (onlineGrid && onlineGrid.scrollHeight - onlineGrid.scrollTop - onlineGrid.clientHeight < 220) moreOnline();
  }
  $: { clearTimeout(_ot); const query = onlineQ.trim(); _ot = setTimeout(() => fetchOnline(query), 300); }
  async function pickOnline(s) {
    if (onlineBusy) return;
    onlineBusy = s.id;
    try {
      const dataURI = await fetchRemoteMedia(s.mp4); // unduh transparan via backend
      if (dataURI) { recents = [dataURI, ...recents.filter((r) => r !== dataURI)]; saveRecents(); dispatch("pick", dataURI); }
    } catch (e) {}
    onlineBusy = null;
  }

  let tab = "online";
  let packInput;
  function pickPack() { packInput && packInput.click(); }
  async function onPackFiles(e) {
    const files = [...(e.target.files || [])];
    e.target.value = "";
    for (const f of files) {
      try { const wp = await toSticker(f); pack = [wp, ...pack.filter((x) => x !== wp)]; } catch (e2) {}
    }
    savePack();
  }
  function removePack(uri) { pack = pack.filter((x) => x !== uri); savePack(); }
  let fileInput;
  let busy = false;
  let preview = null; // webp dataURI hasil cutout

  function pickFile() { fileInput && fileInput.click(); }

  // Gambar (mungkin transparan) → kanvas 512×512, objek di-fit & ditengah → webp.
  function toSticker(srcBlobOrUrl) {
    return new Promise((res, rej) => {
      const img = new Image();
      img.crossOrigin = "anonymous";
      img.onload = () => {
        const C = 512;
        const cv = document.createElement("canvas");
        cv.width = C; cv.height = C;
        const ctx = cv.getContext("2d");
        const scale = Math.min(C / img.width, C / img.height);
        const w = img.width * scale, h = img.height * scale;
        ctx.drawImage(img, (C - w) / 2, (C - h) / 2, w, h);
        res(cv.toDataURL("image/webp", 0.92));
      };
      img.onerror = rej;
      img.src = typeof srcBlobOrUrl === "string" ? srcBlobOrUrl : URL.createObjectURL(srcBlobOrUrl);
    });
  }

  async function onFile(e) {
    const f = e.target.files && e.target.files[0];
    e.target.value = "";
    if (!f) return;
    busy = true; preview = null;
    try {
      // Hapus background (ML in-browser, model di-unduh sekali oleh lib).
      const { removeBackground } = await import("@imgly/background-removal");
      const cut = await removeBackground(f);          // PNG transparan
      preview = await toSticker(cut);                  // → webp 512 transparan
    } catch (err) {
      // fallback: tanpa hapus-BG (kotak) bila ML gagal.
      try { preview = await toSticker(f); pushToast($t("sticker_bg_failed")); }
      catch (e2) { pushToast($t("err_generic")); }
    }
    busy = false;
  }

  function sendPreview() {
    if (!preview) return;
    recents = [preview, ...recents.filter((r) => r !== preview)];
    saveRecents();
    dispatch("pick", preview);
    preview = null;
  }
  function sendRecent(uri) {
    recents = [uri, ...recents.filter((r) => r !== uri)];
    saveRecents();
    dispatch("pick", uri);
  }
  function sendPack(uri) {
    recents = [uri, ...recents.filter((r) => r !== uri)];
    saveRecents();
    dispatch("pick", uri);
  }
</script>

<div class="stk-panel">
  <div class="stk-tabs">
    <button class:active={tab === "online"} on:click={() => (tab = "online")}>{$t("stk_online")}</button>
    <button class:active={tab === "recents"} on:click={() => (tab = "recents")}>{$t("stk_recents")}</button>
    <button class:active={tab === "pack"} on:click={() => (tab = "pack")}>{$t("stk_pack")}</button>
    <button class:active={tab === "create"} on:click={() => (tab = "create")}>{$t("stk_create")}</button>
  </div>

  {#if tab === "online"}
    <div class="pk-searchbox">
      <input class="stk-search" placeholder="{$t('search')} stiker" bind:value={onlineQ}
        on:input={() => (sAcOpen = true)}
        on:keydown={(e) => e.key === "Enter" && onlineQ.trim() && sCommit(onlineQ.trim())} />
      {#if onlineQ.trim() && sSugg.length}
        <div class="ac-pop">
          {#each sSugg as s}<button class="ac-item" on:click={() => sCommit(s)}>{s}</button>{/each}
        </div>
      {/if}
    </div>
    {#if !onlineQ.trim() && sHist.length}
      <div class="hist-row">
        {#each sHist as h}
          <button class="hist-chip" on:click={() => sCommit(h)}>{h}<span class="hx" role="button" tabindex="0" on:click|stopPropagation={() => (sHist = removeHistory("sticker", h))} on:keydown={(e) => e.key === "Enter" && (sHist = removeHistory("sticker", h))}>×</span></button>
        {/each}
        <button class="hist-clear" on:click={() => (sHist = clearHistory("sticker"))}>{$t("clear")}</button>
      </div>
    {/if}
    <div class="stk-grid" bind:this={onlineGrid} on:scroll={onOnlineScroll}>
      {#if onlineLoading}
        <div class="stk-empty">…</div>
      {:else}
        {#each online as s (s.id)}
          <button class="stk-cell" on:click={() => pickOnline(s)} disabled={onlineBusy === s.id}><img src={s.preview} alt="" loading="lazy" /></button>
        {/each}
        {#if online.length === 0}<div class="stk-empty">{$t("no_match")}</div>{/if}
        {#if onlineMore}<div class="stk-empty">…</div>{/if}
      {/if}
    </div>
    <div class="stk-credit">Powered by Tenor</div>
  {:else if tab === "recents"}
    <div class="stk-grid">
      {#each recents as uri}
        <button class="stk-cell" on:click={() => sendRecent(uri)}><img src={uri} alt="" /></button>
      {/each}
      {#if recents.length === 0}<div class="stk-empty">{$t("stk_no_recents")}</div>{/if}
    </div>
  {:else if tab === "pack"}
    <button class="btn-ghost" style="margin:0 0 8px" on:click={pickPack}>+ {$t("stk_import")}</button>
    <input type="file" accept="image/png,image/webp,image/*" multiple bind:this={packInput} on:change={onPackFiles} style="display:none" />
    <div class="stk-grid">
      {#each pack as uri}
        <button class="stk-cell" on:click={() => sendPack(uri)} on:contextmenu|preventDefault={() => removePack(uri)} title={$t("stk_remove_hint")}><img src={uri} alt="" /></button>
      {/each}
      {#if pack.length === 0}<div class="stk-empty">{$t("stk_pack_empty")}</div>{/if}
    </div>
  {:else}
    <div class="stk-create">
      {#if busy}
        <div class="stk-busy">{$t("stk_processing")}</div>
      {:else if preview}
        <img class="stk-preview" src={preview} alt="" />
        <div class="stk-actions">
          <button class="btn-ghost" on:click={() => (preview = null)}>{$t("cancel")}</button>
          <button class="btn-accent" on:click={sendPreview}>{$t("send")}</button>
        </div>
      {:else}
        <button class="stk-pick" on:click={pickFile}>
          <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
          <span>{$t("stk_pick")}</span>
          <small>{$t("stk_hint")}</small>
        </button>
      {/if}
      <input type="file" accept="image/*" bind:this={fileInput} on:change={onFile} style="display:none" />
    </div>
  {/if}
</div>

<style>
  .stk-panel { position:absolute; bottom:68px; left:8px; right:8px; max-width:520px; z-index:40; background:var(--bg); border:1px solid var(--line); border-radius:14px; box-shadow:0 8px 30px rgba(0,0,0,.18); padding:10px; }
  .stk-tabs { display:flex; gap:6px; margin-bottom:10px; }
  .stk-tabs button { flex:1; padding:8px; border:0; background:var(--bg2); border-radius:9px; cursor:pointer; color:var(--text2); font-size:13px; font-weight:600; }
  .stk-tabs button.active { background:var(--accent); color:#fff; }
  .stk-search { width:100%; border:0; border-radius:9px; padding:8px 12px; background:var(--bg2); color:var(--text); font:inherit; outline:none; margin-bottom:8px; }
  .stk-credit { text-align:center; font-size:10px; color:var(--text2); margin-top:6px; letter-spacing:.5px; }
  .stk-grid { display:grid; grid-template-columns:repeat(5,1fr); gap:6px; max-height:320px; overflow-y:auto; overflow-x:hidden; }
  .stk-cell { padding:6px; border:0; background:var(--bg2); border-radius:10px; cursor:pointer; aspect-ratio:1; }
  .stk-cell img { width:100%; height:100%; object-fit:contain; }
  .stk-empty, .stk-busy { grid-column:1/-1; text-align:center; color:var(--text2); padding:28px 12px; }
  .stk-create { min-height:180px; display:grid; align-items:center;justify-items:center; }
  .stk-pick { display:flex; flex-direction:column; align-items:center; gap:6px; padding:28px; border:2px dashed var(--line); border-radius:14px; background:none; cursor:pointer; color:var(--text2); width:100%; }
  .stk-pick svg { width:30px; height:30px; fill:none; stroke:var(--accent); stroke-width:2; }
  .stk-pick small { font-size:11px; }
  .stk-preview { max-width:200px; max-height:200px; object-fit:contain; }
  .stk-actions { display:flex; gap:10px; justify-content:center; margin-top:12px; }
</style>
