<script>
  import { onMount } from "svelte";
  import { getCommunities, leaveCommunity, colorFor } from "../../services/data.js";
  import { activeChatId, railView, pushToast } from "../../stores.js";
  import { t } from "../i18n.js";
  import { initial } from "../util.js";

  let comms = [];
  let loading = true;
  let openSet = new Set(); // jid komunitas yang diperluas

  async function load() {
    loading = true;
    comms = await getCommunities();
    // perluas semua secara default
    openSet = new Set(comms.map((c) => c.jid));
    loading = false;
  }
  onMount(load);

  function toggle(jid) {
    openSet.has(jid) ? openSet.delete(jid) : openSet.add(jid);
    openSet = openSet;
  }
  function openGroup(g) { activeChatId.set(g.jid); railView.set("chats"); }
  function leave(c) {
    if (!confirm($t("comm_leave_confirm").replace("%s", c.name))) return;
    leaveCommunity(c.jid);
    comms = comms.filter((x) => x.jid !== c.jid);
    pushToast($t("comm_left"), "ok");
  }
</script>

<header class="pane-head"><h2>{$t("rail_communities")}</h2></header>

<div style="flex:1; overflow-y:auto">
  {#if loading}
    <div class="empty-list" style="text-align:center;padding:24px;color:var(--text2)">…</div>
  {:else}
    {#each comms as c (c.jid)}
      <div class="comm">
        <div class="comm-head" on:click={() => toggle(c.jid)} role="button" tabindex="0" on:keydown={(e) => e.key === "Enter" && toggle(c.jid)}>
          <span class="comm-av" style="background:{colorFor(c.jid)}">{initial(c.name)}</span>
          <div class="comm-meta">
            <div class="comm-name">{c.name}</div>
            <div class="comm-sub">{(c.groups || []).length} {$t("comm_groups")}</div>
          </div>
          <button class="ch-act" title={$t("comm_leave")} on:click|stopPropagation={() => leave(c)}>⎋</button>
          <span class="comm-chev {openSet.has(c.jid) ? 'open' : ''}">▾</span>
        </div>
        {#if openSet.has(c.jid)}
          <div class="comm-groups">
            {#each c.groups || [] as g (g.jid)}
              <div class="comm-grow" on:click={() => openGroup(g)} role="button" tabindex="0" on:keydown={(e) => e.key === "Enter" && openGroup(g)}>
                <span class="comm-gico">#</span>
                <span class="comm-gname">{g.name}{#if g.isDefault}<span class="comm-tag">{$t("comm_announce")}</span>{/if}</span>
              </div>
            {/each}
            {#if (c.groups || []).length === 0}
              <div style="padding:8px 16px;font-size:12.5px;color:var(--text2)">{$t("comm_no_groups")}</div>
            {/if}
          </div>
        {/if}
      </div>
    {/each}
    {#if comms.length === 0}
      <div class="empty-list" style="text-align:center;padding:28px 16px;color:var(--text2)">{$t("comm_empty")}</div>
    {/if}
  {/if}
</div>

<style>
  .comm { border-bottom:1px solid var(--line); }
  .comm-head { display:flex; align-items:center; gap:13px; padding:11px 14px; cursor:pointer; }
  .comm-head:hover { background:var(--hover); }
  .comm-av { width:46px; height:46px; border-radius:16px; display:grid; align-items:center;justify-items:center; color:#fff; font-weight:600; font-size:18px; flex:0 0 auto; }
  .comm-meta { flex:1; min-width:0; }
  .comm-name { font-weight:600; font-size:15px; }
  .comm-sub { font-size:12.5px; color:var(--text2); }
  .comm-chev { color:var(--text2); transition:transform .15s; }
  .comm-chev.open { transform:rotate(180deg); }
  .ch-act { background:none; border:0; cursor:pointer; font-size:15px; opacity:.6; padding:4px; }
  .ch-act:hover { opacity:1; }
  .comm-groups { padding:2px 0 8px; }
  .comm-grow { display:flex; align-items:center; gap:12px; padding:8px 14px 8px 26px; cursor:pointer; }
  .comm-grow:hover { background:var(--hover); }
  .comm-gico { width:32px; height:32px; border-radius:10px; background:var(--bg2); display:grid; align-items:center;justify-items:center; color:var(--text2); font-weight:600; flex:0 0 auto; }
  .comm-gname { font-size:14px; display:flex; align-items:center; gap:7px; }
  .comm-tag { font-size:10px; background:var(--accent); color:#fff; border-radius:6px; padding:1px 6px; }
</style>
