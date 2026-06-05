<script>
  import Avatar from "../common/Avatar.svelte";
  import { chats, activeChatId, infoOpen, blockContact, leaveGroup, fetchGroupInfo, pushToast } from "../../stores.js";
  import { avatarUrl } from "../../services/data.js";
  import { initial } from "../util.js";
  import { t } from "../i18n.js";

  $: chat = $chats.find((c) => c.id === $activeChatId);

  let groupInfo = null, gLoaded = null;
  $: if (chat && chat.group && chat.id !== gLoaded) loadGroup(chat.id);
  async function loadGroup(id) { gLoaded = id; groupInfo = null; groupInfo = await fetchGroupInfo(id); }

  const close = () => infoOpen.set(false);
  function doBlock() {
    if (!chat) return;
    blockContact(chat.id, true);
    pushToast(chat.name + " diblokir", "ok");
  }
  function doLeave() {
    if (!chat) return;
    leaveGroup(chat.id);
    pushToast("Keluar dari " + chat.name, "ok");
    infoOpen.set(false);
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
      <div class="iname">{chat.name}</div>
      <div class="iphone">{chat.group ? (groupInfo ? groupInfo.participants.length + " anggota" : chat.status) : chat.phone || chat.status}</div>
    </div>

    {#if chat.group && groupInfo && groupInfo.participants}
      <div class="info-block">
        <div class="lbl">{groupInfo.participants.length} anggota</div>
        <div class="members">
          {#each groupInfo.participants as p (p.jid)}
            <div class="member">
              <Avatar name={p.name} color={chat.color} photo={avatarUrl(p.jid)} sm={true} />
              <span class="m-name">{p.name || p.jid.split("@")[0]}</span>
              {#if p.isAdmin}<span class="m-admin">admin</span>{/if}
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
          <span class="grow">Keluar grup</span>
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
