<script>
  import { onMount } from "svelte";
  import { railView, updateMyName, updateMyAbout } from "../../stores.js";
  import { getProfile, fetchProfile, myQR } from "../../services/data.js";
  import { initial } from "../util.js";
  import { t } from "../i18n.js";

  let me = getProfile(); // mock instan
  onMount(async () => { me = await fetchProfile(); });

  let qrImg = null, qrOpen = false;
  async function showQR() { qrOpen = true; if (!qrImg) qrImg = await myQR(false); }

  const pencil = '<svg viewBox="0 0 24 24"><path d="M4 20h4L18 10l-4-4L4 16v4z"/><path d="M14 6l4 4"/></svg>';

  let editName = false, nameVal = "";
  let editAbout = false, aboutVal = "";
  function startName() { nameVal = me.name; editName = true; }
  function saveName() {
    nameVal = nameVal.trim();
    if (nameVal && nameVal !== me.name) { me = { ...me, name: nameVal }; updateMyName(nameVal); }
    editName = false;
  }
  function startAbout() { aboutVal = me.about; editAbout = true; }
  function saveAbout() {
    me = { ...me, about: aboutVal.trim() }; updateMyAbout(aboutVal.trim()); editAbout = false;
  }
</script>

<header class="pane-head" style="gap:22px">
  <button class="icon-btn" title={$t("back")} on:click={() => railView.set("settings")}>
    <svg viewBox="0 0 24 24"><path d="M15 5l-7 7 7 7"/></svg>
  </button>
  <h2 style="font-size:17px">{$t("profile")}</h2>
</header>

<div style="flex:1; overflow-y:auto">
  <div class="profile-hero">
    <div class="avatar big" style="--c:{me.color}"><span>{initial(me.name)}</span></div>
  </div>

  <div class="profile-field">
    <div class="pf-lbl">{$t("profile_name")}</div>
    {#if editName}
      <input class="pf-edit" bind:value={nameVal} on:blur={saveName}
        on:keydown={(e) => e.key === "Enter" && saveName()} autofocus />
    {:else}
      <div class="pf-val" role="button" tabindex="0" on:click={startName} on:keydown={(e) => e.key === "Enter" && startName()}>
        {me.name} {@html pencil}
      </div>
    {/if}
    <div class="pf-note">{$t("profile_name_note")}</div>
  </div>

  <div class="profile-field">
    <div class="pf-lbl">{$t("profile_info")}</div>
    {#if editAbout}
      <input class="pf-edit" bind:value={aboutVal} on:blur={saveAbout}
        on:keydown={(e) => e.key === "Enter" && saveAbout()} autofocus />
    {:else}
      <div class="pf-val" role="button" tabindex="0" on:click={startAbout} on:keydown={(e) => e.key === "Enter" && startAbout()}>
        {me.about || "—"} {@html pencil}
      </div>
    {/if}
  </div>

  <div class="profile-field">
    <div class="pf-lbl">{$t("profile_phone")}</div>
    <div class="pf-val">{me.phone}</div>
  </div>

  <div class="profile-field">
    <button class="pf-qr-btn" on:click={showQR}>
      <svg viewBox="0 0 24 24"><rect x="4" y="4" width="6" height="6"/><rect x="14" y="4" width="6" height="6"/><rect x="4" y="14" width="6" height="6"/><path d="M14 14h2v2M18 14h2M20 18v2h-2M16 20h-2"/></svg>
      {$t("my_qr")}
    </button>
  </div>
</div>

{#if qrOpen}
  <button class="modal-backdrop" aria-label={$t("close")} on:click={() => (qrOpen = false)}></button>
  <div class="qr-modal" role="dialog" aria-modal="true">
    <div class="qr-title">{$t("my_qr")}</div>
    {#if qrImg}<img class="qr-modal-img" src={qrImg} alt="QR" />{:else}<div class="qr-loading">…</div>{/if}
    <div class="qr-sub">{$t("my_qr_sub")}</div>
  </div>
{/if}

<style>
  .pf-qr-btn { display: flex; align-items: center; gap: 10px; width: 100%; background: var(--bg2); border: 0; border-radius: 10px; padding: 12px 14px; color: var(--accent); font: inherit; font-weight: 600; cursor: pointer; }
  .pf-qr-btn svg { width: 22px; height: 22px; fill: none; stroke: currentColor; stroke-width: 2; }
  .qr-modal { position: fixed; z-index: 96; top: 50%; left: 50%; transform: translate(-50%, -50%); width: min(360px, 90vw); background: var(--bg); border: 1px solid var(--line); border-radius: 16px; box-shadow: 0 16px 50px rgba(0,0,0,.35); padding: 22px; text-align: center; }
  .qr-title { font-weight: 700; color: var(--text); margin-bottom: 14px; }
  .qr-modal-img { width: 260px; height: 260px; border-radius: 12px; background: #fff; padding: 8px; }
  .qr-loading { padding: 80px; color: var(--text2); }
  .qr-sub { margin-top: 12px; color: var(--text2); font-size: 13px; }
</style>
