<script>
  import { mediaDraft, sendMediaMessage } from "../../stores.js";
  import { t } from "../i18n.js";

  let caption = "";
  let last = null;
  // reset caption tiap draft baru
  $: if ($mediaDraft && $mediaDraft.dataURI !== last) { last = $mediaDraft.dataURI; caption = ""; }

  function close() { mediaDraft.set(null); caption = ""; }
  async function send() {
    const d = $mediaDraft;
    if (!d) return;
    mediaDraft.set(null);
    await sendMediaMessage(d.chatId, d.kind, caption.trim(), d.name || "", d.dataURI, d.viewOnce || false);
    caption = "";
  }
  function onKey(e) {
    if (!$mediaDraft) return;
    if (e.key === "Escape") close();
    else if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); send(); }
  }
</script>

<svelte:window on:keydown={onKey} />

{#if $mediaDraft}
  <div class="mp-overlay" on:click|self={close}>
    <button class="mp-x" title={$t("close")} on:click={close}>✕</button>
    <div class="mp-stage">
      {#if $mediaDraft.kind === "video"}
        <video class="mp-media" src={$mediaDraft.dataURI} controls></video>
      {:else}
        <img class="mp-media" src={$mediaDraft.dataURI} alt="" />
      {/if}
    </div>
    <div class="mp-bar">
      <input class="mp-caption" placeholder={$t("add_caption")} bind:value={caption} autofocus />
      <button class="mp-send" on:click={send} title={$t("send")}>
        <svg viewBox="0 0 24 24"><path d="M3 11l18-8-8 18-2-7-8-3z"/></svg>
      </button>
    </div>
  </div>
{/if}

<style>
  .mp-overlay { position:fixed; inset:0; z-index:70; background:rgba(11,20,26,.97); display:flex; flex-direction:column; }
  .mp-x { position:absolute; top:16px; left:18px; background:none; border:0; color:#fff; font-size:22px; cursor:pointer; z-index:2; }
  .mp-stage { flex:1; display:grid; place-items:center; overflow:hidden; padding:48px 16px 8px; }
  .mp-media { max-width:90vw; max-height:72vh; object-fit:contain; border-radius:8px; }
  .mp-bar { display:flex; align-items:center; gap:10px; padding:14px 18px 22px; max-width:760px; width:100%; margin:0 auto; }
  .mp-caption { flex:1; border:0; border-radius:22px; padding:12px 18px; background:var(--bg2,#1f2c34); color:var(--text,#e9edef); font:inherit; outline:none; }
  .mp-send { width:48px; height:48px; border-radius:50%; border:0; background:var(--accent); color:#fff; cursor:pointer; display:grid; place-items:center; flex:0 0 auto; }
  .mp-send svg { width:22px; height:22px; fill:currentColor; }
</style>
