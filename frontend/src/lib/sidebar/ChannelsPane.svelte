<script>
  import { onMount } from "svelte";
  import { getChannels, getChannelMessages, followChannel, followChannelByJID, getRecommendedChannels, unfollowChannel, muteChannel, reactChannel, colorFor, avatarUrl, createChannel, postChannel } from "../../services/data.js";
  import { pushToast } from "../../stores.js";
  import { t } from "../i18n.js";
  import { initial } from "../util.js";

  let channels = [];
  let loading = true;
  let active = null;      // saluran terbuka (feed)
  let feed = [];
  let feedLoading = false;
  let followOpen = false;
  let link = "";

  // --- Jelajah (rekomendasi global) ---
  let tab = "following"; // following | discover
  let recommended = [], discQ = "", discLoading = false, _dt, following = new Set();
  $: following = new Set(channels.map((c) => c.jid));
  async function loadDiscover(q) { discLoading = true; recommended = await getRecommendedChannels(q || ""); discLoading = false; }
  $: if (tab === "discover") { clearTimeout(_dt); const q = discQ.trim(); _dt = setTimeout(() => loadDiscover(q), 350); }
  async function followRec(c) {
    followChannelByJID(c.jid);
    pushToast($t("ch_followed"), "ok");
    setTimeout(load, 600);
  }

  async function load() {
    loading = true;
    channels = await getChannels();
    loading = false;
  }
  onMount(load);

  async function open(c) {
    active = c; feed = []; feedLoading = true;
    feed = await getChannelMessages(c.jid);
    feedLoading = false;
  }
  function back() { active = null; feed = []; }

  async function doFollow() {
    const l = link.trim();
    if (!l) return;
    followOpen = false; link = "";
    const c = await followChannel(l);
    if (c && c.jid) { pushToast($t("ch_followed"), "ok"); load(); }
    else pushToast($t("ch_follow_fail"), "error");
  }
  function doUnfollow(c) {
    unfollowChannel(c.jid);
    channels = channels.filter((x) => x.jid !== c.jid);
    if (active && active.jid === c.jid) back();
    pushToast($t("ch_unfollowed"), "ok");
  }
  let postVal = "";
  async function doPost() {
    if (!postVal.trim() || !active) return;
    postChannel(active.jid, postVal.trim());
    postVal = "";
    setTimeout(async () => { feed = await getChannelMessages(active.jid); }, 1200);
  }
  let createOpen = false, cName = "", cDesc = "", cBusy = false;
  async function doCreate() {
    if (!cName.trim()) return;
    cBusy = true;
    const jid = await createChannel(cName.trim(), cDesc.trim());
    cBusy = false;
    if (jid) { pushToast($t("ch_created"), "ok"); createOpen = false; cName = ""; cDesc = ""; load(); }
    else pushToast($t("err_generic"));
  }
  function toggleMute(c) {
    c.muted = !c.muted; channels = channels;
    muteChannel(c.jid, c.muted);
  }
</script>

{#if active}
  <!-- Feed read-only satu saluran -->
  <header class="pane-head" style="gap:14px">
    <button class="icon-btn" title={$t("back")} on:click={back}>
      <svg viewBox="0 0 24 24"><path d="M15 5l-7 7 7 7"/></svg>
    </button>
    <span class="ch-av sm" style="background:{colorFor(active.jid)}">{initial(active.name)}
      {#if avatarUrl(active.jid) || active.picture}<img src={avatarUrl(active.jid) || active.picture} alt="" on:error={(e) => e.target.remove()} />{/if}
    </span>
    <div style="min-width:0">
      <h2 style="font-size:16px;display:flex;gap:5px;align-items:center">{active.name}{#if active.verified}<span class="ch-verif">✓</span>{/if}</h2>
      <div style="font-size:12px;color:var(--text2)">{(active.subscribers ?? 0).toLocaleString()} {$t("ch_subs")}</div>
    </div>
  </header>
  <div class="ch-feed">
    {#if feedLoading}
      <div class="empty-list" style="text-align:center;padding:24px;color:var(--text2)">…</div>
    {:else}
      {#each feed as m (m.id)}
        <div class="ch-post">
          {#if m.thumb}<img class="ch-post-media" src={m.thumb} alt="" />{/if}
          {#if m.text}<div class="ch-post-text">{m.text}</div>{/if}
          <div class="ch-post-meta">
            <span>{m.time}{m.views ? ` · 👁 ${m.views.toLocaleString()}` : ""}</span>
            <span class="ch-react">
              {#each ["👍","❤️","😂","😮","🙏"] as e}
                <button on:click={() => { reactChannel(active.jid, m.id, m.serverId, e); pushToast(e, "ok"); }}>{e}</button>
              {/each}
            </span>
          </div>
        </div>
      {/each}
      {#if feed.length === 0}
        <div class="empty-list" style="text-align:center;padding:24px;color:var(--text2)">{$t("ch_no_posts")}</div>
      {/if}
    {/if}
  </div>
  {#if active.role === "owner" || active.role === "admin"}
    <div class="ch-composer">
      <input placeholder={$t("ch_post_ph")} bind:value={postVal} on:keydown={(e) => e.key === "Enter" && doPost()} />
      <button class="ch-post-send" disabled={!postVal.trim()} on:click={doPost} aria-label={$t("send")}>
        <svg viewBox="0 0 24 24"><path d="M3 11l18-8-8 18-2-7-8-3z"/></svg>
      </button>
    </div>
  {/if}
{:else}
  <!-- Daftar saluran: Diikuti / Jelajahi -->
  <header class="pane-head">
    <h2>{$t("rail_channels")}</h2>
    <button class="icon-btn" title={$t("ch_create")} style="margin-left:auto" on:click={() => createOpen = true}>
      <svg viewBox="0 0 24 24"><path d="M19 9v6M16 12h6"/><circle cx="9" cy="8" r="4"/><path d="M2 20a7 7 0 0 1 12-5"/></svg>
    </button>
    <button class="icon-btn" title={$t("ch_follow")} on:click={() => followOpen = true}>
      <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
    </button>
  </header>

  {#if createOpen}
    <button class="modal-backdrop" aria-label={$t("close")} on:click={() => (createOpen = false)}></button>
    <div class="nc-modal" role="dialog">
      <div class="nc-head"><span>{$t("ch_create")}</span><button class="icon-btn" on:click={() => (createOpen = false)} aria-label={$t("close")}><svg viewBox="0 0 24 24"><path d="M6 6l12 12M18 6L6 18"/></svg></button></div>
      <input class="nc-name" placeholder={$t("ch_name")} bind:value={cName} />
      <input class="nc-name" placeholder={$t("ch_desc")} bind:value={cDesc} />
      <button class="nc-create" disabled={cBusy || !cName.trim()} on:click={doCreate}>{$t("ch_create")}</button>
    </div>
  {/if}
  <div class="ch-tabs">
    <button class:on={tab === "following"} on:click={() => (tab = "following")}>{$t("ch_following")}</button>
    <button class:on={tab === "discover"} on:click={() => (tab = "discover")}>{$t("ch_discover")}</button>
  </div>

  {#if tab === "discover"}
    <div style="padding:0 12px 8px"><input class="ch-search" placeholder={$t("search")} bind:value={discQ} /></div>
    <div style="flex:1; overflow-y:auto">
      {#if discLoading}
        <div class="empty-list" style="text-align:center;padding:24px;color:var(--text2)">…</div>
      {:else}
        {#each recommended as c (c.jid)}
          <div class="ch-row" on:click={() => open(c)} role="button" tabindex="0" on:keydown={(e) => e.key === "Enter" && open(c)}>
            <span class="ch-av" style="background:{colorFor(c.jid)}">{initial(c.name)}
              {#if avatarUrl(c.jid) || c.picture}<img src={avatarUrl(c.jid) || c.picture} alt="" on:error={(e) => e.target.remove()} />{/if}
            </span>
            <div class="ch-meta">
              <div class="ch-name">{c.name}{#if c.verified}<span class="ch-verif">✓</span>{/if}</div>
              <div class="ch-sub">{(c.subscribers ?? 0).toLocaleString()} {$t("ch_subs")}</div>
            </div>
            {#if following.has(c.jid)}
              <span class="ch-followed">✓</span>
            {:else}
              <button class="btn-accent ch-follow-btn" on:click|stopPropagation={() => followRec(c)}>{$t("ch_follow_short")}</button>
            {/if}
          </div>
        {/each}
        {#if recommended.length === 0}
          <div class="empty-list" style="text-align:center;padding:28px 16px;color:var(--text2)">{$t("ch_discover_empty")}</div>
        {/if}
      {/if}
    </div>
  {:else}
  <div style="flex:1; overflow-y:auto">
    {#if loading}
      <div class="empty-list" style="text-align:center;padding:24px;color:var(--text2)">…</div>
    {:else}
      {#each channels as c (c.jid)}
        <div class="ch-row" on:click={() => open(c)} role="button" tabindex="0" on:keydown={(e) => e.key === "Enter" && open(c)}>
          <span class="ch-av" style="background:{colorFor(c.jid)}">{initial(c.name)}
            {#if avatarUrl(c.jid) || c.picture}<img src={avatarUrl(c.jid) || c.picture} alt="" on:error={(e) => e.target.remove()} />{/if}
          </span>
          <div class="ch-meta">
            <div class="ch-name">{c.name}{#if c.verified}<span class="ch-verif">✓</span>{/if}</div>
            <div class="ch-sub">{(c.subscribers ?? 0).toLocaleString()} {$t("ch_subs")}</div>
          </div>
          <button class="ch-act" title={c.muted ? $t("unmute") : $t("mute")} on:click|stopPropagation={() => toggleMute(c)}>{c.muted ? "🔕" : "🔔"}</button>
          <button class="ch-act" title={$t("ch_unfollow")} on:click|stopPropagation={() => doUnfollow(c)}>✕</button>
        </div>
      {/each}
      {#if channels.length === 0}
        <div class="empty-list" style="text-align:center;padding:28px 16px;color:var(--text2)">{$t("ch_empty")}</div>
      {/if}
    {/if}
  </div>
  {/if}
{/if}

<!-- Ikuti via tautan -->
{#if followOpen}
  <div class="nc-overlay" on:click|self={() => followOpen = false}>
    <div class="nc-card" style="max-width:420px">
      <h3 style="margin:0 0 12px">{$t("ch_follow")}</h3>
      <input bind:value={link} placeholder="https://whatsapp.com/channel/…"
        style="width:100%;border:1px solid var(--line);border-radius:12px;padding:11px 12px;background:var(--bg2);color:var(--text);font:inherit" />
      <div style="display:flex;gap:10px;justify-content:flex-end;margin-top:14px">
        <button class="btn-ghost" on:click={() => followOpen = false}>{$t("cancel")}</button>
        <button class="btn-accent" on:click={doFollow} disabled={!link.trim()}>{$t("ch_follow")}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .ch-tabs { display:flex; gap:6px; padding:2px 12px 10px; }
  .ch-tabs button { flex:1; padding:8px; border:0; background:var(--bg2); border-radius:9px; cursor:pointer; color:var(--text2); font:inherit; font-size:13px; font-weight:600; }
  .ch-tabs button.on { background:var(--accent); color:#fff; }
  .ch-search { width:100%; border:0; border-radius:10px; padding:9px 14px; background:var(--bg2); color:var(--text); font:inherit; outline:none; }
  .ch-follow-btn { padding:6px 14px; font-size:13px; flex:0 0 auto; }
  .ch-followed { color:var(--accent); font-weight:700; flex:0 0 auto; padding:0 8px; }
  .ch-row { display:flex; align-items:center; gap:13px; padding:10px 14px; cursor:pointer; }
  .ch-row:hover { background:var(--hover); }
  .ch-av { position:relative; overflow:hidden; width:48px; height:48px; border-radius:50%; display:grid; align-items:center;justify-items:center; color:#fff; font-weight:600; font-size:18px; flex:0 0 auto; }
  .ch-av img { position:absolute; inset:0; width:100%; height:100%; object-fit:cover; }
  .ch-av.sm { width:38px; height:38px; font-size:15px; }
  .ch-meta { flex:1; min-width:0; }
  .ch-name { font-weight:600; font-size:15px; display:flex; align-items:center; gap:5px; }
  .ch-sub { font-size:12.5px; color:var(--text2); }
  .ch-verif { color:#fff; background:var(--accent); border-radius:50%; width:15px; height:15px; display:inline-grid; align-items:center;justify-items:center; font-size:10px; }
  .ch-act { background:none; border:0; cursor:pointer; font-size:15px; opacity:.6; padding:4px; }
  .ch-act:hover { opacity:1; }
  .ch-feed { flex:1; overflow-y:auto; padding:12px; display:flex; flex-direction:column; gap:12px; }
  .ch-composer { display:flex; align-items:center; gap:8px; padding:10px 12px; border-top:1px solid var(--line); }
  .ch-composer input { flex:1; border:0; border-radius:20px; padding:10px 14px; background:var(--bg2); color:var(--text); font:inherit; outline:none; }
  .ch-post-send { width:40px; height:40px; border-radius:50%; border:0; background:var(--accent); color:#fff; cursor:pointer; flex:0 0 auto; display:flex; align-items:center; justify-content:center; }
  .ch-post-send svg { width:19px; height:19px; fill:currentColor; }
  .ch-post-send:disabled { opacity:.5; }
  .ch-post { background:var(--bubble-in, var(--bg2)); border:1px solid var(--line); border-radius:14px; padding:10px 12px; }
  .ch-post-media { width:100%; border-radius:10px; margin-bottom:8px; object-fit:cover; }
  .ch-post-text { font-size:14.5px; white-space:pre-wrap; word-break:break-word; }
  .ch-post-meta { font-size:11.5px; color:var(--text2); margin-top:6px; }
</style>
