<script>
  import Avatar from "../common/Avatar.svelte";
  import { chats, activeChatId, infoOpen, blockContact, leaveGroup, fetchGroupInfo, pushToast, clearChatMessages, lightbox, askConfirm } from "../../stores.js";
  import { avatarUrl, updateGroupParticipants, setGroupSubject, setGroupDescription, groupInviteLink, setGroupPhoto, setDisappearing, exportChat, setGroupAnnounce, setGroupLocked, setGroupJoinApproval, setGroupAddMode, getGroupRequests, updateGroupRequest, getChatMedia } from "../../services/data.js";
  import { initial } from "../util.js";
  import { t } from "../i18n.js";

  $: chat = $chats.find((c) => c.id === $activeChatId);

  let groupInfo = null, gLoaded = null;
  let descOpen = false; // deskripsi grup/about: klem + "baca selengkapnya"
  $: $activeChatId, (descOpen = false); // reset saat ganti chat
  $: if (chat && chat.group && chat.id !== gLoaded) loadGroup(chat.id);
  async function loadGroup(id) { gLoaded = id; groupInfo = null; groupInfo = await fetchGroupInfo(id); }
  $: amAdmin = !!(groupInfo && groupInfo.amAdmin);

  const close = () => infoOpen.set(false);
  function doBlock() {
    if (!chat) return;
    askConfirm($t("block_confirm", { name: chat.name }), () => {
      blockContact(chat.id, true);
      pushToast($t("blocked_toast").replace("%s", chat.name), "ok");
    });
  }
  function doLeave() {
    if (!chat) return;
    askConfirm($t("leave_confirm", { name: chat.name }), () => {
      leaveGroup(chat.id);
      pushToast($t("left_toast").replace("%s", chat.name), "ok");
      infoOpen.set(false);
    });
  }

  // --- Aksi admin grup ---
  function reloadSoon() { setTimeout(() => loadGroup(chat.id), 1200); }
  // prompt() native tak jalan di WebKitGTK → modal input inline.
  let pm = null; // {title, value, placeholder, ok(value)}
  function submitPm() { if (!pm) return; const v = pm.value; const fn = pm.ok; pm = null; fn(v); }
  function editSubject() {
    pm = { title: $t("group_edit_name"), value: chat.name, placeholder: chat.name, ok: (name) => {
      if (name && name.trim() && name !== chat.name) { setGroupSubject(chat.id, name.trim()); reloadSoon(); }
    } };
  }
  function editDesc() {
    const cur = groupInfo ? (groupInfo.topic || "") : "";
    pm = { title: $t("group_edit_desc"), value: cur, placeholder: "", ok: (d) => {
      if (d != null && d.trim() !== cur) { setGroupDescription(chat.id, d.trim()); reloadSoon(); }
    } };
  }
  function addMember() {
    pm = { title: $t("group_add_prompt"), value: "", placeholder: "62812…", ok: (num) => {
      const digits = (num || "").replace(/[^0-9]/g, "");
      if (digits.length < 6) return;
      updateGroupParticipants(chat.id, [digits + "@s.whatsapp.net"], "add");
      reloadSoon();
    } };
  }
  // Setelan admin grup → ubah lalu muat ulang info.
  function toggle(fn, val) { fn(chat.id, val); reloadSoon(); }
  // Permintaan bergabung (mode approval).
  let requests = [], reqFor = null;
  $: if (groupInfo && amAdmin && groupInfo.joinApproval && reqFor !== chat.id) loadRequests();
  async function loadRequests() { reqFor = chat.id; requests = await getGroupRequests(chat.id); }
  function decideReq(jid, approve) {
    updateGroupRequest(chat.id, [jid], approve);
    requests = requests.filter((r) => r.jid !== jid);
    reloadSoon();
  }
  function memberAction(p, action) {
    if (action === "remove") {
      askConfirm($t("remove_member_confirm", { name: p.name || p.jid }), () => {
        updateGroupParticipants(chat.id, [p.jid], action); reloadSoon();
      });
      return;
    }
    updateGroupParticipants(chat.id, [p.jid], action);
    reloadSoon();
  }
  async function copyInvite() {
    const link = await groupInviteLink(chat.id, false);
    if (link) { navigator.clipboard?.writeText(link); pushToast($t("invite_copied"), "ok"); }
  }
  function resetInvite() {
    askConfirm($t("invite_reset_confirm"), async () => {
      const link = await groupInviteLink(chat.id, true); // reset=true → tautan baru
      if (link) { navigator.clipboard?.writeText(link); pushToast($t("invite_reset_done"), "ok"); }
    });
  }
  async function doExport() {
    const txt = await exportChat(chat.id);
    if (!txt) { pushToast($t("export_empty")); return; }
    const a = document.createElement("a");
    a.href = URL.createObjectURL(new Blob([txt], { type: "text/plain" }));
    a.download = `${(chat.name || "chat").replace(/[^\w-]+/g, "_")}.txt`;
    a.click();
    setTimeout(() => URL.revokeObjectURL(a.href), 4000);
  }
  function doClear() {
    askConfirm($t("clear_chat_confirm"), () => {
      clearChatMessages(chat.id);
      pushToast($t("clear_chat"), "ok");
    });
  }
  // Lapor: whatsmeow tak punya API report → konfirmasi lalu blokir + toast.
  function doReport() {
    askConfirm($t("report_confirm", { name: chat.name }), () => {
      blockContact(chat.id, true);
      pushToast($t("reported_toast"), "ok");
      infoOpen.set(false);
    });
  }
  // Galeri media chat (foto/video/stiker/gif/dok) untuk panel info.
  let media = [], mediaFor = null, mediaOpen = false;
  $: if (chat && chat.id !== mediaFor) loadMedia(chat.id);
  async function loadMedia(id) { mediaFor = id; mediaOpen = false; media = await getChatMedia(id); }
  function openMedia(m) {
    if (m.type === "image" || m.type === "sticker") lightbox.set({ url: `/media/${chat.id}/${m.id}`, type: "image" });
    else if (m.type === "video" || m.type === "gif") lightbox.set({ url: `/media/${chat.id}/${m.id}`, type: "video" });
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
      {#if avatarUrl(chat.id) || chat.photo}
        <img class="avatar big photo zoomable" src={avatarUrl(chat.id) || chat.photo} alt={chat.name}
          role="button" tabindex="0"
          on:click={() => lightbox.set({ url: avatarUrl(chat.id) || chat.photo, type: "image", caption: chat.name })}
          on:keydown={(e) => (e.key === "Enter" || e.key === " ") && (e.preventDefault(), lightbox.set({ url: avatarUrl(chat.id) || chat.photo, type: "image", caption: chat.name }))}
          on:error={(e) => (e.target.style.display = 'none')} />
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

    {#if chat.group && groupInfo}
      <div class="info-block">
        <div class="lbl">{$t("info_groupdesc")}{#if amAdmin}<button class="edit-pen" title={$t("group_edit_desc")} on:click={editDesc}><svg viewBox="0 0 24 24"><path d="M4 20h4L18 10l-4-4L4 16z"/><path d="M14 6l4 4"/></svg></button>{/if}</div>
        <div class="val desc" class:clamp={!descOpen} dir="auto">{groupInfo.topic || "—"}</div>
        {#if (groupInfo.topic || "").length > 140}<button class="read-more" on:click={() => (descOpen = !descOpen)}>{descOpen ? $t("read_less") : $t("read_more")}</button>{/if}
      </div>
    {/if}
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
          <button class="info-row" on:click={resetInvite}>
            <svg viewBox="0 0 24 24"><path d="M4 12a8 8 0 0 1 14-5l2 2M20 12a8 8 0 0 1-14 5l-2-2M18 4v5h-5M6 20v-5h5"/></svg>
            <span class="grow">{$t("invite_reset")}</span>
          </button>
        </div>

        <!-- Setelan admin grup -->
        <div class="info-block">
          <div class="lbl">{$t("group_admin_settings")}</div>
          <button class="info-row" on:click={() => toggle(setGroupAnnounce, !groupInfo.announce)}>
            <span class="grow">{$t("group_announce")}</span>
            <span class="switch {groupInfo.announce ? '' : 'off'}"></span>
          </button>
          <button class="info-row" on:click={() => toggle(setGroupLocked, !groupInfo.locked)}>
            <span class="grow">{$t("group_locked")}</span>
            <span class="switch {groupInfo.locked ? '' : 'off'}"></span>
          </button>
          <button class="info-row" on:click={() => toggle(setGroupJoinApproval, !groupInfo.joinApproval)}>
            <span class="grow">{$t("group_join_approval")}</span>
            <span class="switch {groupInfo.joinApproval ? '' : 'off'}"></span>
          </button>
          <button class="info-row" on:click={() => toggle(setGroupAddMode, !groupInfo.adminAddOnly)}>
            <span class="grow">{$t("group_admin_add")}</span>
            <span class="switch {groupInfo.adminAddOnly ? '' : 'off'}"></span>
          </button>
        </div>

        <!-- Permintaan bergabung (approval) -->
        {#if groupInfo.joinApproval && requests.length}
          <div class="info-block">
            <div class="lbl">{$t("group_requests")} ({requests.length})</div>
            <div class="members">
              {#each requests as r (r.jid)}
                <div class="member">
                  <Avatar name={r.name} color={chat.color} photo={avatarUrl(r.jid)} sm={true} />
                  <span class="m-name">{r.name || r.jid.split("@")[0]}</span>
                  <span class="m-actions">
                    <button title={$t("ok")} on:click={() => decideReq(r.jid, true)}>✓</button>
                    <button title={$t("call_reject")} class="danger" on:click={() => decideReq(r.jid, false)}>✕</button>
                  </span>
                </div>
              {/each}
            </div>
          </div>
        {/if}
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
        <div class="val desc" class:clamp={!descOpen} dir="auto">{chat.about || "—"}</div>
        {#if (chat.about || "").length > 140}<button class="read-more" on:click={() => (descOpen = !descOpen)}>{descOpen ? $t("read_less") : $t("read_more")}</button>{/if}
      </div>
    {/if}

    {#if media.length}
      <div class="info-block">
        <button class="lbl media-lbl" on:click={() => (mediaOpen = !mediaOpen)}>
          {$t("info_media")} ({media.length})
          <span class="chev">{mediaOpen ? "▾" : "▸"}</span>
        </button>
        <div class="media-grid">
          {#each (mediaOpen ? media : media.slice(0, 9)) as m (m.id)}
            {#if m.type === "document"}
              <div class="media-cell doc" title={m.text}><svg viewBox="0 0 24 24"><path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5"/></svg></div>
            {:else}
              <button class="media-cell" on:click={() => openMedia(m)}>
                <img src={`/media/${chat.id}/${m.id}`} alt="" loading="lazy" />
                {#if m.type === "video" || m.type === "gif"}<span class="media-play">▶</span>{/if}
              </button>
            {/if}
          {/each}
        </div>
      </div>
    {/if}

    <div class="info-row" style="align-items:center">
      <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 3"/></svg>
      <div class="grow">{$t("disappearing_msg")}</div>
      <select class="lang-select" on:change={(e) => setDisappearing(chat.id, +e.target.value)}>
        <option value="0">{$t("disappearing_off")}</option>
        <option value="86400">{$t("disappearing_24h")}</option>
        <option value="604800">{$t("disappearing_7d")}</option>
        <option value="7776000">{$t("disappearing_90d")}</option>
      </select>
    </div>

    <div class="info-row" style="align-items:flex-start">
      <svg viewBox="0 0 24 24"><rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/></svg>
      <div class="grow">
        <div>{$t("info_enc")}</div>
        <div class="sub">{$t("info_enc_sub")}</div>
      </div>
    </div>

    <div class="info-group">
      <button class="info-row" on:click={doExport}>
        <svg viewBox="0 0 24 24"><path d="M12 4v11M7 11l5 5 5-5M5 20h14"/></svg>
        <span class="grow">{$t("export_chat")}</span>
      </button>
      <button class="info-row danger" on:click={doClear}>
        <svg viewBox="0 0 24 24"><path d="M10 3h4l1 4h5v3H4V7h5z"/><path d="M6 10v9a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2v-9"/></svg>
        <span class="grow">{$t("clear_chat")}</span>
      </button>
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
        <button class="info-row danger" on:click={doReport}>
          <svg viewBox="0 0 24 24"><path d="M10 3h4l1 4h5v3H4V7h5z"/><path d="M6 10v9a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2v-9"/></svg>
          <span class="grow">{$t("report", { name: chat.name })}</span>
        </button>
      {/if}
    </div>
  </aside>
{/if}

{#if pm}
  <div class="nc-overlay" role="presentation" on:click|self={() => (pm = null)}>
    <div class="nc-card" style="max-width:380px">
      <h3 style="margin:0 0 14px">{pm.title}</h3>
      <input class="poll-in" placeholder={pm.placeholder} bind:value={pm.value}
        on:keydown={(e) => e.key === "Enter" && submitPm()} />
      <!-- svelte-ignore a11y-autofocus -->
      <div class="poll-foot">
        <button class="btn-ghost" on:click={() => (pm = null)}>{$t("cancel")}</button>
        <button class="btn-accent" on:click={submitPm}>{$t("save")}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .info-hero { position: relative; }
  .hero-photo-btn { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); width: 40px; height: 40px; border-radius: 50%; background: rgba(0,0,0,.45); border: 0; color: #fff; cursor: pointer; display: grid; align-items: center; justify-items: center; }
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
