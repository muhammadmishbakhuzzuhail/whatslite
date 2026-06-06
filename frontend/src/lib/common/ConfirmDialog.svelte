<script>
  // Dialog konfirmasi global. confirm()/prompt() native TAK jalan di WebKitGTK,
  // jadi pakai modal inline. Dipicu via askConfirm(text, onYes) di stores.
  import { confirmDialog, resolveConfirm } from "../../stores.js";
  import { t } from "../i18n.js";
  function onKey(e) {
    if (!$confirmDialog) return;
    if (e.key === "Escape") resolveConfirm(false);
    else if (e.key === "Enter") resolveConfirm(true);
  }
</script>

<svelte:window on:keydown={onKey} />

{#if $confirmDialog}
  <button class="cf-backdrop" aria-label={$t("close")} on:click={() => resolveConfirm(false)}></button>
  <div class="cf-box" role="dialog" aria-modal="true">
    <div class="cf-text">{$confirmDialog.text}</div>
    <div class="cf-actions">
      <button class="btn-ghost" on:click={() => resolveConfirm(false)}>{$t("cancel")}</button>
      <button class="btn-accent" on:click={() => resolveConfirm(true)}>{$t("ok")}</button>
    </div>
  </div>
{/if}

<style>
  .cf-backdrop { position: fixed; inset: 0; z-index: 95; border: 0; background: rgba(0,0,0,.4); cursor: default; }
  .cf-box { position: fixed; z-index: 96; top: 50%; left: 50%; transform: translate(-50%, -50%);
    width: min(360px, 90vw); background: var(--bg); border: 1px solid var(--line);
    border-radius: 16px; box-shadow: 0 16px 50px rgba(0,0,0,.35); padding: 20px; }
  .cf-text { color: var(--text); line-height: 1.5; margin-bottom: 18px; }
  .cf-actions { display: flex; justify-content: flex-end; gap: 10px; }
  .cf-actions button { padding: 9px 18px; cursor: pointer; }
</style>
