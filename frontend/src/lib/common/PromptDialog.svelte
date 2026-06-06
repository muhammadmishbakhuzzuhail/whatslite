<script>
  // Dialog input teks global (prompt() native tak jalan di WebKitGTK).
  import { promptDialog, resolvePrompt } from "../../stores.js";
  import { t } from "../i18n.js";
  let val = "";
  let last = null;
  $: if ($promptDialog && $promptDialog !== last) { last = $promptDialog; val = $promptDialog.value || ""; }
  function ok() { resolvePrompt(true, val); }
  function onKey(e) {
    if (!$promptDialog) return;
    if (e.key === "Escape") resolvePrompt(false);
    else if (e.key === "Enter") ok();
  }
</script>

<svelte:window on:keydown={onKey} />

{#if $promptDialog}
  <button class="cf-backdrop" aria-label={$t("close")} on:click={() => resolvePrompt(false)}></button>
  <div class="cf-box" role="dialog" aria-modal="true">
    <div class="pd-title">{$promptDialog.title}</div>
    <!-- svelte-ignore a11y-autofocus -->
    <input class="pd-input" bind:value={val} autofocus on:keydown={(e) => e.key === "Enter" && ok()} />
    <div class="cf-actions">
      <button class="btn-ghost" on:click={() => resolvePrompt(false)}>{$t("cancel")}</button>
      <button class="btn-accent" on:click={ok} disabled={!val.trim()}>{$t("ok")}</button>
    </div>
  </div>
{/if}

<style>
  .cf-backdrop { position: fixed; inset: 0; z-index: 95; border: 0; background: rgba(0,0,0,.4); cursor: default; }
  .cf-box { position: fixed; z-index: 96; top: 50%; left: 50%; transform: translate(-50%, -50%);
    width: min(360px, 90vw); background: var(--bg); border: 1px solid var(--line);
    border-radius: 16px; box-shadow: 0 16px 50px rgba(0,0,0,.35); padding: 20px; }
  .pd-title { color: var(--text); font-weight: 600; margin-bottom: 12px; }
  .pd-input { width: 100%; box-sizing: border-box; border: 1px solid var(--line); border-radius: 10px;
    padding: 10px 12px; background: var(--bg2); color: var(--text); font: inherit; outline: none; margin-bottom: 16px; }
  .cf-actions { display: flex; justify-content: flex-end; gap: 10px; }
  .cf-actions button { padding: 9px 18px; cursor: pointer; }
</style>
