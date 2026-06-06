<script>
  // ContactProfile — panel profil kontak (klik mention / pengirim grup).
  // Tampilkan avatar/nama/nomor/about + aksi Pesan & Simpan-nama (label LOKAL,
  // tidak sinkron ke buku-alamat HP/WA).
  import { profileJid, closeProfile, messageContact, saveContactLabel, removeContactLabel, pushToast, lightbox } from "../../stores.js";
  import { getContactProfile, avatarUrl, senderColorFor, getBusinessProfile, getCommonGroups } from "../../services/data.js";
  import { activeChatId, railView } from "../../stores.js";
  import { initial } from "../util.js";
  import { t } from "../i18n.js";

  let prof = null, loadedFor = null, biz = null, common = [];
  $: if ($profileJid && $profileJid !== loadedFor) load($profileJid);
  async function load(jid) {
    loadedFor = jid; prof = null; biz = null; common = [];
    prof = await getContactProfile(jid);
    biz = await getBusinessProfile(jid); // {isBiz, address, email, category}
    common = await getCommonGroups(jid);
  }
  function openGroup(jid) { activeChatId.set(jid); railView.set("chats"); closeProfile(); }

  // prompt() tak jalan di WebKitGTK → modal inline.
  let renameOpen = false, renameVal = "";
  function openSave() { renameVal = prof?.name || ""; renameOpen = true; }
  function doSave() {
    const v = renameVal.trim();
    if (!prof || !v) return;
    saveContactLabel(prof.jid, v);
    prof = { ...prof, name: v, saved: true };
    renameOpen = false;
    pushToast($t("contact_saved").replace("%s", v), "ok");
  }
  function doRemove() {
    if (!prof) return;
    removeContactLabel(prof.jid);
    prof = { ...prof, saved: false };
    pushToast($t("contact_label_removed"), "ok");
  }
</script>

{#if $profileJid}
  <aside class="info-panel">
    <div class="info-head">
      <button class="icon-btn" title={$t("close")} on:click={closeProfile}>
        <svg viewBox="0 0 24 24"><path d="M6 6l12 12M18 6L6 18"/></svg>
      </button>
      <span class="title">{$t("info_contact")}</span>
    </div>

    {#if prof}
      <div class="info-hero">
        {#if avatarUrl(prof.jid)}
          <img class="avatar big photo zoomable" src={avatarUrl(prof.jid)} alt={prof.name}
            role="button" tabindex="0"
            on:click={() => lightbox.set({ url: avatarUrl(prof.jid), type: "image", caption: prof.name })}
            on:keydown={(e) => (e.key === "Enter" || e.key === " ") && (e.preventDefault(), lightbox.set({ url: avatarUrl(prof.jid), type: "image", caption: prof.name }))}
            on:error={(e) => (e.target.style.display = 'none')} />
        {:else if /[\p{L}]/u.test(initial(prof.name))}
          <div class="avatar big" style="--c:{senderColorFor(prof.jid)}"><span>{initial(prof.name)}</span></div>
        {:else}
          <div class="avatar big def" style="--c:{senderColorFor(prof.jid)}">
            <svg class="person" viewBox="0 0 24 24"><circle cx="12" cy="8.5" r="4"/><path d="M4.5 20c0-4.2 3.8-6.5 7.5-6.5s7.5 2.3 7.5 6.5z"/></svg>
          </div>
        {/if}
        <div class="iname">{prof.name}</div>
        {#if prof.phone}<div class="iphone">{prof.phone}</div>{/if}
        {#if prof.saved}<div class="saved-chip">{$t("contact_saved_chip")}</div>{/if}
        {#if biz && biz.isBiz}<div class="biz-chip">✔ {$t("business_account")}{biz.category ? ` · ${biz.category}` : ""}</div>{/if}
      </div>

      {#if biz && biz.isBiz && (biz.address || biz.email)}
        <div class="info-block">
          <div class="lbl">{$t("business_info")}</div>
          {#if biz.address}<div class="val">📍 {biz.address}</div>{/if}
          {#if biz.email}<div class="val">✉️ {biz.email}</div>{/if}
        </div>
      {/if}

      {#if prof.about}
        <div class="info-block">
          <div class="lbl">{$t("info_about")}</div>
          <div class="val">{prof.about}</div>
        </div>
      {/if}

      <div class="info-group">
        <button class="info-row" on:click={() => messageContact(prof.jid)}>
          <svg viewBox="0 0 24 24"><path d="M4 5h16v11H8l-4 4z"/></svg>
          <span class="grow">{$t("message_action")}</span>
        </button>
        <button class="info-row" on:click={openSave}>
          <svg viewBox="0 0 24 24"><circle cx="9" cy="8" r="4"/><path d="M2 20c0-3.5 3-6 7-6M17 11v6M14 14h6"/></svg>
          <span class="grow">{prof.saved ? $t("rename_contact") : $t("save_contact")}</span>
        </button>
        {#if prof.saved}
          <button class="info-row danger" on:click={doRemove}>
            <svg viewBox="0 0 24 24"><path d="M4 7h16M9 7V5h6v2M6 7l1 13h10l1-13"/></svg>
            <span class="grow">{$t("remove_label")}</span>
          </button>
        {/if}
      </div>
      {#if common.length}
        <div class="info-block">
          <div class="lbl">{$t("common_groups")} ({common.length})</div>
          {#each common as g (g.jid)}
            <button class="info-row" on:click={() => openGroup(g.jid)} style="width:100%">
              <svg viewBox="0 0 24 24"><circle cx="9" cy="9" r="3"/><path d="M2 20c0-3 3-5 7-5M16 8a3 3 0 0 1 0 6M15 20c0-2 2-4 5-4"/></svg>
              <span class="grow" style="text-align:left">{g.name || g.jid.split("@")[0]}</span>
            </button>
          {/each}
        </div>
      {/if}

      <div class="local-note">{$t("local_label_note")}</div>
    {:else}
      <div class="prof-loading">…</div>
    {/if}
  </aside>
{/if}

{#if renameOpen}
  <div class="nc-modal" role="presentation" on:click|self={() => (renameOpen = false)}>
    <div class="nc-card" style="max-width:360px">
      <h3 style="margin:0 0 14px">{prof?.saved ? $t("rename_contact") : $t("save_contact")}</h3>
      <input class="poll-in" placeholder={$t("profile_name")} bind:value={renameVal}
        on:keydown={(e) => e.key === "Enter" && doSave()} />
      <div class="poll-foot">
        <button class="btn-ghost" on:click={() => (renameOpen = false)}>{$t("cancel")}</button>
        <button class="btn-accent" on:click={doSave} disabled={!renameVal.trim()}>{$t("save")}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .saved-chip { margin-top: 6px; font-size: 12px; color: var(--accent); }
  .local-note { padding: 10px 16px; font-size: 11px; color: var(--text2); line-height: 1.4; }
  .prof-loading { padding: 40px; text-align: center; color: var(--text2); }
</style>
