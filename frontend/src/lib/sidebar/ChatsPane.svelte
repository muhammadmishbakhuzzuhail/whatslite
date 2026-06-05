<script>
  import SearchBar from "./SearchBar.svelte";
  import Filters from "./Filters.svelte";
  import ChatList from "./ChatList.svelte";
  import { t } from "../i18n.js";
  import { syncing, newChatOpen } from "../../stores.js";
  let showNotif = true;
</script>

<header class="sidebar-head">
  <h1>{$t("rail_chats")}{#if $syncing}<span class="sync-tag">{$t("syncing")}</span>{/if}</h1>
  <div class="head-actions">
    <button class="icon-btn" title={$t("new_chat")} on:click={() => newChatOpen.set(true)}>
      <svg viewBox="0 0 24 24"><path d="M12 5H7a3 3 0 0 0-3 3v9a3 3 0 0 0 3 3h9a3 3 0 0 0 3-3v-5"/><path d="M18.5 3.5a2.1 2.1 0 0 1 3 3L13 15l-4 1 1-4 8.5-8.5z"/></svg>
    </button>
    <button class="icon-btn" title={$t("menu")}>
      <svg viewBox="0 0 24 24"><circle cx="12" cy="5" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="12" cy="19" r="1.6"/></svg>
    </button>
  </div>
</header>

<SearchBar />
<Filters />

{#if showNotif}
  <div class="notif-banner">
    <svg viewBox="0 0 24 24"><path d="M6 8a6 6 0 0 1 12 0c0 5 2 6 2 6H4s2-1 2-6z"/><path d="M10 20a2 2 0 0 0 4 0"/><path d="M3 3l18 18"/></svg>
    <div class="nb-text">{$t("notif_off")} <span class="nb-link">{$t("notif_turnon")}</span></div>
    <button class="nb-close" title={$t("close")} on:click={() => (showNotif = false)}>
      <svg viewBox="0 0 24 24"><path d="M6 6l12 12M18 6L6 18"/></svg>
    </button>
  </div>
{/if}

<ChatList />
