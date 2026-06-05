<script>
  import { onMount } from "svelte";
  import { infoMsg } from "../../stores.js";
  import { getMessageInfo, onEvent } from "../../services/data.js";
  import { t } from "../i18n.js";

  let info = null;
  let loading = false;

  $: if ($infoMsg) loadInfo($infoMsg);
  async function loadInfo(ref) {
    loading = true; info = null;
    info = await getMessageInfo(ref.chat, ref.id);
    loading = false;
  }
  function close() { infoMsg.set(null); }

  // Receipt baru saat modal terbuka → muat ulang diam-diam (live update daftar baca).
  onMount(() => {
    onEvent("wa:receipt", (e) => {
      const ref = $infoMsg;
      if (!ref || !e || e.chat !== ref.chat) return;
      if (e.ids && e.ids.length && !e.ids.includes(ref.id)) return;
      getMessageInfo(ref.chat, ref.id).then((d) => { if ($infoMsg && d) info = d; });
    });
  });

  const statusLabel = { sent: "status_sent", delivered: "status_delivered", read: "status_read" };
  const typeKey = { text: "t_text", image: "t_photo", video: "t_video", sticker: "t_sticker", voice: "t_voice" };
</script>

{#if $infoMsg}
  <div class="nc-modal" on:click|self={close}>
    <div class="nc-card" style="max-width:380px">
      <h3 style="margin:0 0 14px">{$t("msg_info")}</h3>
      {#if loading}
        <div style="color:var(--text2);padding:12px 0">…</div>
      {:else if info}
        <div class="mi-grid">
          <div class="mi-k">{$t("mi_status")}</div>
          <div class="mi-v">
            <span class="mi-dot {info.status}"></span>
            {$t(statusLabel[info.status] || "status_sent")}
          </div>
          <div class="mi-k">{$t("mi_type")}</div>
          <div class="mi-v">{typeKey[info.type] ? $t(typeKey[info.type]) : info.type}</div>
          <div class="mi-k">{$t("mi_sent")}</div>
          <div class="mi-v">{info.sent}</div>
          {#if info.sender && !info.fromMe}
            <div class="mi-k">{$t("mi_from")}</div>
            <div class="mi-v">{info.sender}</div>
          {/if}
          <div class="mi-k">ID</div>
          <div class="mi-v" style="font-family:monospace;font-size:11px;word-break:break-all">{info.id}</div>
        </div>
        {#if info.fromMe && (info.readBy?.length || info.deliveredTo?.length)}
          {#if info.readBy?.length}
            <div class="mi-sec"><span class="mi-dot read"></span>{$t("mi_read_by")}</div>
            {#each info.readBy as r}<div class="mi-rcpt"><span>{r.name}</span><span class="mi-rt">{r.time}</span></div>{/each}
          {/if}
          {#if info.deliveredTo?.length}
            <div class="mi-sec"><span class="mi-dot delivered"></span>{$t("mi_delivered_to")}</div>
            {#each info.deliveredTo as r}<div class="mi-rcpt"><span>{r.name}</span><span class="mi-rt">{r.time}</span></div>{/each}
          {/if}
        {:else if info.fromMe}
          <p class="mi-note">{$t("mi_note")}</p>
        {/if}
      {:else}
        <div style="color:var(--text2)">{$t("mi_unavailable")}</div>
      {/if}
      <div style="display:flex;justify-content:flex-end;margin-top:16px">
        <button class="btn-accent" on:click={close}>{$t("close")}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .mi-grid { display:grid; grid-template-columns:auto 1fr; gap:8px 16px; align-items:baseline; }
  .mi-k { color:var(--text2); font-size:13px; }
  .mi-v { font-size:14px; display:flex; align-items:center; gap:7px; }
  .mi-dot { width:8px; height:8px; border-radius:50%; background:var(--text2); }
  .mi-dot.delivered { background:#3d8bd3; }
  .mi-dot.read { background:var(--accent); }
  .mi-note { font-size:12px; color:var(--text2); margin:14px 0 0; line-height:1.5; }
  .mi-sec { display:flex; align-items:center; gap:7px; font-size:12px; font-weight:600; color:var(--text2); margin:14px 0 6px; text-transform:uppercase; letter-spacing:.4px; }
  .mi-rcpt { display:flex; justify-content:space-between; font-size:13.5px; padding:3px 0; }
  .mi-rt { color:var(--text2); font-size:12px; }
</style>
