<script>
  import { reactionTarget, reactMessage, theme } from "../../stores.js";
  import { t } from "../i18n.js";

  let ready = false;
  $: if ($reactionTarget && !ready) load();
  async function load() { await import("emoji-picker-element"); ready = true; }

  function close() { reactionTarget.set(null); }
  function onPick(e) {
    const tgt = $reactionTarget;
    if (tgt && e.detail?.unicode) reactMessage(tgt.chatId, tgt.idx, e.detail.unicode);
    close();
  }
  function onKey(e) { if ($reactionTarget && e.key === "Escape") close(); }
</script>

<svelte:window on:keydown={onKey} />

{#if $reactionTarget}
  <button class="rp-backdrop" aria-label={$t("close")} on:click={close}></button>
  <div class="rp-pop">
    {#if ready}
      <emoji-picker class={$theme === "dark" ? "dark" : "light"} on:emoji-click={onPick}></emoji-picker>
    {:else}
      <div class="rp-load">…</div>
    {/if}
  </div>
{/if}

<style>
  .rp-backdrop { position: fixed; inset: 0; z-index: 75; background: transparent; border: 0; }
  .rp-pop { position: fixed; z-index: 76; top: 50%; left: 50%; transform: translate(-50%, -50%);
    border-radius: 14px; box-shadow: var(--shadow-lg); overflow: hidden; }
  .rp-pop emoji-picker { width: min(352px, 92vw); height: 400px;
    --background: var(--bg); --border-color: var(--line); --emoji-size: 1.4rem; }
  .rp-load { background: var(--bg); padding: 30px; color: var(--text2); width: 300px; text-align: center; }
</style>
