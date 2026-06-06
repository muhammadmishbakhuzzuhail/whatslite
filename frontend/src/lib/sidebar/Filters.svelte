<script>
  import { filter, chats, folders, addFolder, askPrompt } from "../../stores.js";
  import { t } from "../i18n.js";
  // WhatsApp menampilkan jumlah di chip Unread & Groups ("Unread 90", "Groups 56").
  $: unread = $chats.filter((c) => c.unread).length;
  $: groups = $chats.filter((c) => c.group).length;
  const set = (v) => filter.set(v);
  function newFolder() { askPrompt($t("folder_new"), "", (name) => { addFolder(name); set("folder:" + name); }); }
</script>

<div class="filters">
  <button class="chip {$filter === 'Semua' ? 'active' : ''}" on:click={() => set("Semua")}>{$t("filter_all")}</button>
  <button class="chip {$filter === 'Belum dibaca' ? 'active' : ''}" on:click={() => set("Belum dibaca")}>
    {$t("filter_unread")}{#if unread}<span class="chip-n">{unread}</span>{/if}
  </button>
  <button class="chip {$filter === 'Favorit' ? 'active' : ''}" on:click={() => set("Favorit")}>{$t("filter_favorites")}</button>
  <button class="chip {$filter === 'Grup' ? 'active' : ''}" on:click={() => set("Grup")}>
    {$t("filter_groups")}{#if groups}<span class="chip-n">{groups}</span>{/if}
  </button>
  {#each $folders as f (f.name)}
    <button class="chip {$filter === 'folder:' + f.name ? 'active' : ''}" on:click={() => set("folder:" + f.name)}>{f.name}</button>
  {/each}
  <button class="chip plus" title={$t("folder_new")} on:click={newFolder}>+</button>
</div>
