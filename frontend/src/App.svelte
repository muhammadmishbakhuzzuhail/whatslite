<script>
  import Rail from "./lib/Rail.svelte";
  import Sidebar from "./lib/sidebar/Sidebar.svelte";
  import Conversation from "./lib/chat/Conversation.svelte";
  import InfoPanel from "./lib/chat/InfoPanel.svelte";
  import ContactProfile from "./lib/chat/ContactProfile.svelte";
  import Login from "./lib/Login.svelte";
  import AppLock from "./lib/AppLock.svelte";
  import ForwardModal from "./lib/chat/ForwardModal.svelte";
  import MessageInfoModal from "./lib/chat/MessageInfoModal.svelte";
  import Lightbox from "./lib/chat/Lightbox.svelte";
  import MediaPreviewModal from "./lib/chat/MediaPreviewModal.svelte";
  import ReactionPicker from "./lib/chat/ReactionPicker.svelte";
  import NewChatModal from "./lib/sidebar/NewChatModal.svelte";
  import IncomingCall from "./lib/chat/IncomingCall.svelte";
  import ConfirmDialog from "./lib/common/ConfirmDialog.svelte";
  import PromptDialog from "./lib/common/PromptDialog.svelte";
  import FolderPicker from "./lib/sidebar/FolderPicker.svelte";
  import Toast from "./lib/Toast.svelte";
  import { effectiveTheme, infoOpen, loggedIn, lockState, inChatSearch, activeChatId, newChatOpen, lightbox, forwardDraft, profileJid, chats, muteChat, markChatUnread } from "./stores.js";
  import { locale, t } from "./lib/i18n.js";

  $: document.documentElement.setAttribute("data-theme", $effectiveTheme);
  $: document.documentElement.setAttribute("lang", $locale);

  let shortcutsOpen = false;
  // Pindah chat relatif (+1 / -1) dalam urutan sidebar.
  function cycleChat(dir) {
    const list = $chats;
    if (!list.length) return;
    const i = list.findIndex((c) => c.id === $activeChatId);
    const n = ((i < 0 ? 0 : i) + dir + list.length) % list.length;
    activeChatId.set(list[n].id);
  }
  // Shortcut keyboard global (gaya WhatsApp Web).
  function onKey(e) {
    const mod = e.ctrlKey || e.metaKey;
    const k = e.key.toLowerCase();
    // Saat fokus di input/textarea/contenteditable → hanya Escape yang lewat,
    // shortcut lain JANGAN dibajak (cegah Ctrl+F/K/] mengganggu pengetikan).
    const el = e.target;
    const typing = el && (el.tagName === "INPUT" || el.tagName === "TEXTAREA" || el.isContentEditable);
    if (typing && e.key !== "Escape") return;
    if (e.key === "Escape") {
      if (shortcutsOpen) { shortcutsOpen = false; return; }
      if ($lightbox) lightbox.set(null);
      else if ($forwardDraft) forwardDraft.set(null);
      else if ($inChatSearch) inChatSearch.set(false);
      else if ($profileJid) profileJid.set(null);
      else if ($infoOpen) infoOpen.set(false);
      return;
    }
    if (!mod) return;
    if (mod && !e.shiftKey && k === "f") { if ($activeChatId) { e.preventDefault(); inChatSearch.set(true); } }
    else if (mod && !e.shiftKey && (k === "k" || k === "n")) { e.preventDefault(); newChatOpen.set(true); }
    else if (mod && e.shiftKey && k === "m") { e.preventDefault(); if ($activeChatId) { const c = $chats.find((x) => x.id === $activeChatId); muteChat($activeChatId, !(c && c.muted)); } }
    else if (mod && e.shiftKey && k === "u") { e.preventDefault(); if ($activeChatId) { const c = $chats.find((x) => x.id === $activeChatId); markChatUnread($activeChatId, !(c && c.unread)); } }
    else if (mod && (k === "]" || (e.shiftKey && k === "arrowdown"))) { e.preventDefault(); cycleChat(1); }
    else if (mod && (k === "[" || (e.shiftKey && k === "arrowup"))) { e.preventDefault(); cycleChat(-1); }
    else if (mod && k === "/") { e.preventDefault(); shortcutsOpen = !shortcutsOpen; }
  }
  const SHORTCUTS = [
    ["mod+F", "kbd_search"], ["mod+K", "kbd_new_chat"], ["mod+Shift+M", "kbd_mute"],
    ["mod+Shift+U", "kbd_unread"], ["mod+]", "kbd_next"], ["mod+[", "kbd_prev"],
    ["mod+/", "kbd_help"], ["Esc", "kbd_close"],
  ];
</script>

<svelte:window on:keydown={onKey} />

{#if $lockState !== "off"}
  <AppLock />
{:else if !$loggedIn}
  <Login />
{:else}
  <div class="app">
    <Rail />
    <Sidebar />
    <Conversation />
    {#if $infoOpen}<InfoPanel />{/if}
    {#if $profileJid}<ContactProfile />{/if}
    <ForwardModal />
    <MessageInfoModal />
    <Lightbox />
    <MediaPreviewModal />
    <ReactionPicker />
    <NewChatModal />
  </div>
  <IncomingCall />
{/if}
<ConfirmDialog />
<PromptDialog />
<FolderPicker />
{#if shortcutsOpen}
  <button class="modal-backdrop" aria-label={$t("close")} on:click={() => (shortcutsOpen = false)}></button>
  <div class="kbd-modal" role="dialog" aria-modal="true">
    <div class="kbd-title">{$t("kbd_title")}</div>
    {#each SHORTCUTS as [combo, key]}
      <div class="kbd-row"><span>{$t(key)}</span><kbd>{combo.replace("mod", navigator.platform.startsWith("Mac") ? "⌘" : "Ctrl")}</kbd></div>
    {/each}
  </div>
{/if}
<Toast />

<style>
  .kbd-modal { position: fixed; z-index: 96; top: 50%; left: 50%; transform: translate(-50%, -50%); width: min(380px, 92vw); background: var(--bg); border: 1px solid var(--line); border-radius: 16px; box-shadow: 0 16px 50px rgba(0,0,0,.35); padding: 20px; }
  .kbd-title { font-weight: 700; color: var(--text); margin-bottom: 14px; }
  .kbd-row { display: flex; justify-content: space-between; align-items: center; padding: 7px 0; color: var(--text); border-bottom: 1px solid var(--line); }
  .kbd-row kbd { background: var(--bg2); border-radius: 6px; padding: 3px 8px; font-size: 12px; font-family: monospace; color: var(--text2); }
</style>
