<script>
  import { onMount } from "svelte";
  import { railView, activeChatId, jumpMsg } from "../../stores.js";
  import { getStarred, colorFor } from "../../services/data.js";
  import { t } from "../i18n.js";

  let items = [];
  onMount(async () => { items = await getStarred(); });
  function open(h) { activeChatId.set(h.chatJid); railView.set("chats"); jumpMsg.set(h.msgId); }
  function initial(s) { for (const c of s || "") if (/[\p{L}\p{N}]/u.test(c)) return c.toUpperCase(); return "?"; }
</script>

<header class="pane-head" style="gap:22px">
  <button class="icon-btn" title={$t("back")} on:click={() => railView.set("settings")}>
    <svg viewBox="0 0 24 24"><path d="M15 5l-7 7 7 7"/></svg>
  </button>
  <h2 style="font-size:17px">{$t("starred_msg")}</h2>
</header>

<div class="chat-list" style="padding:4px 6px">
  {#each items as h (h.chatJid + h.msgId)}
    <button class="hit-row" on:click={() => open(h)}>
      <span class="hit-av" style="background:{colorFor(h.chatJid)}">{initial(h.chatName)}</span>
      <span class="hit-main">
        <span class="hit-top"><span class="hit-name">{h.chatName}</span><span class="hit-time">{h.time}</span></span>
        <span class="hit-text">⭐ {h.text || `(${$t("media_generic")})`}</span>
      </span>
    </button>
  {/each}
  {#if items.length === 0}
    <div class="empty-list" style="padding:28px 16px;text-align:center;color:var(--text2)">{$t("no_starred")}</div>
  {/if}
</div>
