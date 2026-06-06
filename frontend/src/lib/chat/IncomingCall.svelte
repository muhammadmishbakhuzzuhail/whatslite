<script>
  // Banner panggilan masuk. whatsmeow tak punya media call → tak bisa menjawab;
  // hanya bisa MENOLAK atau menutup banner (panggilan tetap berdering di HP).
  import Avatar from "../common/Avatar.svelte";
  import { incomingCall, rejectIncomingCall, dismissIncomingCall } from "../../stores.js";
  import { avatarUrl } from "../../services/data.js";
  import { t } from "../i18n.js";

  let timer = null, last = null;
  // Auto-tutup setelah 35 dtk (panggilan biasanya berhenti berdering).
  $: if ($incomingCall && $incomingCall !== last) {
    last = $incomingCall;
    clearTimeout(timer);
    timer = setTimeout(() => dismissIncomingCall(), 35000);
  }
</script>

{#if $incomingCall}
  <div class="ic-wrap" role="dialog" aria-modal="true" aria-label={$t("call_incoming")}>
    <Avatar name={$incomingCall.name} photo={avatarUrl($incomingCall.jid)} group={$incomingCall.group} />
    <div class="ic-meta">
      <div class="ic-name">{$incomingCall.name}</div>
      <div class="ic-sub">{$incomingCall.video ? $t("call_video") : $t("call_voice")} · {$t("call_incoming")}</div>
    </div>
    <div class="ic-actions">
      <button class="ic-btn reject" title={$t("call_reject")} on:click={() => rejectIncomingCall($incomingCall)}>
        <svg viewBox="0 0 24 24"><path d="M5 4h3l2 5-2.5 1.5a11 11 0 0 0 5 5L15 13l5 2v3a2 2 0 0 1-2 2A16 16 0 0 1 3 6a2 2 0 0 1 2-2z"/></svg>
      </button>
      <button class="ic-btn close" title={$t("close")} on:click={dismissIncomingCall}>
        <svg viewBox="0 0 24 24"><path d="M6 6l12 12M18 6L6 18"/></svg>
      </button>
    </div>
  </div>
{/if}

<style>
  .ic-wrap { position: fixed; top: 18px; left: 50%; transform: translateX(-50%); z-index: 90;
    display: flex; align-items: center; gap: 12px; min-width: 340px; max-width: 92vw;
    background: var(--bg); border: 1px solid var(--line); border-radius: 16px;
    box-shadow: 0 12px 40px rgba(0,0,0,.32); padding: 12px 14px; }
  .ic-meta { flex: 1; min-width: 0; }
  .ic-name { font-weight: 700; color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .ic-sub { font-size: 12.5px; color: var(--text2); }
  .ic-actions { display: flex; gap: 8px; }
  .ic-btn { width: 42px; height: 42px; border-radius: 50%; border: 0; cursor: pointer;
    display: flex; align-items: center; justify-content: center; }
  .ic-btn svg { width: 20px; height: 20px; }
  .ic-btn.reject { background: #ef5350; color: #fff; }
  .ic-btn.reject svg { fill: currentColor; }
  .ic-btn.close { background: var(--bg2); color: var(--text2); }
  .ic-btn.close svg { fill: none; stroke: currentColor; stroke-width: 2; }
</style>
