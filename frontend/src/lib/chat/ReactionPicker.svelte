<script>
  import { reactionTarget, reactMessage, theme } from "../../stores.js";
  import { t } from "../i18n.js";

  let ready = false;
  const W = 352, H = 400, GAP = 6;
  let top = 0, left = 0;

  $: if ($reactionTarget) place($reactionTarget);
  function place(tgt) {
    if (!ready) load();
    const vw = window.innerWidth, vh = window.innerHeight;
    const ax = tgt.x != null ? tgt.x : vw / 2;
    const ay = tgt.y != null ? tgt.y : vh / 2;
    const w = Math.min(W, vw * 0.92);
    // Default: di KANAN tombol; bila mepet tepi kanan → ke kiri tombol.
    let l = ax + GAP;
    if (l + w > vw - 8) l = ax - w - GAP * 4;
    left = Math.max(8, Math.min(l, vw - w - 8));
    // Vertikal: dekat tombol, dijepit ke viewport.
    top = Math.max(8, Math.min(ay - 40, vh - H - 8));
  }
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
  <div class="rp-pop" style="top:{top}px; left:{left}px">
    {#if ready}
      <emoji-picker class={$theme === "dark" ? "dark" : "light"} on:emoji-click={onPick}></emoji-picker>
    {:else}
      <div class="rp-load">…</div>
    {/if}
  </div>
{/if}

<style>
  .rp-backdrop { position: fixed; inset: 0; z-index: 75; background: transparent; border: 0; }
  /* Di-anchor ke tombol (kanan), bukan modal tengah → muncul di samping "+". */
  .rp-pop { position: fixed; z-index: 76; border-radius: 14px; box-shadow: var(--shadow-lg); overflow: hidden; }
  .rp-pop emoji-picker { width: min(352px, 92vw); height: 400px;
    --background: var(--bg); --border-color: var(--line); --emoji-size: 1.4rem; }
  .rp-load { background: var(--bg); padding: 30px; color: var(--text2); width: 300px; text-align: center; }
</style>
