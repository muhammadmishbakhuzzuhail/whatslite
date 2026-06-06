<script>
  import Avatar from "../common/Avatar.svelte";
  import Ticks from "../common/Ticks.svelte";
  import { activeChatId, railView, pinChat, muteChat, archiveChat, markChatUnread, removeChat, typingChats } from "../../stores.js";
  import { avatarUrl } from "../../services/data.js";
  import { t } from "../i18n.js";

  export let chat;
  $: active = $activeChatId === chat.id;
  $: typing = $typingChats[chat.id]; // dari event presence (bukan flag statis chat.typing)
  function open() {
    activeChatId.set(chat.id);
    railView.set("chats");
  }

  let menuOpen = false;
  function toggleMenu(e) { e.stopPropagation(); menuOpen = !menuOpen; }
  function act(fn) { return (e) => { e.stopPropagation(); fn(); menuOpen = false; }; }
</script>

<div
  class="chat-row {active ? 'active' : ''} {chat.unread ? 'unread' : ''}"
  role="button"
  tabindex="0"
  on:click={open}
  on:keydown={(e) => e.key === "Enter" && open()}
>
  <Avatar name={chat.name} color={chat.color} photo={avatarUrl(chat.id) || chat.photo} group={chat.group} />
  <div class="row-main">
    <div class="row-top">
      <span class="row-name">{chat.name}</span>
      <span class="row-time">{chat.time}</span>
      <button class="row-menu-btn" aria-label={$t("menu")} on:click={toggleMenu}>
        <svg viewBox="0 0 24 24"><path d="M7 10l5 5 5-5"/></svg>
      </button>
    </div>
    <div class="row-bottom">
      <span class="row-preview">
        {#if typing}
          <span class="typing">{typeof typing === "string" ? `${typing} ${$t("typing")}` : $t("typing")}</span>
        {:else}
          {#if chat.sent}<Ticks status={chat.status || "sent"} />{/if}
          <span>{chat.preview}</span>
        {/if}
      </span>
      {#if chat.badge}
        <span class="badge">{chat.badge}</span>
      {:else if chat.pinned || chat.muted}
        <span class="row-icons">
          {#if chat.muted}
            <span class="mute"><svg viewBox="0 0 24 24"><path d="M5 9v6h3l4 4V5L8 9H5z"/><path d="M16 8a5 5 0 0 1 0 8"/><path d="M3 3l18 18"/></svg></span>
          {/if}
          {#if chat.pinned}
            <span class="pin"><svg viewBox="0 0 24 24"><path d="M12 17v5M7 4h10l-1 6 3 3H5l3-3-1-6z"/></svg></span>
          {/if}
        </span>
      {/if}
    </div>
  </div>

  {#if menuOpen}
    <div class="row-menu">
      <button class="mi" on:click={act(() => pinChat(chat.id, !chat.pinned))}>{chat.pinned ? $t("unpin") : $t("pin_msg")}</button>
      <button class="mi" on:click={act(() => muteChat(chat.id, !chat.muted))}>{chat.muted ? $t("unmute") : $t("mute")}</button>
      <button class="mi" on:click={act(() => markChatUnread(chat.id, !chat.unread))}>{chat.unread ? $t("mark_read") : $t("mark_unread")}</button>
      <button class="mi" on:click={act(() => archiveChat(chat.id, true))}>{$t("archived")}</button>
      <button class="mi danger" on:click={act(() => removeChat(chat.id))}>{$t("delete")}</button>
    </div>
    <button class="menu-backdrop" aria-label={$t("close")} on:click={(e) => { e.stopPropagation(); menuOpen = false; }}></button>
  {/if}
</div>
