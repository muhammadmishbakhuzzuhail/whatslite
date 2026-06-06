<script>
  import { lightbox, pushToast } from "../../stores.js";
  import { t } from "../i18n.js";

  // Album: lightbox bisa membawa {items:[{url,type,caption}], i} → prev/next.
  let i = 0;
  $: items = $lightbox && $lightbox.items ? $lightbox.items : ($lightbox ? [$lightbox] : []);
  $: if ($lightbox && $lightbox.items) i = Math.min($lightbox.i || 0, items.length - 1);
  $: cur = items[i] || null;
  function prev() { if (i > 0) { i--; reset(); } }
  function next() { if (i < items.length - 1) { i++; reset(); } }

  // Zoom + geser (pan) untuk gambar.
  let scale = 1, tx = 0, ty = 0, dragging = false, sx = 0, sy = 0;
  function reset() { scale = 1; tx = 0; ty = 0; }
  function close() { lightbox.set(null); reset(); i = 0; }
  let _lastUrl = null;
  $: if (cur && cur.url !== _lastUrl) { _lastUrl = cur.url; reset(); }
  function onKey(e) {
    if (!$lightbox) return;
    if (e.key === "Escape") close();
    else if (e.key === "ArrowLeft") prev();
    else if (e.key === "ArrowRight") next();
  }
  function onWheel(e) {
    if (!$lightbox || $lightbox.type === "video") return;
    e.preventDefault();
    scale = Math.min(5, Math.max(1, scale * (e.deltaY < 0 ? 1.15 : 0.87)));
    if (scale === 1) { tx = 0; ty = 0; }
  }
  function dblZoom() { if (scale > 1) reset(); else scale = 2.5; }
  function down(e) { if (scale > 1) { dragging = true; sx = e.clientX - tx; sy = e.clientY - ty; } }
  function move(e) { if (dragging) { tx = e.clientX - sx; ty = e.clientY - sy; } }
  function up() { dragging = false; }
  async function save() {
    const lb = cur;
    if (!lb) return;
    try {
      const blob = await fetch(lb.url).then((r) => r.blob());
      const ext = lb.type === "video" ? "mp4" : (blob.type.split("/")[1] || "jpg").split(";")[0];
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = `wa-${Date.now()}.${ext}`;
      a.click();
      setTimeout(() => URL.revokeObjectURL(a.href), 4000);
    } catch (e) { pushToast($t("err_generic")); }
  }
</script>

<svelte:window on:keydown={onKey} />

{#if $lightbox}
  <div class="lb" on:click|self={close} on:wheel={onWheel} on:mousemove={move} on:mouseup={up} on:mouseleave={up}>
    <button class="lb-save" title={$t("save")} on:click={save}>
      <svg viewBox="0 0 24 24"><path d="M12 4v11M7 11l5 5 5-5M5 20h14"/></svg>
    </button>
    <button class="lb-x" title={$t("close")} on:click={close}>✕</button>
    {#if items.length > 1}
      {#if i > 0}<button class="lb-nav prev" on:click|stopPropagation={prev} aria-label="prev">‹</button>{/if}
      {#if i < items.length - 1}<button class="lb-nav next" on:click|stopPropagation={next} aria-label="next">›</button>{/if}
      <div class="lb-count">{i + 1}/{items.length}</div>
    {/if}
    {#if cur && cur.type === "video"}
      <video class="lb-media" src={cur.url} controls autoplay></video>
    {:else if cur}
      <img class="lb-media" src={cur.url} alt="" draggable="false"
        style="transform:translate({tx}px,{ty}px) scale({scale}); cursor:{scale > 1 ? (dragging ? 'grabbing' : 'grab') : 'zoom-in'}"
        on:dblclick={dblZoom} on:mousedown={down} />
    {/if}
    {#if cur && cur.caption}<div class="lb-cap">{cur.caption}</div>{/if}
  </div>
{/if}

<style>
  .lb { position:fixed; inset:0; z-index:80; background:rgba(0,0,0,.92); display:grid; align-items:center;justify-items:center; }
  .lb-media { max-width:94vw; max-height:90vh; object-fit:contain; border-radius:6px; }
  .lb-x { position:absolute; top:18px; right:22px; background:rgba(255,255,255,.12); border:0; color:#fff; width:38px; height:38px; border-radius:50%; font-size:18px; cursor:pointer; }
  .lb-x:hover { background:rgba(255,255,255,.22); }
  .lb-save { position:absolute; top:18px; right:70px; background:rgba(255,255,255,.12); border:0; color:#fff; width:38px; height:38px; border-radius:50%; cursor:pointer; display:grid; align-items:center;justify-items:center; }
  .lb-save svg { width:20px; height:20px; fill:none; stroke:currentColor; stroke-width:2; stroke-linecap:round; stroke-linejoin:round; }
  .lb-save:hover { background:rgba(255,255,255,.22); }
  .lb-cap { position:absolute; bottom:26px; left:0; right:0; text-align:center; color:#fff; font-size:14px; padding:0 24px; text-shadow:0 1px 4px rgba(0,0,0,.7); }
  .lb-nav { position:absolute; top:50%; transform:translateY(-50%); background:rgba(255,255,255,.12); border:0; color:#fff; width:46px; height:46px; border-radius:50%; font-size:28px; line-height:1; cursor:pointer; }
  .lb-nav:hover { background:rgba(255,255,255,.22); }
  .lb-nav.prev { left:18px; } .lb-nav.next { right:18px; }
  .lb-count { position:absolute; top:22px; left:0; right:0; text-align:center; color:#fff; font-size:13px; opacity:.8; }
</style>
