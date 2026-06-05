<script>
  import { onMount } from "svelte";
  import ChatRow from "./ChatRow.svelte";
  import { railView } from "../../stores.js";
  import { getArchivedChats } from "../../services/data.js";
  import { t } from "../i18n.js";

  let items = [];
  let loading = true;
  onMount(async () => { items = await getArchivedChats(); loading = false; });
</script>

<header class="pane-head" style="gap:14px">
  <button class="icon-btn" title={$t("back")} on:click={() => railView.set("chats")}>
    <svg viewBox="0 0 24 24"><path d="M15 5l-7 7 7 7"/></svg>
  </button>
  <h2 style="font-size:17px">{$t("archived_chats")}</h2>
</header>

<div class="chat-list" style="flex:1;overflow-y:auto">
  {#if loading}
    <div class="empty-list" style="text-align:center;padding:24px;color:var(--text2)">…</div>
  {:else}
    {#each items as c (c.id)}<ChatRow chat={c} />{/each}
    {#if items.length === 0}
      <div class="empty-list" style="text-align:center;padding:28px 16px;color:var(--text2)">{$t("no_archived")}</div>
    {/if}
  {/if}
</div>
