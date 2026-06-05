<script>
  import { onMount } from "svelte";
  import { getChannels, getChannelMessages, followChannel, unfollowChannel, muteChannel, reactChannel, colorFor } from "../../services/data.js";
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
    {#if active.picture}
      <img class="ch-av sm" src={active.picture} alt="" on:error={(e) => (e.target.style.display = 'none')} />
    {:else}
      <span class="ch-av sm" style="background:{colorFor(active.jid)}">{initial(active.name)}</span>
    {/if}
    <div style="min-width:0">
      <h2 style="font-size:16px;display:flex;gap:5px;align-items:center">{active.name}{#if active.verified}<span class="ch-verif">✓</span>{/if}</h2>
      <div style="font-size:12px;color:var(--text2)">{active.subscribers.toLocaleString()} {$t("ch_subs")}</div>
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
{:else}
  <!-- Daftar saluran diikuti -->
  <header class="pane-head">
    <h2>{$t("rail_channels")}</h2>
    <button class="icon-btn" title={$t("ch_follow")} style="margin-left:auto" on:click={() => followOpen = true}>
      <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
    </button>
  </header>
  <div style="flex:1; overflow-y:auto">
    {#if loading}
      <div class="empty-list" style="text-align:center;padding:24px;color:var(--text2)">…</div>
    {:else}
      {#each channels as c (c.jid)}
        <div class="ch-row" on:click={() => open(c)} role="button" tabindex="0" on:keydown={(e) => e.key === "Enter" && open(c)}>
          {#if c.picture}
            <img class="ch-av" src={c.picture} alt="" on:error={(e) => (e.target.style.display = 'none')} />
          {:else}
            <span class="ch-av" style="background:{colorFor(c.jid)}">{initial(c.name)}</span>
          {/if}
          <div class="ch-meta">
            <div class="ch-name">{c.name}{#if c.verified}<span class="ch-verif">✓</span>{/if}</div>
            <div class="ch-sub">{c.subscribers.toLocaleString()} {$t("ch_subs")}</div>
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

<!-- Ikuti via tautan -->
{#if followOpen}
  <div class="nc-modal" on:click|self={() => followOpen = false}>
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
  .ch-row { display:flex; align-items:center; gap:13px; padding:10px 14px; cursor:pointer; }
  .ch-row:hover { background:var(--hover); }
  .ch-av { width:48px; height:48px; border-radius:50%; display:grid; align-items:center;justify-items:center; color:#fff; font-weight:600; font-size:18px; object-fit:cover; flex:0 0 auto; }
  .ch-av.sm { width:38px; height:38px; font-size:15px; }
  .ch-meta { flex:1; min-width:0; }
  .ch-name { font-weight:600; font-size:15px; display:flex; align-items:center; gap:5px; }
  .ch-sub { font-size:12.5px; color:var(--text2); }
  .ch-verif { color:#fff; background:var(--accent); border-radius:50%; width:15px; height:15px; display:inline-grid; align-items:center;justify-items:center; font-size:10px; }
  .ch-act { background:none; border:0; cursor:pointer; font-size:15px; opacity:.6; padding:4px; }
  .ch-act:hover { opacity:1; }
  .ch-feed { flex:1; overflow-y:auto; padding:12px; display:flex; flex-direction:column; gap:12px; }
  .ch-post { background:var(--bubble-in, var(--bg2)); border:1px solid var(--line); border-radius:14px; padding:10px 12px; }
  .ch-post-media { width:100%; border-radius:10px; margin-bottom:8px; object-fit:cover; }
  .ch-post-text { font-size:14.5px; white-space:pre-wrap; word-break:break-word; }
  .ch-post-meta { font-size:11.5px; color:var(--text2); margin-top:6px; }
</style>
