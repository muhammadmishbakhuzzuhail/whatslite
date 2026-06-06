<script>
  import { onMount } from "svelte";
  import { railView } from "../../stores.js";
  import { getStorageUsage } from "../../services/data.js";
  import { t } from "../i18n.js";

  let u = null, loading = true;
  onMount(async () => { u = await getStorageUsage(); loading = false; });

  function fmt(b) {
    if (!b) return "0 B";
    const k = 1024;
    if (b < k) return b + " B";
    if (b < k * k) return (b / k).toFixed(0) + " KB";
    if (b < k * k * k) return (b / k / k).toFixed(1) + " MB";
    return (b / k / k / k).toFixed(2) + " GB";
  }
  $: total = u ? u.dbBytes + u.mediaBytes : 0;
  const label = { text: "t_text", image: "t_photo", video: "t_video", sticker: "t_sticker", voice: "t_voice", gif: "t_video", document: "attach_document" };
  function kindName(k) { return label[k] ? $t(label[k]) : k; }
</script>

<header class="pane-head" style="gap:22px">
  <button class="icon-btn" title={$t("back")} on:click={() => railView.set("settings")}>
    <svg viewBox="0 0 24 24"><path d="M15 5l-7 7 7 7"/></svg>
  </button>
  <h2 style="font-size:17px">{$t("storage")}</h2>
</header>

<div style="flex:1; overflow-y:auto; padding:8px 4px">
  {#if loading}
    <div class="su-empty">…</div>
  {:else if u}
    <div class="su-totals">
      <div class="su-total">{fmt(total)}<span>{$t("storage")}</span></div>
      <div class="su-split">
        <div><b>{fmt(u.dbBytes)}</b><span>{$t("messages_label")}</span></div>
        <div><b>{fmt(u.mediaBytes)}</b><span>{$t("info_media")}</span></div>
      </div>
      <div class="su-count">{u.msgCount.toLocaleString()} {$t("messages_label")}</div>
    </div>
    <div class="su-list">
      {#each u.kinds as k}
        <div class="su-row">
          <span class="su-k">{kindName(k.kind)}</span>
          <span class="su-bar"><span style="width:{total ? Math.max(2, (k.bytes / total) * 100) : 0}%"></span></span>
          <span class="su-b">{fmt(k.bytes)} · {k.count}</span>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .su-empty { text-align:center; color:var(--text2); padding:40px; }
  .su-totals { text-align:center; padding:18px 16px; }
  .su-total { font-size:30px; font-weight:700; color:var(--text); display:flex; flex-direction:column; gap:2px; }
  .su-total span { font-size:13px; font-weight:400; color:var(--text2); }
  .su-split { display:flex; justify-content:center; gap:28px; margin-top:14px; }
  .su-split div { display:flex; flex-direction:column; }
  .su-split b { color:var(--accent); font-size:16px; }
  .su-split span { font-size:12px; color:var(--text2); }
  .su-count { margin-top:10px; font-size:12.5px; color:var(--text2); }
  .su-list { padding:8px 16px; }
  .su-row { display:flex; align-items:center; gap:10px; padding:8px 0; }
  .su-k { width:90px; font-size:13px; color:var(--text); }
  .su-bar { flex:1; height:8px; border-radius:5px; background:var(--bg2); overflow:hidden; }
  .su-bar span { display:block; height:100%; background:var(--accent); }
  .su-b { font-size:12px; color:var(--text2); white-space:nowrap; }
</style>
