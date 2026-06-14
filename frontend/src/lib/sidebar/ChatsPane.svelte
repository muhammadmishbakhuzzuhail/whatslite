<script>
  import SearchBar from "./SearchBar.svelte";
  import Filters from "./Filters.svelte";
  import ChatList from "./ChatList.svelte";
  import { t } from "../i18n.js";
  import { syncing, newChatOpen, railView, markAllRead, pushToast } from "../../stores.js";
  let menuOpen = false;
  function go(v) { railView.set(v); menuOpen = false; }
  function allRead() { markAllRead(); menuOpen = false; pushToast($t("all_read_done"), "ok"); }
</script>

<header class="sidebar-head">
  <h1>{$t("rail_chats")}{#if $syncing}<span class="sync-tag">{$t("syncing")}</span>{/if}</h1>
  <div class="head-actions">
    <button class="icon-btn" title={$t("new_chat")} on:click={() => newChatOpen.set(true)}>
      <svg viewBox="0 0 24 24"><path d="M12 5H7a3 3 0 0 0-3 3v9a3 3 0 0 0 3 3h9a3 3 0 0 0 3-3v-5"/><path d="M18.5 3.5a2.1 2.1 0 0 1 3 3L13 15l-4 1 1-4 8.5-8.5z"/></svg>
    </button>
    <div class="hdr-menu-wrap">
      <button class="icon-btn" title={$t("menu")} on:click={() => (menuOpen = !menuOpen)}>
        <svg viewBox="0 0 24 24"><circle cx="12" cy="5" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="12" cy="19" r="1.6"/></svg>
      </button>
      {#if menuOpen}
        <div class="row-menu hdr-menu">
          <button class="mi" on:click={() => { newChatOpen.set(true); menuOpen = false; }}>{$t("group_new")}</button>
          <button class="mi" on:click={allRead}>{$t("mark_all_read")}</button>
          <button class="mi" on:click={() => go("starred")}>{$t("starred_msg")}</button>
          <button class="mi" on:click={() => go("scheduled")}>{$t("scheduled_reminders")}</button>
          <button class="mi" on:click={() => go("archived")}>{$t("archived")}</button>
          <button class="mi" on:click={() => go("settings")}>{$t("rail_settings")}</button>
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

<SearchBar />
<Filters />

<ChatList />
