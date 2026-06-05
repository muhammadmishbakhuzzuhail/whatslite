<script>
  import { createEventDispatcher } from "svelte";
  import { pushToast } from "../../stores.js";
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

  let tab = recents.length ? "recents" : (pack.length ? "pack" : "create");
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
    <button class:active={tab === "recents"} on:click={() => (tab = "recents")}>{$t("stk_recents")}</button>
    <button class:active={tab === "pack"} on:click={() => (tab = "pack")}>{$t("stk_pack")}</button>
    <button class:active={tab === "create"} on:click={() => (tab = "create")}>{$t("stk_create")}</button>
  </div>

  {#if tab === "recents"}
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
  .stk-panel { position:absolute; bottom:68px; left:8px; right:8px; max-width:380px; z-index:40; background:var(--bg); border:1px solid var(--line); border-radius:14px; box-shadow:0 8px 30px rgba(0,0,0,.18); padding:10px; }
  .stk-tabs { display:flex; gap:6px; margin-bottom:10px; }
  .stk-tabs button { flex:1; padding:8px; border:0; background:var(--bg2); border-radius:9px; cursor:pointer; color:var(--text2); font-size:13px; font-weight:600; }
  .stk-tabs button.active { background:var(--accent); color:#fff; }
  .stk-grid { display:grid; grid-template-columns:repeat(4,1fr); gap:6px; max-height:280px; overflow-y:auto; }
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
