<script>
  import Avatar from "../common/Avatar.svelte";
  import { chats, newChatOpen, activeChatId, railView, createGroup, pushToast } from "../../stores.js";
  import { avatarUrl, joinGroupLink, isOnWhatsApp, addViaQR, fetchProfile } from "../../services/data.js";
  import { onMount } from "svelte";
  import { t } from "../i18n.js";

  let mode = "list"; // "list" | "group" | "join" (gabung via tautan) | "number" (chat via nomor)
  let q = "", name = "", selected = new Set();
  let linkVal = "", numVal = "", qrVal = "", busy = false, selfJid = "";
  onMount(async () => { const p = await fetchProfile(); selfJid = p && p.jid ? p.jid : ""; });
  function noteToSelf() { if (selfJid) { activeChatId.set(selfJid); railView.set("chats"); close(); } }
  async function doQR() {
    if (!qrVal.trim()) return;
    busy = true;
    const jid = await addViaQR(qrVal.trim());
    busy = false;
    if (jid) { activeChatId.set(jid); railView.set("chats"); close(); }
    else pushToast($t("qr_invalid"));
  }

  async function doJoin() {
    if (!linkVal.trim()) return;
    busy = true;
    const jid = await joinGroupLink(linkVal.trim());
    busy = false;
    if (jid) { activeChatId.set(jid); railView.set("chats"); close(); }
    else pushToast($t("join_failed"));
  }
  async function doNumber() {
    const digits = numVal.replace(/[^0-9]/g, "");
    if (digits.length < 8) return;
    busy = true;
    const res = await isOnWhatsApp([digits]);
    busy = false;
    const r = res && res[0];
    if (r && r.registered && r.jid) { activeChatId.set(r.jid); railView.set("chats"); close(); }
    else pushToast($t("number_not_wa"));
  }

  $: contacts = $chats.filter((c) => !c.group); // kandidat: chat 1:1
  $: filtered = contacts.filter((c) => (c.name || "").toLowerCase().includes(q.toLowerCase()));

  function close() { newChatOpen.set(false); mode = "list"; q = ""; name = ""; selected = new Set(); linkVal = ""; numVal = ""; qrVal = ""; }
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
      <span>{mode === "group" ? $t("group_create") : $t("new_chat")}</span>
      <button class="icon-btn" on:click={close} aria-label={$t("close")}>
        <svg viewBox="0 0 24 24"><path d="M6 6l12 12M18 6L6 18"/></svg>
      </button>
    </div>

    {#if mode === "list"}
      <button class="nc-action" on:click={() => (mode = "group")}>
        <span class="nc-ico"><svg viewBox="0 0 24 24"><circle cx="9" cy="8" r="3"/><path d="M3 19a6 6 0 0 1 12 0"/><path d="M17 7v6M14 10h6"/></svg></span>
        {$t("group_new")}
      </button>
      <button class="nc-action" on:click={() => (mode = "number")}>
        <span class="nc-ico"><svg viewBox="0 0 24 24"><path d="M5 4h3l2 5-2.5 1.5a11 11 0 0 0 5 5L15 13l5 2v3a2 2 0 0 1-2 2A16 16 0 0 1 3 6a2 2 0 0 1 2-2z"/></svg></span>
        {$t("chat_by_number")}
      </button>
      {#if selfJid}
        <button class="nc-action" on:click={noteToSelf}>
          <span class="nc-ico"><svg viewBox="0 0 24 24"><path d="M12 3v18M3 12h18"/><circle cx="12" cy="12" r="9"/></svg></span>
          {$t("note_to_self")}
        </button>
      {/if}
      <button class="nc-action" on:click={() => (mode = "join")}>
        <span class="nc-ico"><svg viewBox="0 0 24 24"><path d="M9 15l6-6M8 13l-2 2a3 3 0 0 0 4 4l2-2M16 11l2-2a3 3 0 0 0-4-4l-2 2"/></svg></span>
        {$t("join_via_link")}
      </button>
      <button class="nc-action" on:click={() => (mode = "qr")}>
        <span class="nc-ico"><svg viewBox="0 0 24 24"><rect x="4" y="4" width="6" height="6"/><rect x="14" y="4" width="6" height="6"/><rect x="4" y="14" width="6" height="6"/><path d="M14 14h6v6h-6z"/></svg></span>
        {$t("add_via_qr")}
      </button>
    {:else if mode === "group"}
      <input class="nc-name" placeholder={$t("group_name")} bind:value={name} />
    {/if}

    {#if mode === "join"}
      <input class="nc-name" placeholder="https://chat.whatsapp.com/…" bind:value={linkVal}
        on:keydown={(e) => e.key === "Enter" && doJoin()} />
      <button class="nc-create" disabled={busy || !linkVal.trim()} on:click={doJoin}>{$t("join_group")}</button>
    {:else if mode === "number"}
      <input class="nc-name" type="tel" inputmode="tel" placeholder="62812…" bind:value={numVal}
        on:keydown={(e) => e.key === "Enter" && doNumber()} />
      <button class="nc-create" disabled={busy || numVal.replace(/[^0-9]/g, '').length < 8} on:click={doNumber}>{$t("start_chat")}</button>
    {:else if mode === "qr"}
      <input class="nc-name" placeholder="https://wa.me/qr/…" bind:value={qrVal}
        on:keydown={(e) => e.key === "Enter" && doQR()} />
      <button class="nc-create" disabled={busy || !qrVal.trim()} on:click={doQR}>{$t("start_chat")}</button>
    {:else}
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
        {$t("group_create")} ({selected.size})
      </button>
    {/if}
    {/if}
  </div>
{/if}
