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
  import Toast from "./lib/Toast.svelte";
  import { theme, infoOpen, loggedIn, lockState, inChatSearch, activeChatId, newChatOpen, lightbox, forwardDraft, profileJid } from "./stores.js";
  import { locale } from "./lib/i18n.js";

  $: document.documentElement.setAttribute("data-theme", $theme);
  $: document.documentElement.setAttribute("lang", $locale);

  // Shortcut keyboard global.
  function onKey(e) {
    const mod = e.ctrlKey || e.metaKey;
    if (mod && e.key.toLowerCase() === "f") {
      if ($activeChatId) { e.preventDefault(); inChatSearch.set(true); }
    } else if (mod && e.key.toLowerCase() === "k") {
      e.preventDefault(); newChatOpen.set(true);
    } else if (e.key === "Escape") {
      // tutup overlay teratas (urutan prioritas).
      if ($lightbox) lightbox.set(null);
      else if ($forwardDraft) forwardDraft.set(null);
      else if ($inChatSearch) inChatSearch.set(false);
      else if ($profileJid) profileJid.set(null);
      else if ($infoOpen) infoOpen.set(false);
    }
  }
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
{/if}
<Toast />
