<script>
  import { onMount } from "svelte";
  import { railView, updateMyName, updateMyAbout } from "../../stores.js";
  import { getProfile, fetchProfile } from "../../services/data.js";
  import { initial } from "../util.js";
  import { t } from "../i18n.js";

  let me = getProfile(); // mock instan
  onMount(async () => { me = await fetchProfile(); });

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
</div>
