<script>
  import Avatar from "../common/Avatar.svelte";
  import { infoOpen, chatStatus, typingChats, inChatSearch } from "../../stores.js";
  import { avatarUrl } from "../../services/data.js";
  import { t } from "../i18n.js";
  export let chat;
  const openInfo = () => infoOpen.set(true);
  $: typing = $typingChats[chat.id];
  $: live = $chatStatus[chat.id];
  $: subtitle = typing ? $t("typing") : live || (chat.group ? chat.members || chat.status : chat.status) || "";
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
    <button class="icon-btn" title={$t("video_call")}>
      <svg viewBox="0 0 24 24"><rect x="3" y="6" width="13" height="12" rx="2"/><path d="M16 10l5-3v10l-5-3z"/></svg>
    </button>
    <button class="icon-btn" title={$t("search")} on:click={() => inChatSearch.set(true)}>
      <svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/></svg>
    </button>
    <button class="icon-btn" title={$t("menu")}>
      <svg viewBox="0 0 24 24"><circle cx="12" cy="5" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="12" cy="19" r="1.6"/></svg>
    </button>
  </div>
</header>
