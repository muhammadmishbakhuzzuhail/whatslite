<script>
  import { railView, theme, pinSet, beginSetPin, removePin, lockNow, logout, translateLang, soundOn, showDeleted, accent, virtList } from "../../stores.js";
  import { getProfile, getRetention, setRetention, setDefaultDisappearing, getProxy, setProxy, getBackgroundClose, setBackgroundClose, quitApp, getKeepDeleted, setKeepDeleted, appVersion } from "../../services/data.js";
  import { uiScale, quickReactions, setQuickReaction, askPrompt, dnd } from "../../stores.js";
  function editQR(i) { askPrompt($t("quick_react_edit"), $quickReactions[i], (v) => setQuickReaction(i, [...v][0] || $quickReactions[i])); }
  import { TRANSLATE_LANGS } from "../langs.js";
  import LangPicker from "../common/LangPicker.svelte";
  import { initial } from "../util.js";
  import { onMount } from "svelte";
  import { t, locale } from "../i18n.js";
  const me = getProfile();
  const THEME_MODES = ["light", "dark", "system"];
  const RETENTIONS = [30, 90, 180, 0]; // 0 = selamanya
  let retDays = 90;
  let proxyVal = "", proxySaved = "", bgClose = false, keepDel = true, appVer = "dev";
  onMount(async () => { retDays = await getRetention(); proxyVal = proxySaved = await getProxy(); bgClose = await getBackgroundClose(); keepDel = await getKeepDeleted(); appVer = await appVersion(); });
  function toggleBg() { bgClose = !bgClose; setBackgroundClose(bgClose); }
  function toggleKeepDel() { keepDel = !keepDel; setKeepDeleted(keepDel); }
  function saveProxy() { if (proxyVal !== proxySaved) { setProxy(proxyVal.trim()); proxySaved = proxyVal.trim(); } }
  function pickRetention(d) { retDays = d; setRetention(d); }
  const DISAPPEAR = [[0, "disappearing_off"], [86400, "disappearing_24h"], [604800, "disappearing_7d"], [7776000, "disappearing_90d"]];
  let defDis = 0;
  function pickDisappear(s) { defDis = s; setDefaultDisappearing(s); }
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
    <!-- Tema app (terang / gelap / ikuti sistem) -->
    <div class="settings-item" style="align-items:flex-start">
      {@html icons.theme}
      <div class="grow">
        <div class="si-name">{$t("theme")}</div>
        <div class="theme-modes">
          {#each THEME_MODES as m}
            <button class="theme-mode {$theme === m ? 'on' : ''}" on:click={() => theme.set(m)}>{$t("theme_" + m)}</button>
          {/each}
        </div>
      </div>
    </div>

    <!-- Warna aksen kustom -->
    <div class="settings-item" style="align-items:flex-start">
      {@html icons.theme}
      <div class="grow">
        <div class="si-name">{$t("accent_color")}</div>
        <div class="theme-modes" style="gap:8px">
          {#each ["", "#06b67f", "#5b6ef5", "#e5614e", "#f2a33c", "#9b59b6", "#e9418a"] as c}
            <button class="acc-sw {(($accent || '') === c) ? 'on' : ''}" title={c || $t("wallpaper_default")}
              style="background:{c || 'var(--accent)'}" on:click={() => accent.set(c)}>{c ? '' : '✕'}</button>
          {/each}
        </div>
      </div>
    </div>

    <!-- Pemilih bahasa (i18n) -->
    <div class="settings-item lang-item">
      {@html icons.globe}
      <div class="grow">
        <div class="si-name">{$t("language")}</div>
        <div class="si-desc">{$t("language_d")}</div>
      </div>
      <LangPicker options={TRANSLATE_LANGS} value={$locale} onSelect={(c) => locale.set(c)} />
    </div>

    <!-- Bahasa tujuan terjemahan pesan (deteksi sumber otomatis) -->
    <div class="settings-item lang-item">
      <svg viewBox="0 0 24 24"><path d="M4 5h7M9 3v2c0 4-2 7-5 9M5 9c0 3 3 5 6 5"/><path d="M14 19l3-7 3 7M15.5 16h3"/></svg>
      <div class="grow">
        <div class="si-name">{$t("translate_lang")}</div>
        <div class="si-desc">{$t("translate_lang_d")}</div>
      </div>
      <LangPicker options={TRANSLATE_LANGS} value={$translateLang} onSelect={(c) => translateLang.set(c)} />
    </div>

    <!-- Suara notifikasi -->
    <div class="settings-item" role="button" tabindex="0" on:click={() => soundOn.update((v) => !v)} on:keydown={(e) => e.key === "Enter" && soundOn.update((v) => !v)}>
      <svg viewBox="0 0 24 24"><path d="M11 5L6 9H2v6h4l5 4zM15 9a3 3 0 0 1 0 6M18 6a7 7 0 0 1 0 12"/></svg>
      <div class="grow"><div class="si-name">{$t("notif_sound")}</div><div class="si-desc">{$soundOn ? $t("active") : $t("off")}</div></div>
      <span class="switch {$soundOn ? '' : 'off'}"></span>
    </div>

    <!-- Virtualisasi daftar pesan (eksperimental) -->
    <div class="settings-item" role="button" tabindex="0" on:click={() => virtList.update((v) => !v)} on:keydown={(e) => e.key === "Enter" && virtList.update((v) => !v)}>
      <svg viewBox="0 0 24 24"><path d="M4 6h16M4 12h16M4 18h16"/></svg>
      <div class="grow"><div class="si-name">{$t("virt_list")}</div><div class="si-desc">{$t("virt_list_d")}</div></div>
      <span class="switch {$virtList ? '' : 'off'}"></span>
    </div>

    <!-- Anti-delete: lihat pesan yang ditarik pengirim -->
    <div class="settings-item" role="button" tabindex="0" on:click={toggleKeepDel} on:keydown={(e) => e.key === "Enter" && toggleKeepDel()}>
      <svg viewBox="0 0 24 24"><path d="M3 6h18"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><path d="M6 6l1 14a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-14"/></svg>
      <div class="grow"><div class="si-name">{$t("keep_deleted")}</div><div class="si-desc">{$t("keep_deleted_d")}</div></div>
      <span class="switch {keepDel ? '' : 'off'}"></span>
    </div>
    {#if keepDel}
    <div class="settings-item" role="button" tabindex="0" on:click={() => showDeleted.update((v) => !v)} on:keydown={(e) => e.key === "Enter" && showDeleted.update((v) => !v)}>
      <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M5.6 5.6l12.8 12.8"/></svg>
      <div class="grow"><div class="si-name">{$t("show_deleted")}</div><div class="si-desc">{$t("show_deleted_d")}</div></div>
      <span class="switch {$showDeleted ? '' : 'off'}"></span>
    </div>
    {/if}

    <!-- Retensi pesan (jaga DB ramping) -->
    <div class="settings-item" style="align-items:flex-start">
      {@html icons.disk}
      <div class="grow">
        <div class="si-name">{$t("retention")}</div>
        <div class="si-desc">{$t("retention_d")}</div>
        <div class="theme-modes">
          {#each RETENTIONS as d}
            <button class="theme-mode {retDays === d ? 'on' : ''}" on:click={() => pickRetention(d)}>{d === 0 ? $t("retention_forever") : $t("retention_days", { n: d })}</button>
          {/each}
        </div>
      </div>
    </div>

    <!-- Timer hilang-otomatis default (chat baru) -->
    <div class="settings-item" style="align-items:flex-start">
      <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 3"/></svg>
      <div class="grow">
        <div class="si-name">{$t("default_disappearing")}</div>
        <div class="si-desc">{$t("default_disappearing_d")}</div>
        <div class="theme-modes">
          {#each DISAPPEAR as [s, key]}
            <button class="theme-mode {defDis === s ? 'on' : ''}" on:click={() => pickDisappear(s)}>{$t(key)}</button>
          {/each}
        </div>
      </div>
    </div>

    <!-- Proxy (privasi/jaringan; berlaku setelah restart) -->
    <div class="settings-item" style="align-items:flex-start">
      <svg viewBox="0 0 24 24"><path d="M4 12h16M12 4a15 15 0 0 1 0 16M12 4a15 15 0 0 0 0 16"/><circle cx="12" cy="12" r="9"/></svg>
      <div class="grow">
        <div class="si-name">{$t("proxy")}</div>
        <div class="si-desc">{$t("proxy_d")}</div>
        <input class="stk-search" style="margin-top:6px" placeholder="socks5://127.0.0.1:9050" bind:value={proxyVal} on:blur={saveProxy} on:keydown={(e) => e.key === "Enter" && saveProxy()} />
      </div>
    </div>

    <!-- Penyimpanan & data (rincian) -->
    <div class="settings-item" role="button" tabindex="0" on:click={() => railView.set("storage")} on:keydown={(e) => e.key === "Enter" && railView.set("storage")}>
      {@html icons.disk}
      <div class="grow"><div class="si-name">{$t("storage")}</div><div class="si-desc">{$t("storage_d")}</div></div>
    </div>

    <!-- Privasi -->
    <div class="settings-item" role="button" tabindex="0" on:click={() => railView.set("privacy")} on:keydown={(e) => e.key === "Enter" && railView.set("privacy")}>
      <svg viewBox="0 0 24 24"><rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/></svg>
      <div class="grow"><div class="si-name">{$t("privacy")}</div><div class="si-desc">{$t("privacy_d")}</div></div>
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

    <!-- DND / jam tenang -->
    <div class="settings-item" style="align-items:flex-start">
      <svg viewBox="0 0 24 24"><path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/></svg>
      <div class="grow">
        <div class="si-name">{$t("dnd")}</div>
        <div class="si-desc">{$t("dnd_d")}</div>
        {#if $dnd.on}
          <div style="display:flex;gap:8px;align-items:center;margin-top:8px;color:var(--text2);font-size:13px">
            <input type="time" bind:value={$dnd.from} style="background:var(--bg2);border:1px solid var(--line);border-radius:8px;padding:4px 8px;color:var(--text)" />
            –
            <input type="time" bind:value={$dnd.to} style="background:var(--bg2);border:1px solid var(--line);border-radius:8px;padding:4px 8px;color:var(--text)" />
          </div>
        {/if}
      </div>
      <span class="switch {$dnd.on ? '' : 'off'}" role="button" tabindex="0" on:click={() => ($dnd.on = !$dnd.on)} on:keydown={(e) => e.key === "Enter" && ($dnd.on = !$dnd.on)}></span>
    </div>

    <!-- Reaksi cepat kustom -->
    <div class="settings-item" style="align-items:flex-start">
      <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M8 14s1.5 2 4 2 4-2 4-2M9 9h.01M15 9h.01"/></svg>
      <div class="grow">
        <div class="si-name">{$t("quick_react")}</div>
        <div class="theme-modes" style="gap:6px">
          {#each $quickReactions as e, i}
            <button class="qre-sw" on:click={() => editQR(i)} title={$t("quick_react_edit")}>{e}</button>
          {/each}
        </div>
      </div>
    </div>

    <!-- Ukuran tampilan (zoom UI) -->
    <div class="settings-item" style="align-items:flex-start">
      <svg viewBox="0 0 24 24"><path d="M7 8V5h3M17 8V5h-3M7 16v3h3M17 16v3h-3"/></svg>
      <div class="grow">
        <div class="si-name">{$t("ui_scale")} · {$uiScale}%</div>
        <input type="range" min="80" max="140" step="5" bind:value={$uiScale} style="width:100%;margin-top:8px;accent-color:var(--accent)" />
      </div>
    </div>

    <!-- Jalan di latar belakang saat ditutup -->
    <div class="settings-item" role="button" tabindex="0" on:click={toggleBg} on:keydown={(e) => e.key === "Enter" && toggleBg()}>
      <svg viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="14" rx="2"/><path d="M8 21h8M12 18v3"/></svg>
      <div class="grow"><div class="si-name">{$t("bg_run")}</div><div class="si-desc">{$t("bg_run_d")}</div></div>
      <span class="switch {bgClose ? '' : 'off'}"></span>
    </div>
    {#if bgClose}
      <div class="settings-item danger" role="button" tabindex="0" on:click={quitApp} on:keydown={(e) => e.key === "Enter" && quitApp()}>
        <svg viewBox="0 0 24 24"><path d="M18.36 6.64a9 9 0 1 1-12.73 0M12 2v10"/></svg>
        <div class="grow"><div class="si-name">{$t("quit_app")}</div></div>
      </div>
    {/if}

    <!-- Keluar / ganti akun -->
    <div class="settings-item danger" role="button" tabindex="0" on:click={logout} on:keydown={(e) => e.key === "Enter" && logout()}>
      {@html icons.logout}
      <div class="grow"><div class="si-name">{$t("logout")}</div></div>
    </div>

    <div class="settings-foot">WhatsLite {appVer}</div>
  </div>
</div>

<style>
  .settings-foot { text-align: center; padding: 18px 0 8px; font-size: 12px; opacity: 0.45; }
</style>
