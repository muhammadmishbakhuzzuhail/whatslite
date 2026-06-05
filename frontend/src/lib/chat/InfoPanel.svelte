<script>
  import Avatar from "../common/Avatar.svelte";
  import { chats, activeChatId, infoOpen, blockContact, leaveGroup, fetchGroupInfo, pushToast } from "../../stores.js";
  import { avatarUrl, updateGroupParticipants, setGroupSubject, groupInviteLink, setGroupPhoto } from "../../services/data.js";
  import { initial } from "../util.js";
  import { t } from "../i18n.js";

  $: chat = $chats.find((c) => c.id === $activeChatId);

  let groupInfo = null, gLoaded = null;
  $: if (chat && chat.group && chat.id !== gLoaded) loadGroup(chat.id);
  async function loadGroup(id) { gLoaded = id; groupInfo = null; groupInfo = await fetchGroupInfo(id); }
  $: amAdmin = !!(groupInfo && groupInfo.amAdmin);

  const close = () => infoOpen.set(false);
  function doBlock() {
    if (!chat) return;
    blockContact(chat.id, true);
    pushToast($t("blocked_toast").replace("%s", chat.name), "ok");
  }
  function doLeave() {
    if (!chat) return;
    leaveGroup(chat.id);
    pushToast($t("left_toast").replace("%s", chat.name), "ok");
    infoOpen.set(false);
  }

  // --- Aksi admin grup ---
  function reloadSoon() { setTimeout(() => loadGroup(chat.id), 1200); }
  function editSubject() {
    const name = prompt($t("group_edit_name"), chat.name);
    if (name && name.trim() && name !== chat.name) { setGroupSubject(chat.id, name.trim()); reloadSoon(); }
  }
  function addMember() {
    const num = prompt($t("group_add_prompt"));
    if (!num) return;
    const digits = num.replace(/[^0-9]/g, "");
    if (digits.length < 6) return;
    updateGroupParticipants(chat.id, [digits + "@s.whatsapp.net"], "add");
    reloadSoon();
  }
  function memberAction(p, action) {
    updateGroupParticipants(chat.id, [p.jid], action);
    reloadSoon();
  }
  async function copyInvite() {
    const link = await groupInviteLink(chat.id, false);
    if (link) { navigator.clipboard?.writeText(link); pushToast($t("invite_copied"), "ok"); }
  }
  let photoInput;
  function pickPhoto() { photoInput && photoInput.click(); }
  function onPhoto(e) {
    const f = e.target.files && e.target.files[0];
    if (!f) return;
    const r = new FileReader();
    r.onload = () => { setGroupPhoto(chat.id, r.result); reloadSoon(); };
    r.readAsDataURL(f);
    e.target.value = "";
  }
</script>

{#if chat}
  <aside class="info-panel">
    <div class="info-head">
      <button class="icon-btn" title={$t("close")} on:click={close}>
        <svg viewBox="0 0 24 24"><path d="M6 6l12 12M18 6L6 18"/></svg>
      </button>
      <span class="title">{chat.group ? $t("info_group") : $t("info_contact")}</span>
    </div>

    <div class="info-hero">
      {#if chat.photo}
        <img class="avatar big photo" src={chat.photo} alt={chat.name} />
      {:else if chat.group}
        <div class="avatar big group" style="--c:{chat.color}">
          <svg viewBox="0 0 24 24"><path d="M16 11c1.66 0 3-1.34 3-3s-1.34-3-3-3-3 1.34-3 3 1.34 3 3 3zm-8 0c1.66 0 3-1.34 3-3S9.66 5 8 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V18h14v-1.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.99 1.97 3.45V18h6v-1.5c0-2.33-4.67-3.5-7-3.5z"/></svg>
        </div>
      {:else}
        <div class="avatar big" style="--c:{chat.color}"><span>{initial(chat.name)}</span></div>
      {/if}
      {#if chat.group && amAdmin}
        <button class="hero-photo-btn" title={$t("group_set_photo")} on:click={pickPhoto}>
          <svg viewBox="0 0 24 24"><path d="M4 7h3l2-2h6l2 2h3v12H4z"/><circle cx="12" cy="13" r="3.5"/></svg>
        </button>
        <input type="file" accept="image/*" bind:this={photoInput} on:change={onPhoto} style="display:none" />
      {/if}
      <div class="iname">
        {chat.name}
        {#if chat.group && amAdmin}<button class="edit-pen" title={$t("group_edit_name")} on:click={editSubject}><svg viewBox="0 0 24 24"><path d="M4 20h4L18 10l-4-4L4 16z"/><path d="M14 6l4 4"/></svg></button>{/if}
      </div>
      <div class="iphone">{chat.group ? (groupInfo ? $t("members_n").replace("%n", groupInfo.participants.length) : chat.status) : chat.phone || chat.status}</div>
    </div>

    {#if chat.group && groupInfo && groupInfo.participants}
      {#if amAdmin}
        <div class="info-group">
          <button class="info-row" on:click={addMember}>
            <svg viewBox="0 0 24 24"><circle cx="9" cy="8" r="4"/><path d="M2 20c0-3.5 3-6 7-6M18 11v6M15 14h6"/></svg>
            <span class="grow">{$t("group_add_member")}</span>
          </button>
          <button class="info-row" on:click={copyInvite}>
            <svg viewBox="0 0 24 24"><path d="M9 15l6-6M8 13l-2 2a3 3 0 0 0 4 4l2-2M16 11l2-2a3 3 0 0 0-4-4l-2 2"/></svg>
            <span class="grow">{$t("invite_link")}</span>
          </button>
        </div>
      {/if}
      <div class="info-block">
        <div class="lbl">{$t("members_n").replace("%n", groupInfo.participants.length)}</div>
        <div class="members">
          {#each groupInfo.participants as p (p.jid)}
            <div class="member">
              <Avatar name={p.name} color={chat.color} photo={avatarUrl(p.jid)} sm={true} />
              <span class="m-name">{p.name || p.jid.split("@")[0]}</span>
              {#if p.isAdmin}<span class="m-admin">{$t("member_admin")}</span>{/if}
              {#if amAdmin}
                <span class="m-actions">
                  {#if p.isAdmin}
                    <button title={$t("demote")} on:click={() => memberAction(p, "demote")}>▼</button>
                  {:else}
                    <button title={$t("promote")} on:click={() => memberAction(p, "promote")}>▲</button>
                  {/if}
                  <button title={$t("remove_member")} class="danger" on:click={() => memberAction(p, "remove")}>✕</button>
                </span>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {:else}
      <div class="info-block">
        <div class="lbl">{chat.group ? $t("info_groupdesc") : $t("info_about")}</div>
        <div class="val">{chat.about || "—"}</div>
      </div>
    {/if}

    <div class="info-row" style="align-items:flex-start">
      <svg viewBox="0 0 24 24"><rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/></svg>
      <div class="grow">
        <div>{$t("info_enc")}</div>
        <div class="sub">{$t("info_enc_sub")}</div>
      </div>
    </div>

    <div class="info-group">
      {#if chat.group}
        <button class="info-row danger" on:click={doLeave}>
          <svg viewBox="0 0 24 24"><path d="M15 4h3a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-3"/><path d="M10 17l-5-5 5-5M5 12h11"/></svg>
          <span class="grow">{$t("leave_group")}</span>
        </button>
      {:else}
        <button class="info-row danger" on:click={doBlock}>
          <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M5.5 5.5l13 13"/></svg>
          <span class="grow">{$t("block", { name: chat.name })}</span>
        </button>
        <button class="info-row danger">
          <svg viewBox="0 0 24 24"><path d="M10 3h4l1 4h5v3H4V7h5z"/><path d="M6 10v9a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2v-9"/></svg>
          <span class="grow">{$t("report", { name: chat.name })}</span>
        </button>
      {/if}
    </div>
  </aside>
{/if}

<style>
  .info-hero { position: relative; }
  .hero-photo-btn { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); width: 40px; height: 40px; border-radius: 50%; background: rgba(0,0,0,.45); border: 0; color: #fff; cursor: pointer; display: grid; place-items: center; }
  .hero-photo-btn svg { width: 20px; height: 20px; fill: none; stroke: currentColor; stroke-width: 2; }
  .iname { display: inline-flex; align-items: center; gap: 8px; }
  .edit-pen { background: none; border: 0; color: var(--text2); cursor: pointer; padding: 2px; }
  .edit-pen svg { width: 16px; height: 16px; fill: none; stroke: currentColor; stroke-width: 2; }
  .edit-pen:hover { color: var(--accent); }
  .member { display: flex; align-items: center; gap: 10px; }
  .m-name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .m-actions { display: inline-flex; gap: 4px; }
  .m-actions button { background: none; border: 0; cursor: pointer; color: var(--text2); font-size: 13px; padding: 3px 6px; border-radius: 6px; }
  .m-actions button:hover { background: var(--hover); color: var(--text); }
  .m-actions button.danger:hover { color: #e0463e; }
</style>
