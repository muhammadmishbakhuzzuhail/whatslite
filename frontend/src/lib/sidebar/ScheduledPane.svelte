<script>
  import { onMount } from "svelte";
  import { railView, activeChatId } from "../../stores.js";
  import { getScheduled, cancelScheduled, getReminders, cancelReminder, onEvent } from "../../services/data.js";
  import { t } from "../i18n.js";

  let scheduled = [], reminders = [];
  async function load() { scheduled = await getScheduled(); reminders = await getReminders(); }
  onMount(() => {
    load();
    const o1 = onEvent("wa:scheduled", load);
    const o2 = onEvent("wa:reminders", load);
    return () => { o1 && o1(); o2 && o2(); };
  });
  function when(ts) {
    const d = new Date(ts * 1000);
    return d.toLocaleString([], { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });
  }
  function openChat(jid) { activeChatId.set(jid); railView.set("chats"); }
</script>

<header class="pane-head" style="gap:22px">
  <button class="icon-btn" title={$t("back")} on:click={() => railView.set("settings")}>
    <svg viewBox="0 0 24 24"><path d="M15 5l-7 7 7 7"/></svg>
  </button>
  <h2 style="font-size:17px">{$t("scheduled_reminders")}</h2>
</header>

<div style="flex:1; overflow-y:auto; padding:8px 0">
  <div class="sc-sec">{$t("schedule_msg")} ({scheduled.length})</div>
  {#each scheduled as s (s.id)}
    <div class="sc-row">
      <button class="sc-main" on:click={() => openChat(s.chatJid)}>
        <div class="sc-name">{s.chatName || s.chatJid.split("@")[0]}</div>
        <div class="sc-text">{s.text}</div>
        <div class="sc-when">⏰ {when(s.sendAt)}</div>
      </button>
      <button class="sc-x" title={$t("cancel")} on:click={() => cancelScheduled(s.id)}>✕</button>
    </div>
  {/each}
  {#if scheduled.length === 0}<div class="sc-empty">{$t("no_scheduled")}</div>{/if}

  <div class="sc-sec">{$t("reminders")} ({reminders.length})</div>
  {#each reminders as r (r.id)}
    <div class="sc-row">
      <button class="sc-main" on:click={() => openChat(r.chatJid)}>
        <div class="sc-name">{r.chatName || r.chatJid.split("@")[0]}</div>
        <div class="sc-text">{r.note || r.msgId}</div>
        <div class="sc-when">🔔 {when(r.remindAt)}</div>
      </button>
      <button class="sc-x" title={$t("cancel")} on:click={() => cancelReminder(r.id)}>✕</button>
    </div>
  {/each}
  {#if reminders.length === 0}<div class="sc-empty">{$t("no_reminders")}</div>{/if}
</div>

<style>
  .sc-sec { font-size:12px; font-weight:700; text-transform:uppercase; letter-spacing:.4px; color:var(--text2); padding:14px 16px 6px; }
  .sc-row { display:flex; align-items:center; gap:8px; padding:6px 12px; }
  .sc-main { flex:1; text-align:left; background:none; border:0; cursor:pointer; min-width:0; padding:6px 4px; }
  .sc-name { font-weight:600; color:var(--text); font-size:14px; }
  .sc-text { color:var(--text2); font-size:13px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
  .sc-when { color:var(--accent); font-size:12px; margin-top:2px; }
  .sc-x { background:var(--bg2); border:0; color:var(--text2); width:30px; height:30px; border-radius:50%; cursor:pointer; flex:0 0 auto; }
  .sc-empty { text-align:center; color:var(--text2); padding:14px; font-size:13px; }
</style>
