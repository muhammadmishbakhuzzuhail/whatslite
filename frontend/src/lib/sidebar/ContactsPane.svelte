<script>
  // ContactsPane — daftar kontak (buku-alamat + label lokal). Klik → buka chat.
  import { onMount } from "svelte";
  import { messageContact, openProfile } from "../../stores.js";
  import { getContacts, avatarUrl, senderColorFor } from "../../services/data.js";
  import { initial } from "../util.js";
  import { t } from "../i18n.js";

  let contacts = [];
  let q = "";
  onMount(load);
  async function load() { contacts = await getContacts(); }
  $: filtered = contacts.filter((c) =>
    (c.name || "").toLowerCase().includes(q.trim().toLowerCase()) ||
    (c.phone || "").includes(q.trim())
  );
</script>

<header class="pane-head"><h2>{$t("rail_contacts")}</h2></header>
<div class="ct-search-wrap">
  <input class="ct-search" placeholder={$t("search")} bind:value={q} />
</div>
<div class="ct-list">
  {#each filtered as c (c.jid)}
    <div class="ct-row" role="button" tabindex="0"
      on:click={() => messageContact(c.jid)}
      on:keydown={(e) => e.key === "Enter" && messageContact(c.jid)}>
      {#if avatarUrl(c.jid)}
        <img class="avatar sm photo" src={avatarUrl(c.jid)} alt={c.name} on:error={(e) => (e.target.style.display = 'none')} />
      {:else}
        <div class="avatar sm" style="--c:{senderColorFor(c.jid)}"><span>{initial(c.name)}</span></div>
      {/if}
      <div class="ct-meta">
        <div class="ct-name">{c.name}</div>
        {#if c.phone && !c.saved}<div class="ct-sub">{c.phone}</div>{/if}
      </div>
      <button class="ct-info" title={$t("info_contact")} on:click|stopPropagation={() => openProfile(c.jid)}>
        <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M12 11v5"/><circle cx="12" cy="7.5" r=".6"/></svg>
      </button>
    </div>
  {/each}
  {#if filtered.length === 0}<div class="ct-empty">{$t("no_match")}</div>{/if}
</div>

<style>
  .ct-search-wrap { padding: 6px 12px 10px; }
  .ct-search { width: 100%; border: 0; border-radius: 10px; padding: 9px 14px; background: var(--bg2); color: var(--text); font: inherit; outline: none; }
  .ct-list { flex: 1; overflow-y: auto; }
  .ct-row { display: flex; align-items: center; gap: 12px; padding: 8px 14px; cursor: pointer; }
  .ct-row:hover { background: var(--hover); }
  .ct-meta { flex: 1; min-width: 0; }
  .ct-name { font-size: 15px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .ct-sub { font-size: 12.5px; color: var(--text2); }
  .ct-info { background: none; border: 0; color: var(--text2); cursor: pointer; padding: 6px; border-radius: 50%; flex: 0 0 auto; }
  .ct-info:hover { background: var(--bg2); color: var(--accent); }
  .ct-info svg { width: 20px; height: 20px; fill: none; stroke: currentColor; stroke-width: 2; }
  .ct-empty { text-align: center; color: var(--text2); padding: 40px 16px; }
</style>
