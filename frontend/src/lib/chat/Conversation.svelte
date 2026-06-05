<script>
  import ConvHeader from "./ConvHeader.svelte";
  import MessageList from "./MessageList.svelte";
  import Composer from "./Composer.svelte";
  import { chats, activeChatId, allMessages, loadMessages, openChat, pinnedVersion, jumpMsg, pinMessageAction } from "../../stores.js";
  import { t } from "../i18n.js";

  $: chat = $chats.find((c) => c.id === $activeChatId);
  $: if ($activeChatId != null) { loadMessages($activeChatId); openChat($activeChatId); }
  $: messages = $allMessages[$activeChatId] || [];
  // Foto profil (peer & pengirim grup) dimuat lazy via /avatar di komponen Avatar.

  // Pesan tersemat: turunan dari pesan termuat (reaktif thd pin/unpin + versi).
  $: pinned = ($pinnedVersion, (messages || []).filter((m) => m.pinned));
  $: topPin = pinned.length ? pinned[pinned.length - 1] : null;
  function jumpPin() { if (topPin) jumpMsg.set(topPin.id); }
  function unpinTop() {
    if (!topPin) return;
    const idx = messages.findIndex((m) => m.id === topPin.id);
    if (idx >= 0) pinMessageAction($activeChatId, idx, false);
  }
</script>

<section class="conversation">
  {#if chat}
    <ConvHeader {chat} />
    {#if topPin}
      <div class="pinned-strip">
        <svg viewBox="0 0 24 24"><path d="M12 17v5M7 4h10l-1 6 3 3H5l3-3-1-6z"/></svg>
        <div class="ps-text" role="button" tabindex="0" on:click={jumpPin} on:keydown={(e) => e.key === "Enter" && jumpPin()}>
          <b>{$t("pinned")}{pinned.length > 1 ? ` (${pinned.length})` : ""}:</b> {topPin.text || `(${$t("media_generic")})`}
        </div>
        <button title={$t("unpin")} on:click={unpinTop} style="margin-left:auto;background:none;border:0;color:var(--text2);cursor:pointer;font-size:14px;padding:4px 8px">✕</button>
      </div>
    {/if}
    <MessageList {messages} group={!!chat.group} chatId={chat.id} peerName={chat.name} />
    <Composer chatId={chat.id} group={!!chat.group} />
  {:else}
    <div class="conv-splash">
      <div class="splash-logo">
        <svg viewBox="0 0 24 24"><path d="M12 3C6.5 3 2 6.8 2 11.5c0 2.3 1.1 4.4 2.9 5.9-.1 1.2-.6 2.6-1.4 3.6 1.6-.2 3.2-.8 4.4-1.6 1.2.4 2.6.6 4.1.6 5.5 0 10-3.8 10-8.5S17.5 3 12 3z"/></svg>
      </div>
      <h2>{$t("splash_title")}</h2>
      <p>{$t("splash_sub")}</p>
      <div class="splash-enc">
        <svg viewBox="0 0 24 24"><rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/></svg>
        {$t("splash_enc")}
      </div>
    </div>
  {/if}
</section>
