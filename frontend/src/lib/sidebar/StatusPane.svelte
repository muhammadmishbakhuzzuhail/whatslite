<script>
  import { onMount, onDestroy } from "svelte";
  import { getStatuses, postTextStatus, postMediaStatus, getStatusViewers, colorFor, avatarUrl, reactStatus, replyStatus } from "../../services/data.js";
  import { pushToast } from "../../stores.js";
  import { t } from "../i18n.js";
  import { initial } from "../util.js";
  import Avatar from "../common/Avatar.svelte";

  let groups = [];
  // Urut abjad + pemisah huruf A–Z/# (seperti tab Kontak).
  $: lettered = (() => {
    const others = groups.filter((g) => !g.mine);
    const sorted = [...others].sort((a, b) => (a.name || "").toLowerCase().localeCompare((b.name || "").toLowerCase()));
    const m = {};
    for (const g of sorted) {
      let L = (g.name || "#").trim().charAt(0).toUpperCase();
      if (!/[A-Z]/.test(L)) L = "#";
      (m[L] = m[L] || []).push(g);
    }
    return Object.keys(m).sort().map((L) => ({ L, items: m[L] }));
  })();
  let loading = true;

  // status yg sudah dilihat (persist lokal) → cincin abu vs hijau.
  const SEEN_KEY = "wa-status-seen";
  let seen = new Set();
  try { seen = new Set(JSON.parse(localStorage.getItem(SEEN_KEY) || "[]")); } catch (e) {}
  function persistSeen() { try { localStorage.setItem(SEEN_KEY, JSON.stringify([...seen])); } catch (e) {} }
  function allSeen(g) { return g.items.every((it) => seen.has(it.id)); }

  async function load() {
    loading = true;
    groups = await getStatuses();
    loading = false;
  }
  onMount(load);

  // --- balas / react status (hanya status orang lain) ---
  let replyVal = "";
  function sendReply() {
    if (!viewG || !cur || !replyVal.trim()) return;
    replyStatus(viewG.jid, cur.id, cur.text || "", replyVal.trim());
    replyVal = "";
    pushToast($t("status_replied"), "ok");
  }
  function sendReact(emoji) {
    if (!viewG || !cur) return;
    reactStatus(viewG.jid, cur.id, emoji);
    pushToast(emoji, "ok");
  }

  // --- viewer (fullscreen tap-through) ---
  let viewG = null;   // grup aktif
  let viewI = 0;      // index item aktif
  let progress = 0;   // 0..100 batang berjalan
  let timer = null;
  const DUR = 5000;

  function openGroup(g) {
    viewG = g; viewI = 0;
    startItem();
  }
  function startItem() {
    const it = viewG.items[viewI];
    if (it) { seen.add(it.id); persistSeen(); groups = groups; }
    viewers = []; showViewers = false;
    if (it && viewG.mine) getStatusViewers(it.id).then((v) => (viewers = v));
    progress = 0;
    clearInterval(timer);
    const step = 50;
    timer = setInterval(() => {
      progress += (step / DUR) * 100;
      if (progress >= 100) next();
    }, step);
  }
  function next() {
    if (!viewG) return;
    if (viewI < viewG.items.length - 1) { viewI++; startItem(); }
    else close();
  }
  function prev() {
    if (!viewG) return;
    if (viewI > 0) { viewI--; startItem(); }
  }
  function close() { clearInterval(timer); viewG = null; }
  onDestroy(() => clearInterval(timer));

  // Penonton status sendiri.
  let viewers = [];
  let showViewers = false;
  function toggleViewers() {
    showViewers = !showViewers;
    if (showViewers) { clearInterval(timer); } else { startItem(); }
  }

  function onKey(e) {
    if (!viewG) return;
    if (e.key === "Escape") close();
    else if (e.key === "ArrowRight") next();
    else if (e.key === "ArrowLeft") prev();
  }

  // --- compose status teks ---
  let composeOpen = false;
  let draft = "";
  // Warna latar status teks (ARGB). null = default akun.
  const STATUS_BG = [
    { argb: 0, css: "linear-gradient(135deg,#1f2c34,#0b141a)" },
    { argb: 0xff06b67f, css: "#06b67f" }, { argb: 0xff5b6ef5, css: "#5b6ef5" },
    { argb: 0xffe5614e, css: "#e5614e" }, { argb: 0xfff2a33c, css: "#f2a33c" },
    { argb: 0xff9b59b6, css: "#9b59b6" }, { argb: 0xff2d3436, css: "#2d3436" },
  ];
  let bgArgb = 0;
  $: bgCss = (STATUS_BG.find((b) => b.argb === bgArgb) || STATUS_BG[0]).css;
  async function post() {
    const txt = draft.trim();
    if (!txt) return;
    composeOpen = false; draft = "";
    const id = await postTextStatus(txt, bgArgb, 0);
    bgArgb = 0;
    pushToast(id ? $t("status_posted") : $t("status_failed"), id ? "ok" : "error");
    setTimeout(load, 1200);
  }

  $: cur = viewG ? viewG.items[viewI] : null;

  // --- status media (gambar/video) ---
  let mediaInput;
  function pickMedia() { mediaInput && mediaInput.click(); }
  function onMedia(e) {
    const f = e.target.files && e.target.files[0];
    e.target.value = "";
    if (!f) return;
    const kind = f.type.startsWith("video/") ? "video" : "image";
    const r = new FileReader();
    r.onload = async () => {
      const id = await postMediaStatus(kind, "", r.result);
      pushToast(id ? $t("status_posted") : $t("status_failed"), id ? "ok" : "error");
      setTimeout(load, 1500);
    };
    r.readAsDataURL(f);
  }
</script>

<svelte:window on:keydown={onKey} />

<header class="pane-head"><h2>{$t("rail_status")}</h2></header>

<div style="flex:1; overflow-y:auto">
  <!-- Status saya / tambah -->
  <div class="status-row">
    <button class="status-av-wrap" style="background:none;border:0;cursor:pointer;padding:0" on:click={() => composeOpen = true}>
      <span class="status-av" style="background:{colorFor('me')}">{initial("?")}</span>
      <span class="status-add">+</span>
    </button>
    <button class="status-meta" style="background:none;border:0;cursor:pointer;text-align:left;flex:1" on:click={() => composeOpen = true}>
      <span class="status-name">{$t("my_status")}</span>
      <span class="status-sub">{$t("status_add_hint")}</span>
    </button>
    <button class="icon-btn" title={$t("status_photo")} on:click={pickMedia}>
      <svg viewBox="0 0 24 24"><path d="M4 7h3l2-2h6l2 2h3v12H4z"/><circle cx="12" cy="13" r="3.5"/></svg>
    </button>
    <input type="file" accept="image/*,video/*" bind:this={mediaInput} on:change={onMedia} style="display:none" />
  </div>

  {#if loading}
    <div class="empty-list" style="padding:24px 16px;text-align:center;color:var(--text2)">…</div>
  {:else}
    {#each lettered as grp (grp.L)}
      <div class="ct-letter">{grp.L}</div>
      {#each grp.items as g (g.jid)}
        <button class="status-row" on:click={() => openGroup(g)}>
          <span class="status-av-wrap">
            <span class="ring {allSeen(g) ? 'seen' : ''}" style="--n:{g.count}">
              <Avatar name={g.name} color={colorFor(g.jid)} photo={avatarUrl(g.jid)} />
            </span>
          </span>
          <span class="status-meta">
            <span class="status-name">{g.name}</span>
            <span class="status-sub">{g.time}{g.count > 1 ? ` · ${g.count}` : ""}</span>
          </span>
        </button>
      {/each}
    {/each}
    {#if lettered.length === 0}
      <div class="empty-list" style="padding:28px 16px;text-align:center;color:var(--text2)">{$t("status_empty")}</div>
    {/if}
  {/if}
</div>

<!-- Compose status teks -->
{#if composeOpen}
  <div class="nc-modal" on:click|self={() => composeOpen = false}>
    <div class="nc-card" style="max-width:420px">
      <h3 style="margin:0 0 12px">{$t("status_new")}</h3>
      <div class="st-preview" style="background:{bgCss}">{draft || $t("status_placeholder")}</div>
      <textarea bind:value={draft} rows="3" placeholder={$t("status_placeholder")}
        style="width:100%;resize:none;border:1px solid var(--line);border-radius:12px;padding:12px;background:var(--bg2);color:var(--text);font:inherit;margin-top:10px"></textarea>
      <div class="st-bg-row">
        {#each STATUS_BG as b}
          <button class="st-bg-sw {bgArgb === b.argb ? 'on' : ''}" style="background:{b.css}" on:click={() => (bgArgb = b.argb)} aria-label="bg"></button>
        {/each}
      </div>
      <div style="display:flex;gap:10px;justify-content:flex-end;margin-top:14px">
        <button class="btn-ghost" on:click={() => composeOpen = false}>{$t("cancel")}</button>
        <button class="btn-accent" on:click={post} disabled={!draft.trim()}>{$t("send")}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Viewer tap-through -->
{#if viewG && cur}
  <div class="st-viewer">
    <div class="st-bars">
      {#each viewG.items as _, i}
        <span class="st-bar"><span class="st-fill" style="width:{i < viewI ? 100 : i === viewI ? progress : 0}%"></span></span>
      {/each}
    </div>
    <div class="st-head">
      <Avatar name={viewG.name} color={colorFor(viewG.jid)} photo={avatarUrl(viewG.jid)} sm={true} />
      <span class="st-htext"><b>{viewG.name}</b><span>{cur.time}</span></span>
      <button class="st-x" on:click={close}>✕</button>
    </div>

    <div class="st-stage">
      <div class="st-zone left" on:click={prev}></div>
      <div class="st-zone right" on:click={next}></div>
      {#if cur.thumb}
        <img class="st-media" src={cur.thumb} alt="" />
        {#if cur.text}<div class="st-caption">{cur.text}</div>{/if}
      {:else}
        <div class="st-text" style="background:{colorFor(viewG.jid)}">{cur.text || ""}</div>
      {/if}
    </div>

    {#if viewG.mine}
      <button class="st-viewers-btn" on:click={toggleViewers}>
        <svg viewBox="0 0 24 24"><path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7-10-7-10-7z"/><circle cx="12" cy="12" r="3"/></svg>
        {viewers.length} {$t("status_seen_by")}
      </button>
      {#if showViewers}
        <div class="st-viewers-sheet">
          <div class="st-vs-head">{$t("status_seen_by")} · {viewers.length}</div>
          {#each viewers as v}
            <div class="st-vs-row"><span>{v.name}</span><span class="st-vs-time">{v.time}</span></div>
          {/each}
          {#if viewers.length === 0}<div class="st-vs-empty">{$t("status_no_viewers")}</div>{/if}
        </div>
      {/if}
    {:else}
      <div class="st-react-row">
        {#each ["❤️", "😂", "😮", "😢", "🙏", "👍"] as e}
          <button class="st-react" on:click={() => sendReact(e)}>{e}</button>
        {/each}
      </div>
      <div class="st-reply">
        <input placeholder={$t("status_reply_ph")} bind:value={replyVal}
          on:keydown={(e) => e.key === "Enter" && sendReply()} />
        <button class="st-reply-send" disabled={!replyVal.trim()} on:click={sendReply} aria-label={$t("send")}>
          <svg viewBox="0 0 24 24"><path d="M3 11l18-8-8 18-2-7-8-3z"/></svg>
        </button>
      </div>
    {/if}
  </div>
{/if}

<style>
  .ct-letter { position: sticky; top: 0; background: var(--bg); color: var(--accent); font-size: 12px; font-weight: 700; padding: 5px 16px; z-index: 1; }
  .status-row { display:flex; align-items:center; gap:14px; width:100%; padding:10px 14px; background:none; border:0; cursor:pointer; text-align:left; }
  .status-row:hover { background:var(--hover); }
  .status-av-wrap { position:relative; flex:0 0 auto; }
  .status-av { width:48px; height:48px; border-radius:50%; display:grid; align-items:center;justify-items:center; color:#fff; font-weight:600; font-size:18px; object-fit:cover; }
  .ring { display:block; padding:2.5px; border-radius:50%; background:conic-gradient(var(--accent) 0, var(--accent) 100%); }
  .ring.seen { background:var(--line); }
  .status-add { position:absolute; right:-2px; bottom:-2px; width:18px; height:18px; border-radius:50%; background:var(--accent); color:#fff; display:grid; align-items:center;justify-items:center; font-size:14px; border:2px solid var(--bg); }
  .status-meta { display:flex; flex-direction:column; min-width:0; }
  .status-name { font-weight:600; font-size:15px; }
  .status-sub { font-size:12.5px; color:var(--text2); }

  .st-preview { min-height:90px; border-radius:12px; display:flex; align-items:center; justify-content:center; text-align:center; color:#fff; font-size:20px; font-weight:500; padding:16px; word-break:break-word; text-shadow:0 1px 3px rgba(0,0,0,.4); }
  .st-bg-row { display:flex; gap:8px; margin-top:10px; flex-wrap:wrap; }
  .st-bg-sw { width:30px; height:30px; border-radius:50%; border:2px solid transparent; cursor:pointer; }
  .st-bg-sw.on { border-color:var(--accent); }
  .st-react-row { display:flex; justify-content:center; gap:10px; padding:8px 12px 0; }
  .st-react { background:rgba(255,255,255,.12); border:0; border-radius:50%; width:42px; height:42px; font-size:22px; cursor:pointer; }
  .st-react:hover { background:rgba(255,255,255,.22); transform:scale(1.1); }
  .st-reply { display:flex; align-items:center; gap:8px; padding:10px 14px 16px; }
  .st-reply input { flex:1; border:0; border-radius:22px; padding:11px 16px; background:rgba(255,255,255,.12); color:#fff; outline:none; font:inherit; }
  .st-reply input::placeholder { color:rgba(255,255,255,.6); }
  .st-reply-send { width:42px; height:42px; border-radius:50%; border:0; background:var(--accent); color:#fff; cursor:pointer; flex:0 0 auto; display:flex; align-items:center; justify-content:center; }
  .st-reply-send svg { width:20px; height:20px; fill:currentColor; }
  .st-reply-send:disabled { opacity:.5; }
  .st-viewer { position:fixed; inset:0; z-index:60; background:#0b141a; display:flex; flex-direction:column; }
  .st-bars { display:flex; gap:4px; padding:10px 12px 4px; }
  .st-bar { flex:1; height:3px; border-radius:3px; background:rgba(255,255,255,.3); overflow:hidden; }
  .st-fill { display:block; height:100%; background:#fff; }
  .st-head { display:flex; align-items:center; gap:10px; padding:6px 14px 10px; color:#fff; }
  .st-htext { display:flex; flex-direction:column; line-height:1.2; flex:1; min-width:0; }
  .st-htext b { font-size:14px; } .st-htext span { font-size:11.5px; opacity:.7; }
  .st-x { background:none; border:0; color:#fff; font-size:20px; cursor:pointer; }
  .st-stage { position:relative; flex:1; display:grid; align-items:center;justify-items:center; overflow:hidden; }
  .st-zone { position:absolute; top:0; bottom:0; width:35%; z-index:2; cursor:pointer; }
  .st-zone.left { left:0; } .st-zone.right { right:0; width:65%; }
  .st-media { max-width:100%; max-height:100%; object-fit:contain; }
  .st-caption { position:absolute; bottom:24px; left:0; right:0; text-align:center; color:#fff; font-size:15px; padding:0 24px; text-shadow:0 1px 4px rgba(0,0,0,.6); }
  .st-text { width:100%; height:100%; display:grid; align-items:center;justify-items:center; color:#fff; font-size:26px; font-weight:500; text-align:center; padding:0 32px; }
  .st-viewers-btn { display:flex; align-items:center; justify-content:center; gap:7px; background:none; border:0; color:#fff; padding:14px; font-size:13px; cursor:pointer; opacity:.85; }
  .st-viewers-btn svg { width:18px; height:18px; fill:none; stroke:currentColor; stroke-width:2; }
  .st-viewers-sheet { position:absolute; left:0; right:0; bottom:0; max-height:50%; overflow-y:auto; background:#111b21; color:#fff; border-radius:16px 16px 0 0; padding:14px 18px; }
  .st-vs-head { font-weight:600; font-size:13px; opacity:.7; margin-bottom:10px; }
  .st-vs-row { display:flex; justify-content:space-between; padding:7px 0; font-size:14px; }
  .st-vs-time { opacity:.6; font-size:12px; }
  .st-vs-empty { opacity:.6; font-size:13px; padding:8px 0; }
</style>
