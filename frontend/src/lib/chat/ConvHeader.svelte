<script>
  import Avatar from "../common/Avatar.svelte";
  import { infoOpen, chatStatus, typingChats, inChatSearch, pinChat, muteChat, archiveChat, clearChatMessages, blockContact, leaveGroup, pushToast, askConfirm } from "../../stores.js";
  import { avatarUrl } from "../../services/data.js";
  import { t } from "../i18n.js";
  export let chat;
  const openInfo = () => infoOpen.set(true);
  $: typing = $typingChats[chat.id];
  $: live = $chatStatus[chat.id];
  // typing string (grup) → "Budi mengetik…"; true (1:1) → "mengetik…".
  $: typeLabel = typeof typing === "string" ? `${typing} ${$t("typing")}` : $t("typing");
  $: subtitle = typing ? typeLabel : live || (chat.group ? chat.members || chat.status : chat.status) || "";

  // Menu konteks header (dulu tombol mati).
  let menuOpen = false;
  function act(fn) { return () => { fn(); menuOpen = false; }; }
  function doClear() { menuOpen = false; askConfirm($t("clear_chat_confirm"), () => clearChatMessages(chat.id)); }
  function doBlock() { menuOpen = false; askConfirm($t("block_confirm", { name: chat.name }), () => { blockContact(chat.id, true); pushToast($t("blocked_toast").replace("%s", chat.name), "ok"); }); }
  function doLeave() { menuOpen = false; askConfirm($t("leave_confirm", { name: chat.name }), () => leaveGroup(chat.id)); }
</script>

<header class="conv-head">
  <div class="conv-peer" role="button" tabindex="0" on:click={openInfo} on:keydown={(e) => e.key === "Enter" && openInfo()}>
    <Avatar name={chat.name} color={chat.color} photo={avatarUrl(chat.id) || chat.photo} group={chat.group} sm={true} />
    <div class="conv-meta">
      <div class="conv-name">{chat.name}</div>
      <div class="conv-status {typing ? 'typing' : ''}">{subtitle}</div>
    </div>
  </div>
  <div class="conv-actions">
    <button class="icon-btn" title={$t("search")} on:click={() => inChatSearch.set(true)}>
      <svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/></svg>
    </button>
    <div class="hdr-menu-wrap">
      <button class="icon-btn" title={$t("menu")} on:click={() => (menuOpen = !menuOpen)}>
        <svg viewBox="0 0 24 24"><circle cx="12" cy="5" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="12" cy="19" r="1.6"/></svg>
      </button>
      {#if menuOpen}
        <div class="row-menu hdr-menu">
          <button class="mi" on:click={act(openInfo)}>{$t(chat.group ? "info_group" : "info_contact")}</button>
          <button class="mi" on:click={act(() => pinChat(chat.id, !chat.pinned))}>{chat.pinned ? $t("unpin") : $t("pin_msg")}</button>
          <button class="mi" on:click={act(() => muteChat(chat.id, !chat.muted))}>{chat.muted ? $t("unmute") : $t("mute")}</button>
          <button class="mi" on:click={act(() => archiveChat(chat.id, true))}>{$t("archived")}</button>
          <button class="mi" on:click={doClear}>{$t("clear_chat")}</button>
          {#if chat.group}
            <button class="mi danger" on:click={doLeave}>{$t("leave_group")}</button>
          {:else}
            <button class="mi danger" on:click={doBlock}>{$t("block", { name: chat.name })}</button>
          {/if}
        </div>
        <button class="menu-backdrop" aria-label={$t("close")} on:click={() => (menuOpen = false)}></button>
      {/if}
    </div>
  </div>
</header>

<style>
  .hdr-menu-wrap { position: relative; }
  .hdr-menu { position: absolute; top: 100%; right: 0; margin-top: 4px; z-index: 30; }
</style>
