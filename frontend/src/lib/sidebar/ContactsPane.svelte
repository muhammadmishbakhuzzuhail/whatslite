<script>
  // ContactsPane — daftar kontak ala buku-alamat: pemisah huruf, subtitle
  // about/status (lazy), titik online, tombol simpan kontak baru.
  import { onMount } from "svelte";
  import { messageContact, openProfile, saveContactLabel, chatStatus, pushToast } from "../../stores.js";
  import { getContacts, getContactAbout, subscribePresence, avatarUrl, senderColorFor } from "../../services/data.js";
  import Avatar from "../common/Avatar.svelte";
  import { t } from "../i18n.js";

  let contacts = [];
  let q = "";
  let abouts = {}; // jid -> about text (lazy)

  onMount(load);
  async function load() {
    contacts = await getContacts();
    // Berlangganan presence (titik online) — batasi agar tak membanjiri.
    contacts.slice(0, 100).forEach((c) => subscribePresence(c.jid));
  }

  $: filtered = contacts.filter((c) =>
    (c.name || "").toLowerCase().includes(q.trim().toLowerCase()) ||
    (c.phone || "").includes(q.trim())
  );
  // Kelompokkan per huruf awal (A–Z, selain itu "#").
  $: groups = (() => {
    const g = {};
    for (const c of filtered) {
      let L = (c.name || "#").trim().charAt(0).toUpperCase();
      if (!/[A-Z]/.test(L)) L = "#";
      (g[L] = g[L] || []).push(c);
    }
    return Object.keys(g).sort().map((L) => ({ L, items: g[L] }));
  })();

  // Lazy-ambil about saat baris terlihat (hindari panggilan jaringan utk semua).
  function observe(node, jid) {
    const io = new IntersectionObserver((es) => {
      if (es[0].isIntersecting && abouts[jid] === undefined) {
        abouts[jid] = ""; // tandai sedang diambil
        getContactAbout(jid).then((a) => { abouts = { ...abouts, [jid]: a || "" }; });
        io.disconnect();
      }
    });
    io.observe(node);
    return { destroy() { io.disconnect(); } };
  }
  function subtitle(c) {
    const a = abouts[c.jid];
    if (a) return a;                 // about/status kontak
    return c.phone || "";            // fallback nomor
  }
  const isOnline = (jid) => $chatStatus[jid] === "online";

  function saveNew() {
    const num = prompt($t("contact_new_number"));
    if (!num) return;
    const digits = num.replace(/[^0-9]/g, "");
    if (digits.length < 6) { pushToast($t("err_generic")); return; }
    const name = prompt($t("save_contact_prompt"));
    if (!name || !name.trim()) return;
    saveContactLabel(digits + "@s.whatsapp.net", name.trim());
    pushToast($t("contact_saved").replace("%s", name.trim()), "ok");
    setTimeout(load, 400);
  }
</script>

<header class="pane-head"><h2>{$t("rail_contacts")}</h2></header>
<div class="ct-top">
  <input class="ct-search" placeholder={$t("search")} bind:value={q} />
  <button class="ct-new" on:click={saveNew} title={$t("contact_save_new")}>
    <svg viewBox="0 0 24 24"><circle cx="9" cy="8" r="4"/><path d="M2 20c0-3.5 3-6 7-6M17 11v6M14 14h6"/></svg>
    <span>{$t("contact_save_new")}</span>
  </button>
</div>
<div class="ct-list">
  {#each groups as grp (grp.L)}
    <div class="ct-letter">{grp.L}</div>
    {#each grp.items as c (c.jid)}
      <div class="ct-row" role="button" tabindex="0" use:observe={c.jid}
        on:click={() => messageContact(c.jid)}
        on:keydown={(e) => e.key === "Enter" && messageContact(c.jid)}>
        <div class="ct-av">
          <Avatar name={c.name} color={senderColorFor(c.jid)} photo={avatarUrl(c.jid)} sm={true} />
          {#if isOnline(c.jid)}<span class="ct-dot" title="online"></span>{/if}
        </div>
        <div class="ct-meta">
          <div class="ct-name">{c.name}{#if !c.saved && c.phone}<span class="ct-num">{c.phone}</span>{/if}</div>
          {#if subtitle(c)}<div class="ct-sub">{subtitle(c)}</div>{/if}
        </div>
        <button class="ct-info" title={$t("info_contact")} on:click|stopPropagation={() => openProfile(c.jid)}>
          <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M12 11v5"/><circle cx="12" cy="7.5" r=".6"/></svg>
        </button>
      </div>
    {/each}
  {/each}
  {#if filtered.length === 0}<div class="ct-empty">{$t("no_match")}</div>{/if}
</div>

<style>
  .ct-top { display: flex; gap: 8px; padding: 6px 12px 10px; align-items: center; }
  .ct-search { flex: 1; border: 0; border-radius: 10px; padding: 9px 14px; background: var(--bg2); color: var(--text); font: inherit; outline: none; }
  .ct-new { display: inline-flex; align-items: center; gap: 6px; border: 0; background: var(--accent); color: #fff; border-radius: 10px; padding: 8px 11px; cursor: pointer; font: inherit; font-size: 13px; white-space: nowrap; }
  .ct-new svg { width: 17px; height: 17px; fill: none; stroke: currentColor; stroke-width: 2; }
  .ct-list { flex: 1; overflow-y: auto; }
  .ct-letter { position: sticky; top: 0; background: var(--bg); color: var(--accent); font-size: 12px; font-weight: 700; padding: 5px 16px; z-index: 1; }
  .ct-row { display: flex; align-items: center; gap: 12px; padding: 8px 14px; cursor: pointer; }
  .ct-row:hover { background: var(--hover); }
  .ct-av { position: relative; flex: 0 0 auto; }
  .ct-dot { position: absolute; right: -1px; bottom: -1px; width: 12px; height: 12px; border-radius: 50%; background: #28c840; border: 2px solid var(--bg); }
  .ct-meta { flex: 1; min-width: 0; }
  .ct-name { font-size: 15px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .ct-num { font-weight: 400; color: var(--text2); font-size: 12.5px; margin-left: 6px; }
  .ct-sub { font-size: 12.5px; color: var(--text2); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .ct-info { background: none; border: 0; color: var(--text2); cursor: pointer; padding: 6px; border-radius: 50%; flex: 0 0 auto; }
  .ct-info:hover { background: var(--bg2); color: var(--accent); }
  .ct-info svg { width: 20px; height: 20px; fill: none; stroke: currentColor; stroke-width: 2; }
  .ct-empty { text-align: center; color: var(--text2); padding: 40px 16px; }
</style>
