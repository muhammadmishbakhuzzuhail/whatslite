<script>
  import { mediaDraft, sendMediaMessage } from "../../stores.js";
  import { t } from "../i18n.js";

  let captions = []; // caption PER item (ala WhatsApp: tiap foto punya caption sendiri)
  let names = [];    // nama file PER item (dokumen bisa di-rename sebelum kirim)
  let editURIs = []; // dataURI hasil edit (crop/rotate) PER item; kosong = pakai asli
  let idx = 0;
  let last = null;
  let once = false; // sekali-lihat (view-once): toggle di preview, bukan attach menu
  $: if ($mediaDraft && $mediaDraft !== last) { last = $mediaDraft; const its = $mediaDraft.items || []; captions = its.map(() => ""); names = its.map((it) => it.name || ""); editURIs = its.map((it) => it.dataURI); idx = 0; once = !!$mediaDraft.viewOnce; cropping = false; cropBox = null; }
  $: items = $mediaDraft?.items || [];
  $: cur = items[idx];

  // --- Edit foto: rotate (bake langsung) + crop (drag persegi) ---
  // Rotate di-bake tiap klik → crop selalu bekerja di gambar tegak (tak perlu
  // hitung rotasi saat crop → bebas-bug). Hasil disimpan di editURIs[idx].
  let imgEl, cropping = false, cropBox = null, dragStart = null;
  function loadImg(uri) { return new Promise((res, rej) => { const im = new Image(); im.onload = () => res(im); im.onerror = rej; im.src = uri; }); }
  async function rotate(dir) {
    const im = await loadImg(editURIs[idx]);
    const c = document.createElement("canvas");
    c.width = im.naturalHeight; c.height = im.naturalWidth;
    const x = c.getContext("2d");
    x.translate(c.width / 2, c.height / 2);
    x.rotate((dir * Math.PI) / 2);
    x.drawImage(im, -im.naturalWidth / 2, -im.naturalHeight / 2);
    editURIs[idx] = c.toDataURL("image/jpeg", 0.92); editURIs = [...editURIs];
    cropping = false; cropBox = null;
  }
  function cropDown(e) { if (!cropping) return; const r = imgEl.getBoundingClientRect(); dragStart = { x: e.clientX - r.left, y: e.clientY - r.top, r }; cropBox = { x: dragStart.x, y: dragStart.y, w: 0, h: 0 }; }
  function cropMove(e) { if (!dragStart) return; const r = dragStart.r; let x = Math.max(0, Math.min(r.width, e.clientX - r.left)), y = Math.max(0, Math.min(r.height, e.clientY - r.top)); cropBox = { x: Math.min(dragStart.x, x), y: Math.min(dragStart.y, y), w: Math.abs(x - dragStart.x), h: Math.abs(y - dragStart.y) }; }
  function cropUp() { dragStart = null; }
  async function applyCrop() {
    if (!cropBox || cropBox.w < 8 || cropBox.h < 8) { cropping = false; cropBox = null; return; }
    const im = await loadImg(editURIs[idx]);
    const r = imgEl.getBoundingClientRect();
    const scale = Math.min(r.width / im.naturalWidth, r.height / im.naturalHeight); // object-fit:contain
    const ox = (r.width - im.naturalWidth * scale) / 2, oy = (r.height - im.naturalHeight * scale) / 2;
    let cx = Math.max(0, (cropBox.x - ox) / scale), cy = Math.max(0, (cropBox.y - oy) / scale);
    let cw = Math.min(im.naturalWidth - cx, cropBox.w / scale), ch = Math.min(im.naturalHeight - cy, cropBox.h / scale);
    if (cw < 4 || ch < 4) { cropping = false; cropBox = null; return; }
    const c = document.createElement("canvas"); c.width = Math.round(cw); c.height = Math.round(ch);
    c.getContext("2d").drawImage(im, cx, cy, cw, ch, 0, 0, cw, ch);
    editURIs[idx] = c.toDataURL("image/jpeg", 0.92); editURIs = [...editURIs];
    cropping = false; cropBox = null;
  }

  function close() { mediaDraft.set(null); captions = []; idx = 0; once = false; }
  async function send() {
    const d = $mediaDraft;
    if (!d) return;
    const caps = captions;
    mediaDraft.set(null);
    // Tiap item dikirim dgn caption-nya sendiri.
    for (let i = 0; i < d.items.length; i++) {
      const it = d.items[i];
      await sendMediaMessage(d.chatId, it.kind, (caps[i] || "").trim(), (names[i] || it.name || "").trim(), editURIs[i] || it.dataURI, once);
    }
    captions = []; once = false;
  }
  function onKey(e) {
    if (!$mediaDraft) return;
    if (e.key === "Escape") close();
    else if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); send(); }
  }
</script>

<svelte:window on:keydown={onKey} on:mousemove={cropMove} on:mouseup={cropUp} />

{#if $mediaDraft && cur}
  <div class="mp-overlay" on:click|self={close}>
    <button class="mp-x" title={$t("close")} on:click={close}>✕</button>
    <div class="mp-stage">
      {#if cur.kind === "video"}
        <video class="mp-media" src={cur.dataURI} controls></video>
      {:else if cur.kind === "document"}
        <div class="mp-doc">
          <input class="mp-doc-name mp-doc-rename" bind:value={names[idx]} placeholder={$t("file_name")} title={$t("rename")} />
          {#if /\.(mp3|wav|ogg|m4a|aac|flac|opus)$/i.test(cur.name)}
            <div class="mp-doc-ico"><svg viewBox="0 0 24 24"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg></div>
            <audio class="mp-doc-audio" src={cur.dataURI} controls></audio>
          {:else if /\.(png|jpe?g|gif|webp|bmp)$/i.test(cur.name)}
            <img class="mp-doc-img" src={cur.dataURI} alt="" />
          {:else}
            <div class="mp-doc-ico"><svg viewBox="0 0 24 24"><path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5"/></svg></div>
          {/if}
        </div>
      {:else}
        <div class="mp-imgwrap" class:cropping>
          <img class="mp-media" bind:this={imgEl} src={editURIs[idx] || cur.dataURI} alt="" draggable="false" on:mousedown={cropDown} />
          {#if cropping && cropBox}
            <div class="mp-cropbox" style="left:{cropBox.x}px;top:{cropBox.y}px;width:{cropBox.w}px;height:{cropBox.h}px"></div>
          {/if}
        </div>
      {/if}
    </div>
    {#if cur.kind === "image"}
      <div class="mp-edit">
        {#if cropping}
          <button class="mp-eb" on:click={() => { cropping = false; cropBox = null; }}>{$t("cancel")}</button>
          <button class="mp-eb on" on:click={applyCrop}>{$t("apply")}</button>
        {:else}
          <button class="mp-eb" title={$t("rotate")} on:click={() => rotate(-1)}>
            <svg viewBox="0 0 24 24"><path d="M3 12a9 9 0 1 0 3-6.7L3 8"/><path d="M3 3v5h5"/></svg>
          </button>
          <button class="mp-eb" title={$t("rotate")} on:click={() => rotate(1)}>
            <svg viewBox="0 0 24 24" style="transform:scaleX(-1)"><path d="M3 12a9 9 0 1 0 3-6.7L3 8"/><path d="M3 3v5h5"/></svg>
          </button>
          <button class="mp-eb" title={$t("crop")} on:click={() => { cropping = true; cropBox = null; }}>
            <svg viewBox="0 0 24 24"><path d="M6 2v14a2 2 0 0 0 2 2h14"/><path d="M2 6h14a2 2 0 0 1 2 2v14"/></svg>
          </button>
        {/if}
      </div>
    {/if}
    {#if items.length > 1}
      <div class="mp-strip">
        {#each items as it, i}
          <button class="mp-thumb {i === idx ? 'on' : ''}" on:click={() => (idx = i)}>
            {#if it.kind === "video"}<video src={it.dataURI} muted></video>
            {:else if it.kind === "document" && /\.(mp3|wav|ogg|m4a|aac|flac|opus)$/i.test(it.name)}<span class="mp-thumb-doc">🎵</span>
            {:else if it.kind === "document" && !/\.(png|jpe?g|gif|webp|bmp)$/i.test(it.name)}<span class="mp-thumb-doc">📄</span>
            {:else}<img src={it.dataURI} alt="" />{/if}
            {#if (captions[i] || "").trim()}<span class="mp-thumb-cap" title={$t("add_caption")}>💬</span>{/if}
          </button>
        {/each}
      </div>
    {/if}
    <div class="mp-bar">
      <input class="mp-caption" placeholder={$t("add_caption")} bind:value={captions[idx]} autofocus />
      <!-- Tombol sekali-lihat (SATU) di kanan area teks — hanya utk foto/video. -->
      {#if cur.kind !== "document"}
        <button class="mp-once {once ? 'on' : ''}" on:click={() => (once = !once)} title={$t("view_once")}>
          <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><text x="12" y="16" text-anchor="middle" font-size="11" font-weight="700" fill="currentColor" stroke="none">1</text></svg>
        </button>
      {/if}
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
  .mp-stage { flex:1; display:grid; align-items:center;justify-items:center; overflow:auto; padding:48px 16px 8px; }
  .mp-media { max-width:94vw; max-height:80vh; object-fit:contain; border-radius:8px; }
  /* Pratinjau dokumen: nama file di atas, isi (gambar/pdf/ikon) di bawah, bisa scroll. */
  .mp-doc { display:flex; flex-direction:column; align-items:center; gap:14px; width:100%; max-width:680px; }
  .mp-doc-name { align-self:stretch; text-align:center; color:#fff; font-weight:600; font-size:15px;
    background:rgba(255,255,255,.1); border-radius:10px; padding:10px 14px; word-break:break-all; }
  .mp-doc-rename { border:1px solid rgba(255,255,255,.18); outline:none; font-family:inherit; }
  .mp-doc-rename:focus { border-color:var(--accent); background:rgba(255,255,255,.16); }
  .mp-doc-img { max-width:90vw; max-height:70vh; object-fit:contain; border-radius:8px; }
  .mp-imgwrap { position:relative; display:inline-flex; }
  .mp-imgwrap.cropping .mp-media { cursor:crosshair; user-select:none; }
  .mp-cropbox { position:absolute; border:2px solid #fff; box-shadow:0 0 0 9999px rgba(0,0,0,.45); pointer-events:none; }
  .mp-edit { display:flex; gap:10px; justify-content:center; padding:6px 16px 0; }
  .mp-eb { display:flex; align-items:center; gap:6px; background:rgba(255,255,255,.14); color:#fff; border:0; border-radius:20px; padding:8px 14px; cursor:pointer; font:inherit; font-size:13.5px; }
  .mp-eb svg { width:20px; height:20px; fill:none; stroke:currentColor; stroke-width:2; }
  .mp-eb.on { background:var(--accent); }
  .mp-doc-ico { width:120px; height:120px; color:#fff; opacity:.85; }
  .mp-doc-ico svg { width:100%; height:100%; fill:none; stroke:currentColor; stroke-width:1.5; }
  .mp-doc-audio { width:min(90vw,420px); }
  .mp-thumb-doc { font-size:24px; }
  .mp-strip { display:flex; gap:8px; justify-content:center; padding:6px 16px; overflow-x:auto; }
  .mp-thumb { position:relative; width:54px; height:54px; border-radius:8px; overflow:hidden; border:2px solid transparent; padding:0; background:none; cursor:pointer; flex:0 0 auto; }
  .mp-thumb.on { border-color:var(--accent); }
  .mp-thumb img, .mp-thumb video { width:100%; height:100%; object-fit:cover; }
  .mp-thumb-cap { position:absolute; bottom:1px; right:2px; font-size:10px; line-height:1;
    filter:drop-shadow(0 1px 1px rgba(0,0,0,.6)); }
  .mp-bar { display:flex; align-items:center; gap:10px; padding:14px 18px 22px; max-width:760px; width:100%; margin:0 auto; }
  .mp-caption { flex:1; border:0; border-radius:22px; padding:12px 18px; background:var(--bg2,#1f2c34); color:var(--text,#e9edef); font:inherit; outline:none; }
  .mp-send { position:relative; width:48px; height:48px; border-radius:50%; border:0; background:var(--accent); color:#fff; cursor:pointer; display:grid; align-items:center;justify-items:center; flex:0 0 auto; }
  .mp-send svg { width:22px; height:22px; fill:currentColor; }
  .mp-count { position:absolute; top:-4px; right:-4px; background:#fff; color:var(--accent); font-size:11px; font-weight:700; border-radius:9px; min-width:18px; height:18px; display:grid; align-items:center;justify-items:center; }
  .mp-once { width:48px; height:48px; border-radius:50%; border:0; background:rgba(255,255,255,.14); color:#fff; cursor:pointer; display:flex; align-items:center; justify-content:center; flex:0 0 auto; }
  .mp-once svg { width:24px; height:24px; fill:none; stroke:currentColor; stroke-width:2; }
  .mp-once.on { background:var(--accent); color:#fff; }
</style>
