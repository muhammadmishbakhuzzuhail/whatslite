<script>
  import { toasts } from "../stores.js";
</script>

{#if $toasts.length}
  <div class="toast-stack">
    {#each $toasts as t (t.id)}
      <div class="toast {t.type}">{t.text}</div>
    {/each}
  </div>
{/if}

<style>
  .toast-stack { position: fixed; bottom: 22px; left: 50%; transform: translateX(-50%);
    z-index: 60; display: flex; flex-direction: column; gap: 8px; align-items: center; }
  .toast { background: #2a2f36; color: #fff; padding: 10px 16px; border-radius: 999px;
    font-size: 13.5px; box-shadow: 0 6px 20px rgba(0,0,0,.3); max-width: 70vw;
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .toast.error { background: #c0392b; }
  .toast.ok { background: #2e7d52; }
</style>
