<script>
  import { lightbox } from "../../stores.js";
  import { t } from "../i18n.js";

  function close() { lightbox.set(null); }
  function onKey(e) { if ($lightbox && e.key === "Escape") close(); }
  async function save() {
    const lb = $lightbox;
    if (!lb) return;
    try {
      const blob = await fetch(lb.url).then((r) => r.blob());
      const ext = lb.type === "video" ? "mp4" : (blob.type.split("/")[1] || "jpg").split(";")[0];
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = `wa-${Date.now()}.${ext}`;
      a.click();
      setTimeout(() => URL.revokeObjectURL(a.href), 4000);
    } catch (e) {}
  }
</script>

<svelte:window on:keydown={onKey} />

{#if $lightbox}
  <div class="lb" on:click|self={close}>
    <button class="lb-save" title={$t("save")} on:click={save}>
      <svg viewBox="0 0 24 24"><path d="M12 4v11M7 11l5 5 5-5M5 20h14"/></svg>
    </button>
    <button class="lb-x" title={$t("close")} on:click={close}>✕</button>
    {#if $lightbox.type === "video"}
      <video class="lb-media" src={$lightbox.url} controls autoplay></video>
    {:else}
      <img class="lb-media" src={$lightbox.url} alt="" />
    {/if}
    {#if $lightbox.caption}<div class="lb-cap">{$lightbox.caption}</div>{/if}
  </div>
{/if}

<style>
  .lb { position:fixed; inset:0; z-index:80; background:rgba(0,0,0,.92); display:grid; place-items:center; }
  .lb-media { max-width:94vw; max-height:90vh; object-fit:contain; border-radius:6px; }
  .lb-x { position:absolute; top:18px; right:22px; background:rgba(255,255,255,.12); border:0; color:#fff; width:38px; height:38px; border-radius:50%; font-size:18px; cursor:pointer; }
  .lb-x:hover { background:rgba(255,255,255,.22); }
  .lb-save { position:absolute; top:18px; right:70px; background:rgba(255,255,255,.12); border:0; color:#fff; width:38px; height:38px; border-radius:50%; cursor:pointer; display:grid; place-items:center; }
  .lb-save svg { width:20px; height:20px; fill:none; stroke:currentColor; stroke-width:2; stroke-linecap:round; stroke-linejoin:round; }
  .lb-save:hover { background:rgba(255,255,255,.22); }
  .lb-cap { position:absolute; bottom:26px; left:0; right:0; text-align:center; color:#fff; font-size:14px; padding:0 24px; text-shadow:0 1px 4px rgba(0,0,0,.7); }
</style>
