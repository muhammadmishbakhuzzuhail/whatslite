<script>
  import { onMount } from "svelte";
  import { railView, pushToast } from "../../stores.js";
  import { getPrivacy, setPrivacy, getBlockedContacts, block, colorFor } from "../../services/data.js";
  import { t } from "../i18n.js";
  import { initial } from "../util.js";

  let priv = {};
  let blocked = [];
  let loading = true;

  async function load() {
    loading = true;
    [priv, blocked] = await Promise.all([getPrivacy(), getBlockedContacts()]);
    loading = false;
  }
  onMount(load);

  // opsi per jenis setelan
  const visOpts = [
    { v: "all", k: "pv_everyone" },
    { v: "contacts", k: "pv_contacts" },
    { v: "none", k: "pv_nobody" },
  ];
  function change(name, e) { priv = { ...priv, [name]: e.target.value }; setPrivacy(name, e.target.value); }
  function toggleReceipts(e) {
    const v = e.target.checked ? "all" : "none";
    priv = { ...priv, readreceipts: v };
    setPrivacy("readreceipts", v);
  }
  function unblock(c) {
    block(c.jid, false);
    blocked = blocked.filter((x) => x.jid !== c.jid);
    pushToast($t("unblocked_toast").replace("%s", c.name), "ok");
  }

  const rows = [
    { name: "lastseen", k: "pv_last_seen" },
    { name: "profile", k: "pv_profile_photo" },
    { name: "status", k: "pv_status" },
    { name: "groupadd", k: "pv_groups" },
  ];
</script>

<header class="pane-head" style="gap:14px">
  <button class="icon-btn" title={$t("back")} on:click={() => railView.set("settings")}>
    <svg viewBox="0 0 24 24"><path d="M15 5l-7 7 7 7"/></svg>
  </button>
  <h2 style="font-size:17px">{$t("privacy")}</h2>
</header>

<div style="flex:1; overflow-y:auto">
  {#if loading}
    <div class="empty-list" style="text-align:center;padding:24px;color:var(--text2)">…</div>
  {:else}
    {#each rows as r}
      <div class="pv-row">
        <span class="pv-name">{$t(r.k)}</span>
        <select class="lang-select" value={priv[r.name] || "all"} on:change={(e) => change(r.name, e)}>
          {#each visOpts as o}<option value={o.v}>{$t(o.k)}</option>{/each}
        </select>
      </div>
    {/each}

    <div class="pv-row">
      <span class="pv-name">{$t("pv_read_receipts")}</span>
      <span class="switch {priv.readreceipts === 'none' ? 'off' : ''}" role="switch" tabindex="0"
        on:click={() => toggleReceipts({ target: { checked: priv.readreceipts === 'none' } })}
        on:keydown={(e) => e.key === 'Enter' && toggleReceipts({ target: { checked: priv.readreceipts === 'none' } })}></span>
    </div>

    <div class="list-label" style="margin-top:14px">{$t("pv_blocked")} ({blocked.length})</div>
    {#each blocked as c (c.jid)}
      <div class="pv-blocked">
        <span class="pv-av" style="background:{colorFor(c.jid)}">{initial(c.name)}</span>
        <span class="pv-bname">{c.name}</span>
        <button class="btn-ghost" on:click={() => unblock(c)}>{$t("pv_unblock")}</button>
      </div>
    {/each}
    {#if blocked.length === 0}
      <div class="empty-list" style="padding:14px 16px;color:var(--text2);font-size:13px">{$t("pv_no_blocked")}</div>
    {/if}
  {/if}
</div>

<style>
  .pv-row { display:flex; align-items:center; gap:12px; padding:13px 16px; border-bottom:1px solid var(--line); }
  .pv-name { flex:1; font-size:14.5px; }
  .pv-blocked { display:flex; align-items:center; gap:12px; padding:9px 16px; }
  .pv-av { width:38px; height:38px; border-radius:50%; display:grid; align-items:center;justify-items:center; color:#fff; font-weight:600; flex:0 0 auto; }
  .pv-bname { flex:1; min-width:0; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
</style>
