<script>
  import { onMount } from "svelte";
  import Avatar from "../common/Avatar.svelte";
  import { calls, refreshCalls, activeChatId, railView } from "../../stores.js";
  import { avatarUrl } from "../../services/data.js";
  import { t } from "../i18n.js";

  let loading = true;
  onMount(async () => { await refreshCalls(); loading = false; });

  function when(ts) {
    if (!ts) return "";
    const d = new Date(ts * 1000), now = new Date();
    const day = (x) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
    const diff = Math.round((day(now) - day(d)) / 86400000);
    const hm = d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
    if (diff <= 0) return hm;
    if (diff === 1) return `${$t("yesterday")} ${hm}`;
    return d.toLocaleDateString(undefined, { day: "numeric", month: "short" });
  }
  function openChat(c) { activeChatId.set(c.jid); railView.set("chats"); }
</script>

<header class="sidebar-head"><h1>{$t("rail_calls")}</h1></header>

<div class="chat-list">
  {#if loading}
    <div class="ct-empty">…</div>
  {:else if $calls.length === 0}
    <div class="ct-empty">{$t("calls_empty")}</div>
  {:else}
    {#each $calls as c (c.id)}
      <div class="chat-row" role="button" tabindex="0" on:click={() => openChat(c)} on:keydown={(e) => e.key === "Enter" && openChat(c)}>
        <Avatar name={c.name} photo={avatarUrl(c.jid)} group={c.group} />
        <div class="row-main">
          <div class="row-top">
            <span class="row-name">{c.name}</span>
            <span class="row-time">{when(c.ts)}</span>
          </div>
          <div class="row-bottom">
            <span class="row-preview call-line {c.status === 'missed' ? 'missed' : ''}">
              <svg class="call-ico" viewBox="0 0 24 24"><path d="M7 17L17 7M17 7H9M17 7v8"/></svg>
              {c.video ? $t("call_video") : $t("call_voice")} · {c.status === "missed" ? $t("call_missed") : $t("call_rejected")}
            </span>
          </div>
        </div>
      </div>
    {/each}
  {/if}
</div>

<style>
  .call-line { display: flex; align-items: center; gap: 6px; }
  .call-line.missed { color: #ef5350; }
  .call-ico { width: 15px; height: 15px; fill: none; stroke: currentColor; stroke-width: 2; transform: rotate(0deg); }
  .ct-empty { text-align: center; color: var(--text2); padding: 40px 16px; }
</style>
