<script>
  import { railView } from "../stores.js";
  import { getProfile } from "../services/data.js";
  import { initial } from "./util.js";
  import { t } from "./i18n.js";
  const me = getProfile();
  const go = (v) => railView.set(v);
</script>

<nav class="rail">
  <div class="rail-top">
    <button class="rail-btn {$railView === 'chats' ? 'active' : ''}" title={$t("rail_chats")} on:click={() => go("chats")}>
      <svg viewBox="0 0 24 24"><path d="M12 3C6.5 3 2 6.8 2 11.5c0 2.3 1.1 4.4 2.9 5.9-.1 1.2-.6 2.6-1.4 3.6 1.6-.2 3.2-.8 4.4-1.6 1.2.4 2.6.6 4.1.6 5.5 0 10-3.8 10-8.5S17.5 3 12 3z"/></svg>
      <span class="rail-badge">5</span>
    </button>
    <button class="rail-btn" title={$t("rail_calls")}>
      <svg viewBox="0 0 24 24"><path d="M5 4h3l2 5-2.5 1.5a11 11 0 0 0 5 5L15 13l5 2v3a2 2 0 0 1-2 2A16 16 0 0 1 3 6a2 2 0 0 1 2-2z"/></svg>
      <span class="rail-badge">1</span>
    </button>
    <button class="rail-btn {$railView === 'status' ? 'active' : ''}" title={$t("rail_status")} on:click={() => go("status")}>
      <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9" stroke-dasharray="3 3"/></svg>
    </button>
    <button class="rail-btn {$railView === 'channels' ? 'active' : ''}" title={$t("rail_channels")} on:click={() => go("channels")}>
      <svg viewBox="0 0 24 24"><path d="M4 9v6h4l5 4V5L8 9H4z"/><path d="M16 8a5 5 0 0 1 0 8"/></svg>
    </button>
    <button class="rail-btn {$railView === 'communities' ? 'active' : ''}" title={$t("rail_communities")} on:click={() => go("communities")}>
      <svg viewBox="0 0 24 24"><circle cx="8" cy="9" r="3"/><circle cx="16" cy="9" r="2.2"/><path d="M3 19c0-2.5 2.2-4.5 5-4.5s5 2 5 4.5"/><path d="M14 19c0-1.8.9-3.3 2.3-3.9"/></svg>
    </button>
    <button class="rail-btn meta-ai" title={$t("rail_meta")}><span class="ai-ring"></span></button>
  </div>
  <div class="rail-bottom">
    <button class="rail-btn {$railView === 'settings' ? 'active' : ''}" title={$t("rail_settings")} on:click={() => go("settings")}>
      <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M12 2v3M12 19v3M2 12h3M19 12h3M5 5l2 2M17 17l2 2M19 5l-2 2M7 17l-2 2"/></svg>
    </button>
    <button class="rail-avatar" title={$t("rail_profile")} on:click={() => go("profile")} style="background:{me.color}"><span>{initial(me.name)}</span></button>
  </div>
</nav>
