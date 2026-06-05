<script>
  import ChatRow from "./ChatRow.svelte";
  import { chats, search, filter, activeChatId, railView, searchMessages, jumpMsg } from "../../stores.js";
  import { getArchivedCount, colorFor } from "../../services/data.js";
  import { t } from "../i18n.js";
  const archivedCount = getArchivedCount();
  // Foto profil LAZY per-avatar (via /avatar/<jid>) — bukan eager.

  $: q = $search.trim().toLowerCase();
  $: filtered = $chats.filter((c) => {
    const matchQ = !q || c.name.toLowerCase().includes(q) || (c.preview || "").toLowerCase().includes(q);
    let matchF = true;
    if ($filter === "Belum dibaca") matchF = !!c.unread;
    else if ($filter === "Grup") matchF = !!c.group;
    else if ($filter === "Favorit") matchF = !!c.pinned;
    return matchQ && matchF;
  });
  $: pinned = filtered.filter((c) => c.pinned);
  $: others = filtered.filter((c) => !c.pinned);
  $: plain = !q && $filter === "Semua";

  // Pencarian ISI pesan (FTS5 di BE) — debounce.
  let hits = [];
  let _t;
  $: runSearch($search.trim());
  function runSearch(query) {
    clearTimeout(_t);
    if (query.length < 2) { hits = []; return; }
    _t = setTimeout(async () => { hits = await searchMessages(query); }, 220);
  }
  function openHit(h) { activeChatId.set(h.chatJid); railView.set("chats"); jumpMsg.set(h.msgId); }
  function initial(s) { for (const ch of s || "") if (/[\p{L}\p{N}]/u.test(ch)) return ch.toUpperCase(); return "?"; }
</script>

<div class="chat-list">
  {#if q}
    <!-- Mode pencarian: chat (nama) + pesan (isi) -->
    {#if filtered.length}
      <div class="list-label">{$t("rail_chats")}</div>
      {#each filtered as c (c.id)}<ChatRow chat={c} />{/each}
    {/if}
    {#if hits.length}
      <div class="list-label">Pesan</div>
      {#each hits as h (h.chatJid + h.msgId)}
        <button class="hit-row" on:click={() => openHit(h)}>
          <span class="hit-av" style="background:{colorFor(h.chatJid)}">{initial(h.chatName)}</span>
          <span class="hit-main">
            <span class="hit-top"><span class="hit-name">{h.chatName}</span><span class="hit-time">{h.time}</span></span>
            <span class="hit-text">{h.text}</span>
          </span>
        </button>
      {/each}
    {/if}
    {#if filtered.length === 0 && hits.length === 0}
      <div class="empty-list">{$t("no_match")}</div>
    {/if}
  {:else}
    {#if plain}
      <div class="archived" role="button" tabindex="0">
        <div class="arc-ico"><svg viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="4" rx="1"/><path d="M5 8v11a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V8"/><path d="M10 12h4"/></svg></div>
        <div class="arc-label">{$t("archived")}</div>
        {#if archivedCount > 0}<div class="arc-count">{archivedCount}</div>{/if}
      </div>
    {/if}
    {#if pinned.length}
      <div class="list-label">
        <svg viewBox="0 0 24 24"><path d="M12 17v5M7 4h10l-1 6 3 3H5l3-3-1-6z"/></svg> {$t("pinned")}
      </div>
      {#each pinned as c (c.id)}<ChatRow chat={c} />{/each}
    {/if}
    {#if others.length}
      {#if pinned.length}<div class="list-label">{$t("all_chats")}</div>{/if}
      {#each others as c (c.id)}<ChatRow chat={c} />{/each}
    {/if}
    {#if filtered.length === 0}
      <div class="empty-list">{$t("no_match")}</div>
    {/if}
  {/if}
</div>

<style>
  .empty-list { padding: 28px 16px; text-align: center; color: var(--text2); font-size: 14px; }
  .hit-row { display: flex; align-items: center; gap: 12px; width: 100%; border: 0; background: 0;
    cursor: pointer; padding: 9px 12px; border-radius: var(--r); text-align: left; }
  .hit-row:hover { background: var(--hover); }
  .hit-av { width: 40px; height: 40px; border-radius: 50%; flex-shrink: 0; display: grid;
    place-items: center; font-weight: 700; font-size: 16px; color: #fff; background: var(--accent); }
  .hit-main { flex: 1; min-width: 0; display: flex; flex-direction: column; }
  .hit-top { display: flex; justify-content: space-between; align-items: baseline; }
  .hit-name { font-size: 15px; font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .hit-time { font-size: 12px; color: var(--text2); flex-shrink: 0; margin-left: 8px; }
  .hit-text { font-size: 13.5px; color: var(--text2); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
</style>
