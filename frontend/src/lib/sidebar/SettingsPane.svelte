<script>
  import { railView, theme, pinSet, beginSetPin, removePin, lockNow, logout, translateLang } from "../../stores.js";
  import { getProfile, getSettingsItems } from "../../services/data.js";
  import { TRANSLATE_LANGS } from "../langs.js";
  import { initial } from "../util.js";
  import { t, locale, languages } from "../i18n.js";
  const me = getProfile();
  const settingsItems = getSettingsItems();
  const toggleTheme = () => theme.update((v) => (v === "dark" ? "light" : "dark"));
  const toggleLock = () => ($pinSet ? removePin() : beginSetPin());

  const icons = {
    key: '<svg viewBox="0 0 24 24"><circle cx="8" cy="8" r="4"/><path d="M11 11l8 8M16 16l2-2M19 19l2-2"/></svg>',
    lock: '<svg viewBox="0 0 24 24"><rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/></svg>',
    chat: '<svg viewBox="0 0 24 24"><path d="M21 12a8 8 0 0 1-11.3 7.3L4 21l1.7-5.7A8 8 0 1 1 21 12z"/></svg>',
    bell: '<svg viewBox="0 0 24 24"><path d="M6 9a6 6 0 0 1 12 0c0 5 2 6 2 6H4s2-1 2-6z"/><path d="M10 20a2 2 0 0 0 4 0"/></svg>',
    disk: '<svg viewBox="0 0 24 24"><rect x="4" y="4" width="16" height="16" rx="2"/><path d="M8 4v6h8V4M8 16h.01"/></svg>',
    help: '<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M9.5 9a2.5 2.5 0 0 1 4 2c0 1.5-2 2-2 3.2"/><circle cx="11.6" cy="17" r=".6"/></svg>',
    globe: '<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.5 2.5 2.5 15 0 18M12 3C9.5 5.5 9.5 18.5 12 21"/></svg>',
    theme: '<svg viewBox="0 0 24 24"><path d="M21 13A9 9 0 1 1 11 3a7 7 0 0 0 10 10z"/></svg>',
    applock: '<svg viewBox="0 0 24 24"><rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/></svg>',
    logout: '<svg viewBox="0 0 24 24"><path d="M15 4h3a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-3"/><path d="M10 17l-5-5 5-5M5 12h11"/></svg>',
  };
</script>

<header class="pane-head"><h2>{$t("settings")}</h2></header>

<div style="flex:1; overflow-y:auto">
  <div class="settings-profile" role="button" tabindex="0" on:click={() => railView.set("profile")} on:keydown={(e) => e.key === "Enter" && railView.set("profile")}>
    <div class="avatar" style="--c:{me.color}"><span>{initial(me.name)}</span></div>
    <div class="sp-meta">
      <div class="sp-name">{me.name}</div>
      <div class="sp-about">{me.about}</div>
    </div>
  </div>

  <div class="settings-list">
    {#each settingsItems as s}
      <div class="settings-item" role="button" tabindex="0">
        {@html icons[s.icon]}
        <div>
          <div class="si-name">{$t(s.key)}</div>
          <div class="si-desc">{$t(s.key + "_d")}</div>
        </div>
      </div>
    {/each}

    <!-- Tema (terang/gelap) -->
    <div class="settings-item" role="button" tabindex="0" on:click={toggleTheme} on:keydown={(e) => e.key === "Enter" && toggleTheme()}>
      {@html icons.theme}
      <div class="grow">
        <div class="si-name">{$t("theme")}</div>
        <div class="si-desc">{$theme === "dark" ? $t("theme_dark") : $t("theme_light")}</div>
      </div>
      <span class="switch {$theme === 'dark' ? '' : 'off'}"></span>
    </div>

    <!-- Pemilih bahasa (i18n) -->
    <div class="settings-item lang-item">
      {@html icons.globe}
      <div class="grow">
        <div class="si-name">{$t("language")}</div>
      </div>
      <select class="lang-select" bind:value={$locale}>
        {#each languages as l}<option value={l.code}>{l.label}</option>{/each}
      </select>
    </div>

    <!-- Bahasa tujuan terjemahan pesan (deteksi sumber otomatis) -->
    <div class="settings-item lang-item">
      <svg viewBox="0 0 24 24"><path d="M4 5h7M9 3v2c0 4-2 7-5 9M5 9c0 3 3 5 6 5"/><path d="M14 19l3-7 3 7M15.5 16h3"/></svg>
      <div class="grow">
        <div class="si-name">Bahasa terjemahan</div>
        <div class="si-desc">Pesan diterjemahkan ke bahasa ini</div>
      </div>
      <select class="lang-select" bind:value={$translateLang}>
        {#each TRANSLATE_LANGS as l}<option value={l.code}>{l.name}</option>{/each}
      </select>
    </div>

    <!-- Pesan berbintang -->
    <div class="settings-item" role="button" tabindex="0" on:click={() => railView.set("starred")} on:keydown={(e) => e.key === "Enter" && railView.set("starred")}>
      <svg viewBox="0 0 24 24"><path d="M12 3l2.6 5.5 6 .8-4.4 4.2 1.1 6L12 16.8 6.7 19.5l1.1-6L3.4 9.3l6-.8z"/></svg>
      <div class="grow"><div class="si-name">{$t("starred_msg")}</div></div>
    </div>

    <!-- Kunci aplikasi -->
    <div class="settings-item" role="button" tabindex="0" on:click={toggleLock} on:keydown={(e) => e.key === "Enter" && toggleLock()}>
      {@html icons.applock}
      <div class="grow">
        <div class="si-name">{$t("applock")}</div>
        <div class="si-desc">{$pinSet ? $t("active") : $t("off")}</div>
      </div>
      <span class="switch {$pinSet ? '' : 'off'}"></span>
    </div>
    {#if $pinSet}
      <div class="settings-item" role="button" tabindex="0" on:click={lockNow} on:keydown={(e) => e.key === "Enter" && lockNow()}>
        {@html icons.applock}
        <div class="grow"><div class="si-name">{$t("lock_now")}</div></div>
      </div>
    {/if}

    <!-- Keluar / ganti akun -->
    <div class="settings-item danger" role="button" tabindex="0" on:click={logout} on:keydown={(e) => e.key === "Enter" && logout()}>
      {@html icons.logout}
      <div class="grow"><div class="si-name">{$t("logout")}</div></div>
    </div>
  </div>
</div>
