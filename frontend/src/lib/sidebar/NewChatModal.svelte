<script>
  import Avatar from "../common/Avatar.svelte";
  import { chats, newChatOpen, activeChatId, railView, createGroup } from "../../stores.js";
  import { avatarUrl } from "../../services/data.js";
  import { t } from "../i18n.js";

  let mode = "list"; // "list" (mulai chat) | "group" (buat grup)
  let q = "", name = "", selected = new Set();

  $: contacts = $chats.filter((c) => !c.group); // kandidat: chat 1:1
  $: filtered = contacts.filter((c) => (c.name || "").toLowerCase().includes(q.toLowerCase()));

  function close() { newChatOpen.set(false); mode = "list"; q = ""; name = ""; selected = new Set(); }
  function pick(c) {
    if (mode === "group") { toggle(c.id); return; }
    activeChatId.set(c.id); railView.set("chats"); close();
  }
  function toggle(id) {
    selected.has(id) ? selected.delete(id) : selected.add(id);
    selected = new Set(selected);
  }
  async function makeGroup() {
    if (!name.trim() || selected.size === 0) return;
    const jid = await createGroup(name.trim(), [...selected]);
    if (jid) { activeChatId.set(jid); railView.set("chats"); }
    close();
  }
</script>

{#if $newChatOpen}
  <button class="modal-backdrop" aria-label={$t("close")} on:click={close}></button>
  <div class="nc-modal" role="dialog">
    <div class="nc-head">
      <span>{mode === "group" ? "Buat grup" : $t("new_chat")}</span>
      <button class="icon-btn" on:click={close} aria-label={$t("close")}>
        <svg viewBox="0 0 24 24"><path d="M6 6l12 12M18 6L6 18"/></svg>
      </button>
    </div>

    {#if mode === "list"}
      <button class="nc-action" on:click={() => (mode = "group")}>
        <span class="nc-ico"><svg viewBox="0 0 24 24"><circle cx="9" cy="8" r="3"/><path d="M3 19a6 6 0 0 1 12 0"/><path d="M17 7v6M14 10h6"/></svg></span>
        Grup baru
      </button>
    {:else}
      <input class="nc-name" placeholder="Nama grup" bind:value={name} />
    {/if}

    <input class="nc-search" placeholder={$t("search")} bind:value={q} />

    <div class="nc-list">
      {#each filtered as c (c.id)}
        <button class="nc-row" on:click={() => pick(c)}>
          <Avatar name={c.name} color={c.color} photo={avatarUrl(c.id) || c.photo} sm={true} />
          <span class="nc-cname">{c.name}</span>
          {#if mode === "group"}
            <span class="nc-check" class:on={selected.has(c.id)}></span>
          {/if}
        </button>
      {/each}
      {#if filtered.length === 0}<div class="nc-empty">{$t("no_match")}</div>{/if}
    </div>

    {#if mode === "group"}
      <button class="nc-create" disabled={!name.trim() || selected.size === 0} on:click={makeGroup}>
        Buat grup ({selected.size})
      </button>
    {/if}
  </div>
{/if}
