<script>
  // ContactProfile — panel profil kontak (klik mention / pengirim grup).
  // Tampilkan avatar/nama/nomor/about + aksi Pesan & Simpan-nama (label LOKAL,
  // tidak sinkron ke buku-alamat HP/WA).
  import { profileJid, closeProfile, messageContact, saveContactLabel, removeContactLabel, pushToast } from "../../stores.js";
  import { getContactProfile, avatarUrl, senderColorFor } from "../../services/data.js";
  import { initial } from "../util.js";
  import { t } from "../i18n.js";

  let prof = null, loadedFor = null;
  $: if ($profileJid && $profileJid !== loadedFor) load($profileJid);
  async function load(jid) {
    loadedFor = jid; prof = null;
    prof = await getContactProfile(jid);
  }

  function doSave() {
    if (!prof) return;
    const name = prompt($t("save_contact_prompt"), prof.name || "");
    if (name === null) return;
    const v = name.trim();
    if (!v) return;
    saveContactLabel(prof.jid, v);
    prof = { ...prof, name: v, saved: true };
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
          <img class="avatar big photo" src={avatarUrl(prof.jid)} alt={prof.name} on:error={(e) => (e.target.style.display = 'none')} />
        {:else}
          <div class="avatar big" style="--c:{senderColorFor(prof.jid)}"><span>{initial(prof.name)}</span></div>
        {/if}
        <div class="iname">{prof.name}</div>
        {#if prof.phone}<div class="iphone">{prof.phone}</div>{/if}
        {#if prof.saved}<div class="saved-chip">{$t("contact_saved_chip")}</div>{/if}
      </div>

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
        <button class="info-row" on:click={doSave}>
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
      <div class="local-note">{$t("local_label_note")}</div>
    {:else}
      <div class="prof-loading">…</div>
    {/if}
  </aside>
{/if}

<style>
  .saved-chip { margin-top: 6px; font-size: 12px; color: var(--accent); }
  .local-note { padding: 10px 16px; font-size: 11px; color: var(--text2); line-height: 1.4; }
  .prof-loading { padding: 40px; text-align: center; color: var(--text2); }
</style>
