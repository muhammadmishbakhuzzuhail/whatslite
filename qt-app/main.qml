// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// Scaffold UI WhatsLite versi QML — replikasi layout & warna Svelte:
// Rail(64) | Sidebar chat(400) | Conversation(timeline + composer).
// Token warna disalin dari frontend/src/styles/app.css (tema light).
// Data dari engine via AppController (chatsModel/msgsModel/app).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtQuick.Dialogs
import Qt5Compat.GraphicalEffects
import Qt.labs.settings

ApplicationWindow {
    id: win
    width: 1100; height: 740; visible: true
    title: "WhatsLite (QML)"

    property var ctxMsg: ({})        // pesan target context-menu
    property real reactX: -1         // koord (window) tombol/bubble asal → posisi reactionPopup
    property real reactY: -1
    property var selectedChat: ({})  // chat aktif (utk header)
    property string activeView: "chats" // chats | calls | starred
    property string myName: (app.profile && app.profile.name) ? app.profile.name : "Saya"   // GetProfile (AppController.profile)
    property string myAbout: (app.profile && app.profile.about) ? app.profile.about : ""    // status/about profil sendiri

    // Wallpaper per-chat (Svelte setWallpaper/localStorage) — persist via Qt.labs.settings.
    property var wallpapers: ({})
    Settings { id: wpStore; category: "wallpapers"; property string data: "{}" }
    Component.onCompleted: { try { wallpapers = JSON.parse(wpStore.data) || {} } catch (e) { wallpapers = {} } }
    function setWallpaper(cid, c) {
        if (!cid) return
        var m = Object.assign({}, wallpapers)
        if (c) m[cid] = c; else delete m[cid]
        wallpapers = m; wpStore.data = JSON.stringify(m)
    }
    function chatWallpaper(cid) { return (cid && wallpapers[cid]) ? wallpapers[cid] : "" }
    property string chatFilter: "Semua" // filter chip aktif
    property string channelTab: "following" // ChannelsPane .ch-tabs: following | discover
    property var commOpen: ({})       // CommunitiesPane: jid/nama komunitas → diperluas (chevron)
    function commToggle(key) {         // toggle perluasan satu komunitas (delegate daur-ulang → simpan di win)
        var o = win.commOpen; o[key] = !o[key]; win.commOpen = o
        commOpenChanged()
    }
    property string lightboxSrc: ""  // media fullscreen (kosong = tutup)
    property string lightboxCaption: "" // keterangan media di lightbox (Lightbox.svelte .lb-cap)
    // Album lightbox (Lightbox.svelte items[]): daftar URL + indeks aktif → prev/next.
    // Kosong → mode gambar tunggal (pakai lightboxSrc). NOTE: pemicu album dari bubble
    // belum disambungkan (engine Qt belum kirim items[]); UI nav siap & ter-guard >1.
    property var lightboxItems: []
    property int lightboxIndex: 0
    // Draf pratinjau media sebelum kirim (MediaPreviewModal.svelte). Engine Qt tak
    // punya store mediaDraft → diisi lokal saat pilih gambar; null = tutup.
    // Bentuk: { chatId, items:[{kind, url, name}] }. items kosong = popup tutup.
    property var mediaDraft: null
    property int mediaDraftIdx: 0
    property bool mediaDraftOnce: false
    property bool locked: (typeof startLock !== "undefined") && startLock // app-lock PIN
    property var replyTo: null        // pesan yang sedang dibalas (banner composer)
    property var ctxChat: ({})        // chat target context-menu baris
    // --- Folder chat kustom (client-side, sama desain dgn FolderPicker.svelte /
    // stores.js: folders adalah daftar lokal {name, jids[]}; engine Qt tak punya
    // store folder, jadi disimpan in-memory di sini). folderPickFor = jid yg sedang
    // diatur foldernya (null = modal tutup). folderPickName = nama chat utk judul.
    property var folders: []          // [{name, jids:[]}]
    property string folderPickFor: "" // jid chat dlm picker ("" = tutup)
    property string folderPickName: "" // nama chat (judul modal)
    function openFolderPicker(jid, name) {
        if (!jid) return
        folderPickName = name || ""
        folderPickFor = jid
        folderPickerPopup.open()
    }
    function addFolder(name) {
        if (!name) return
        for (var i = 0; i < folders.length; i++) if (folders[i].name === name) return
        var f = folders.slice(); f.push({ name: name, jids: [] }); folders = f
    }
    function deleteFolder(name) {
        folders = folders.filter(function(x){ return x.name !== name })
    }
    function chatInFolder(name, jid) {
        for (var i = 0; i < folders.length; i++)
            if (folders[i].name === name) return folders[i].jids.indexOf(jid) >= 0
        return false
    }
    function toggleChatFolder(name, jid) {
        var f = folders.map(function(x){
            if (x.name !== name) return x
            var has = x.jids.indexOf(jid) >= 0
            return { name: x.name, jids: has ? x.jids.filter(function(j){ return j !== jid })
                                             : x.jids.concat([jid]) }
        })
        folders = f
    }
    // Ikon SVG disalin dari komponen Svelte (Rail/SearchBar/Composer) — faithful.
    readonly property var ico: ({
        "chats": '<path d="M12 3C6.5 3 2 6.8 2 11.5c0 2.3 1.1 4.4 2.9 5.9-.1 1.2-.6 2.6-1.4 3.6 1.6-.2 3.2-.8 4.4-1.6 1.2.4 2.6.6 4.1.6 5.5 0 10-3.8 10-8.5S17.5 3 12 3z"/>',
        "calls": '<path d="M5 4h3l2 5-2.5 1.5a11 11 0 0 0 5 5L15 13l5 2v3a2 2 0 0 1-2 2A16 16 0 0 1 3 6a2 2 0 0 1 2-2z"/>',
        "status": '<circle cx="12" cy="12" r="9" stroke-dasharray="3 3"/>',
        "channels": '<path d="M4 9v6h4l5 4V5L8 9H4z"/><path d="M16 8a5 5 0 0 1 0 8"/>',
        "communities": '<circle cx="8" cy="9" r="3"/><circle cx="16" cy="9" r="2.2"/><path d="M3 19c0-2.5 2.2-4.5 5-4.5s5 2 5 4.5"/><path d="M14 19c0-1.8.9-3.3 2.3-3.9"/>',
        "contacts": '<circle cx="12" cy="8" r="4"/><path d="M4 20c0-4 3.6-6.5 8-6.5s8 2.5 8 6.5"/>',
        "starred": '<path d="M12 3l2.6 5.6 6 .7-4.4 4.1 1.2 6L12 16.6 6.6 19.4l1.2-6L3.4 9.3l6-.7z"/>',
        "archived": '<rect x="3" y="6" width="18" height="4" rx="1"/><path d="M5 10h14v9H5zM10 14h4"/>',
        "scheduled": '<circle cx="12" cy="13" r="7"/><path d="M12 9v4l3 2M9 3h6"/>',
        // Composer schedule button (Composer.svelte) — alarm-clock with top stem.
        "schedule": '<circle cx="12" cy="13" r="8"/><path d="M12 9v4l3 2M9 2h6"/>',
        "settings": '<circle cx="12" cy="12" r="3"/><path d="M12 2v3M12 19v3M2 12h3M19 12h3M5 5l2 2M17 17l2 2M19 5l-2 2M7 17l-2 2"/>',
        "search": '<circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/>',
        "plus": '<path d="M12 5v14M5 12h14"/>',
        "send": '<path d="M3 11l18-8-8 18-2-7-8-3z"/>',
        "emoji": '<circle cx="12" cy="12" r="9"/><circle cx="9" cy="10" r="1"/><circle cx="15" cy="10" r="1"/><path d="M8.5 14.5a4 4 0 0 0 7 0"/>',
        "mic": '<rect x="9" y="3" width="6" height="11" rx="3"/><path d="M5 11a7 7 0 0 0 14 0M12 18v3"/>',
        // Stop (kotak) — tombol saat sedang merekam suara (Composer.svelte recording icon).
        "stop": '<rect x="6" y="6" width="12" height="12" rx="2"/>',
        "pollq": '<path d="M5 5h14M5 12h9M5 19h5"/>',
        "play": '<path d="M8 5v14l11-7z"/>',
        "locpin": '<path d="M12 21s7-6 7-11a7 7 0 0 0-14 0c0 5 7 11 7 11z"/><circle cx="12" cy="10" r="2.5"/>',
        "download": '<path d="M12 4v11M7 11l5 5 5-5M5 20h14"/>',
        "close": '<path d="M6 6l12 12M18 6L6 18"/>',
        // Panah kembali (pane-head .icon-btn pada PrivacyPane/StoragePane.svelte).
        "chevleft": '<path d="M15 5l-7 7 7 7"/>',
        "logout": '<path d="M16 17l5-5-5-5M21 12H9M9 4H6a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h3"/>',
        "sticker": '<path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h8l6-6V5a2 2 0 0 0-2-2z"/><path d="M14 21v-4a2 2 0 0 1 2-2h4"/>',
        "gifb": '<rect x="3" y="5" width="18" height="14" rx="2"/><path d="M8 9v6M11 9v6h2M16 9h-2v6M16 12h-1"/>',
        "document": '<path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><path d="M14 3v6h6"/>',
        // Quick replies (Composer.svelte attach-menu zap/lightning icon) — disalin persis.
        "zap": '<path d="M13 2L3 14h7l-1 8 10-12h-7z"/>',
        // .doc-ico svg (Bubble.svelte cabang document) — disalin persis.
        "docfile": '<path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5"/>',
        "overflow": '<circle cx="12" cy="5" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="12" cy="19" r="1.6"/>',
        "newchat": '<path d="M12 5H7a3 3 0 0 0-3 3v9a3 3 0 0 0 3 3h9a3 3 0 0 0 3-3v-5"/><path d="M18.5 3.5a2.1 2.1 0 0 1 3 3L13 15l-4 1 1-4 8.5-8.5z"/>',
        "pin": '<path d="M12 17v5M7 4h10l-1 6 3 3H5l3-3-1-6z"/>',
        "mute": '<path d="M5 9v6h3l4 4V5L8 9H5z"/><path d="M16 8a5 5 0 0 1 0 8"/><path d="M3 3l18 18"/>',
        // Lonceng (ChannelsPane .ch-act 🔔/🔕): bel = aktif, mute = senyap.
        "bell": '<path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.7 21a2 2 0 0 1-3.4 0"/>',
        "check": '<path d="M3 7.5l3.5 3.5L14 4"/>',
        "checks": '<path d="M1 7.5l3.2 3.2L10 4"/><path d="M7 10.7L12.8 4"/>',
        // Panah arah panggilan (CallsPane.svelte .call-ico). missed/in = arah masuk.
        "callArrowOut": '<path d="M7 17L17 7M17 7H9M17 7v8"/>',
        "callArrowIn": '<path d="M17 7L7 17M7 17h8M7 17V9"/>',
        // Centang badge terverifikasi (ChannelsPane.svelte .ch-verif → ✓ putih solid).
        "verif": '<path d="M5 12l4 4 10-10"/>',
        // Tombol info kontak (ContactsPane.svelte .ct-info → lingkaran-i).
        "ctInfo": '<circle cx="12" cy="12" r="9"/><path d="M12 11v5"/><circle cx="12" cy="7.5" r="0.6"/>',
        // --- Ikon Setelan (disalin dari SettingsPane.svelte icons/inline svg) ---
        "theme": '<path d="M21 13A9 9 0 1 1 11 3a7 7 0 0 0 10 10z"/>',
        "globe": '<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.5 2.5 2.5 15 0 18M12 3C9.5 5.5 9.5 18.5 12 21"/>',
        "globe2": '<path d="M4 12h16M12 4a15 15 0 0 1 0 16M12 4a15 15 0 0 0 0 16"/><circle cx="12" cy="12" r="9"/>',
        "disk": '<rect x="4" y="4" width="16" height="16" rx="2"/><path d="M8 4v6h8V4M8 16h.01"/>',
        "lock": '<rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/>',
        "trash": '<path d="M3 6h18"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><path d="M6 6l1 14a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-14"/>',
        // rec-cancel (Composer.svelte) trash — simpler path, distinct from settings trash.
        "trashSimple": '<path d="M4 7h16M9 7V5h6v2M6 7l1 13h10l1-13"/>',
        "clock": '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 3"/>',
        // Ticks.svelte 'sending' clock (viewBox 0 0 16 16).
        "clockSm": '<circle cx="8" cy="8" r="6"/><path d="M8 4.7V8l2.2 1.3"/>',
        // Chevron bawah (Combo indicator / chat-row menu .row-menu-btn / shared-media .chev).
        "chevdown": '<path d="M6 9l6 6 6-6"/>',
        "star2": '<path d="M12 3l2.6 5.5 6 .8-4.4 4.2 1.1 6L12 16.8 6.7 19.5l1.1-6L3.4 9.3l6-.8z"/>',
        "window": '<rect x="3" y="4" width="18" height="14" rx="2"/><path d="M8 21h8M12 18v3"/>',
        // --- Ikon Setelan tambahan (SettingsPane.svelte inline svg) ---
        "translate": '<path d="M4 5h7M9 3v2c0 4-2 7-5 9M5 9c0 3 3 5 6 5"/><path d="M14 19l3-7 3 7M15.5 16h3"/>',
        "speaker": '<path d="M11 5L6 9H2v6h4l5 4zM15 9a3 3 0 0 1 0 6M18 6a7 7 0 0 1 0 12"/>',
        "hamburger": '<path d="M4 6h16M4 12h16M4 18h16"/>',
        "eyeoff": '<circle cx="12" cy="12" r="9"/><path d="M5.6 5.6l12.8 12.8"/>',
        "moon": '<path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/>',
        "emojiface": '<circle cx="12" cy="12" r="9"/><path d="M8 14s1.5 2 4 2 4-2 4-2M9 9h.01M15 9h.01"/>',
        "zoom": '<path d="M7 8V5h3M17 8V5h-3M7 16v3h3M17 16v3h-3"/>',
        "power": '<path d="M18.36 6.64a9 9 0 1 1-12.73 0M12 2v10"/>',
        // --- Ikon panel info/profil (disalin dari InfoPanel.svelte & ContactProfile.svelte) ---
        "editpen": '<path d="M4 20h4L18 10l-4-4L4 16z"/><path d="M14 6l4 4"/>',
        "addmember": '<circle cx="9" cy="8" r="4"/><path d="M2 20c0-3.5 3-6 7-6M18 11v6M15 14h6"/>',
        // Buat saluran (ChannelsPane.svelte .icon-btn ch_create) — orang + plus.
        "ch_create": '<path d="M19 9v6M16 12h6"/><circle cx="9" cy="8" r="4"/><path d="M2 20a7 7 0 0 1 12-5"/>',
        "invitelink": '<path d="M9 15l6-6M8 13l-2 2a3 3 0 0 0 4 4l2-2M16 11l2-2a3 3 0 0 0-4-4l-2 2"/>',
        "resetlink": '<path d="M4 12a8 8 0 0 1 14-5l2 2M20 12a8 8 0 0 1-14 5l-2-2M18 4v5h-5M6 20v-5h5"/>',
        "wallpaperico": '<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 15l5-4 4 3 5-5 4 4"/>',
        "clearchat": '<path d="M10 3h4l1 4h5v3H4V7h5z"/><path d="M6 10v9a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2v-9"/>',
        "block": '<circle cx="12" cy="12" r="9"/><path d="M5.5 5.5l13 13"/>',
        "message": '<path d="M4 5h16v11H8l-4 4z"/>',
        "removelabel": '<path d="M4 7h16M9 7V5h6v2M6 7l1 13h10l1-13"/>',
        "commongroup": '<circle cx="9" cy="9" r="3"/><path d="M2 20c0-3 3-5 7-5M16 8a3 3 0 0 1 0 6M15 20c0-2 2-4 5-4"/>',
        "herophoto": '<path d="M4 7h3l2-2h6l2 2h3v12H4z"/><circle cx="12" cy="13" r="3.5"/>',
        "leavegroup": '<path d="M15 4h3a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-3"/><path d="M10 17l-5-5 5-5M5 12h11"/>',
        // --- Ikon pratinjau media (MediaPreviewModal.svelte inline svg) ---
        "rotate": '<path d="M3 12a9 9 0 1 0 3-6.7L3 8"/><path d="M3 3v5h5"/>',
        "crop": '<path d="M6 2v14a2 2 0 0 0 2 2h14"/><path d="M2 6h14a2 2 0 0 1 2 2v14"/>',
        // Sekali-lihat (view-once): lingkaran dgn "1" — Composer/MediaPreview.
        "viewonce": '<circle cx="12" cy="12" r="9"/>'
    })
    // Palet warna avatar per-kontak (dari mock.js Svelte) + hash nama → stabil.
    readonly property var avPalette: ["#6a9e3d", "#c95a8b", "#e0794f", "#b86ac9", "#3d8bd3", "#2aa89e", "#5a6ac9", "#d8902a"]
    function avatarColor(s) {
        s = s || "?"
        var h = 0
        for (var i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0
        return avPalette[h % avPalette.length]
    }
    // Hitung chat ber-field true (chip-n WhatsApp: Unread/Groups bawa jumlah).
    // dep = chatList.count → bindings recompute saat daftar berubah.
    function chatCount(field, dep) {
        var n = 0
        for (var i = 0; i < dep; i++) {
            var m = chatsModel.get(i)
            if (m && m[field] === true) n++
        }
        return n
    }
    // Filter jenis pencarian aktif (.sc-types: all|image|video|document|link|voice).
    // NOTE: backend Qt SearchMessages hanya menerima query (1 arg) → filter jenis
    // belum tersambung ke engine; chip hanya state UI (parity visual).
    property string searchType: "all"
    // Chat yang NAMA-nya cocok query (.list-label "Chats" di ChatList.svelte search).
    // dep = chatList.count agar binding recompute saat model berubah.
    function nameMatches(q, dep) {
        var out = []
        var s = (q || "").trim().toLowerCase()
        if (s.length === 0) return out
        for (var i = 0; i < dep; i++) {
            var m = chatsModel.get(i)
            if (m && (m.name || m.id || "").toLowerCase().indexOf(s) !== -1) out.push(m)
        }
        return out
    }
    // Huruf-awal utk pemisah abjad ContactsPane/StatusPane (.ct-letter): A–Z,
    // selain itu "#". Dipakai delegate utk menyisipkan header huruf inline
    // (model diasumsikan sudah terurut abjad dari engine — lihat LIMITATIONS).
    function letterOf(name) {
        var L = ((name || "#").trim().charAt(0) || "#").toUpperCase()
        return /[A-Z]/.test(L) ? L : "#"
    }
    // True bila baris index i memulai grup huruf baru (i===0 atau huruf berbeda
    // dari baris sebelumnya pada model m → render .ct-letter di atas baris).
    function startsLetterGroup(mdl, i, name) {
        if (i <= 0) return true
        var prev = mdl.get(i - 1)
        return letterOf(prev ? prev.name : "") !== letterOf(name)
    }

    // Tinggi bar waveform voice (app.css: nth 3n→40%, odd→60%, even→95%). i 0-based.
    function barH(i) {
        var c = i + 1
        if (c % 3 === 0) return 0.40
        return (c % 2 === 1) ? 0.60 : 0.95
    }

    // --- Token tema (light + dark) — cocok dgn app.css [data-theme] ---
    QtObject {
        id: theme
        property bool dark: (typeof startDark !== "undefined") ? startDark : false
        // Token DISALIN PERSIS dari frontend/src/styles/app.css (desain app
        // existing) — sumber valid in-repo, bukan tebakan eksternal.
        readonly property color railBg: dark ? "#11161d" : "#f4f6fa"
        readonly property color railIco: dark ? "#8a97a3" : "#6b7785"
        // Override aksen kustom (Setelan → .acc-sw). "" = pakai default tema.
        property string accentOverride: ""
        readonly property color accent: accentOverride !== "" ? accentOverride : (dark ? "#06c98c" : "#06b67f")
        readonly property color accentDeep: accentOverride !== "" ? Qt.darker(accentOverride, 1.18) : (dark ? "#06b67f" : "#048a60")
        readonly property color sidebarBg: dark ? "#0e1318" : "#ffffff"
        readonly property color bg: dark ? "#1a232a" : "#ffffff"
        readonly property color bg2: dark ? "#222e35" : "#f0f2f5"
        readonly property color headBg: dark ? "#11171e" : "#ffffff"
        readonly property color line: dark ? "#2a3942" : "#e4e8ee"
        readonly property color divider: dark ? "#1c252d" : "#eceff3"
        readonly property color searchBg: dark ? "#1b232b" : "#eef1f6"
        readonly property color wallpaper: dark ? "#0a0f14" : "#eef1f6"
        readonly property color inBg: dark ? "#1d262e" : "#ffffff"
        readonly property color outBg: dark ? "#114b39" : "#d6f3c4"
        readonly property color text: dark ? "#e7ecf0" : "#0f1722"
        readonly property color text2: dark ? "#8a97a3" : "#6b7785"
        readonly property color hover: dark ? "#161d24" : "#f2f4f8"
        readonly property color tick: "#2eaadc"
        readonly property color selected: dark ? "#12302a" : "#e7f6ef"
        // Kutipan balasan (app.css --quote-bar/--quote-bg).
        readonly property color quoteBar: dark ? "#06c98c" : "#06b67f"
        readonly property color quoteBg: dark ? Qt.rgba(6/255, 201/255, 140/255, 0.12) : Qt.rgba(6/255, 182/255, 127/255, 0.09)
        // Radii (app.css): r-sm 10, r 14, r-lg 18, pill 999.
        readonly property real rSm: 10
        readonly property real r: 14
        readonly property real rLg: 18
    }

    // --- Kontrol bertema (app.css). QtQuick Controls Basic = chrome putih → tak
    // cocok tema gelap. Inline component override agar baca token theme. ---
    // .btn-ghost (bg2/text) & .btn-accent (accent/#fff): radius 10, pad 9/16, w600.
    component Btn: Button {
        id: _btn
        property bool accent: false
        property bool danger: false
        leftPadding: 16; rightPadding: 16; topPadding: 9; bottomPadding: 9
        font.pixelSize: 14; font.weight: Font.DemiBold
        background: Rectangle {
            radius: 10
            color: _btn.accent ? (_btn.down || _btn.hovered ? theme.accentDeep : theme.accent)
                               : (_btn.hovered ? theme.hover : theme.bg2)
            opacity: _btn.enabled ? 1 : 0.5
        }
        contentItem: Text {
            text: _btn.text; font: _btn.font
            color: _btn.accent ? "#ffffff" : (_btn.danger ? "#e35d6a" : theme.text)
            opacity: _btn.enabled ? 1 : 0.5
            horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
        }
    }
    // .switch: 38x22 radius12; track accent(on)/text2(off); knob 18x18 #fff inset 2.
    component Tog: Switch {
        id: _sw
        implicitWidth: 38; implicitHeight: 22
        indicator: Rectangle {
            implicitWidth: 38; implicitHeight: 22; radius: 12
            color: _sw.checked ? theme.accent : theme.text2
            Behavior on color { ColorAnimation { duration: 120 } }
            Rectangle {
                width: 18; height: 18; radius: 9; color: "#ffffff"; y: 2
                x: _sw.checked ? parent.width - width - 2 : 2
                Behavior on x { NumberAnimation { duration: 120 } }
            }
        }
        contentItem: Item {}
    }
    // Combo bertema: field bg2, popup gelap (app.css surface input).
    component Combo: ComboBox {
        id: _cb
        font.pixelSize: 13
        implicitHeight: 34
        // .lang-select: bg var(--search-bg), border var(--divider) radius 9,
        // hover border-color var(--accent) (transition .12s).
        background: Rectangle {
            radius: 9; color: theme.searchBg
            border.color: _cb.hovered ? theme.accent : theme.divider; border.width: 1
            Behavior on border.color { ColorAnimation { duration: 120 } }
        }
        contentItem: Text {
            leftPadding: 11; rightPadding: 30; text: _cb.displayText; font: _cb.font
            color: theme.text; verticalAlignment: Text.AlignVCenter; elide: Text.ElideRight
        }
        indicator: Icon {
            x: _cb.width - 24; y: (_cb.height - height) / 2; width: 14; height: 14
            svg: win.ico["chevdown"]; color: theme.text2
            rotation: _cb.popup.visible ? 180 : 0
            Behavior on rotation { NumberAnimation { duration: 120 } }
        }
        popup: Popup {
            y: _cb.height + 2; width: _cb.width; padding: 1
            background: Rectangle { color: theme.searchBg; radius: 8; border.color: theme.divider }
            contentItem: ListView {
                clip: true; implicitHeight: contentHeight; model: _cb.popup.visible ? _cb.delegateModel : null
                currentIndex: _cb.highlightedIndex
                ScrollIndicator.vertical: ScrollIndicator {}
            }
        }
        delegate: ItemDelegate {
            id: _del
            width: _cb.width
            // .lp-opt.on: bg color-mix(accent 14%) + .lp-check accent ✓
            property bool _on: _cb.currentIndex === index
            contentItem: RowLayout {
                spacing: 8
                Text { Layout.fillWidth: true
                    text: _cb.textRole ? (Array.isArray(_cb.model) ? modelData[_cb.textRole] : model[_cb.textRole]) : modelData
                    color: theme.text; font.pixelSize: 13; verticalAlignment: Text.AlignVCenter; elide: Text.ElideRight }
                Text { visible: _del._on; text: "✓"; color: theme.accent; font.pixelSize: 13; font.bold: true }
            }
            background: Rectangle {
                color: _del._on ? Qt.rgba(theme.accent.r, theme.accent.g, theme.accent.b, 0.14)
                               : (_del.hovered ? theme.hover : "transparent")
            }
        }
    }

    // .attach-menu: panel gelap (bg/line, radius 14, shadow-lg, pad 6, min-w 200).
    // Basic Menu = chrome putih → teks tak terbaca di tema gelap. Override panel.
    component TMenu: Menu {
        id: _menu
        padding: 6
        implicitWidth: Math.max(200, contentWidth + leftPadding + rightPadding)
        background: Rectangle {
            implicitWidth: 200
            radius: 14
            color: theme.bg
            border.width: 1; border.color: theme.line
            layer.enabled: true
            layer.effect: DropShadow {
                horizontalOffset: 0; verticalOffset: 8; radius: 30; samples: 41
                color: Qt.rgba(0, 0, 0, theme.dark ? 0.45 : 0.18)
            }
        }
    }
    // .attach-item: hover bg hover, teks text 15px, radius 8, pad ~9 12.
    // `iconName` (opsional) → .am-item: leading line-SVG 20px accent + gap 12 + teks 14.5px (int 14).
    component TMenuItem: MenuItem {
        id: _mi
        property string iconName: ""
        implicitHeight: iconName !== "" ? 42 : 38
        leftPadding: iconName !== "" ? 14 : 12; rightPadding: 14; topPadding: iconName !== "" ? 11 : 9; bottomPadding: iconName !== "" ? 11 : 9
        spacing: 12
        contentItem: RowLayout {
            spacing: 12
            // .am-item svg { width:20; height:20; color: var(--accent) }
            Icon {
                visible: _mi.iconName !== ""
                Layout.preferredWidth: 20; Layout.preferredHeight: 20
                svg: _mi.iconName !== "" ? (win.ico[_mi.iconName] || "") : ""
                color: theme.accent
            }
            Text {
                Layout.fillWidth: true
                text: _mi.text; color: theme.text
                font.pixelSize: _mi.iconName !== "" ? 14 : 15   // .am-item 14.5px → int 14
                verticalAlignment: Text.AlignVCenter; elide: Text.ElideRight
                opacity: _mi.enabled ? 1 : 0.5
            }
        }
        background: Rectangle {
            radius: 8
            color: (_mi.hovered || _mi.highlighted) ? theme.hover : "transparent"
        }
    }

    // .settings-item: flex, align center, gap 20, padding 14 20, border-bottom 1px
    // divider, hover bg hover; leading svg 24x24 text2; .si-name 16; .si-desc 13
    // text2 mt 2; .grow flex:1. danger → #e35d6a. `extra` = blok di bawah teks
    // (theme-modes / input); `trailing` = kontrol kanan (Tog / Combo).
    component SettingsItem: Rectangle {
        id: _si
        property string icon: ""
        property string name: ""
        property string desc: ""
        property bool danger: false
        property bool clickable: true     // false utk lang-item & baris extra/toggle
        property bool topAlign: false     // align-items:flex-start utk baris ber-extra
        property Item extra: null         // slot blok di bawah (theme-modes/input)
        property Item trailing: null      // slot kontrol kanan (Tog/Combo)
        signal activated()
        readonly property color icoColor: danger ? "#e35d6a" : theme.text2
        readonly property color nameColor: danger ? "#e35d6a" : theme.text

        Layout.fillWidth: true
        implicitHeight: _row.implicitHeight + 28 // padding 14 atas + 14 bawah
        color: (_si.clickable && _siHov.hovered) ? theme.hover : "transparent"
        Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: theme.divider }
        HoverHandler { id: _siHov; enabled: _si.clickable }
        TapHandler { enabled: _si.clickable; onTapped: _si.activated() }

        // Re-parent slot extra → kolom .grow; trailing → ujung RowLayout. Slot
        // di-instantiate di scope pemanggil; di sini cukup di-pasang ke layout.
        Component.onCompleted: {
            if (_si.extra) { _si.extra.parent = _grow; }
            if (_si.trailing) { _si.trailing.parent = _row; }
        }

        RowLayout {
            id: _row
            anchors.left: parent.left; anchors.right: parent.right
            anchors.leftMargin: 20; anchors.rightMargin: 20
            anchors.verticalCenter: _si.topAlign ? undefined : parent.verticalCenter
            anchors.top: _si.topAlign ? parent.top : undefined
            anchors.topMargin: _si.topAlign ? 14 : 0
            spacing: 20
            Icon {
                Layout.alignment: _si.topAlign ? Qt.AlignTop : Qt.AlignVCenter
                Layout.preferredWidth: 24; Layout.preferredHeight: 24
                svg: win.ico[_si.icon] || ""; color: _si.icoColor
            }
            ColumnLayout {
                id: _grow
                Layout.fillWidth: true; spacing: 0
                Text { text: _si.name; color: _si.nameColor; font.pixelSize: 16 }
                Text {
                    visible: _si.desc !== ""; text: _si.desc
                    color: theme.text2; font.pixelSize: 13; Layout.topMargin: 2
                    Layout.fillWidth: true; wrapMode: Text.WordWrap
                }
            }
        }
    }

    // .theme-mode: flex:1, padding 7 4, radius 9, border 1px line, bg bg2;
    // .on → border accent + bg color-mix(accent 12%) + color text.
    component ThemeMode: Rectangle {
        id: _tm
        property string text: ""
        property bool on: false
        signal clicked()
        Layout.fillWidth: true
        implicitHeight: 30 // 7+16+7
        radius: 9
        color: _tm.on ? Qt.rgba(theme.accent.r, theme.accent.g, theme.accent.b, 0.12) : theme.bg2
        border.width: 1
        border.color: _tm.on ? theme.accent : theme.line
        Text {
            anchors.centerIn: parent
            text: _tm.text; font.pixelSize: 13
            color: _tm.on ? theme.text : theme.text2
        }
        TapHandler { onTapped: _tm.clicked() }
    }

    // Slider bertema (input[type=range] accent-color:var(--accent)). Basic Slider
    // = chrome terang → override track/handle ke token accent.
    component TSlider: Slider {
        id: _sl
        implicitHeight: 20
        background: Rectangle {
            x: _sl.leftPadding; y: _sl.topPadding + _sl.availableHeight / 2 - height / 2
            width: _sl.availableWidth; height: 4; radius: 2; color: theme.bg2
            Rectangle {
                width: _sl.visualPosition * parent.width; height: parent.height
                color: theme.accent; radius: 2
            }
        }
        handle: Rectangle {
            x: _sl.leftPadding + _sl.visualPosition * (_sl.availableWidth - width)
            y: _sl.topPadding + _sl.availableHeight / 2 - height / 2
            width: 16; height: 16; radius: 8
            color: theme.accent; border.color: "#ffffff"; border.width: 2
        }
    }

    // --- i18n: default English, dapat ganti runtime. Kamus JSON per bahasa di
    // i18n/<code>.json (en/id ditulis tangan; bahasa lain tinggal tambah file).
    // Kunci hilang → fallback en → fallback kunci. Pola sama dgn FE Svelte. ---
    QtObject {
        id: i18n
        property string lang: "en"
        property var en: ({})
        property var dict: ({})
        function t(k) { return dict[k] || en[k] || k }
        function setLang(code) {
            lang = code
            dict = (code === "en") ? en : app.readJson(srcDir + "/i18n/" + code + ".json")
        }
        Component.onCompleted: {
            en = app.readJson(srcDir + "/i18n/en.json"); dict = en
            if (typeof startLang !== "undefined" && startLang !== "" && startLang !== "en") setLang(startLang)
        }
    }

    // --- Helper format document (pakai metadata docSize/docMime/docPages) ---
    function fmtSize(b) {
        if (!b || b <= 0) return ""
        if (b < 1024) return b + " B"
        if (b < 1048576) return Math.round(b / 1024) + " KB"
        return (b / 1048576).toFixed(1) + " MB"
    }
    function extLabel(m) {
        m = m || ""
        if (m.indexOf("pdf") >= 0) return "PDF"
        if (m.indexOf("word") >= 0 || m.indexOf("msword") >= 0) return "DOC"
        if (m.indexOf("sheet") >= 0 || m.indexOf("excel") >= 0) return "XLS"
        if (m.indexOf("zip") >= 0) return "ZIP"
        return "File"
    }
    function fmtDoc(m) {
        var p = []
        var e = extLabel(m.docMime); if (e) p.push(e)
        var s = fmtSize(m.docSize); if (s) p.push(s)
        if (m.docPages > 0) p.push(m.docPages + " hal")
        return p.join(" · ")
    }

    // toast — pil bawah-tengah sementara (Toast.svelte). type: "" | "ok" | "error".
    property string toastText: ""
    property string toastType: ""
    function toast(msg, type) {
        toastText = msg || ""
        toastType = type || ""
        toastTimer.restart()
    }

    // prompt — dialog input teks reusable. cb(nilai) dipanggil saat simpan.
    function prompt(label, def, cb) {
        promptDialog.label = label
        promptInput.text = def || ""
        promptDialog.cb = cb
        promptDialog.open()
    }

    // loadView — muat pane read-only via loadInto generik (peta view→method+model).
    function loadView(v) {
        var methods = {
            calls: "GetCalls", starred: "GetStarred", status: "GetStatuses",
            contacts: "GetContacts", channels: "GetChannels", communities: "GetCommunities",
            archived: "GetArchivedChats", scheduled: "GetScheduled"
        }
        var models = {
            calls: callsModel, starred: starredModel, status: statusModel,
            contacts: contactsModel, channels: channelsModel, communities: communitiesModel,
            archived: archivedModel, scheduled: scheduledModel
        }
        if (methods[v])
            app.loadInto(methods[v], models[v])
    }

    RowLayout {
        anchors.fill: parent
        spacing: 0

        // ===== Rail (64px) =====
        Rectangle {
            Layout.preferredWidth: 64
            Layout.fillHeight: true
            color: theme.railBg
            ColumnLayout {
                anchors.fill: parent
                anchors.topMargin: 14
                spacing: 6
                Repeater {
                    model: [
                        { view: "chats" },
                        { view: "status" },
                        { view: "channels" },
                        { view: "communities" },
                        { view: "starred" },
                        { view: "calls" },
                        { view: "contacts" },
                        { view: "archived" },
                        { view: "scheduled" }
                    ]
                    // .rail-btn: 44×44; default lingkaran (radius 50%), aktif → rounded-rect
                    // radius 14 + bg rgba(0,168,132,.15) + ikon rail-active (app.css).
                    delegate: Rectangle {
                        id: railBtn
                        property bool active: activeView === modelData.view
                        Layout.alignment: Qt.AlignHCenter
                        width: 44; height: 44
                        radius: active ? 14 : 22
                        color: active ? Qt.rgba(0, 168/255, 132/255, 0.15)
                                      : (railMa.containsMouse ? (theme.dark ? Qt.rgba(1, 1, 1, 0.08) : Qt.rgba(0, 0, 0, 0.06))
                                                              : "transparent")
                        // .rail-btn transition: background .15s
                        Behavior on color { ColorAnimation { duration: 150 } }
                        Icon {
                            anchors.centerIn: parent; width: 24; height: 24
                            svg: win.ico[modelData.view] || ""
                            color: railBtn.active ? theme.accent : theme.railIco
                        }
                        // Badge belum-dibaca pada tombol Chats (.rail-badge app.css).
                        Rectangle {
                            visible: modelData.view === "chats" && win.chatCount("unread", chatList.count) > 0
                            anchors.top: parent.top; anchors.topMargin: 3
                            anchors.right: parent.right; anchors.rightMargin: 5
                            height: 16; width: Math.max(16, badgeTxt.implicitWidth + 8); radius: 8
                            color: theme.accent
                            Text {
                                id: badgeTxt; anchors.centerIn: parent
                                text: { var n = win.chatCount("unread", chatList.count); return n > 99 ? "99+" : n }
                                color: "#ffffff"; font.pixelSize: 10; font.bold: true
                            }
                        }
                        MouseArea {
                            id: railMa; anchors.fill: parent; hoverEnabled: true
                            onClicked: {
                                if (modelData.view === "settings") { settingsPopup.open(); return }
                                activeView = modelData.view
                                win.loadView(modelData.view)
                            }
                        }
                    }
                }
                Item { Layout.fillHeight: true }
                // Rail.svelte .rail-bottom: settings rail-btn DI BAWAH (dekat avatar) + .rail-avatar.
                Rectangle {
                    id: railSettings
                    Layout.alignment: Qt.AlignHCenter; Layout.bottomMargin: 8
                    width: 44; height: 44; radius: 22
                    color: setMa.containsMouse ? (theme.dark ? Qt.rgba(1, 1, 1, 0.08) : Qt.rgba(0, 0, 0, 0.06)) : "transparent"
                    Behavior on color { ColorAnimation { duration: 150 } }
                    Icon { anchors.centerIn: parent; width: 24; height: 24; svg: win.ico["settings"]; color: theme.railIco }
                    MouseArea { id: setMa; anchors.fill: parent; hoverEnabled: true; onClicked: settingsPopup.open() }
                }
                // Avatar profil di dasar rail (.rail-avatar 40px, inisial putih).
                Avatar {
                    Layout.alignment: Qt.AlignHCenter
                    Layout.bottomMargin: 14
                    implicitWidth: 40; implicitHeight: 40; fontSize: 16
                    name: win.myName; jid: "self"; base: app.mediaBase
                    accent: win.avatarColor(win.myName)
                    MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                        onClicked: { app.loadDetail("GetContactProfile", "self"); detailPopup.open() } }
                }
            }
        }

        // ===== Sidebar (daftar chat) =====
        Rectangle {
            Layout.preferredWidth: 400
            Layout.fillHeight: true
            color: theme.sidebarBg
            border.color: theme.line
            ColumnLayout {
                anchors.fill: parent
                spacing: 0
                // Header sidebar — KONDISIONAL per activeView (Svelte: setiap pane punya
                // header sendiri). Peta judul i18n (sesuai key Svelte tiap pane).
                property string paneTitle: ({
                    chats: i18n.t("rail_chats"), status: i18n.t("rail_status"),
                    channels: i18n.t("rail_channels"), communities: i18n.t("rail_communities"),
                    calls: i18n.t("rail_calls"), contacts: i18n.t("rail_contacts"),
                    starred: i18n.t("starred_msg"), archived: i18n.t("archived_chats"),
                    scheduled: i18n.t("scheduled_reminders")
                })[activeView] || i18n.t("rail_chats")

                // (1) chats + calls → .sidebar-head 60px (h1 23/700) + newchat + overflow.
                Rectangle {
                    visible: activeView === "chats" || activeView === "calls"
                    Layout.fillWidth: true; Layout.preferredHeight: 60
                    color: theme.headBg
                    RowLayout {
                        anchors.fill: parent; anchors.leftMargin: 18; anchors.rightMargin: 10; spacing: 4
                        Text {
                            Layout.fillWidth: true
                            text: paneTitle
                            font.pixelSize: 23; font.bold: true; color: theme.text
                        }
                        Rectangle {
                            Layout.preferredWidth: 40; Layout.preferredHeight: 40; radius: 20; color: chatNewMa.containsMouse ? theme.hover : "transparent"
                            Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["newchat"]; color: theme.railIco }
                            MouseArea { id: chatNewMa; anchors.fill: parent; hoverEnabled: true; onClicked: app.act("CreateGroup", []) }
                        }
                        Rectangle {
                            Layout.preferredWidth: 40; Layout.preferredHeight: 40; radius: 20; color: menuMa.containsMouse ? theme.hover : "transparent"
                            Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["overflow"]; color: theme.railIco }
                            MouseArea { id: menuMa; anchors.fill: parent; hoverEnabled: true; onClicked: settingsPopup.open() }
                        }
                    }
                }
                // (2) status/contacts/communities → .pane-head 56px (h2 19/600), tanpa aksi.
                Rectangle {
                    visible: activeView === "status" || activeView === "contacts" || activeView === "communities"
                    Layout.fillWidth: true; Layout.preferredHeight: 56
                    color: theme.headBg
                    Text {
                        anchors.verticalCenter: parent.verticalCenter
                        anchors.left: parent.left; anchors.leftMargin: 16
                        text: paneTitle
                        font.pixelSize: 19; font.weight: Font.DemiBold; color: theme.text
                    }
                }
                // (3) channels → .pane-head 56px (h2 19/600) + ch_create + ch_follow.
                Rectangle {
                    visible: activeView === "channels"
                    Layout.fillWidth: true; Layout.preferredHeight: 56
                    color: theme.headBg
                    RowLayout {
                        anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 8; spacing: 0
                        Text {
                            Layout.fillWidth: true
                            text: paneTitle
                            font.pixelSize: 19; font.weight: Font.DemiBold; color: theme.text
                        }
                        Rectangle {
                            Layout.preferredWidth: 40; Layout.preferredHeight: 40; radius: 20; color: chCreateMa.containsMouse ? theme.hover : "transparent"
                            Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["ch_create"]; color: theme.railIco }
                            MouseArea { id: chCreateMa; anchors.fill: parent; hoverEnabled: true; onClicked: app.act("CreateChannel", []) }
                        }
                        Rectangle {
                            Layout.preferredWidth: 40; Layout.preferredHeight: 40; radius: 20; color: chFollowMa.containsMouse ? theme.hover : "transparent"
                            Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["plus"]; color: theme.railIco }
                            MouseArea { id: chFollowMa; anchors.fill: parent; hoverEnabled: true; onClicked: { win.channelTab = "discover"; app.loadIntoA("GetRecommendedChannels", [""], channelsModel) } }
                        }
                    }
                }
                // (4) starred/archived/scheduled → .pane-head 56px: back icon-btn + h2 17px.
                //     gap 22 (starred/scheduled) / 14 (archived).
                Rectangle {
                    visible: activeView === "starred" || activeView === "archived" || activeView === "scheduled"
                    Layout.fillWidth: true; Layout.preferredHeight: 56
                    color: theme.headBg
                    RowLayout {
                        anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 16
                        spacing: (activeView === "archived") ? 14 : 22
                        Rectangle {
                            Layout.preferredWidth: 40; Layout.preferredHeight: 40; radius: 20; color: backMa.containsMouse ? theme.hover : "transparent"
                            Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["chevleft"]; color: theme.railIco }
                            MouseArea { id: backMa; anchors.fill: parent; hoverEnabled: true; onClicked: activeView = "chats" }
                        }
                        Text {
                            Layout.fillWidth: true
                            text: paneTitle
                            font.pixelSize: 17; font.weight: Font.DemiBold; color: theme.text
                        }
                    }
                }
                // Search (FTS pesan)
                Rectangle {
                    Layout.fillWidth: true; Layout.preferredHeight: 38  // = Svelte .search (pad 9 + icon 18)
                    // .search-wrap padding 8px 12px.
                    Layout.topMargin: 8; Layout.bottomMargin: 8; Layout.leftMargin: 12; Layout.rightMargin: 12
                    radius: 19; color: theme.searchBg
                    Icon {
                        anchors.verticalCenter: parent.verticalCenter
                        anchors.left: parent.left; anchors.leftMargin: 14
                        width: 18; height: 18; svg: win.ico["search"]; color: theme.text2
                    }
                    TextInput {
                        id: searchInput
                        anchors.fill: parent; anchors.leftMargin: 44; anchors.rightMargin: 14
                        verticalAlignment: TextInput.AlignVCenter
                        color: theme.text; font.pixelSize: 14; clip: true
                        onTextChanged: app.search(text, searchModel)
                    }
                    Text {
                        visible: searchInput.text === ""
                        anchors.verticalCenter: parent.verticalCenter
                        anchors.left: parent.left; anchors.leftMargin: 44
                        text: i18n.t("search_placeholder"); color: theme.text2; font.pixelSize: 14
                    }
                }
                // Filter chips (Semua / Belum dibaca N / Favorit / Grup N / +) — ala WhatsApp.
                Flow {
                    // .filters padding 4px 16px 10px.
                    Layout.fillWidth: true; Layout.leftMargin: 16; Layout.rightMargin: 16; Layout.topMargin: 4; Layout.bottomMargin: 10; spacing: 8
                    visible: activeView === "chats" && searchInput.text === ""
                    Repeater {
                        // f = field pencacah (chip-n WhatsApp: Unread/Groups bawa jumlah).
                        model: [{ label: i18n.t("filter_all"), v: "Semua", f: "" },
                                { label: i18n.t("filter_unread"), v: "Belum dibaca", f: "unread" },
                                { label: i18n.t("filter_favorites"), v: "Favorit", f: "" },
                                { label: i18n.t("filter_groups"), v: "Grup", f: "group" }]
                        delegate: Rectangle {
                            property bool on: win.chatFilter === modelData.v
                            property int n: modelData.f ? win.chatCount(modelData.f, chatList.count) : 0
                            radius: 13; height: 26; implicitWidth: crow.implicitWidth + 26  // = Svelte .chip (pad 5/13)
                            // app.css: aktif = bg accent + teks putih (solid), bukan outline.
                            color: on ? theme.accent : theme.searchBg
                            Row {
                                id: crow; anchors.centerIn: parent; spacing: 5
                                Text { text: modelData.label; font.pixelSize: 13; color: on ? "#ffffff" : theme.text2 }
                                Text { visible: n > 0; text: n; font.pixelSize: 13; font.weight: Font.DemiBold
                                    opacity: 0.7; color: on ? "#ffffff" : theme.text2 }
                            }
                            MouseArea { anchors.fill: parent; onClicked: win.chatFilter = modelData.v }
                        }
                    }
                    // Chip "+" (buat folder) — app.css .chip.plus.
                    Rectangle {
                        radius: 13; height: 26; width: 34; color: theme.searchBg
                        Text { anchors.centerIn: parent; text: "+"; font.pixelSize: 17; color: theme.text2 }
                        MouseArea { anchors.fill: parent
                            onClicked: win.prompt(i18n.t("folder_new"), "", function(v){ if (v) { app.act("AddFolder", [v]); win.addFolder(v) } }) }
                    }
                }
                // .ch-tabs (ChannelsPane.svelte) — Diikuti / Jelajahi: dua tombol flex,
                // aktif = accent + #fff, lainnya bg2/text2 13/600. padding 2/12 10.
                RowLayout {
                    Layout.fillWidth: true; Layout.leftMargin: 12; Layout.rightMargin: 12
                    Layout.topMargin: 2; Layout.bottomMargin: 10; spacing: 6
                    visible: activeView === "channels" && searchInput.text === ""
                    Repeater {
                        model: [{ k: "following", label: i18n.t("ch_following") },
                                { k: "discover", label: i18n.t("ch_discover") }]
                        delegate: Rectangle {
                            property bool on: win.channelTab === modelData.k
                            Layout.fillWidth: true; Layout.preferredHeight: 34   // .ch-tabs button pad 8
                            radius: 9; color: on ? theme.accent : theme.bg2
                            Text { anchors.centerIn: parent; text: modelData.label; font.pixelSize: 13; font.weight: Font.DemiBold
                                color: on ? "#ffffff" : theme.text2 }
                            MouseArea { anchors.fill: parent; onClicked: {
                                win.channelTab = modelData.k
                                if (modelData.k === "discover") app.loadIntoA("GetRecommendedChannels", [""], channelsModel)
                                else win.loadView("channels")
                            } }
                        }
                    }
                }
                // .ch-search — input cari pada tab Jelajahi (radius10 pad 9/14 bg2).
                Rectangle {
                    Layout.fillWidth: true; Layout.leftMargin: 12; Layout.rightMargin: 12; Layout.bottomMargin: 8
                    Layout.preferredHeight: 37; radius: 10; color: theme.bg2
                    visible: activeView === "channels" && win.channelTab === "discover" && searchInput.text === ""
                    TextInput {
                        id: channelDiscSearch
                        anchors.fill: parent; anchors.leftMargin: 14; anchors.rightMargin: 14
                        verticalAlignment: TextInput.AlignVCenter; clip: true
                        color: theme.text; font.pixelSize: 14
                        onTextChanged: app.loadIntoA("GetRecommendedChannels", [text], channelsModel)
                    }
                    Text { visible: channelDiscSearch.text === ""
                        anchors.verticalCenter: parent.verticalCenter; anchors.left: parent.left; anchors.leftMargin: 14
                        text: i18n.t("search"); color: theme.text2; font.pixelSize: 14 }
                }
                // Daftar (swap per activeView): chats / calls / starred
                Item {
                    Layout.fillWidth: true; Layout.fillHeight: true
                    // --- Chats ---
                    ListView {
                        id: chatList
                        anchors.fill: parent
                        // .chat-list padding 4px 8px → row inset 8 + .chat-row pad 12 = avatar @20px.
                        anchors.leftMargin: 8; anchors.rightMargin: 8; anchors.topMargin: 4
                        visible: activeView === "chats" && searchInput.text === ""
                        clip: true; model: chatsModel; reuseItems: true
                        section.property: "sec"
                        section.criteria: ViewSection.FullString
                        section.delegate: Rectangle {
                            width: chatList.width; height: 28; color: theme.sidebarBg
                            Text {
                                anchors.verticalCenter: parent.verticalCenter; anchors.left: parent.left; anchors.leftMargin: 16
                                text: section === "pin" ? "DISEMATKAN" : "SEMUA CHAT"
                                color: theme.text2; font.pixelSize: 12; font.weight: Font.Medium
                            }
                        }
                        header: ItemDelegate {
                            width: chatList.width; height: 48  // = Svelte .archived (icon 22 + 2×12 pad)
                            onClicked: { activeView = "archived"; app.loadInto("GetArchivedChats", archivedModel) }
                            background: Rectangle { anchors.margins: 3; radius: theme.r; color: hovered ? theme.hover : "transparent" }
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 22; anchors.rightMargin: 16; spacing: 16
                                Icon { Layout.preferredWidth: 22; Layout.preferredHeight: 22; svg: win.ico["archived"]; color: theme.accent }
                                Text { Layout.fillWidth: true; text: "Diarsipkan"; color: theme.text; font.pixelSize: 15 }
                            }
                        }
                        delegate: ItemDelegate {
                            width: chatList.width; height: 70; clip: true  // = Svelte .chat-row (49 avatar + 2×10 pad + 1 mb)
                            property bool isActive: (win.selectedChat.id !== undefined) && win.selectedChat.id === model.m.id
                            onClicked: { win.selectedChat = model.m; app.openChat(model.m.id) }
                            background: Rectangle { anchors.margins: 3; radius: theme.r
                                color: isActive ? theme.selected : (hovered ? theme.hover : "transparent") }
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 12; anchors.rightMargin: 12; spacing: 13
                                Avatar {
                                    Layout.preferredWidth: 49; Layout.preferredHeight: 49; Layout.alignment: Qt.AlignVCenter
                                    name: model.m.name; jid: model.m.id; base: app.mediaBase; accent: win.avatarColor(model.m.name)
                                    group: model.m.group === true
                                }
                                ColumnLayout {
                                    Layout.fillWidth: true; Layout.alignment: Qt.AlignVCenter; spacing: 2
                                    // Baris 1: nama (kiri) + waktu (kanan)
                                    RowLayout {
                                        Layout.fillWidth: true; spacing: 8
                                        Text {
                                            Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1; wrapMode: Text.NoWrap
                                            text: model.m.name || model.m.id || ""
                                            font.pixelSize: 16; color: theme.text
                                            font.weight: model.m.unread ? Font.Bold : Font.Medium
                                        }
                                        Text {
                                            text: model.m.time || ""
                                            // .row-time text2; .chat-row.unread .row-time accent (keyed on unread, not badge).
                                            color: model.m.unread ? theme.accent : theme.text2; font.pixelSize: 12
                                        }
                                        // .row-menu-btn: chevron yg muncul saat hover (width 0→20, opacity 0→1).
                                        Rectangle {
                                            Layout.preferredWidth: hovered ? 20 : 0
                                            Layout.preferredHeight: 20; Layout.leftMargin: hovered ? 4 : 0
                                            radius: 6; clip: true
                                            // .row-menu-btn: no bg; :hover only changes icon color to var(--text).
                                            color: "transparent"
                                            opacity: hovered ? 1 : 0
                                            Behavior on Layout.preferredWidth { NumberAnimation { duration: 120 } }
                                            Behavior on opacity { NumberAnimation { duration: 120 } }
                                            Icon { anchors.centerIn: parent; width: 18; height: 18
                                                svg: win.ico["chevdown"]; color: rmHov.hovered ? theme.text : theme.text2 }
                                            HoverHandler { id: rmHov }
                                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                                                onClicked: { win.ctxChat = model.m; chatMenu.popup() } }
                                        }
                                    }
                                    // Baris 2: preview (kiri) + badge unread (kanan)
                                    RowLayout {
                                        Layout.fillWidth: true; spacing: 4
                                        // State preview (ChatRow.svelte): typing > draft > ticks+preview.
                                        readonly property bool rowTyping: !!model.m.typing
                                        readonly property bool rowDraft: !rowTyping && !!model.m.draft && !isActive
                                        // Mengetik… (.row-preview .typing: accent + italic).
                                        Text {
                                            visible: parent.rowTyping
                                            Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1; wrapMode: Text.NoWrap
                                            text: (model.m.typing && model.m.typing.name ? model.m.typing.name + " " : "")
                                                  + i18n.t((model.m.typing && model.m.typing.rec) ? "rec_voice" : "typing")
                                            color: theme.accent; font.italic: true; font.pixelSize: 14
                                        }
                                        // Draf: … (.row-preview .draft: #ef5350 weight 600 untuk label).
                                        Text {
                                            visible: parent.rowDraft
                                            text: i18n.t("draft") + ":"
                                            color: "#ef5350"; font.weight: Font.DemiBold; font.pixelSize: 14
                                        }
                                        Text {
                                            visible: parent.rowDraft
                                            Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1; wrapMode: Text.NoWrap
                                            text: model.m.draft || ""; color: theme.text2; font.pixelSize: 14
                                        }
                                        // Ticks preview (pesan terakhir keluar): "sending" → jam, "sent" → centang tunggal, "delivered"/"read" → ganda.
                                        Icon {
                                            visible: !parent.rowTyping && !parent.rowDraft && model.m.sent === true
                                            readonly property bool sending: model.m.status === "sending"
                                            // .row-preview .ticks always 16x12; .ticks.clock only overrides stroke color, not size.
                                            Layout.preferredWidth: 16; Layout.preferredHeight: 12; Layout.alignment: Qt.AlignVCenter
                                            vbox: sending ? "0 0 16 16" : "0 0 18 14"
                                            svg: sending ? win.ico["clockSm"]
                                               : model.m.status === "sent" ? win.ico["check"] : win.ico["checks"]
                                            color: model.m.status === "read" ? theme.tick : theme.text2
                                        }
                                        Text {
                                            visible: !parent.rowTyping && !parent.rowDraft
                                            Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1; wrapMode: Text.NoWrap
                                            text: model.m.preview || ""; font.pixelSize: 14
                                            // Unread → preview lebih terang + medium (app.css .chat-row.unread .row-preview).
                                            color: model.m.unread ? theme.text : theme.text2
                                            font.weight: model.m.unread ? Font.Medium : Font.Normal
                                        }
                                        Rectangle {
                                            visible: (model.m.badge || 0) > 0
                                            // .badge margin-left:8 (row spacing 4 + 4).
                                            Layout.leftMargin: 4
                                            radius: 10; color: theme.accent
                                            implicitWidth: Math.max(20, bdg.implicitWidth + 12); implicitHeight: 20
                                            Text { id: bdg; anchors.centerIn: parent; color: "white"; font.pixelSize: 12; font.weight: Font.DemiBold
                                                text: model.m.badge > 99 ? "99+" : (model.m.badge || 0) }
                                        }
                                        // Pin/mute (saat tak ada badge) — ala WhatsApp.
                                        Row {
                                            visible: !((model.m.badge || 0) > 0) && (model.m.pinned === true || model.m.muted === true)
                                            spacing: 4
                                            Icon { visible: model.m.muted === true; width: 16; height: 16; svg: win.ico["mute"]; color: theme.text2 }
                                            Icon { visible: model.m.pinned === true; width: 16; height: 16; rotation: 45; svg: win.ico["pin"]; color: theme.text2 }
                                        }
                                    }
                                }
                            }
                            // Klik-kanan baris chat → menu kelola chat.
                            MouseArea {
                                anchors.fill: parent
                                acceptedButtons: Qt.RightButton
                                onClicked: { win.ctxChat = model.m; chatMenu.popup() }
                            }
                        }
                    }
                    // --- Riwayat panggilan (CallsPane.svelte → .chat-row: avatar 49,
                    //     dua baris nama+waktu / panah-arah + 'Video/Voice · status'). ---
                    ListView {
                        anchors.fill: parent
                        visible: activeView === "calls" && searchInput.text === ""
                        clip: true; model: callsModel
                        delegate: ItemDelegate {
                            id: callRow
                            width: ListView.view.width; height: 70; clip: true  // = .chat-row metrics
                            onClicked: { win.selectedChat = { name: model.m.name, id: model.m.jid }; activeView = "chats"; app.openChat(model.m.jid) }
                            background: Rectangle { anchors.margins: 3; radius: theme.r; color: hovered ? theme.hover : "transparent" }
                            readonly property bool missed: model.m.status === "missed"
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 12; anchors.rightMargin: 12; spacing: 13
                                Avatar {
                                    Layout.preferredWidth: 49; Layout.preferredHeight: 49; Layout.alignment: Qt.AlignVCenter
                                    name: model.m.name; jid: model.m.jid; base: app.mediaBase; accent: win.avatarColor(model.m.name)
                                    group: model.m.group === true
                                }
                                ColumnLayout {
                                    Layout.fillWidth: true; Layout.alignment: Qt.AlignVCenter; spacing: 3
                                    RowLayout {
                                        Layout.fillWidth: true; spacing: 6
                                        Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1; wrapMode: Text.NoWrap
                                            text: model.m.name || ""; color: theme.text; font.pixelSize: 16; font.weight: Font.Medium }   // .row-name 16.5/500
                                        Text { text: model.m.time || ""; color: theme.text2; font.pixelSize: 12 }   // .row-time
                                    }
                                    RowLayout {
                                        Layout.fillWidth: true; spacing: 6
                                        Icon {   // .call-ico 15px — satu panah diagonal statis (CallsPane.svelte); merah hanya saat missed
                                            Layout.preferredWidth: 15; Layout.preferredHeight: 15; Layout.alignment: Qt.AlignVCenter
                                            svg: win.ico["callArrowOut"]
                                            color: callRow.missed ? "#ef5350" : theme.text2
                                        }
                                        Text {
                                            Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1; wrapMode: Text.NoWrap
                                            text: (model.m.video ? i18n.t("call_video") : i18n.t("call_voice")) + " · "
                                                  + (callRow.missed ? i18n.t("call_missed") : i18n.t("call_rejected"))
                                            color: callRow.missed ? "#ef5350" : theme.text2; font.pixelSize: 14   // .row-preview 14
                                        }
                                    }
                                }
                            }
                        }
                    }
                    // --- Pesan berbintang (StarredPane.svelte → .hit-row: .hit-av 40px,
                    //     .hit-top nama 15/500 + waktu 12, .hit-text '⭐ '+teks 13.5). ---
                    ListView {
                        anchors.fill: parent
                        visible: activeView === "starred" && searchInput.text === ""
                        clip: true; model: starredModel
                        delegate: ItemDelegate {
                            width: ListView.view.width; height: 62; clip: true
                            onClicked: { win.selectedChat = { name: model.m.chatName, id: model.m.chatJid }; activeView = "chats"; app.openChat(model.m.chatJid) }
                            background: Rectangle { anchors.margins: 3; radius: theme.r; color: hovered ? theme.hover : "transparent" }
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 12; anchors.rightMargin: 12; spacing: 12
                                Avatar {   // .hit-av 40px lingkaran berwarna + inisial
                                    Layout.preferredWidth: 40; Layout.preferredHeight: 40; Layout.alignment: Qt.AlignVCenter; fontSize: 16
                                    name: model.m.chatName; jid: model.m.chatJid; base: app.mediaBase; accent: win.avatarColor(model.m.chatName)
                                }
                                ColumnLayout {
                                    Layout.fillWidth: true; Layout.alignment: Qt.AlignVCenter; spacing: 2
                                    RowLayout {
                                        Layout.fillWidth: true; spacing: 6
                                        Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1; wrapMode: Text.NoWrap
                                            text: model.m.chatName || ""; color: theme.text; font.pixelSize: 15; font.weight: Font.Medium }   // .hit-name
                                        Text { text: model.m.time || ""; color: theme.text2; font.pixelSize: 12 }   // .hit-time
                                    }
                                    Text {   // .hit-text 13.5 — bintang mendahului preview, bukan nama
                                        Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1; wrapMode: Text.NoWrap
                                        text: "⭐ " + (model.m.text || ("(" + i18n.t("media_generic") + ")"))
                                        color: theme.text2; font.pixelSize: 13
                                    }
                                }
                            }
                        }
                    }
                    // --- Status (StatusPane.svelte → 'My status' add row di atas, .ct-letter
                    //     pemisah abjad, lalu .status-row: cincin 48px av + 2.5px pad
                    //     (accent belum-dilihat / --line sudah), nama 15/600, sub time[·N]). ---
                    ListView {
                        id: statusList
                        anchors.fill: parent
                        visible: activeView === "status" && searchInput.text === ""
                        clip: true; model: statusModel
                        // 'My status' .status-row: status-av + '+' badge, nama + status_add_hint.
                        header: Rectangle {
                            width: ListView.view.width; height: 68
                            color: myStMa.containsMouse ? theme.hover : "transparent"
                            MouseArea { id: myStMa; anchors.fill: parent; hoverEnabled: true }
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 14; anchors.rightMargin: 12; spacing: 14
                                Item {   // .status-av-wrap + '+' badge
                                    Layout.preferredWidth: 48; Layout.preferredHeight: 48; Layout.alignment: Qt.AlignVCenter
                                    Avatar {   // .status-av 48px (akun "me")
                                        anchors.fill: parent; fontSize: 18
                                        name: "?"; jid: "me"; base: app.mediaBase; accent: win.avatarColor("me")
                                    }
                                    Rectangle {   // .status-add 18px accent, '+' putih, border bg
                                        width: 18; height: 18; radius: 9; color: theme.accent
                                        border.width: 2; border.color: theme.sidebarBg
                                        anchors.right: parent.right; anchors.bottom: parent.bottom
                                        anchors.rightMargin: -2; anchors.bottomMargin: -2
                                        Text { anchors.centerIn: parent; text: "+"; color: "#ffffff"; font.pixelSize: 14 }
                                    }
                                }
                                ColumnLayout {
                                    Layout.fillWidth: true; Layout.alignment: Qt.AlignVCenter; spacing: 2
                                    Text { text: i18n.t("my_status"); color: theme.text; font.pixelSize: 15; font.weight: Font.DemiBold }
                                    Text { text: i18n.t("status_add_hint"); color: theme.text2; font.pixelSize: 13 }
                                }
                            }
                        }
                        delegate: Column {
                            width: ListView.view.width
                            Item {   // .ct-letter pemisah abjad (accent 12/700, pad 5/16)
                                width: parent.width
                                height: win.startsLetterGroup(statusModel, index, model.m.name) ? 26 : 0
                                visible: height > 0; clip: true
                                Rectangle { anchors.fill: parent; color: theme.bg }
                                Text {
                                    anchors.verticalCenter: parent.verticalCenter; anchors.left: parent.left; anchors.leftMargin: 16
                                    text: win.letterOf(model.m.name); color: theme.accent; font.pixelSize: 12; font.weight: Font.Bold
                                }
                            }
                            Rectangle {
                                width: parent.width; height: 68; clip: true   // .status-row pad 10/14
                                color: stMa.containsMouse ? theme.hover : "transparent"
                                MouseArea { id: stMa; anchors.fill: parent; hoverEnabled: true
                                    onClicked: { win.selectedChat = { name: model.m.name, id: model.m.id }; activeView = "chats"; app.openChat(model.m.id) } }
                                RowLayout {
                                    anchors.fill: parent; anchors.leftMargin: 14; anchors.rightMargin: 12; spacing: 14
                                    Rectangle {   // .ring — latar penuh (accent/--line), inset 2.5 → cincin
                                        Layout.preferredWidth: 53; Layout.preferredHeight: 53; Layout.alignment: Qt.AlignVCenter
                                        radius: width / 2
                                        color: model.m.seen ? theme.line : theme.accent
                                        Avatar {   // .status-av 48px
                                            anchors.centerIn: parent; width: 48; height: 48; fontSize: 18
                                            name: model.m.name; jid: model.m.id; base: app.mediaBase; accent: win.avatarColor(model.m.name)
                                        }
                                    }
                                    ColumnLayout {
                                        Layout.fillWidth: true; Layout.alignment: Qt.AlignVCenter; spacing: 2
                                        Text { text: model.m.name || ""; color: theme.text; font.pixelSize: 15; font.weight: Font.DemiBold }   // .status-name 15/600
                                        Text {   // .status-sub 12.5 — waktu, lalu ' · N' hanya bila count>1
                                            text: (model.m.time || "") + ((model.m.count || 0) > 1 ? " · " + model.m.count : "")
                                            color: theme.text2; font.pixelSize: 13
                                        }
                                    }
                                }
                            }
                        }
                    }
                    // --- Kontak (ContactsPane.svelte: .ct-top search+new, .ct-letter
                    //     pemisah abjad, .ct-row av 40 + titik online + .ct-info). ---
                    ListView {
                        id: contactsList
                        anchors.fill: parent
                        visible: activeView === "contacts" && searchInput.text === ""
                        clip: true; model: contactsModel
                        // .ct-top { display:flex; gap:8px; padding:6px 12px 10px } — search + tombol new accent.
                        header: Item {
                            width: ListView.view.width; height: 53   // 6+37(input)+10
                            RowLayout {
                                anchors.fill: parent
                                anchors.leftMargin: 12; anchors.rightMargin: 12
                                anchors.topMargin: 6; anchors.bottomMargin: 10; spacing: 8
                                Rectangle {   // .ct-search flex:1 radius10 pad 9/14 bg2
                                    Layout.fillWidth: true; Layout.fillHeight: true
                                    radius: 10; color: theme.bg2
                                    TextInput {
                                        id: contactsSearch
                                        anchors.fill: parent; anchors.leftMargin: 14; anchors.rightMargin: 14
                                        verticalAlignment: TextInput.AlignVCenter; clip: true
                                        color: theme.text; font.pixelSize: 14
                                    }
                                    Text {
                                        visible: contactsSearch.text === ""
                                        anchors.verticalCenter: parent.verticalCenter
                                        anchors.left: parent.left; anchors.leftMargin: 14
                                        text: i18n.t("search"); color: theme.text2; font.pixelSize: 14
                                    }
                                }
                                ItemDelegate {   // .ct-new accent bg, #fff, radius10, pad 8/11, person+plus svg 17
                                    Layout.fillHeight: true; Layout.preferredWidth: ctNewRow.implicitWidth + 22
                                    hoverEnabled: true
                                    background: Rectangle { radius: 10; color: parent.hovered ? theme.accentDeep : theme.accent }
                                    onClicked: contactDialog.open()
                                    Row {
                                        id: ctNewRow; anchors.centerIn: parent; spacing: 6
                                        Icon { anchors.verticalCenter: parent.verticalCenter; width: 17; height: 17
                                            svg: win.ico["addmember"]; color: "#ffffff" }
                                        Text { anchors.verticalCenter: parent.verticalCenter
                                            text: i18n.t("contact_save_new"); color: "#ffffff"; font.pixelSize: 13 }
                                    }
                                }
                            }
                        }
                        delegate: Column {
                            id: ctRow
                            width: ListView.view.width
                            // .ct-letter { color:accent; font 12/700; padding 5px 16px } — sebelum tiap grup huruf.
                            Item {
                                width: parent.width
                                height: win.startsLetterGroup(contactsModel, index, model.m.name) ? 26 : 0
                                visible: height > 0; clip: true
                                Rectangle { anchors.fill: parent; color: theme.bg }
                                Text {
                                    anchors.verticalCenter: parent.verticalCenter; anchors.left: parent.left; anchors.leftMargin: 16
                                    text: win.letterOf(model.m.name); color: theme.accent; font.pixelSize: 12; font.weight: Font.Bold
                                }
                            }
                            Rectangle {   // .ct-row pad 8/14 + 40 av
                                width: parent.width; height: 56; clip: true
                                color: ctMa.containsMouse ? theme.hover : "transparent"
                                MouseArea {
                                    id: ctMa; anchors.fill: parent; hoverEnabled: true
                                    acceptedButtons: Qt.LeftButton | Qt.RightButton
                                    onClicked: function(e) {
                                        if (e.button === Qt.RightButton) { win.ctxChat = { id: model.m.jid, name: model.m.name }; contactMenu.popup() }
                                        else { win.selectedChat = { name: model.m.name, id: model.m.jid }; activeView = "chats"; app.openChat(model.m.jid) }
                                    }
                                }
                                RowLayout {
                                    anchors.fill: parent; anchors.leftMargin: 14; anchors.rightMargin: 14; spacing: 12
                                    Item {   // .ct-av — pembungkus relatif utk titik online
                                        Layout.preferredWidth: 40; Layout.preferredHeight: 40; Layout.alignment: Qt.AlignVCenter
                                        Avatar {   // Avatar sm = 40px
                                            anchors.fill: parent; fontSize: 16
                                            name: model.m.name; jid: model.m.jid; base: app.mediaBase; accent: win.avatarColor(model.m.name)
                                        }
                                        Rectangle {   // .ct-dot — 12px hijau saat online (guard: model.m.online)
                                            visible: model.m.online === true
                                            width: 12; height: 12; radius: 6; color: "#28c840"
                                            border.width: 2; border.color: theme.sidebarBg
                                            anchors.right: parent.right; anchors.bottom: parent.bottom
                                            anchors.rightMargin: -1; anchors.bottomMargin: -1
                                        }
                                    }
                                    ColumnLayout {
                                        Layout.fillWidth: true; Layout.alignment: Qt.AlignVCenter; spacing: 2
                                        RowLayout {   // .ct-name 15 + .ct-num inline (kontak belum disimpan)
                                            Layout.fillWidth: true; spacing: 6
                                            Text { elide: Text.ElideRight; maximumLineCount: 1; Layout.maximumWidth: parent.width - 60
                                                text: model.m.name || ""; color: theme.text; font.pixelSize: 15 }
                                            Text { visible: model.m.saved !== true && (model.m.phone || "") !== ""
                                                text: model.m.phone || ""; color: theme.text2; font.pixelSize: 12 }   // .ct-num 12.5
                                            Item { Layout.fillWidth: true }
                                        }
                                        Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1; visible: text !== ""
                                            text: model.m.about || model.m.phone || ""; color: theme.text2; font.pixelSize: 13 }   // .ct-sub 12.5
                                    }
                                    ItemDelegate {   // .ct-info — tombol info 20px, hover bg2/accent
                                        Layout.preferredWidth: 32; Layout.preferredHeight: 32; Layout.alignment: Qt.AlignVCenter
                                        hoverEnabled: true
                                        background: Rectangle { radius: width / 2; color: parent.hovered ? theme.bg2 : "transparent" }
                                        onClicked: { win.ctxChat = { id: model.m.jid, name: model.m.name }; contactMenu.popup() }
                                        Icon { anchors.centerIn: parent; width: 20; height: 20
                                            svg: win.ico["ctInfo"]; color: parent.hovered ? theme.accent : theme.text2 }
                                    }
                                }
                            }
                        }
                    }
                    // --- Channels / Communities / Archived / Scheduled (pola sama) ---
                    // --- Saluran (ChannelsPane.svelte → .ch-row: av 48px, nama 15/600 +
                    //     badge ✓ terverifikasi, sub = N subscribers, tombol mute + unfollow). ---
                    ListView {
                        anchors.fill: parent; visible: activeView === "channels" && searchInput.text === ""
                        clip: true; model: channelsModel
                        delegate: ItemDelegate {
                            width: ListView.view.width; height: 68; clip: true   // .ch-row pad 10/14 + 48 av
                            onClicked: { win.selectedChat = { name: model.m.name, id: model.m.jid }; activeView = "chats"; app.openChat(model.m.jid) }
                            background: Rectangle { anchors.margins: 3; radius: theme.r; color: hovered ? theme.hover : "transparent" }
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 14; anchors.rightMargin: 14; spacing: 13
                                Avatar {   // .ch-av 48px lingkaran
                                    Layout.preferredWidth: 48; Layout.preferredHeight: 48; Layout.alignment: Qt.AlignVCenter; fontSize: 18
                                    name: model.m.name; jid: model.m.jid; base: app.mediaBase; accent: win.avatarColor(model.m.name)
                                }
                                ColumnLayout {
                                    Layout.fillWidth: true; Layout.alignment: Qt.AlignVCenter; spacing: 2
                                    RowLayout {   // .ch-name 15/600 + ✓ badge
                                        Layout.fillWidth: true; spacing: 5
                                        Text { elide: Text.ElideRight; maximumLineCount: 1; Layout.fillWidth: false; Layout.maximumWidth: parent.width - 20
                                            text: model.m.name || ""; color: theme.text; font.pixelSize: 15; font.weight: Font.DemiBold }
                                        Rectangle {   // .ch-verif — lingkaran accent 15px, ✓ putih (guard: model.m.verified)
                                            visible: model.m.verified === true
                                            width: 15; height: 15; radius: width / 2; color: theme.accent; Layout.alignment: Qt.AlignVCenter
                                            Icon { anchors.centerIn: parent; width: 10; height: 10; vbox: "0 0 24 24"; svg: win.ico["verif"]; color: "white" }
                                        }
                                        Item { Layout.fillWidth: true }
                                    }
                                    Text {   // .ch-sub 12.5 — N subscribers
                                        Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1
                                        text: (model.m.subscribers || 0) + " " + i18n.t("ch_subs")
                                        color: theme.text2; font.pixelSize: 13
                                    }
                                }
                                ItemDelegate {   // .ch-act mute/unmute (guard: model.m.muted)
                                    Layout.preferredWidth: 30; Layout.preferredHeight: 30; Layout.alignment: Qt.AlignVCenter
                                    hoverEnabled: true; opacity: hovered ? 1 : 0.6
                                    background: Rectangle { color: "transparent" }
                                    onClicked: { win.ctxChat = { id: model.m.jid || model.m.id || "", name: model.m.name }; channelMenu.popup() }
                                    Icon { anchors.centerIn: parent; width: 18; height: 18
                                        svg: model.m.muted === true ? win.ico["mute"] : win.ico["bell"]; color: theme.text2 }
                                }
                                ItemDelegate {   // .ch-act unfollow ✕
                                    Layout.preferredWidth: 30; Layout.preferredHeight: 30; Layout.alignment: Qt.AlignVCenter
                                    hoverEnabled: true; opacity: hovered ? 1 : 0.6
                                    background: Rectangle { color: "transparent" }
                                    onClicked: { win.ctxChat = { id: model.m.jid || model.m.id || "", name: model.m.name }; channelMenu.popup() }
                                    Icon { anchors.centerIn: parent; width: 16; height: 16; svg: win.ico["close"]; color: theme.text2 }
                                }
                            }
                            MouseArea { anchors.fill: parent; acceptedButtons: Qt.RightButton
                                onClicked: { win.ctxChat = { id: model.m.jid || model.m.id || "", name: model.m.name }; channelMenu.popup() } }
                        }
                    }
                    // --- Komunitas (CommunitiesPane.svelte → .comm: border-bottom line;
                    //     .comm-head av 46 radius16, nama 15/600, sub = N grup, tombol keluar +
                    //     chevron berputar; saat terbuka → .comm-groups (.comm-grow # 32px,
                    //     nama 14, .comm-tag pengumuman). Default semua diperluas. ---
                    ListView {
                        anchors.fill: parent; visible: activeView === "communities" && searchInput.text === ""
                        clip: true; model: communitiesModel
                        delegate: Column {
                            id: commCol
                            width: ListView.view.width
                            // kunci stabil per komunitas (delegate daur-ulang); default terbuka.
                            property string commKey: model.m.jid || model.m.id || model.m.name || ""
                            property bool isOpen: win.commOpen[commKey] === undefined ? true : win.commOpen[commKey]
                            property var grps: model.m.groups || []
                            Rectangle {   // .comm-head pad 11/14 + 46 av
                                width: parent.width; height: 68; clip: true
                                color: commHeadMa.containsMouse ? theme.hover : "transparent"
                                MouseArea { id: commHeadMa; anchors.fill: parent; hoverEnabled: true
                                    onClicked: win.commToggle(commCol.commKey) }
                                RowLayout {
                                    anchors.fill: parent; anchors.leftMargin: 14; anchors.rightMargin: 14; spacing: 13
                                    Rectangle {   // .comm-av 46px persegi-membulat berwarna + inisial
                                        Layout.preferredWidth: 46; Layout.preferredHeight: 46; Layout.alignment: Qt.AlignVCenter
                                        radius: 16; color: win.avatarColor(model.m.name)
                                        Text { anchors.centerIn: parent; color: "white"; font.pixelSize: 18; font.weight: Font.DemiBold
                                            text: (model.m.name || "?").charAt(0).toUpperCase() }
                                    }
                                    ColumnLayout {
                                        Layout.fillWidth: true; Layout.alignment: Qt.AlignVCenter; spacing: 2
                                        Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1
                                            text: model.m.name || ""; color: theme.text; font.pixelSize: 15; font.weight: Font.DemiBold }   // .comm-name
                                        Text {   // .comm-sub 12.5 — jumlah grup
                                            Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1
                                            text: ((model.m.groups ? model.m.groups.length : (model.m.count || 0))) + " " + i18n.t("comm_groups")
                                            color: theme.text2; font.pixelSize: 13
                                        }
                                    }
                                    ItemDelegate {   // .ch-act tombol keluar (leavegroup)
                                        Layout.preferredWidth: 28; Layout.preferredHeight: 28; Layout.alignment: Qt.AlignVCenter
                                        hoverEnabled: true; opacity: hovered ? 1 : 0.6
                                        background: Rectangle { color: "transparent" }
                                        onClicked: { win.ctxChat = { id: commCol.commKey, name: model.m.name }; commMenu.popup() }
                                        Icon { anchors.centerIn: parent; width: 18; height: 18; svg: win.ico["leavegroup"]; color: theme.text2 }
                                    }
                                    Text {   // .comm-chev ▾ — berputar 180° saat terbuka
                                        text: "▾"; color: theme.text2; font.pixelSize: 14; Layout.alignment: Qt.AlignVCenter
                                        rotation: commCol.isOpen ? 180 : 0
                                        Behavior on rotation { NumberAnimation { duration: 150 } }
                                    }
                                }
                            }
                            // .comm-groups (pad 2/0 8/0) — daftar grup anggota saat terbuka.
                            Column {
                                width: parent.width
                                visible: commCol.isOpen
                                topPadding: 2; bottomPadding: 8
                                Repeater {
                                    model: commCol.grps
                                    delegate: Rectangle {   // .comm-grow pad 8/14 8/26 — # ico + nama + tag
                                        width: commCol.width; height: 48
                                        color: cgMa.containsMouse ? theme.hover : "transparent"
                                        MouseArea { id: cgMa; anchors.fill: parent; hoverEnabled: true
                                            onClicked: { var gid = modelData.jid || modelData.id || ""; if (gid) { win.selectedChat = { name: modelData.name, id: gid }; activeView = "chats"; app.openChat(gid) } } }
                                        RowLayout {
                                            anchors.fill: parent; anchors.leftMargin: 26; anchors.rightMargin: 14; spacing: 12
                                            Rectangle {   // .comm-gico 32px radius10 bg2 '#'
                                                Layout.preferredWidth: 32; Layout.preferredHeight: 32; Layout.alignment: Qt.AlignVCenter
                                                radius: 10; color: theme.bg2
                                                Text { anchors.centerIn: parent; text: "#"; color: theme.text2; font.pixelSize: 15; font.weight: Font.DemiBold }
                                            }
                                            Text {   // .comm-gname 14
                                                Layout.fillWidth: false; elide: Text.ElideRight; maximumLineCount: 1
                                                text: modelData.name || ""; color: theme.text; font.pixelSize: 14 }
                                            Rectangle {   // .comm-tag pengumuman (accent pill 10px)
                                                visible: modelData.isDefault === true
                                                Layout.alignment: Qt.AlignVCenter
                                                radius: 6; color: theme.accent
                                                implicitWidth: commTag.implicitWidth + 12; implicitHeight: commTag.implicitHeight + 2
                                                Text { id: commTag; anchors.centerIn: parent; text: i18n.t("comm_announce"); color: "#ffffff"; font.pixelSize: 10 }
                                            }
                                            Item { Layout.fillWidth: true }
                                        }
                                    }
                                }
                                // Kosong → "Belum ada grup" (mock tak memuat field groups → NOTE limitasi).
                                Item {
                                    width: parent.width; height: commCol.grps.length === 0 ? 34 : 0
                                    visible: height > 0
                                    Text { anchors.verticalCenter: parent.verticalCenter; anchors.left: parent.left; anchors.leftMargin: 16
                                        text: i18n.t("comm_no_groups"); color: theme.text2; font.pixelSize: 13 }
                                }
                            }
                            // .comm border-bottom 1px var(--line)
                            Rectangle { width: parent.width; height: 1; color: theme.line }
                        }
                    }
                    // --- Diarsipkan (ArchivedPane.svelte → <ChatRow>: sama dgn baris chat,
                    //     av 49px, nama 16/medium + waktu, preview 14, hover). ---
                    ListView {
                        anchors.fill: parent; visible: activeView === "archived" && searchInput.text === ""
                        clip: true; model: archivedModel
                        delegate: ItemDelegate {
                            width: ListView.view.width; height: 70; clip: true   // = .chat-row metrics
                            onClicked: { win.selectedChat = model.m; app.openChat(model.m.id) }
                            background: Rectangle { anchors.margins: 3; radius: theme.r; color: hovered ? theme.hover : "transparent" }
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 12; anchors.rightMargin: 12; spacing: 13
                                Avatar {
                                    Layout.preferredWidth: 49; Layout.preferredHeight: 49; Layout.alignment: Qt.AlignVCenter
                                    name: model.m.name; jid: model.m.id; base: app.mediaBase; accent: win.avatarColor(model.m.name)
                                    group: model.m.group === true
                                }
                                ColumnLayout {
                                    Layout.fillWidth: true; Layout.alignment: Qt.AlignVCenter; spacing: 3
                                    RowLayout {
                                        Layout.fillWidth: true; spacing: 6
                                        Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1; wrapMode: Text.NoWrap
                                            text: model.m.name || model.m.id || ""; color: theme.text; font.pixelSize: 16; font.weight: Font.Medium }
                                        Text { text: model.m.time || ""; color: theme.text2; font.pixelSize: 12 }
                                    }
                                    Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1; wrapMode: Text.NoWrap
                                        text: model.m.preview || ""; color: theme.text2; font.pixelSize: 14 }
                                }
                            }
                            MouseArea { anchors.fill: parent; acceptedButtons: Qt.RightButton
                                onClicked: { win.ctxChat = model.m; chatMenu.popup() } }
                        }
                    }
                    // --- Terjadwal (ScheduledPane.svelte → .sc-sec header bagian (12/700
                    //     uppercase, letter-spacing .4), lalu .sc-row: nama 14/600, teks 13,
                    //     'jam' accent 12 + ikon jam, tombol ✕ batal 30px).
                    //     NOTE: scheduledModel = SATU daftar datar tanpa field 'kind' →
                    //     pemisahan Terjadwal/Pengingat diturunkan dari model.m.kind bila ada
                    //     (lights up saat backend isi); jika tidak, semua di bawah 'Terjadwal'
                    //     dan header 'Pengingat (0)' tampil kosong. Tidak menebak backend. ---
                    ListView {
                        id: schedList
                        anchors.fill: parent; visible: activeView === "scheduled" && searchInput.text === ""
                        clip: true; model: scheduledModel
                        // header 'Terjadwal (N)' — .sc-sec uppercase 12/700.
                        header: Column {
                            width: ListView.view.width
                            Item {
                                width: parent.width; height: 34   // .sc-sec pad 14/16 6
                                Text {
                                    anchors.left: parent.left; anchors.leftMargin: 16; anchors.bottom: parent.bottom; anchors.bottomMargin: 6
                                    text: i18n.t("sc_scheduled").toUpperCase() + " (" + schedList.count + ")"
                                    color: theme.text2; font.pixelSize: 12; font.weight: Font.Bold; font.letterSpacing: 0.4
                                }
                            }
                        }
                        delegate: Column {
                            width: ListView.view.width
                            // Pisah bagian 'Pengingat' saat baris ini reminder & sebelumnya bukan (model.m.kind).
                            property bool isReminder: (model.m.kind === "reminder") || (model.m.remindAt !== undefined)
                            property bool startReminders: {
                                if (!isReminder) return false
                                if (index === 0) return true
                                var prev = scheduledModel.get(index - 1)
                                var pr = prev && ((prev.kind === "reminder") || (prev.remindAt !== undefined))
                                return !pr
                            }
                            Item {   // .sc-sec 'PENGINGAT' — hanya saat awal grup reminder
                                width: parent.width; height: startReminders ? 34 : 0
                                visible: height > 0; clip: true
                                Text {
                                    anchors.left: parent.left; anchors.leftMargin: 16; anchors.bottom: parent.bottom; anchors.bottomMargin: 6
                                    text: i18n.t("sc_reminders").toUpperCase()
                                    color: theme.text2; font.pixelSize: 12; font.weight: Font.Bold; font.letterSpacing: 0.4
                                }
                            }
                            Rectangle {
                                width: parent.width; height: 66; clip: true   // .sc-row pad 6/12 + 3 baris
                                color: scMa.containsMouse ? theme.hover : "transparent"
                                MouseArea { id: scMa; anchors.fill: parent; hoverEnabled: true
                                    acceptedButtons: Qt.LeftButton | Qt.RightButton
                                    onClicked: function(e) {
                                        if (e.button === Qt.RightButton) { win.ctxChat = { id: model.m.id || "", name: model.m.chatName }; schedMenu.popup() }
                                        else { win.selectedChat = { name: model.m.chatName, id: model.m.chatJid || model.m.id }; activeView = "chats"; app.openChat(model.m.chatJid || model.m.id) }
                                    }
                                }
                                RowLayout {
                                    anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12; spacing: 8
                                    ColumnLayout {
                                        Layout.fillWidth: true; Layout.alignment: Qt.AlignVCenter; spacing: 2
                                        Text {   // .sc-name 14/600
                                            Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1
                                            text: model.m.chatName || (model.m.chatJid ? model.m.chatJid.split("@")[0] : "")
                                            color: theme.text; font.pixelSize: 14; font.weight: Font.DemiBold
                                        }
                                        Text {   // .sc-text 13 text2
                                            Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1
                                            text: model.m.text || model.m.note || ""; color: theme.text2; font.pixelSize: 13
                                        }
                                        RowLayout {   // .sc-when accent 12 + ikon jam/lonceng
                                            Layout.fillWidth: true; spacing: 5
                                            Icon { Layout.preferredWidth: 13; Layout.preferredHeight: 13; Layout.alignment: Qt.AlignVCenter
                                                svg: isReminder ? win.ico["bell"] : win.ico["clock"]; color: theme.accent }
                                            Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1
                                                text: model.m.time || ""; color: theme.accent; font.pixelSize: 12 }
                                        }
                                    }
                                    ItemDelegate {   // .sc-x — ✕ batal, lingkaran bg2 30px
                                        Layout.preferredWidth: 30; Layout.preferredHeight: 30; Layout.alignment: Qt.AlignVCenter
                                        hoverEnabled: true
                                        background: Rectangle { radius: width / 2; color: theme.bg2 }   // .sc-x: no :hover state in Svelte
                                        onClicked: { win.ctxChat = { id: model.m.id || "", name: model.m.chatName }; schedMenu.popup() }
                                        Icon { anchors.centerIn: parent; width: 14; height: 14; svg: win.ico["close"]; color: theme.text2 }
                                    }
                                }
                            }
                        }
                    }
                    // --- Hasil pencarian (override semua view saat mengetik) ---
                    // Parity ChatList.svelte (cabang search): chip jenis (.sc-types) +
                    // bagian "Chats" (nama cocok) + "Messages" (hit isi) + empty-list.
                    Flickable {
                        id: searchOverlay
                        anchors.fill: parent
                        visible: searchInput.text !== ""
                        clip: true
                        contentWidth: width; contentHeight: searchCol.implicitHeight
                        boundsBehavior: Flickable.StopAtBounds
                        // Chat yang nama-nya cocok query (.list-label "Chats").
                        property var matched: win.nameMatches(searchInput.text, chatList.count)
                        ColumnLayout {
                            id: searchCol
                            width: searchOverlay.width
                            spacing: 0
                            // .sc-types: flex gap 6, padding 8/12 (Svelte style override).
                            Flow {
                                Layout.fillWidth: true
                                Layout.leftMargin: 12; Layout.rightMargin: 12
                                Layout.topMargin: 8; Layout.bottomMargin: 8; spacing: 6
                                Repeater {
                                    model: ["all", "image", "video", "document", "link", "voice"]
                                    delegate: Rectangle {
                                        property bool on: win.searchType === modelData
                                        radius: 16; height: 26  // .chip radius16 pad5/13
                                        implicitWidth: scLbl.implicitWidth + 26
                                        color: on ? theme.accent : theme.searchBg
                                        Text { id: scLbl; anchors.centerIn: parent
                                            text: i18n.t("sct_" + modelData); font.pixelSize: 13
                                            color: on ? "#ffffff" : theme.text2 }
                                        MouseArea { anchors.fill: parent; onClicked: win.searchType = modelData }
                                    }
                                }
                            }
                            // .list-label "CHATS" — hanya filter "all" & ada nama cocok.
                            Text {
                                Layout.fillWidth: true
                                Layout.leftMargin: 16; Layout.rightMargin: 16
                                Layout.topMargin: 8; Layout.bottomMargin: 4
                                visible: win.searchType === "all" && searchOverlay.matched.length > 0
                                text: i18n.t("rail_chats").toUpperCase()
                                color: theme.text2; font.pixelSize: 12; font.letterSpacing: 0.4
                            }
                            // Baris chat (nama cocok) — pakai metrik ChatRow penuh (ChatList.svelte
                            // me-render <ChatRow chat={c}/>): avatar 49, .row-name 16.5/500, gap 13,
                            // pad 10/12, radius var(--r), + .row-time (accent saat unread) & ticks.
                            Repeater {
                                model: (win.searchType === "all") ? searchOverlay.matched : []
                                delegate: Rectangle {
                                    id: scRow
                                    Layout.fillWidth: true
                                    Layout.leftMargin: 0; Layout.rightMargin: 0
                                    Layout.preferredHeight: scRowRow.implicitHeight + 20  // .chat-row padding 10/10
                                    Layout.bottomMargin: 1
                                    radius: theme.r
                                    color: ncHov.hovered ? theme.hover : "transparent"
                                    RowLayout {
                                        id: scRowRow
                                        anchors.left: parent.left; anchors.right: parent.right
                                        anchors.verticalCenter: parent.verticalCenter
                                        anchors.leftMargin: 12; anchors.rightMargin: 12; spacing: 13
                                        Avatar {
                                            Layout.preferredWidth: 49; Layout.preferredHeight: 49; fontSize: 19
                                            name: modelData.name || ""; jid: modelData.id || ""
                                            base: app.mediaBase; accent: win.avatarColor(modelData.name || "?")
                                            group: modelData.group === true
                                        }
                                        ColumnLayout {
                                            Layout.fillWidth: true; spacing: 2
                                            RowLayout {
                                                Layout.fillWidth: true; spacing: 8
                                                Text { Layout.fillWidth: true; elide: Text.ElideRight
                                                    text: modelData.name || modelData.id || ""; color: theme.text; font.pixelSize: 16
                                                    font.weight: modelData.unread ? Font.Bold : Font.Medium }
                                                // .row-time: text2, accent saat unread.
                                                Text { text: modelData.time || ""; font.pixelSize: 12
                                                    color: modelData.unread ? theme.accent : theme.text2 }
                                            }
                                            RowLayout {
                                                Layout.fillWidth: true; spacing: 4
                                                // Ticks pesan terakhir keluar (mirip ChatRow): sending=jam, sent=tunggal, lainnya=ganda.
                                                Icon {
                                                    visible: modelData.sent === true
                                                    readonly property bool sending: modelData.status === "sending"
                                                    Layout.preferredWidth: 16; Layout.preferredHeight: 12; Layout.alignment: Qt.AlignVCenter
                                                    vbox: sending ? "0 0 16 16" : "0 0 18 14"
                                                    svg: sending ? win.ico["clockSm"]
                                                       : modelData.status === "sent" ? win.ico["check"] : win.ico["checks"]
                                                    color: modelData.status === "read" ? theme.tick : theme.text2
                                                }
                                                Text { Layout.fillWidth: true; elide: Text.ElideRight
                                                    text: modelData.preview || ""; font.pixelSize: 14
                                                    color: modelData.unread ? theme.text : theme.text2
                                                    font.weight: modelData.unread ? Font.Medium : Font.Normal }
                                                // .badge unread.
                                                Rectangle {
                                                    visible: (modelData.badge || 0) > 0
                                                    Layout.leftMargin: 4; radius: 10; color: theme.accent
                                                    implicitWidth: Math.max(20, scBdg.implicitWidth + 12); implicitHeight: 20
                                                    Text { id: scBdg; anchors.centerIn: parent; color: "white"; font.pixelSize: 12; font.weight: Font.DemiBold
                                                        text: (modelData.badge || 0) > 99 ? "99+" : (modelData.badge || 0) }
                                                }
                                            }
                                        }
                                    }
                                    HoverHandler { id: ncHov }
                                    MouseArea { anchors.fill: parent; onClicked: {
                                        if ((modelData.id || "") !== "") {
                                            win.selectedChat = modelData
                                            activeView = "chats"; searchInput.text = ""; app.openChat(modelData.id)
                                        }
                                    } }
                                }
                            }
                            // .list-label "MESSAGES" — di atas hit isi pesan.
                            Text {
                                Layout.fillWidth: true
                                Layout.leftMargin: 16; Layout.rightMargin: 16
                                Layout.topMargin: 8; Layout.bottomMargin: 4
                                visible: hitRep.count > 0
                                text: i18n.t("messages_label").toUpperCase()
                                color: theme.text2; font.pixelSize: 12; font.letterSpacing: 0.4
                            }
                            // Hit isi pesan (searchModel ≤50). Repeater agar bisa diselang label.
                            Repeater {
                                id: hitRep
                                model: searchModel
                                delegate: Rectangle {
                                    Layout.fillWidth: true
                                    Layout.leftMargin: 8; Layout.rightMargin: 8
                                    // .hit-row: content-driven, padding 9px 12px (40px av ≈ 58px tall).
                                    Layout.preferredHeight: hitRow.implicitHeight + 18
                                    radius: theme.r
                                    color: hitHov.hovered ? theme.hover : "transparent"
                                    RowLayout {
                                        id: hitRow
                                        anchors.left: parent.left; anchors.right: parent.right
                                        anchors.verticalCenter: parent.verticalCenter
                                        anchors.leftMargin: 12; anchors.rightMargin: 12; spacing: 12
                                        Avatar {
                                            Layout.preferredWidth: 40; Layout.preferredHeight: 40; fontSize: 16
                                            name: model.m.chatName || ""; jid: model.m.chatJid || model.m.chatId || model.m.id || ""
                                            base: app.mediaBase; accent: win.avatarColor(model.m.chatJid || model.m.chatId || model.m.id || "?")
                                        }
                                        ColumnLayout {
                                            Layout.fillWidth: true; spacing: 2
                                            // .hit-top: nama (kiri, elide) + .hit-time (kanan).
                                            RowLayout {
                                                Layout.fillWidth: true; spacing: 8
                                                Text { Layout.fillWidth: true; elide: Text.ElideRight
                                                    text: model.m.chatName || ""; color: theme.text; font.pixelSize: 15; font.weight: Font.Medium }
                                                Text { text: model.m.time || ""; color: theme.text2; font.pixelSize: 12 }
                                            }
                                            Text { Layout.fillWidth: true; elide: Text.ElideRight
                                                text: model.m.text || ""; color: theme.text2; font.pixelSize: 13 }
                                        }
                                    }
                                    HoverHandler { id: hitHov }
                                    MouseArea {
                                        anchors.fill: parent
                                        onClicked: {
                                            var cid = model.m.chatJid || model.m.chatId || model.m.id || ""
                                            if (cid !== "") {
                                                win.selectedChat = { name: model.m.chatName || "", id: cid }
                                                activeView = "chats"; searchInput.text = ""; app.openChat(cid)
                                            }
                                        }
                                    }
                                }
                            }
                            // .empty-list: tak ada chat cocok & tak ada hit → 28/16 center 14px.
                            Text {
                                Layout.fillWidth: true
                                Layout.topMargin: 28; Layout.bottomMargin: 28
                                Layout.leftMargin: 16; Layout.rightMargin: 16
                                visible: searchOverlay.matched.length === 0 && hitRep.count === 0
                                horizontalAlignment: Text.AlignHCenter
                                text: i18n.t("no_match"); color: theme.text2; font.pixelSize: 14
                            }
                        }
                    }
                }
            }
        }

        // ===== Conversation (timeline + composer) =====
        ColumnLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 0
            // Header conv — hanya saat ada chat terpilih (Svelte: {#if chat} … {:else} splash).
            Rectangle {
                visible: win.selectedChat.id !== undefined
                Layout.fillWidth: true; Layout.preferredHeight: 60
                color: theme.headBg
                // .conv-head border-bottom 1px divider (bukan border 4-sisi).
                Rectangle { anchors.left: parent.left; anchors.right: parent.right; anchors.bottom: parent.bottom; height: 1; color: theme.divider }
                RowLayout {
                    anchors.left: parent.left; anchors.leftMargin: 18; anchors.right: parent.right; anchors.rightMargin: 100
                    anchors.verticalCenter: parent.verticalCenter; spacing: 13
                    Avatar {
                        visible: win.selectedChat.id !== undefined
                        Layout.preferredWidth: 40; Layout.preferredHeight: 40; fontSize: 16
                        name: win.selectedChat.name || ""; jid: win.selectedChat.id || ""
                        base: app.mediaBase; accent: win.avatarColor(win.selectedChat.name || "?")
                        group: win.selectedChat.group === true
                    }
                    ColumnLayout {
                        Layout.fillWidth: true; spacing: 0
                        // Header gated on selectedChat.id; chat.name always present (no placeholder).
                        Text { Layout.fillWidth: true; elide: Text.ElideRight
                            text: win.selectedChat.name || ""; font.pixelSize: 16; font.weight: Font.Medium; color: theme.text }
                        Text { visible: win.selectedChat.id !== undefined
                            // typing: rekam suara > "{nama} mengetik…" (grup) > "mengetik…".
                            text: app.typing
                                  ? (app.typingRec ? i18n.t("typing_rec")
                                     : ((win.selectedChat.group && app.typingName !== "") ? app.typingName + " " + i18n.t("typing") : i18n.t("typing")))
                                  // subtitle (ConvHeader.svelte): live || (group ? members||status : status) || ""
                                  : ((win.selectedChat.group ? (win.selectedChat.members || win.selectedChat.status) : win.selectedChat.status) || "")
                            color: app.typing ? theme.accent : theme.text2; font.pixelSize: 13 }
                    }
                }
                MouseArea {
                    anchors.fill: parent
                    enabled: win.selectedChat.id !== undefined
                    onClicked: {
                        if (win.selectedChat.group) app.loadDetail("GetGroupInfo", win.selectedChat.id)
                        else app.loadDetail("GetContactProfile", win.selectedChat.id)
                        detailPopup.open()
                    }
                }
                // Cari dalam chat (ala WhatsApp ConvHeader) — kiri overflow.
                Rectangle {
                    id: convSearchBtn
                    anchors.right: convOverflow.left; anchors.rightMargin: 2
                    anchors.verticalCenter: parent.verticalCenter
                    width: 40; height: 40; radius: 20
                    color: searchHov.hovered ? (theme.dark ? Qt.rgba(1, 1, 1, 0.08) : Qt.rgba(0, 0, 0, 0.06)) : "transparent"
                    visible: win.selectedChat.id !== undefined
                    Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["search"]; color: theme.railIco }
                    HoverHandler { id: searchHov }
                    MouseArea { anchors.fill: parent; onClicked: { activeView = "chats"; searchInput.forceActiveFocus() } }
                }
                // Overflow ⋮ — utilitas (media/pin/poll/profil/grup/status/dll).
                Rectangle {
                    id: convOverflow
                    anchors.right: parent.right; anchors.rightMargin: 18
                    anchors.verticalCenter: parent.verticalCenter
                    width: 40; height: 40; radius: 20
                    color: ovHov.hovered ? (theme.dark ? Qt.rgba(1, 1, 1, 0.08) : Qt.rgba(0, 0, 0, 0.06)) : "transparent"
                    Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["overflow"]; color: theme.railIco }
                    HoverHandler { id: ovHov }
                    MouseArea { anchors.fill: parent; onClicked: overflowMenu.popup() }
                }
            }
            // Timeline — pola tervalidasi (ListView + reuseItems), bubble in/out
            Rectangle {
                visible: win.selectedChat.id !== undefined
                Layout.fillWidth: true; Layout.fillHeight: true
                // Wallpaper per-chat (override) atau default tema.
                color: (win.selectedChat.id && win.wallpapers[win.selectedChat.id]) ? win.wallpapers[win.selectedChat.id] : theme.wallpaper
                // Doodle wallpaper WhatsApp (di-tile) + wash di atasnya (app.css).
                Image {
                    anchors.fill: parent; fillMode: Image.Tile; opacity: 0.5
                    source: srcDir + "/assets/doodle-" + (theme.dark ? "dark" : "light") + ".png"
                }
                Rectangle {
                    anchors.fill: parent
                    color: theme.wallpaper
                    opacity: theme.dark ? 0.84 : 0.5   // doodle-wash app.css
                }
                ListView {
                    id: timeline
                    anchors.fill: parent
                    anchors.margins: 12
                    clip: true
                    model: msgsModel
                    reuseItems: true
                    spacing: 6
                    header: ColumnLayout {
                        width: timeline.width; spacing: 6
                        Btn {
                            Layout.alignment: Qt.AlignHCenter; flat: true; text: "↑ " + i18n.t("load_older")
                            visible: timeline.count > 0
                            onClicked: app.loadOlder()
                        }
                        // Separator tanggal — .day-chip span: bg --in-bg, pad 6/12, radius 8, uppercase, ls .3.
                        Rectangle {
                            Layout.alignment: Qt.AlignHCenter; Layout.topMargin: 8; Layout.bottomMargin: 8
                            visible: timeline.count > 0
                            radius: 8; color: theme.inBg
                            implicitWidth: dlbl.implicitWidth + 24; implicitHeight: 26
                            Text { id: dlbl; anchors.centerIn: parent; text: "HARI INI"; color: theme.text2
                                font.pixelSize: 13; font.weight: Font.Medium; font.letterSpacing: 0.3 }
                        }
                        // Pembatas "belum dibaca" — .unread-divider span: pill TERPUSAT, color text2 (bukan accent).
                        Rectangle {
                            Layout.alignment: Qt.AlignHCenter; Layout.topMargin: 6; Layout.bottomMargin: 6
                            visible: (win.selectedChat.badge || 0) > 0
                            radius: 8; color: theme.inBg
                            implicitWidth: udlbl.implicitWidth + 28; implicitHeight: 26
                            Text { id: udlbl; anchors.centerIn: parent; color: theme.text2
                                font.pixelSize: 12; font.weight: Font.Medium; font.letterSpacing: 0.3
                                text: (win.selectedChat.badge || 0) + " PESAN BELUM DIBACA" }
                        }
                    }
                    delegate: Item {
                        width: timeline.width
                        implicitHeight: bubble.implicitHeight + 4
                        property bool out: (model.m.dir === "out")
                        Rectangle {
                            id: bubble
                            // Stiker + media (foto/video/gif): tanpa kartu bubble.
                            // app.css .bubble.media { background:transparent; padding:0 } → gambar flat,
                            // reply/sender jadi pill di atas, caption/waktu jadi pill di bawah.
                            property bool bare: model.m.type === "sticker" || content.media
                            // .bubble.voice overrides padding to 8px 10px (10px horizontal).
                            property bool voice: model.m.type === "voice"
                            x: parent.out ? parent.width - width - 4 : 4
                            // .bubble: padding 8px 13px → +26 lebar (13×2), +16 tinggi (8×2); voice 10×2.
                            width: content.implicitWidth + (bare ? 0 : (voice ? 20 : 26))
                            implicitHeight: content.implicitHeight + (bare ? 0 : 16)
                            // Tail ala WhatsApp (app.css): radius --r-lg (18); sudut atas dekat pengirim 6px.
                            radius: bare ? 0 : theme.rLg
                            topLeftRadius: bare ? 0 : (parent.out ? theme.rLg : 6)
                            topRightRadius: bare ? 0 : (parent.out ? 6 : theme.rLg)
                            color: bare ? "transparent" : (parent.out ? theme.outBg : theme.inBg)
                            border.color: bare ? "transparent" : theme.line
                            ColumnLayout {
                                id: content
                                property var pmsg: model.m // tangkap pesan (hindari shadowing Repeater)
                                // Media (foto/video/gif): bubble transparan padding:0 (Discord-style).
                                property bool media: ["image", "video", "gif"].indexOf(model.m.type) >= 0
                                anchors.left: parent.left
                                anchors.top: parent.top
                                // .bubble padding 8px 13px; bare (stiker/media) 0; voice 8px 10px.
                                anchors.leftMargin: bubble.bare ? 0 : (bubble.voice ? 10 : 13); anchors.rightMargin: bubble.bare ? 0 : (bubble.voice ? 10 : 13)
                                anchors.topMargin: bubble.bare ? 0 : 8; anchors.bottomMargin: bubble.bare ? 0 : 8
                                spacing: 3
                                // Nama pengirim (grup, pesan masuk) — warna per-pengirim.
                                // Media: nama masuk pill .head (lihat di bawah) → sembunyikan plain di sini.
                                Text {
                                    visible: !content.media && win.selectedChat.group === true && content.pmsg.dir === "in" && (content.pmsg.sender || "") !== ""
                                    text: content.pmsg.sender || ""
                                    color: win.avatarColor(content.pmsg.sender || ""); font.pixelSize: 13; font.weight: Font.DemiBold
                                }
                                // Media HEAD pill (.bubble.media .head): nama + kutipan balasan dalam
                                // satu pill bg in/out di ATAS foto flat (app.css). Hanya saat media.
                                Rectangle {
                                    readonly property bool hasSender: win.selectedChat.group === true && content.pmsg.dir === "in" && (content.pmsg.sender || "") !== ""
                                    readonly property bool hasQuote: (content.pmsg.quoteId || "") !== ""
                                    visible: content.media && (hasSender || hasQuote)
                                    Layout.bottomMargin: 2
                                    implicitWidth: Math.min(headCol.implicitWidth + 20, 360)
                                    implicitHeight: headCol.implicitHeight + 10
                                    radius: 11; color: content.pmsg.dir === "out" ? theme.outBg : theme.inBg
                                    border.width: 1; border.color: theme.line
                                    ColumnLayout {
                                        id: headCol
                                        anchors.left: parent.left; anchors.right: parent.right
                                        anchors.verticalCenter: parent.verticalCenter
                                        anchors.leftMargin: 10; anchors.rightMargin: 10; spacing: 2
                                        Text {
                                            visible: parent.parent.hasSender
                                            text: content.pmsg.sender || ""
                                            color: win.avatarColor(content.pmsg.sender || ""); font.pixelSize: 13; font.weight: Font.DemiBold
                                        }
                                        // .head .quote: bar kiri 3px, tanpa bg, padding 1px 0 1px 8px.
                                        RowLayout {
                                            visible: parent.parent.hasQuote; spacing: 8
                                            Rectangle { Layout.preferredWidth: 3; Layout.fillHeight: true; color: theme.quoteBar }
                                            ColumnLayout {
                                                Layout.fillWidth: true; spacing: 1
                                                Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1
                                                    text: content.pmsg.quoteName || ""; color: theme.quoteBar; font.pixelSize: 13; font.weight: Font.DemiBold }
                                                Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1
                                                    text: content.pmsg.quoteText || ""; color: theme.text2; font.pixelSize: 13 }
                                            }
                                        }
                                    }
                                }
                                // Blok kutipan balasan (bar warna + nama + teks) — NON-media.
                                // .quote: bar 4px --quote-bar, bg --quote-bg, radius 4, padding 5/9, mb 5.
                                Rectangle {
                                    visible: !content.media && (content.pmsg.quoteId || "") !== ""
                                    Layout.fillWidth: true; Layout.bottomMargin: 5; radius: 4
                                    color: theme.quoteBg
                                    implicitHeight: qcol.implicitHeight + 10
                                    Rectangle { anchors.left: parent.left; anchors.top: parent.top; anchors.bottom: parent.bottom; width: 4; color: theme.quoteBar }
                                    ColumnLayout {
                                        id: qcol
                                        anchors.left: parent.left; anchors.leftMargin: 13; anchors.right: parent.right; anchors.rightMargin: 9
                                        anchors.verticalCenter: parent.verticalCenter; spacing: 1
                                        Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1
                                            text: content.pmsg.quoteName || ""; color: theme.quoteBar; font.pixelSize: 13; font.weight: Font.DemiBold }
                                        Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1
                                            text: content.pmsg.quoteText || ""; color: theme.text2; font.pixelSize: 13 }
                                    }
                                }
                                // Document → .doc-card: ikon-box + (nama + meta) + unduh.
                                RowLayout {
                                    visible: model.m.type === "document"
                                    Layout.minimumWidth: 200
                                    Layout.topMargin: 6; Layout.bottomMargin: 6
                                    Layout.leftMargin: 2; Layout.rightMargin: 2
                                    spacing: 11
                                    // .doc-ico: 40×40, radius 9, bg accent 16%, stroke accent.
                                    Rectangle {
                                        Layout.preferredWidth: 40; Layout.preferredHeight: 40
                                        radius: 9
                                        color: Qt.rgba(theme.accent.r, theme.accent.g, theme.accent.b, 0.16)
                                        Icon {
                                            anchors.centerIn: parent; width: 22; height: 22
                                            svg: win.ico["docfile"]; color: theme.accent
                                        }
                                    }
                                    // .doc-card markup (Bubble.svelte): doc-ico + doc-name + doc-dl only (no meta sub-line).
                                    Text {
                                        Layout.fillWidth: true
                                        text: model.m.text || i18n.t("a_document")
                                        color: theme.text; font.pixelSize: 14
                                        elide: Text.ElideRight; maximumLineCount: 1
                                    }
                                    // .doc-dl: ikon unduh 20×20 (text2; accent saat hover).
                                    Icon {
                                        Layout.preferredWidth: 20; Layout.preferredHeight: 20
                                        Layout.alignment: Qt.AlignVCenter
                                        svg: win.ico["download"]
                                        color: docDlMa.containsMouse ? theme.accent : theme.text2
                                        MouseArea { id: docDlMa; anchors.fill: parent; hoverEnabled: true
                                            cursorShape: Qt.PointingHandCursor }
                                    }
                                }
                                // Stiker: kotak 160 (.media-box.sticker) — gambar /media/<id> atau placeholder.
                                Item {
                                    visible: model.m.type === "sticker"
                                    Layout.preferredWidth: 160; Layout.preferredHeight: 160
                                    property bool ok: stk.status === Image.Ready && stk.sourceSize.width > 2
                                    Image {
                                        id: stk; anchors.fill: parent; fillMode: Image.PreserveAspectFit; visible: parent.ok
                                        source: app.mediaBase ? (app.mediaBase + "/media/" + (content.pmsg.id || "")) : ""
                                    }
                                    ColumnLayout {
                                        anchors.centerIn: parent; spacing: 8; visible: !parent.ok
                                        Icon { Layout.alignment: Qt.AlignHCenter; width: 46; height: 46; svg: win.ico["sticker"]; color: theme.text2 }
                                        Text { Layout.alignment: Qt.AlignHCenter; text: i18n.t("t_sticker"); color: theme.text2; font.pixelSize: 12; font.weight: Font.Medium }
                                    }
                                }
                                // Polling: pertanyaan + opsi (klik = vote → VotePoll). app.css .poll-card.
                                ColumnLayout {
                                    visible: model.m.type === "poll"
                                    spacing: 6; Layout.minimumWidth: 230
                                    // .poll-q: ikon-list (stroke accent) + pertanyaan (600).
                                    RowLayout {
                                        spacing: 7
                                        Icon { Layout.preferredWidth: 17; Layout.preferredHeight: 17; Layout.alignment: Qt.AlignTop
                                            svg: win.ico["pollq"]; color: theme.accent }
                                        Text { text: content.pmsg.text || ""; color: theme.text; font.pixelSize: 15; font.weight: Font.DemiBold
                                            wrapMode: Text.WordWrap; Layout.fillWidth: true; Layout.maximumWidth: timeline.width * 0.5 }
                                    }
                                    // .poll-opt: kotak border + radio bulat + teks + jumlah.
                                    Repeater {
                                        model: { try { return JSON.parse(content.pmsg.thumb || "[]") } catch (e) { return [] } }
                                        delegate: Rectangle {
                                            id: pollOpt
                                            Layout.fillWidth: true; Layout.minimumWidth: 214; implicitHeight: 38
                                            radius: 10; color: theme.bg; border.width: 1; border.color: pollHov.hovered ? theme.accent : theme.line
                                            clip: true   // .poll-opt overflow:hidden agar fill ikut radius
                                            // .poll-bar: fill accent 16% di belakang opsi, lebar ~ rasio suara.
                                            // Mock: opsi = string tanpa hitungan → share 0 → lebar 0 (benar).
                                            property real share: 0
                                            Rectangle {
                                                anchors.left: parent.left; anchors.top: parent.top; anchors.bottom: parent.bottom
                                                width: parent.width * pollOpt.share
                                                color: Qt.rgba(theme.accent.r, theme.accent.g, theme.accent.b, 0.16)
                                                Behavior on width { NumberAnimation { duration: 300 } }
                                            }
                                            RowLayout {
                                                anchors.fill: parent; anchors.leftMargin: 11; anchors.rightMargin: 11; spacing: 9
                                                Rectangle { Layout.preferredWidth: 16; Layout.preferredHeight: 16; radius: 8
                                                    color: "transparent"; border.width: 2; border.color: theme.text2 }
                                                Text { Layout.fillWidth: true; text: modelData; color: theme.text; font.pixelSize: 14; elide: Text.ElideRight }
                                                Text { text: "0"; color: theme.text2; font.pixelSize: 12; font.weight: Font.DemiBold }
                                            }
                                            HoverHandler { id: pollHov }
                                            MouseArea { anchors.fill: parent
                                                onClicked: app.act("VotePoll", [win.selectedChat.id, content.pmsg.senderId || "", content.pmsg.id, [modelData]]) }
                                        }
                                    }
                                    // .poll-note: total suara.
                                    Text { text: "0 " + i18n.t("poll_votes_n"); color: theme.text2; font.pixelSize: 12 }
                                }
                                // Gambar/video/GIF: thumbnail bila ada, else placeholder (.img-ph).
                                // .media-box.card: flat (transparan), rounded 14, RATA-KIRI. Tanpa kartu/scrim.
                                Rectangle {
                                    visible: content.media
                                    Layout.preferredWidth: 220; Layout.preferredHeight: 160
                                    radius: 14; clip: true; color: "transparent"
                                    property bool hasMedia: imgM.status === Image.Ready && imgM.sourceSize.width > 2
                                    Image {
                                        id: imgM; anchors.fill: parent; fillMode: Image.PreserveAspectCrop
                                        source: (content.pmsg.thumb || "").indexOf("data:") === 0 ? content.pmsg.thumb
                                                : (model.m.type === "gif" && app.mediaBase ? app.mediaBase + "/media/" + (content.pmsg.id || "") : "")
                                        visible: parent.hasMedia
                                    }
                                    // Badge "GIF" pojok kiri-bawah (ala WhatsApp).
                                    Rectangle {
                                        visible: model.m.type === "gif"
                                        anchors.left: parent.left; anchors.bottom: parent.bottom; anchors.margins: 7
                                        width: gifLbl.implicitWidth + 10; height: 18; radius: 4; color: "#00000077"
                                        Text { id: gifLbl; anchors.centerIn: parent; text: "GIF"; color: "white"; font.pixelSize: 11; font.weight: Font.Bold }
                                    }
                                    // Placeholder media belum diunduh: lingkaran download + label.
                                    ColumnLayout {
                                        anchors.centerIn: parent; spacing: 8; visible: !parent.hasMedia
                                        Rectangle {
                                            Layout.alignment: Qt.AlignHCenter; width: 46; height: 46; radius: 23
                                            color: theme.dark ? Qt.rgba(1, 1, 1, 0.12) : Qt.rgba(0, 0, 0, 0.06)
                                            Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["download"]; color: theme.text2 }
                                        }
                                        Text {
                                            Layout.alignment: Qt.AlignHCenter; color: theme.text2; font.pixelSize: 12; font.weight: Font.Medium
                                            text: model.m.type === "video" ? i18n.t("t_video") : (model.m.type === "gif" ? "GIF" : i18n.t("t_photo"))
                                        }
                                    }
                                    // Play badge video (saat ada thumbnail).
                                    Rectangle {
                                        visible: model.m.type === "video" && parent.hasMedia
                                        anchors.centerIn: parent; width: 54; height: 54; radius: 27; color: "#00000066"
                                        Text { anchors.centerIn: parent; text: "▶"; color: "white"; font.pixelSize: 24 }
                                    }
                                }
                                // Voice note — play + waveform + durasi (app.css .play/.wave/.vtime).
                                RowLayout {
                                    // .bubble.voice min-width:230px.
                                    visible: model.m.type === "voice"; spacing: 9; Layout.minimumWidth: 230
                                    Rectangle {
                                        // .play: background transparent, no :hover rule (no hover-fill).
                                        Layout.preferredWidth: 34; Layout.preferredHeight: 34; radius: 17
                                        color: "transparent"
                                        Icon { anchors.centerIn: parent; width: 26; height: 26; fill: "currentColor"
                                            svg: win.ico["play"]; color: theme.text2 }
                                    }
                                    // Waveform: 22 bar tinggi pola tetap (deterministik). .wave { flex:1 } → isi sisa.
                                    RowLayout {
                                        Layout.fillWidth: true; Layout.preferredHeight: 26; spacing: 3
                                        Repeater {
                                            model: 22
                                            delegate: Rectangle {
                                                Layout.fillWidth: true; Layout.alignment: Qt.AlignVCenter
                                                Layout.preferredHeight: 26 * win.barH(index)
                                                radius: 2; color: theme.text2; opacity: 0.55
                                            }
                                        }
                                    }
                                    Text { text: content.pmsg.text || ""; color: theme.text2; font.pixelSize: 12 }
                                }
                                // Kartu kontak (.ctc-card): avatar + nama + nomor + Salin.
                                RowLayout {
                                    // .ctc-card padding: 8px 4px.
                                    visible: model.m.type === "contact"; spacing: 11; Layout.minimumWidth: 200
                                    Layout.topMargin: 8; Layout.bottomMargin: 8; Layout.leftMargin: 4; Layout.rightMargin: 4
                                    Rectangle {
                                        Layout.preferredWidth: 40; Layout.preferredHeight: 40; radius: 20; color: theme.accent
                                        Text { anchors.centerIn: parent; color: "#ffffff"; font.pixelSize: 18; font.weight: Font.DemiBold
                                            text: (content.pmsg.text || "?").replace(/^👤\s*/, "").charAt(0).toUpperCase() }
                                    }
                                    ColumnLayout {
                                        Layout.fillWidth: true; spacing: 0
                                        Text { Layout.fillWidth: true; elide: Text.ElideRight; color: theme.text; font.pixelSize: 15; font.weight: Font.DemiBold
                                            text: (content.pmsg.text || "").replace(/^👤\s*/, "") }
                                        Text { visible: (content.pmsg.thumb || "") !== ""; text: content.pmsg.thumb || ""; color: theme.text2; font.pixelSize: 12 }
                                    }
                                    Text {
                                        visible: (content.pmsg.thumb || "") !== ""; text: i18n.t("copy"); color: theme.accent
                                        font.pixelSize: 13; font.weight: Font.DemiBold
                                        leftPadding: 8; rightPadding: 8; topPadding: 4; bottomPadding: 4
                                    }
                                }
                                // Kartu lokasi (.loc-card): peta (placeholder bg2) + label pin.
                                Rectangle {
                                    visible: model.m.type === "location"
                                    Layout.preferredWidth: 280; radius: 12; clip: true; color: theme.bg2
                                    implicitHeight: locCol.implicitHeight
                                    ColumnLayout {
                                        id: locCol; anchors.fill: parent; spacing: 0
                                        Rectangle {
                                            Layout.fillWidth: true; Layout.preferredHeight: 140; color: theme.bg2
                                            Icon { anchors.centerIn: parent; width: 32; height: 32; svg: win.ico["locpin"]; color: theme.accent }
                                        }
                                        RowLayout {
                                            Layout.fillWidth: true; Layout.leftMargin: 10; Layout.rightMargin: 10; Layout.topMargin: 8; Layout.bottomMargin: 8; spacing: 6
                                            Icon { Layout.preferredWidth: 18; Layout.preferredHeight: 18; svg: win.ico["locpin"]; color: theme.accent }
                                            Text { Layout.fillWidth: true; elide: Text.ElideRight; text: content.pmsg.text || ""; color: theme.text; font.pixelSize: 14 }
                                        }
                                    }
                                }
                                // Teks biasa (NON-media). Caption media pakai pill .mcap di bawah.
                                Text {
                                    visible: !content.media && ["document", "sticker", "gif", "poll", "voice", "contact", "location"].indexOf(model.m.type) < 0
                                    text: model.m.text || ""
                                    wrapMode: Text.WordWrap; color: theme.text; font.pixelSize: 15
                                    lineHeight: 1.4; lineHeightMode: Text.ProportionalHeight  // .bubble line-height 1.4
                                    Layout.maximumWidth: Math.min(timeline.width * 0.66, 560)
                                }
                                // Media .mcap: caption + waktu/ticks dlm SATU pill bg in/out di bawah foto.
                                Rectangle {
                                    visible: content.media && (model.m.text || "") !== ""
                                    Layout.topMargin: 2
                                    implicitWidth: Math.min(mcapCol.implicitWidth + 22, 360)
                                    implicitHeight: mcapCol.implicitHeight + 12
                                    radius: 11; color: content.pmsg.dir === "out" ? theme.outBg : theme.inBg
                                    border.width: 1; border.color: theme.line
                                    ColumnLayout {
                                        id: mcapCol
                                        anchors.left: parent.left; anchors.right: parent.right
                                        anchors.verticalCenter: parent.verticalCenter
                                        anchors.leftMargin: 11; anchors.rightMargin: 11; spacing: 1
                                        Text {
                                            Layout.fillWidth: true; Layout.maximumWidth: 320
                                            text: model.m.text || ""; wrapMode: Text.WordWrap; color: theme.text; font.pixelSize: 15
                                            lineHeight: 1.4; lineHeightMode: Text.ProportionalHeight
                                        }
                                        RowLayout {
                                            Layout.alignment: Qt.AlignRight; spacing: 3
                                            Text { text: model.m.time || ""; color: theme.text2; font.pixelSize: 11 }
                                            Icon {
                                                visible: content.pmsg.dir === "out"
                                                vbox: "0 0 18 14"; width: 16; height: 12
                                                svg: win.ico["checks"]; color: content.pmsg.status === "read" ? theme.tick : theme.text2
                                            }
                                        }
                                    }
                                }
                                // Media .mtime: TANPA caption → pill waktu saja, rata kanan, bg in/out.
                                Rectangle {
                                    visible: content.media && (model.m.text || "") === ""
                                    Layout.topMargin: 2; Layout.alignment: Qt.AlignRight
                                    implicitWidth: mtimeRow.implicitWidth + 16; implicitHeight: 22
                                    radius: 10; color: content.pmsg.dir === "out" ? theme.outBg : theme.inBg
                                    border.width: 1; border.color: theme.line
                                    RowLayout {
                                        id: mtimeRow; anchors.centerIn: parent; spacing: 3
                                        Text { text: model.m.time || ""; color: theme.text2; font.pixelSize: 11 }
                                        Icon {
                                            visible: content.pmsg.dir === "out"
                                            vbox: "0 0 18 14"; width: 16; height: 12
                                            svg: win.ico["checks"]; color: content.pmsg.status === "read" ? theme.tick : theme.text2
                                        }
                                    }
                                }
                                // Waktu + ticks pojok kanan-bawah bubble (NON-media, ala WhatsApp).
                                RowLayout {
                                    visible: !content.media
                                    Layout.alignment: Qt.AlignRight; spacing: 4
                                    Text { text: model.m.time || ""; color: theme.text2; font.pixelSize: 11 }
                                    Icon {
                                        visible: content.pmsg.dir === "out"
                                        vbox: "0 0 18 14"; width: 16; height: 12
                                        svg: win.ico["checks"]
                                        color: content.pmsg.status === "read" ? theme.tick : theme.text2
                                    }
                                }
                                // Chip reaksi (emoji + jumlah). app.css .reactions: in→kiri, out→kanan.
                                Flow {
                                    visible: content.pmsg.reactions && content.pmsg.reactions.length > 0
                                    spacing: 4
                                    Layout.alignment: content.pmsg.dir === "out" ? Qt.AlignRight : Qt.AlignLeft
                                    Repeater {
                                        model: content.pmsg.reactions || []
                                        delegate: Rectangle {
                                            radius: 11; color: theme.bg2; border.width: 1; border.color: theme.line
                                            implicitWidth: rc.implicitWidth + 12; implicitHeight: 22
                                            Text { id: rc; anchors.centerIn: parent; text: modelData.emoji + " " + modelData.count; font.pixelSize: 12; color: theme.text }
                                        }
                                    }
                                }
                            }
                            // Klik-kanan → menu aksi; klik-kiri media → lightbox.
                            MouseArea {
                                anchors.fill: parent
                                acceptedButtons: Qt.RightButton | Qt.LeftButton
                                onClicked: (mouse) => {
                                    if (mouse.button === Qt.RightButton) {
                                        win.ctxMsg = model.m
                                        var gp = mapToItem(null, mouse.x, mouse.y)
                                        win.reactX = gp.x; win.reactY = gp.y
                                        msgMenu.popup()
                                    }
                                    else if (["image", "sticker", "gif"].indexOf(model.m.type) >= 0) {
                                        win.lightboxCaption = model.m.caption || model.m.text || ""
                                        win.lightboxSrc = (app.mediaBase || "") + "/media/" + (win.selectedChat.id || "") + "/" + model.m.id
                                    }
                                }
                            }
                        }
                    }
                }
            }
            // Banner balas (muncul saat membalas pesan)
            Rectangle {
                Layout.fillWidth: true
                Layout.preferredHeight: (win.replyTo !== null && win.replyTo !== undefined) ? 44 : 0
                visible: win.replyTo !== null && win.replyTo !== undefined
                color: theme.bg2
                RowLayout {
                    anchors.fill: parent; anchors.leftMargin: 12; anchors.rightMargin: 12; spacing: 10
                    Rectangle { width: 3; Layout.preferredHeight: 28; color: theme.accent }
                    ColumnLayout {
                        Layout.fillWidth: true; spacing: 0
                        Text { text: i18n.t("replying"); color: theme.accent; font.pixelSize: 11 }
                        Text { Layout.fillWidth: true; elide: Text.ElideRight
                            text: (win.replyTo ? (win.replyTo.text || "[media]") : ""); color: theme.text2; font.pixelSize: 12 }
                    }
                    Text {
                        text: "✕"; color: theme.text2; font.pixelSize: 16
                        MouseArea { anchors.fill: parent; onClicked: win.replyTo = null }
                    }
                }
            }
            // Composer (.composer: bg head-bg, min-height 64, pad 9/16, gap 10, border-top divider)
            Rectangle {
                id: composer
                visible: win.selectedChat.id !== undefined
                // Tinggi tumbuh dgn isi composer (textarea auto-grow s/d 130): tinggi
                // area input + padding vertikal 9+9; min 64 (.composer min-height).
                property int inputH: 39
                Layout.fillWidth: true; Layout.minimumHeight: 64
                Layout.preferredHeight: Math.max(64, inputH + 18)
                color: theme.headBg
                // Status rekam suara — VISUAL-ONLY (tak ada backend capture). Mic → rec-bar + mic merah.
                property bool recording: false
                property int recElapsed: 0
                Timer {
                    id: recTimer; interval: 1000; repeat: true; running: composer.recording
                    onTriggered: composer.recElapsed += 1
                }
                function recLabel() {
                    // MM:SS with zero-padded minutes (Composer.svelte recLabel).
                    var m = Math.floor(composer.recElapsed / 60), s = composer.recElapsed % 60
                    return (m < 10 ? "0" + m : m) + ":" + (s < 10 ? "0" + s : s)
                }
                Rectangle { anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; height: 1; color: theme.divider }
                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 16; anchors.rightMargin: 16; anchors.topMargin: 9; anchors.bottomMargin: 9
                    spacing: 10
                    // Emoji (placeholder picker) — kiri, ala Composer.svelte.
                    Rectangle {
                        // .icon-btn:hover → overlay translucent (light rgba(0,0,0,.06) / dark rgba(255,255,255,.08)).
                        width: 40; height: 40; radius: 20
                        color: emojiHov.hovered ? (theme.dark ? Qt.rgba(1,1,1,0.08) : Qt.rgba(0,0,0,0.06)) : "transparent"
                        Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["emoji"]; color: theme.railIco }
                        HoverHandler { id: emojiHov }
                        // Tombol emoji → buka grid emoji penuh (reactionPopup) dlm mode sisip,
                        // bukan menu 10-glyph. Posisikan di atas tombol (.emoji-panel bottom:68 left:8).
                        MouseArea { anchors.fill: parent; onClicked: {
                            var p = mapToItem(null, 0, 0)
                            win.reactX = p.x
                            win.reactY = Math.max(8, p.y - reactionPopup.height - 6)
                            reactionPopup.insertMode = true
                            reactionPopup.open()
                        } }
                    }
                    // Lampiran (+) → menu: dokumen/stiker/gif/gambar/video/lokasi/polling/kontak/mention.
                    Rectangle {
                        width: 40; height: 40; radius: 20
                        color: attachHov.hovered ? (theme.dark ? Qt.rgba(1,1,1,0.08) : Qt.rgba(0,0,0,0.06)) : "transparent"
                        Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["plus"]; color: theme.railIco }
                        HoverHandler { id: attachHov }
                        MouseArea { anchors.fill: parent; onClicked: attachMenu.popup() }
                    }
                    // .rec-bar: bar rekam suara (titik merah + timer + slide-to-cancel + delete). Visual-only.
                    Rectangle {
                        visible: composer.recording
                        Layout.fillWidth: true; Layout.fillHeight: true
                        radius: 22; color: theme.searchBg
                        RowLayout {
                            anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 16; spacing: 10
                            // .rec-dot: titik merah berkedip.
                            Rectangle {
                                Layout.preferredWidth: 11; Layout.preferredHeight: 11; radius: 5.5; color: "#e0463e"
                                SequentialAnimation on opacity {
                                    running: composer.recording; loops: Animation.Infinite
                                    NumberAnimation { from: 1; to: 0.3; duration: 500; easing.type: Easing.InOutQuad }
                                    NumberAnimation { from: 0.3; to: 1; duration: 500; easing.type: Easing.InOutQuad }
                                }
                            }
                            // .rec-time: elapsed.
                            Text { text: composer.recLabel(); color: theme.text; font.pixelSize: 15; font.bold: true }
                            // .rec-hint: "Recording…" / slide-to-cancel.
                            Text { Layout.fillWidth: true; text: i18n.t("recording"); color: theme.text2; font.pixelSize: 13 }
                            // .rec-cancel: hapus (trash) → batal.
                            Rectangle {
                                Layout.preferredWidth: 28; Layout.preferredHeight: 28; radius: 14
                                color: recCancelHov.hovered ? theme.hover : "transparent"
                                Icon { anchors.centerIn: parent; width: 20; height: 20; svg: win.ico["trashSimple"]
                                    color: recCancelHov.hovered ? "#e0463e" : theme.text2 }
                                HoverHandler { id: recCancelHov }
                                MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                                    onClicked: { composer.recording = false; composer.recElapsed = 0 } }
                            }
                        }
                    }
                    // .composer .input: pill searchBg radius 22, pad 9/16. Berisi
                    // <textarea> multi-baris auto-grow (line-height 1.4, max-height 130,
                    // overflow-y auto); Enter=kirim, Shift+Enter=baris baru.
                    Rectangle {
                        id: inputPill
                        visible: !composer.recording
                        Layout.fillWidth: true; Layout.alignment: Qt.AlignVCenter
                        Layout.preferredHeight: composer.inputH
                        radius: 22; color: theme.searchBg
                        function send() {
                            if (composerInput.text.trim() === "") return
                            if (win.replyTo && win.replyTo.id)
                                app.replyText(win.replyTo.id, win.replyTo.senderId || "", win.replyTo.text || "", composerInput.text)
                            else
                                app.sendText(composerInput.text)
                            composerInput.text = ""
                            win.replyTo = null
                        }
                        ScrollView {
                            anchors.fill: parent
                            anchors.leftMargin: 16; anchors.rightMargin: 16
                            anchors.topMargin: 9; anchors.bottomMargin: 9
                            clip: true
                            ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
                            TextArea {
                                id: composerInput
                                width: inputPill.width - 32
                                background: null
                                wrapMode: TextArea.Wrap
                                verticalAlignment: TextArea.AlignVCenter
                                color: theme.text; font.pixelSize: 15
                                // line-height 1.4 → spasi antar-baris.
                                topPadding: 0; bottomPadding: 0; leftPadding: 0; rightPadding: 0
                                // textarea auto-grow: clamp(contentHeight, 28, 130) → set inputH (+18 pad pill).
                                onContentHeightChanged: composer.inputH = Math.max(39, Math.min(Math.ceil(contentHeight) + 18, 130))
                                onTextChanged: app.sendTyping(text.length > 0)
                                // Enter=kirim (tanpa Shift); Shift+Enter=baris baru (default TextArea).
                                Keys.onReturnPressed: function(e) {
                                    if (e.modifiers & Qt.ShiftModifier) { e.accepted = false }
                                    else { e.accepted = true; inputPill.send() }
                                }
                                Keys.onEnterPressed: function(e) {
                                    if (e.modifiers & Qt.ShiftModifier) { e.accepted = false }
                                    else { e.accepted = true; inputPill.send() }
                                }
                                Text {
                                    visible: composerInput.text === ""
                                    anchors.verticalCenter: parent.verticalCenter
                                    anchors.left: parent.left
                                    text: i18n.t("type_message"); color: theme.text2; font.pixelSize: 15
                                }
                            }
                        }
                    }
                    // .icon-btn schedule (jam): muncul saat input ada teks (Composer.svelte typing && !editDraft),
                    // di antara input dan tombol mic/send.
                    Rectangle {
                        id: scheduleBtn
                        visible: composerInput.text.trim() !== "" && !composer.recording
                        width: 40; height: 40; radius: 20
                        color: schedHov.hovered ? (theme.dark ? Qt.rgba(1,1,1,0.08) : Qt.rgba(0,0,0,0.06)) : "transparent"
                        Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["schedule"]; color: theme.railIco }
                        HoverHandler { id: schedHov }
                        MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: {} }
                    }
                    // Kosong → mic; ada teks → kirim. Keduanya .icon-btn transparan (text2),
                    // tanpa lingkaran accent (app.css: hanya state rekam yg ber-fill).
                    Rectangle {
                        id: sendBtn
                        property bool hasText: composerInput.text.trim() !== ""
                        width: 40; height: 40; radius: 20
                        // .icon-btn.mic.rec: bg merah #e0392b (hover #c0392b), ikon #fff saat merekam.
                        color: composer.recording ? (sendHov.hovered ? "#c0392b" : "#e0392b")
                                                   : (sendHov.hovered ? (theme.dark ? Qt.rgba(1,1,1,0.08) : Qt.rgba(0,0,0,0.06)) : "transparent")
                        Icon { anchors.centerIn: parent; width: 22; height: 22
                            // teks → kirim; rekam → stop (kirim voice); kosong → mic.
                            svg: sendBtn.hasText ? win.ico["send"]
                               : composer.recording ? win.ico["stop"]
                               : win.ico["mic"]
                            color: composer.recording ? "#ffffff" : theme.railIco }
                        HoverHandler { id: sendHov }
                        MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: {
                            if (sendBtn.hasText) { inputPill.send(); return }
                            if (composer.recording) { composer.recording = false; composer.recElapsed = 0 }  // stop → "kirim" (visual)
                            else { composer.recElapsed = 0; composer.recording = true }  // mic → mulai rekam (visual)
                        } }
                    }
                }
            }
            // Splash kosong (.conv-splash) — saat belum ada chat terpilih ({:else} di Conversation.svelte).
            Item {
                visible: win.selectedChat.id === undefined
                Layout.fillWidth: true; Layout.fillHeight: true
                ColumnLayout {
                    anchors.centerIn: parent
                    width: Math.min(parent.width - 48, 420)
                    spacing: 0
                    // .splash-logo: 200×200 lingkaran head-bg, ikon 96×96 text2 opacity .45, margin-bottom 20.
                    Rectangle {
                        Layout.alignment: Qt.AlignHCenter
                        Layout.preferredWidth: 200; Layout.preferredHeight: 200
                        Layout.bottomMargin: 20
                        radius: 100; color: theme.headBg
                        Icon {
                            anchors.centerIn: parent; width: 96; height: 96
                            svg: win.ico["chats"]; color: theme.text2; opacity: 0.45
                        }
                    }
                    // h2: font 28, weight Light, text, margin-bottom 8.
                    Text {
                        Layout.alignment: Qt.AlignHCenter
                        Layout.bottomMargin: 8
                        text: i18n.t("splash_title")
                        font.pixelSize: 28; font.weight: Font.Light; color: theme.text
                        horizontalAlignment: Text.AlignHCenter
                    }
                    // p: text2, font 14, line-height 1.5, max-width 420, centered, wrap.
                    Text {
                        Layout.alignment: Qt.AlignHCenter
                        Layout.maximumWidth: 420
                        text: i18n.t("splash_sub")
                        color: theme.text2; font.pixelSize: 14
                        lineHeight: 1.5; lineHeightMode: Text.ProportionalHeight
                        wrapMode: Text.WordWrap; horizontalAlignment: Text.AlignHCenter
                    }
                    // .splash-enc: margin-top 34, gap 6, lock 14×14 + teks, semuanya text2 font 13.
                    RowLayout {
                        Layout.alignment: Qt.AlignHCenter
                        Layout.topMargin: 34
                        spacing: 6
                        Icon { Layout.preferredWidth: 14; Layout.preferredHeight: 14; svg: win.ico["lock"]; color: theme.text2 }
                        Text { text: i18n.t("splash_enc"); color: theme.text2; font.pixelSize: 13 }
                    }
                }
            }
        }
    }

    // === Picker stiker (shell ala StickerPicker.svelte: tab + cari + grid) ===
    Popup {
        id: stickerPopup
        width: 520; height: 400
        x: win.width - width - 16
        y: win.height - height - 70
        padding: 10
        property string tab: "online"   // online|recents|pack|create (StickerPicker.svelte default)
        property string cat: "trending" // kategori aktif (.gif-cat.on mengikuti query); "★"=favView
        background: Rectangle { color: theme.bg; radius: 14; border.color: theme.line }
        ColumnLayout {
            anchors.fill: parent; spacing: 10
            // .stk-tabs: 4 tab (aktif = accent).
            RowLayout {
                Layout.fillWidth: true; spacing: 6
                Repeater {
                    model: [{ k: "online", t: i18n.t("stk_online") }, { k: "recents", t: i18n.t("stk_recents") },
                            { k: "pack", t: i18n.t("stk_pack") }, { k: "create", t: i18n.t("stk_create") }]
                    delegate: Rectangle {
                        Layout.fillWidth: true; implicitHeight: 34; radius: 9
                        color: stickerPopup.tab === modelData.k ? theme.accent : theme.bg2
                        Text { anchors.centerIn: parent; text: modelData.t; font.pixelSize: 13; font.weight: Font.DemiBold
                            color: stickerPopup.tab === modelData.k ? "#ffffff" : theme.text2 }
                        MouseArea { anchors.fill: parent; onClicked: {
                            stickerPopup.tab = modelData.k
                            if (modelData.k === "online") app.searchOnline("SearchStickers", "", onlineStkModel) } }
                    }
                }
            }
            // .pk-searchbox + .stk-search (tab online). .stk-search { border:0 }.
            Rectangle {
                Layout.fillWidth: true; implicitHeight: 36; radius: 9; color: theme.bg2; border.width: 0
                visible: stickerPopup.tab === "online"
                RowLayout {
                    // .stk-search: plain input, padding 8px 12px, no leading icon.
                    anchors.fill: parent; anchors.leftMargin: 12; anchors.rightMargin: 12; spacing: 8
                    TextInput { id: stkSearch; Layout.fillWidth: true; color: theme.text; font.pixelSize: 13
                        verticalAlignment: TextInput.AlignVCenter; clip: true
                        onAccepted: app.searchOnline("SearchStickers", text, onlineStkModel) }
                    Text { visible: stkSearch.text === ""; text: i18n.t("search") + " stiker"; color: theme.text2; font.pixelSize: 13
                        anchors.verticalCenter: parent.verticalCenter; anchors.left: parent.left }
                }
            }
            // .gif-cats: kategori chip (tab online). .gif-cat.on mengikuti kategori terpilih (query).
            Flow {
                Layout.fillWidth: true; spacing: 5; visible: stickerPopup.tab === "online"
                Repeater {
                    // StickerPicker.svelte cats (full): trending..good night (16) + ★.
                    model: ["★", "trending", "cute", "love", "happy", "sad", "angry", "cat", "dog", "anime", "meme", "lol", "thanks", "hello", "kiss", "wow", "good night"]
                    delegate: Rectangle {
                        property bool on: stickerPopup.cat === modelData
                        height: 24; radius: 12; width: cat.implicitWidth + 20
                        color: on ? theme.accent : theme.bg2
                        Text { id: cat; anchors.centerIn: parent; text: modelData; font.pixelSize: 12
                            color: parent.on ? "#ffffff" : theme.text2 }
                        MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: {
                            stickerPopup.cat = modelData
                            if (modelData !== "★") app.searchOnline("SearchStickers", modelData === "trending" ? "" : modelData, onlineStkModel)
                        } }
                    }
                }
            }
            // .stk-grid: koleksi (tab pack) / empty (lainnya).
            GridView {
                id: stickerGrid
                Layout.fillWidth: true; Layout.fillHeight: true
                visible: stickerPopup.tab === "pack"
                // .stk-grid minmax(84,1fr) gap 6: sel mengisi 1fr (floor(width/90)) — bukan 84 fix.
                property int stkCols: Math.max(1, Math.floor(width / 90))
                cellWidth: width / stkCols; cellHeight: cellWidth; clip: true
                model: stickerPopup.tab === "pack" ? stickersModel : 0
                delegate: Item {
                    width: GridView.view.cellWidth; height: GridView.view.cellHeight
                    Rectangle {
                        anchors.fill: parent; anchors.margins: 3; radius: 10; color: theme.bg2
                        Image {
                            id: stkImg
                            anchors.fill: parent; anchors.margins: 6; fillMode: Image.PreserveAspectFit
                            source: app.mediaBase ? (app.mediaBase + "/sticker/" + model.m.hash) : ""
                            visible: status === Image.Ready
                        }
                        ColumnLayout {
                            anchors.centerIn: parent; visible: stkImg.status !== Image.Ready
                            Icon { Layout.alignment: Qt.AlignHCenter; width: 30; height: 30; svg: win.ico["sticker"]; color: theme.text2 }
                            Text { Layout.alignment: Qt.AlignHCenter; text: model.m.animated ? "animasi" : "statis"; color: theme.text2; font.pixelSize: 10 }
                        }
                        MouseArea { anchors.fill: parent; onClicked: { app.sendSticker(model.m.hash); stickerPopup.close() } }
                    }
                }
            }
            // Grid hasil online (Tenor/Stickerly) — preview URL remote.
            GridView {
                Layout.fillWidth: true; Layout.fillHeight: true
                visible: stickerPopup.tab === "online"
                property int stkCols: Math.max(1, Math.floor(width / 90))
                cellWidth: width / stkCols; cellHeight: cellWidth; clip: true
                model: stickerPopup.tab === "online" ? onlineStkModel : 0
                delegate: Item {
                    width: GridView.view.cellWidth; height: GridView.view.cellHeight
                    Rectangle {
                        anchors.fill: parent; anchors.margins: 3; radius: 10; color: theme.bg2
                        Image {
                            anchors.fill: parent; anchors.margins: 6; fillMode: Image.PreserveAspectFit
                            source: model.m.preview || ""; asynchronous: true
                        }
                        MouseArea { anchors.fill: parent; onClicked: { app.sendOnline("sticker", model.m.mp4 || model.m.preview); stickerPopup.close() } }
                    }
                }
            }
            // Empty state (tab recents).
            Text {
                Layout.fillWidth: true; Layout.fillHeight: true; visible: stickerPopup.tab === "recents"
                horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
                text: "—"; color: theme.text2; font.pixelSize: 13
            }
            // .stk-create → .stk-pick dashed dropzone (border 2px dashed line, radius 14,
            // accent + icon 30px, stk_pick label + stk_hint small).
            Item {
                Layout.fillWidth: true; Layout.fillHeight: true; visible: stickerPopup.tab === "create"
                Rectangle {
                    anchors.centerIn: parent
                    width: parent.width; height: 180; radius: 14; color: "transparent"
                    // Dashed look approximated via Canvas border.
                    Canvas {
                        anchors.fill: parent
                        onPaint: {
                            var ctx = getContext("2d"); ctx.reset()
                            ctx.strokeStyle = theme.line; ctx.lineWidth = 2; ctx.setLineDash([6, 5])
                            ctx.strokeRect(1, 1, width - 2, height - 2)
                        }
                    }
                    ColumnLayout {
                        anchors.centerIn: parent; spacing: 6
                        Icon { Layout.alignment: Qt.AlignHCenter; width: 30; height: 30; svg: win.ico["plus"]; color: theme.accent }
                        Text { Layout.alignment: Qt.AlignHCenter; text: i18n.t("stk_pick"); color: theme.text2; font.pixelSize: 13 }
                        Text { Layout.alignment: Qt.AlignHCenter; text: i18n.t("stk_hint"); color: theme.text2; font.pixelSize: 11 }
                    }
                }
            }
            // .stk-credit.
            Text { Layout.alignment: Qt.AlignHCenter; text: "Powered by Sticker.ly"; color: theme.text2; font.pixelSize: 10 }
        }
    }

    // === Picker GIF (shell ala GifPicker.svelte: tab + cari + grid) ===
    Popup {
        id: gifPopup
        width: 520; height: 400
        x: win.width - width - 16
        y: win.height - height - 70
        padding: 10
        property string tab: "online"   // GifPicker.svelte: hanya grid online (trending); default online
        property string cat: "trending" // kategori aktif (.gif-cat.on mengikuti query); "★"=favView
        background: Rectangle { color: theme.bg; radius: 14; border.color: theme.line }
        ColumnLayout {
            anchors.fill: parent; spacing: 10
            // GifPicker.svelte has no tabs — single online grid only.
            // Cari (tab online).
            Rectangle {
                Layout.fillWidth: true; implicitHeight: 36; radius: 9; color: theme.bg2; border.color: theme.line
                visible: gifPopup.tab === "online"
                RowLayout {
                    // .gif-search: plain input, padding 8px 12px, no leading icon.
                    anchors.fill: parent; anchors.leftMargin: 12; anchors.rightMargin: 12; spacing: 8
                    TextInput { id: gifSearch; Layout.fillWidth: true; color: theme.text; font.pixelSize: 13
                        verticalAlignment: TextInput.AlignVCenter; clip: true
                        onAccepted: app.searchOnline("SearchGifs", text, onlineGifModel) }
                    Text { visible: gifSearch.text === ""; text: i18n.t("search") + " GIF"; color: theme.text2; font.pixelSize: 13
                        anchors.verticalCenter: parent.verticalCenter; anchors.left: parent.left }
                }
            }
            Flow {
                Layout.fillWidth: true; spacing: 5; visible: gifPopup.tab === "online"
                Repeater {
                    // GifPicker.svelte cats (full, no 'reactions'): trending..clap (12) + ★.
                    model: ["★", "trending", "lol", "love", "sad", "wow", "ok", "thanks", "hi", "bye", "angry", "dance", "clap"]
                    delegate: Rectangle {
                        property bool on: gifPopup.cat === modelData
                        height: 24; radius: 12; width: gcat.implicitWidth + 20
                        color: on ? theme.accent : theme.bg2
                        Text { id: gcat; anchors.centerIn: parent; text: modelData; font.pixelSize: 12
                            color: parent.on ? "#ffffff" : theme.text2 }
                        MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: {
                            gifPopup.cat = modelData
                            if (modelData !== "★") app.searchOnline("SearchGifs", modelData === "trending" ? "" : modelData, onlineGifModel)
                        } }
                    }
                }
            }
            GridView {
                Layout.fillWidth: true; Layout.fillHeight: true; visible: gifPopup.tab === "saved"
                // .gif-grid minmax(150,1fr) gap 6: sel mengisi 1fr (floor(width/156)) — bukan 150 fix.
                property int gifCols: Math.max(1, Math.floor(width / 156))
                cellWidth: width / gifCols; cellHeight: cellWidth; clip: true
                model: gifPopup.tab === "saved" ? gifsModel : 0
                delegate: Item {
                    width: GridView.view.cellWidth; height: GridView.view.cellHeight
                    Rectangle {
                        anchors.fill: parent; anchors.margins: 3; radius: 8; color: theme.bg2; clip: true
                        Image {
                            id: gifImg
                            // .gif-cell img { object-fit:cover } → crop-fill, bukan fit.
                            anchors.fill: parent; fillMode: Image.PreserveAspectCrop; clip: true
                            source: app.mediaBase ? (app.mediaBase + "/savedgif/" + model.m.hash) : ""
                            visible: status === Image.Ready
                        }
                        ColumnLayout {
                            anchors.centerIn: parent; visible: gifImg.status !== Image.Ready
                            Icon { Layout.alignment: Qt.AlignHCenter; width: 30; height: 30; svg: win.ico["gifb"]; color: theme.text2 }
                            Text { Layout.alignment: Qt.AlignHCenter; text: "GIF"; color: theme.text2; font.pixelSize: 11 }
                        }
                        MouseArea { anchors.fill: parent; onClicked: { app.sendGif(model.m.hash); gifPopup.close() } }
                    }
                }
            }
            // Grid hasil online (Tenor) — preview URL remote.
            GridView {
                Layout.fillWidth: true; Layout.fillHeight: true; visible: gifPopup.tab === "online"
                property int gifCols: Math.max(1, Math.floor(width / 156))
                cellWidth: width / gifCols; cellHeight: cellWidth; clip: true
                model: gifPopup.tab === "online" ? onlineGifModel : 0
                delegate: Item {
                    width: GridView.view.cellWidth; height: GridView.view.cellHeight
                    Rectangle {
                        anchors.fill: parent; anchors.margins: 3; radius: 8; color: theme.bg2; clip: true
                        Image {
                            anchors.fill: parent; fillMode: Image.PreserveAspectCrop; clip: true
                            source: model.m.preview || ""; asynchronous: true
                        }
                        MouseArea { anchors.fill: parent; onClicked: { app.sendOnline("gif", model.m.mp4 || model.m.preview); gifPopup.close() } }
                    }
                }
            }
            Text {
                Layout.fillWidth: true; Layout.fillHeight: true; visible: gifPopup.tab === "recents"
                horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
                text: "—"; color: theme.text2; font.pixelSize: 13
            }
            Text { Layout.alignment: Qt.AlignHCenter; text: "Powered by KLIPY"; color: theme.text2; font.pixelSize: 10 }
        }
    }

    // === Menu aksi pesan (klik-kanan bubble) ===
    TMenu {
        id: msgMenu
        TMenuItem { text: "👍  " + i18n.t("m_like"); onTriggered: app.react(win.ctxMsg.id, win.ctxMsg.senderId || "", win.ctxMsg.dir === "out", "👍") }
        TMenuItem { text: "😀  " + i18n.t("m_react"); onTriggered: { reactionPopup.insertMode = false; reactionPopup.open() } }
        TMenuItem { text: "↩️  " + i18n.t("m_reply"); onTriggered: win.replyTo = win.ctxMsg }
        TMenuItem {
            text: "✏️  " + i18n.t("m_edit"); visible: win.ctxMsg.dir === "out"; height: visible ? implicitHeight : 0
            onTriggered: { editInput.text = win.ctxMsg.text || ""; editPopup.open() }
        }
        TMenuItem { text: "📌  " + i18n.t("m_pin"); onTriggered: app.pinMessage(win.ctxMsg.id, win.ctxMsg.senderId || "", win.ctxMsg.dir === "out", true) }
        TMenuItem { text: "⭐  " + i18n.t("m_star"); onTriggered: app.star(win.ctxMsg.id, win.ctxMsg.senderId || "", win.ctxMsg.dir === "out", true) }
        TMenuItem {
            text: "💾  " + i18n.t("m_save_sticker"); visible: win.ctxMsg.type === "sticker"; height: visible ? implicitHeight : 0
            onTriggered: app.saveStickerFromMsg(win.ctxMsg.id)
        }
        TMenuItem {
            text: "🎬  " + i18n.t("m_save_gif"); visible: win.ctxMsg.type === "gif"; height: visible ? implicitHeight : 0
            onTriggered: app.saveGifFromMsg(win.ctxMsg.id)
        }
        TMenuItem { text: "↪️  " + i18n.t("m_forward"); onTriggered: forwardPopup.open() }
        TMenuItem {
            text: "😀  " + i18n.t("m_reactions")
            visible: win.ctxMsg.reactions && win.ctxMsg.reactions.length > 0
            height: visible ? implicitHeight : 0
            onTriggered: reactionDetailPopup.open()
        }
        TMenuItem {
            text: "ℹ️  " + i18n.t("m_info")
            onTriggered: { app.loadDetailA("GetMessageInfo", [win.selectedChat.id || "", win.ctxMsg.id]); msgInfoPopup.open() }
        }
        TMenuItem { text: "🗑️  " + i18n.t("m_delete_all"); onTriggered: app.deleteMsg(win.ctxMsg.id, win.ctxMsg.senderId || "", win.ctxMsg.dir === "out", true) }
    }

    // === Menu kelola chat (klik-kanan baris) ===
    TMenu {
        id: chatMenu
        TMenuItem { text: "✓  " + i18n.t("c_mark_read"); onTriggered: app.markRead(win.ctxChat.id) }
        TMenuItem { text: "📌  " + (win.ctxChat.pinned ? i18n.t("c_unpin") : i18n.t("c_pin")); onTriggered: app.pinChat(win.ctxChat.id, !win.ctxChat.pinned) }
        TMenuItem { text: (win.ctxChat.muted ? "🔔  " + i18n.t("c_unmute") : "🔇  " + i18n.t("c_mute")); onTriggered: app.muteChat(win.ctxChat.id, !win.ctxChat.muted) }
        TMenuItem { text: "🗄️  " + i18n.t("c_archive"); onTriggered: app.archiveChat(win.ctxChat.id, true) }
        TMenuItem { text: "📁  " + i18n.t("folders"); onTriggered: win.openFolderPicker(win.ctxChat.id, win.ctxChat.name || "") }
        TMenuItem { text: "🗑️  " + i18n.t("c_delete"); onTriggered: app.deleteChat(win.ctxChat.id) }
    }

    // === Edit pesan ===
    Popup {
        id: editPopup
        width: 360; height: 180; modal: true; anchors.centerIn: Overlay.overlay; padding: 16
        background: Rectangle { color: theme.bg; radius: 14; border.color: theme.line }
        ColumnLayout {
            anchors.fill: parent; spacing: 12
            Text { text: i18n.t("edit_message"); color: theme.text; font.pixelSize: 16; font.bold: true }
            Rectangle {
                Layout.fillWidth: true; height: 40; radius: 8; color: theme.searchBg; border.color: theme.line
                TextInput { id: editInput; anchors.fill: parent; anchors.margins: 10; color: theme.text; font.pixelSize: 14; clip: true }
            }
            Item { Layout.fillHeight: true }
            RowLayout {
                Layout.alignment: Qt.AlignRight; spacing: 8
                Btn { text: i18n.t("cancel"); onClicked: editPopup.close() }
                Btn { accent: true; text: i18n.t("save"); onClicked: { app.editMessage(win.ctxMsg.id, editInput.text); editPopup.close() } }
            }
        }
    }

    // === Setelan (gear rail) — PANE kiri full-height (parity SettingsPane.svelte) ===
    Popup {
        id: settingsPopup
        x: 0; y: 0; width: 400; height: win.height; modal: true
        padding: 0
        // State lokal: tak ada getter binding utk retensi/disappearing → default
        // app.css/Svelte (retDays 90, defDis 0) + update saat klik.
        property int retDays: 90
        property int defDis: 0
        // State klien-saja (di Svelte = localStorage; Qt belum punya store C++).
        // Mode tema 3-arah: light/dark/system. Qt theme hanya punya `dark`;
        // "system" fallback ke light (note: tak ada deteksi OS di engine ini).
        property string themeMode: theme.dark ? "dark" : "light"
        property string translateLang: "en"
        property bool soundOn: true
        property bool virtList: false
        property bool showDeleted: false
        property bool appLock: false
        property bool dndOn: false
        property string dndFrom: "22:00"
        property string dndTo: "07:00"
        property var quickReactions: ["👍", "❤️", "😂", "😮", "😢", "🙏"]
        property int uiScale: 100
        property bool bgRun: false
        function setThemeMode(m) {
            themeMode = m
            // light/system → terang; dark → gelap (system jatuh ke light).
            theme.dark = (m === "dark")
        }
        onOpened: { app.act("GetProxy", []); app.act("GetRetention", []); app.act("GetBackgroundClose", []) }
        background: Rectangle { color: theme.sidebarBg }

        // Komponen baris .settings-item (ikon + grow + trailing slot) — reusable.
        ColumnLayout {
            anchors.fill: parent; spacing: 0

            // .pane-head (height 56, padding 0 16, bg head-bg; h2 19/600)
            Rectangle {
                Layout.fillWidth: true; Layout.preferredHeight: 56
                color: theme.headBg
                Text {
                    anchors.verticalCenter: parent.verticalCenter
                    anchors.left: parent.left; anchors.leftMargin: 16
                    text: i18n.t("settings"); color: theme.text
                    font.pixelSize: 19; font.weight: Font.DemiBold
                }
            }

            // Konten yang dapat di-scroll
            ScrollView {
                Layout.fillWidth: true; Layout.fillHeight: true; clip: true
                ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
                contentWidth: availableWidth

                ColumnLayout {
                    width: settingsPopup.width
                    spacing: 0

                    // ---- .settings-profile (gap 16, pad 18 16, border-bottom) ----
                    Rectangle {
                        Layout.fillWidth: true
                        implicitHeight: profRow.implicitHeight + 36 // 18 atas + 18 bawah
                        color: profHov.hovered ? theme.hover : "transparent"
                        Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: theme.divider }
                        HoverHandler { id: profHov }
                        RowLayout {
                            id: profRow
                            anchors.left: parent.left; anchors.right: parent.right
                            anchors.verticalCenter: parent.verticalCenter
                            anchors.leftMargin: 16; anchors.rightMargin: 16
                            spacing: 16
                            Avatar {
                                Layout.preferredWidth: 49; Layout.preferredHeight: 49
                                fontSize: 19; name: win.myName; accent: win.avatarColor(win.myName)
                            }
                            ColumnLayout {
                                Layout.fillWidth: true; spacing: 0
                                // .sp-name / .sp-about → bound ke profil "me" yg sama dgn avatar rail.
                                Text { text: win.myName; color: theme.text; font.pixelSize: 18; font.weight: Font.Medium }
                                // .sp-about renders me.about directly (empty when unset; no heading fallback).
                                Text { text: win.myAbout || ""; color: theme.text2; font.pixelSize: 14; visible: text !== "" }
                            }
                        }
                    }

                    // ===== .settings-list =====

                    // 1) Tema — .theme-modes (Light / Dark / System)
                    SettingsItem {
                        icon: "theme"; name: i18n.t("theme"); topAlign: true; clickable: false
                        extra: RowLayout {
                            Layout.fillWidth: true; Layout.topMargin: 8; spacing: 6
                            ThemeMode { text: i18n.t("theme_light"); on: settingsPopup.themeMode === "light"; onClicked: settingsPopup.setThemeMode("light") }
                            ThemeMode { text: i18n.t("theme_dark"); on: settingsPopup.themeMode === "dark"; onClicked: settingsPopup.setThemeMode("dark") }
                            ThemeMode { text: i18n.t("theme_system"); on: settingsPopup.themeMode === "system"; onClicked: settingsPopup.setThemeMode("system") }
                        }
                    }

                    // 2) Warna aksen — .acc-sw swatches (default ✕ + 6 warna)
                    SettingsItem {
                        icon: "theme"; name: i18n.t("accent_color"); topAlign: true; clickable: false
                        extra: RowLayout {
                            Layout.fillWidth: true; Layout.topMargin: 8; spacing: 8
                            Repeater {
                                model: ["", "#06b67f", "#5b6ef5", "#e5614e", "#f2a33c", "#9b59b6", "#e9418a"]
                                // .acc-sw: 26x26 round, border 2px transparent; .on → border var(--text).
                                Rectangle {
                                    required property string modelData
                                    width: 26; height: 26; radius: 13
                                    color: modelData !== "" ? modelData : theme.accent
                                    border.width: 2
                                    border.color: (theme.accentOverride === modelData) ? theme.text : "transparent"
                                    Text {
                                        anchors.centerIn: parent; visible: parent.modelData === ""
                                        text: "✕"; color: theme.text2; font.pixelSize: 12
                                    }
                                    TapHandler { onTapped: theme.accentOverride = parent.modelData }
                                }
                            }
                            Item { Layout.fillWidth: true }
                        }
                    }

                    // 3) Bahasa (lang-item, cursor default)
                    SettingsItem {
                        icon: "globe"; name: i18n.t("language"); desc: i18n.t("language_d")
                        clickable: false
                        trailing: Combo {
                            implicitWidth: 140
                            textRole: "label"; valueRole: "code"
                            model: [
                                { code: "en", label: "English" },
                                { code: "id", label: "Indonesia" },
                                { code: "es", label: "Español" },
                                { code: "ar", label: "العربية" },
                                { code: "ja", label: "日本語" },
                                { code: "zh-CN", label: "中文" }
                            ]
                            currentIndex: { var c = i18n.lang; return c === "id" ? 1 : c === "es" ? 2 : c === "ar" ? 3 : c === "ja" ? 4 : c === "zh-CN" ? 5 : 0 }
                            onActivated: i18n.setLang(currentValue)
                        }
                    }

                    // 4) Bahasa terjemahan (lang-item) — Combo lokal translateLang
                    SettingsItem {
                        icon: "translate"; name: i18n.t("translate_lang"); desc: i18n.t("translate_lang_d")
                        clickable: false
                        trailing: Combo {
                            implicitWidth: 140
                            textRole: "label"; valueRole: "code"
                            model: [
                                { code: "en", label: "English" },
                                { code: "id", label: "Indonesia" },
                                { code: "es", label: "Español" },
                                { code: "ar", label: "العربية" },
                                { code: "ja", label: "日本語" },
                                { code: "zh-CN", label: "中文" }
                            ]
                            currentIndex: { var c = settingsPopup.translateLang; return c === "id" ? 1 : c === "es" ? 2 : c === "ar" ? 3 : c === "ja" ? 4 : c === "zh-CN" ? 5 : 0 }
                            onActivated: settingsPopup.translateLang = currentValue
                        }
                    }

                    // 5) Suara notifikasi (switch)
                    SettingsItem {
                        icon: "speaker"; name: i18n.t("notif_sound")
                        desc: settingsPopup.soundOn ? i18n.t("active") : i18n.t("off")
                        onActivated: settingsPopup.soundOn = !settingsPopup.soundOn
                        trailing: Tog { checked: settingsPopup.soundOn; onToggled: settingsPopup.soundOn = checked }
                    }

                    // 6) Virtualisasi daftar pesan (switch, eksperimental)
                    SettingsItem {
                        icon: "hamburger"; name: i18n.t("virt_list"); desc: i18n.t("virt_list_d")
                        onActivated: settingsPopup.virtList = !settingsPopup.virtList
                        trailing: Tog { checked: settingsPopup.virtList; onToggled: settingsPopup.virtList = checked }
                    }

                    // 7) Keep deleted (anti-delete)
                    SettingsItem {
                        icon: "trash"; name: i18n.t("keep_deleted"); desc: i18n.t("keep_deleted_d")
                        clickable: false
                        trailing: Tog { checked: app.keepDeleted; onToggled: app.setKeepDeleted(checked) }
                    }

                    // 7b) Lihat pesan dihapus (conditional — tampil saat keep-deleted on)
                    SettingsItem {
                        visible: app.keepDeleted
                        icon: "eyeoff"; name: i18n.t("show_deleted"); desc: i18n.t("show_deleted_d")
                        onActivated: settingsPopup.showDeleted = !settingsPopup.showDeleted
                        trailing: Tog { checked: settingsPopup.showDeleted; onToggled: settingsPopup.showDeleted = checked }
                    }

                    // 8) Retensi — chips 30/90/180/selamanya
                    SettingsItem {
                        icon: "disk"; name: i18n.t("retention"); desc: i18n.t("retention_d")
                        topAlign: true; clickable: false
                        extra: RowLayout {
                            Layout.fillWidth: true; Layout.topMargin: 8; spacing: 6
                            ThemeMode { text: i18n.t("retention_days_30"); on: settingsPopup.retDays === 30; onClicked: { settingsPopup.retDays = 30; app.act("SetRetention", [30]) } }
                            ThemeMode { text: i18n.t("retention_days_90"); on: settingsPopup.retDays === 90; onClicked: { settingsPopup.retDays = 90; app.act("SetRetention", [90]) } }
                            ThemeMode { text: i18n.t("retention_days_180"); on: settingsPopup.retDays === 180; onClicked: { settingsPopup.retDays = 180; app.act("SetRetention", [180]) } }
                            ThemeMode { text: i18n.t("retention_forever"); on: settingsPopup.retDays === 0; onClicked: { settingsPopup.retDays = 0; app.act("SetRetention", [0]) } }
                        }
                    }

                    // 6) Timer hilang-otomatis default — chips Off/24h/7d/90d
                    SettingsItem {
                        icon: "clock"; name: i18n.t("default_disappearing"); desc: i18n.t("default_disappearing_d")
                        topAlign: true; clickable: false
                        extra: RowLayout {
                            Layout.fillWidth: true; Layout.topMargin: 8; spacing: 6
                            ThemeMode { text: i18n.t("disappearing_off"); on: settingsPopup.defDis === 0; onClicked: { settingsPopup.defDis = 0; app.act("SetDefaultDisappearing", [0]) } }
                            ThemeMode { text: i18n.t("disappearing_24h"); on: settingsPopup.defDis === 86400; onClicked: { settingsPopup.defDis = 86400; app.act("SetDefaultDisappearing", [86400]) } }
                            ThemeMode { text: i18n.t("disappearing_7d"); on: settingsPopup.defDis === 604800; onClicked: { settingsPopup.defDis = 604800; app.act("SetDefaultDisappearing", [604800]) } }
                            ThemeMode { text: i18n.t("disappearing_90d"); on: settingsPopup.defDis === 7776000; onClicked: { settingsPopup.defDis = 7776000; app.act("SetDefaultDisappearing", [7776000]) } }
                        }
                    }

                    // 7) Proxy — input di bawah deskripsi
                    SettingsItem {
                        icon: "globe2"; name: i18n.t("proxy"); desc: i18n.t("proxy_d")
                        topAlign: true; clickable: false
                        extra: Rectangle {
                            Layout.fillWidth: true; Layout.topMargin: 6; implicitHeight: 34
                            radius: 8; color: theme.bg2; border.color: theme.line; border.width: 1
                            TextInput {
                                id: proxyInput
                                anchors.fill: parent; anchors.leftMargin: 11; anchors.rightMargin: 11
                                verticalAlignment: TextInput.AlignVCenter
                                color: theme.text; font.pixelSize: 13; clip: true; selectByMouse: true
                                onEditingFinished: app.act("SetProxy", [proxyInput.text])
                                Text {
                                    anchors.verticalCenter: parent.verticalCenter
                                    visible: proxyInput.text === "" && !proxyInput.activeFocus
                                    text: "socks5://127.0.0.1:9050"; color: theme.text2; font.pixelSize: 13
                                }
                            }
                        }
                    }

                    // 8) Penyimpanan (link)
                    SettingsItem {
                        icon: "disk"; name: i18n.t("storage"); desc: i18n.t("storage_d")
                        onActivated: { app.loadDetail("GetStorageUsage", ""); settingsPopup.close(); storagePopup.open() }
                    }

                    // 9) Privasi (link)
                    SettingsItem {
                        icon: "lock"; name: i18n.t("privacy"); desc: i18n.t("privacy_d")
                        onActivated: { app.loadDetail("GetPrivacy", ""); settingsPopup.close(); privacyPopup.open() }
                    }

                    // 10) Pesan berbintang (link)
                    SettingsItem {
                        icon: "star2"; name: i18n.t("starred_msg")
                        onActivated: { settingsPopup.close(); activeView = "starred"; win.loadView("starred") }
                    }

                    // 11) Kunci aplikasi (switch) — LOKAL (tak ada backend PIN di engine Qt)
                    SettingsItem {
                        icon: "lock"; name: i18n.t("applock")
                        desc: settingsPopup.appLock ? i18n.t("active") : i18n.t("off")
                        onActivated: settingsPopup.appLock = !settingsPopup.appLock
                        trailing: Tog { checked: settingsPopup.appLock; onToggled: settingsPopup.appLock = checked }
                    }

                    // 11b) Kunci sekarang (conditional — tampil saat applock on)
                    SettingsItem {
                        visible: settingsPopup.appLock
                        icon: "lock"; name: i18n.t("lock_now")
                        onActivated: {} // LOKAL: tak ada lockNow di engine Qt
                    }

                    // 12) Jangan ganggu (switch + dua field jam saat on)
                    SettingsItem {
                        icon: "moon"; name: i18n.t("dnd"); desc: i18n.t("dnd_d")
                        topAlign: true; clickable: false
                        trailing: Tog { checked: settingsPopup.dndOn; onToggled: settingsPopup.dndOn = checked }
                        extra: RowLayout {
                            visible: settingsPopup.dndOn
                            Layout.topMargin: 8; spacing: 8
                            // Field jam (bg2, 1px line, radius 8, pad 4/8) per Svelte.
                            Rectangle {
                                implicitWidth: 70; implicitHeight: 26; radius: 8
                                color: theme.bg2; border.color: theme.line; border.width: 1
                                TextInput {
                                    anchors.fill: parent; anchors.leftMargin: 8; anchors.rightMargin: 8
                                    verticalAlignment: TextInput.AlignVCenter
                                    text: settingsPopup.dndFrom; color: theme.text; font.pixelSize: 13
                                    selectByMouse: true; onEditingFinished: settingsPopup.dndFrom = text
                                }
                            }
                            Text { text: "–"; color: theme.text2; font.pixelSize: 13; Layout.alignment: Qt.AlignVCenter }
                            Rectangle {
                                implicitWidth: 70; implicitHeight: 26; radius: 8
                                color: theme.bg2; border.color: theme.line; border.width: 1
                                TextInput {
                                    anchors.fill: parent; anchors.leftMargin: 8; anchors.rightMargin: 8
                                    verticalAlignment: TextInput.AlignVCenter
                                    text: settingsPopup.dndTo; color: theme.text; font.pixelSize: 13
                                    selectByMouse: true; onEditingFinished: settingsPopup.dndTo = text
                                }
                            }
                        }
                    }

                    // 13) Reaksi cepat kustom (.qre-sw) — klik = edit via prompt
                    SettingsItem {
                        icon: "emojiface"; name: i18n.t("quick_react")
                        topAlign: true; clickable: false
                        extra: RowLayout {
                            Layout.topMargin: 8; spacing: 6
                            Repeater {
                                model: settingsPopup.quickReactions.length
                                // .qre-sw: 36x32, bg2, radius 8, 18px emoji.
                                Rectangle {
                                    required property int index
                                    width: 36; height: 32; radius: 8
                                    color: qreHov.hovered ? theme.hover : theme.bg2
                                    Text { anchors.centerIn: parent; text: settingsPopup.quickReactions[parent.index]; font.pixelSize: 18 }
                                    HoverHandler { id: qreHov }
                                    TapHandler {
                                        onTapped: win.prompt(i18n.t("quick_react_edit"), settingsPopup.quickReactions[parent.index], function(v) {
                                            if (!v) return
                                            var arr = settingsPopup.quickReactions.slice()
                                            arr[parent.index] = Array.from(v)[0] || arr[parent.index]
                                            settingsPopup.quickReactions = arr
                                        })
                                    }
                                }
                            }
                        }
                    }

                    // 14) Ukuran tampilan (zoom UI) — slider 80–140 step 5, LOKAL
                    SettingsItem {
                        icon: "zoom"; name: i18n.t("ui_scale") + " · " + settingsPopup.uiScale + "%"
                        topAlign: true; clickable: false
                        extra: TSlider {
                            Layout.fillWidth: true; Layout.topMargin: 8
                            from: 80; to: 140; stepSize: 5; value: settingsPopup.uiScale
                            onMoved: settingsPopup.uiScale = Math.round(value)
                        }
                    }

                    // 15) Jalan di latar belakang
                    SettingsItem {
                        icon: "window"; name: i18n.t("bg_run"); desc: i18n.t("bg_run_d")
                        clickable: false
                        trailing: Tog { checked: settingsPopup.bgRun; onToggled: { settingsPopup.bgRun = checked; app.act("SetBackgroundClose", [checked]) } }
                    }

                    // 15b) Keluar aplikasi (conditional danger — tampil saat bg-run on)
                    SettingsItem {
                        visible: settingsPopup.bgRun
                        icon: "power"; name: i18n.t("quit_app"); danger: true
                        onActivated: Qt.quit() // engine Qt tak punya app.quit → Qt.quit()
                    }

                    // 16) Keluar (danger)
                    SettingsItem {
                        icon: "logout"; name: i18n.t("logout"); danger: true
                        onActivated: { app.logout(); settingsPopup.close() }
                    }

                    // 13) Footer .settings-foot (pad 18 0 8, 12px, opacity .45)
                    Text {
                        Layout.fillWidth: true; Layout.topMargin: 18; Layout.bottomMargin: 8
                        horizontalAlignment: Text.AlignHCenter
                        text: "WhatsLite dev"; color: theme.text; opacity: 0.45; font.pixelSize: 12
                    }
                }
            }
        }
    }

    // === Detail grup / profil kontak (klik header) — panel dok-kanan ===
    Drawer {
        id: detailPopup
        edge: Qt.RightEdge
        width: 400
        height: parent ? parent.height : 600
        // Panel info dok-kanan (app.css .info-panel: 400px, sidebar-bg, border-left).
        // Grup bila engine kirim participants (real) atau members (mock).
        property bool isGroup: app.detail.participants !== undefined || app.detail.members !== undefined
        // Daftar anggota: real engine→participants, mock→members.
        property var memberList: app.detail.participants || app.detail.members || []
        // Topik/deskripsi grup: real→topic, mock→desc.
        property string groupDesc: app.detail.topic || app.detail.desc || ""
        // Saya admin? real→amAdmin. Mock tak set field → tampilkan UI admin (members ada).
        property bool amAdmin: app.detail.amAdmin === true || (app.detail.amAdmin === undefined && app.detail.members !== undefined)
        // Reset state UI sementara tiap buka (deskripsi klem, filter anggota).
        onOpened: { infoCol.descOpen = false; infoCol.aboutOpen = false; infoCol.memberQ = "" }
        background: Rectangle {
            color: theme.sidebarBg
            Rectangle { width: 1; height: parent.height; color: theme.divider } // border-left
        }
        ColumnLayout {
            anchors.fill: parent; spacing: 0
            // .info-head (56px, head-bg): tutup + judul.
            Rectangle {
                Layout.fillWidth: true; Layout.preferredHeight: 56; color: theme.headBg
                RowLayout {
                    anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 16; spacing: 20
                    Rectangle {
                        width: 40; height: 40; radius: 20
                        color: closeHov.hovered ? (theme.dark ? Qt.rgba(1, 1, 1, 0.08) : Qt.rgba(0, 0, 0, 0.06)) : "transparent"
                        Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["close"]; color: theme.text2 }
                        HoverHandler { id: closeHov }
                        MouseArea { anchors.fill: parent; onClicked: detailPopup.close() }
                    }
                    Text { Layout.fillWidth: true; text: detailPopup.isGroup ? i18n.t("info_group") : i18n.t("info_contact")
                        color: theme.text; font.pixelSize: 16; font.weight: Font.Medium }
                }
            }
            // Konten scroll (hero + blok).
            Flickable {
                Layout.fillWidth: true; Layout.fillHeight: true; clip: true
                contentWidth: width; contentHeight: infoCol.implicitHeight
                ScrollBar.vertical: ScrollBar {}
                ColumnLayout {
                    id: infoCol; width: parent.width; spacing: 0
                    property bool descOpen: false
                    property bool aboutOpen: false
                    property string memberQ: ""

                    // Pemisah antar-bagian: bar 6px warna wallpaper (app.css border-bottom 6px wallpaper).
                    component InfoSep: Rectangle { Layout.fillWidth: true; Layout.preferredHeight: 6; color: theme.wallpaper }

                    // .info-hero: avatar 200, nama (24/500), sub (15 text2). padding 28/24, border-bottom 6px.
                    ColumnLayout {
                        Layout.fillWidth: true; Layout.topMargin: 28; Layout.bottomMargin: 28
                        Layout.leftMargin: 24; Layout.rightMargin: 24; spacing: 0
                        Item {
                            Layout.alignment: Qt.AlignHCenter; Layout.preferredWidth: 200; Layout.preferredHeight: 200
                            Layout.bottomMargin: 16
                            Avatar {
                                anchors.fill: parent; fontSize: 80
                                name: app.detail.name || ""; jid: win.selectedChat.id || ""; base: app.mediaBase
                                accent: win.avatarColor(app.detail.name || "?"); group: detailPopup.isGroup
                            }
                            // Tombol ganti foto grup (admin) — overlay tengah avatar.
                            Rectangle {
                                anchors.centerIn: parent; width: 40; height: 40; radius: 20
                                visible: detailPopup.isGroup && detailPopup.amAdmin
                                // .hero-photo-btn background rgba(0,0,0,.45) constant (no hover-darken).
                                color: "#73000000"
                                Icon { anchors.centerIn: parent; width: 20; height: 20; svg: win.ico["herophoto"]; color: "#ffffff" }
                                MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                                    onClicked: app.act("SetGroupPhoto", [win.selectedChat.id, ""]) }
                            }
                        }
                        // Nama + pensil edit (admin).
                        RowLayout {
                            Layout.alignment: Qt.AlignHCenter; spacing: 8
                            Text { text: app.detail.name || ""; color: theme.text; font.pixelSize: 24; font.weight: Font.Medium }
                            Icon {
                                visible: detailPopup.isGroup && detailPopup.amAdmin
                                Layout.preferredWidth: 16; Layout.preferredHeight: 16
                                svg: win.ico["editpen"]; color: penHov.hovered ? theme.accent : theme.text2
                                HoverHandler { id: penHov }
                                MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                                    onClicked: win.prompt(i18n.t("group_edit_name"), app.detail.name || "", function(v){ if (v && v.trim()) app.act("SetGroupSubject", [win.selectedChat.id, v.trim()]) }) }
                            }
                        }
                        // Sub (jumlah anggota / nomor / about).
                        Text {
                            Layout.alignment: Qt.AlignHCenter; Layout.topMargin: 4; color: theme.text2; font.pixelSize: 15
                            text: detailPopup.isGroup ? (detailPopup.memberList.length + " " + i18n.t("members_n"))
                                                       : (app.detail.phone || app.detail.about || "")
                        }
                        // Chip "kontak tersimpan" / akun bisnis (kontak).
                        Text {
                            visible: !detailPopup.isGroup && app.detail.saved === true
                            Layout.alignment: Qt.AlignHCenter; Layout.topMargin: 6
                            text: i18n.t("contact_saved_chip"); color: theme.accent; font.pixelSize: 12
                        }
                        Rectangle {
                            visible: !detailPopup.isGroup && app.detail.isBiz === true
                            Layout.alignment: Qt.AlignHCenter; Layout.topMargin: 6
                            implicitWidth: bizT.implicitWidth + 20; implicitHeight: 24; radius: 20
                            color: Qt.rgba(theme.accent.r, theme.accent.g, theme.accent.b, 0.15)
                            Text { id: bizT; anchors.centerIn: parent
                                // biz-chip: '✔ Business account' + ' · {category}' when set.
                                text: "✔ " + i18n.t("business_account") + ((app.detail.category || "") !== "" ? " · " + app.detail.category : "")
                                color: theme.accent; font.pixelSize: 12; font.weight: Font.DemiBold }
                        }
                    }
                    InfoSep {}

                    // .info-block deskripsi (grup) — klem 5 baris + baca selengkapnya (>140 char).
                    ColumnLayout {
                        Layout.fillWidth: true; Layout.leftMargin: 24; Layout.rightMargin: 24
                        Layout.topMargin: 14; Layout.bottomMargin: 14; spacing: 5
                        visible: detailPopup.isGroup && detailPopup.groupDesc !== ""
                        RowLayout {
                            Layout.fillWidth: true; spacing: 8
                            Text { text: i18n.t("info_groupdesc"); color: theme.accent; font.pixelSize: 13 }
                            Item { Layout.fillWidth: true }
                            Icon {
                                visible: detailPopup.amAdmin
                                Layout.preferredWidth: 16; Layout.preferredHeight: 16
                                svg: win.ico["editpen"]; color: descPenHov.hovered ? theme.accent : theme.text2
                                HoverHandler { id: descPenHov }
                                MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                                    onClicked: win.prompt(i18n.t("group_edit_desc"), detailPopup.groupDesc, function(v){ if (v != null) app.act("SetGroupDescription", [win.selectedChat.id, v.trim()]) }) }
                            }
                        }
                        Text {
                            Layout.fillWidth: true; wrapMode: Text.WordWrap; text: detailPopup.groupDesc || "—"
                            color: theme.text; font.pixelSize: 15
                            maximumLineCount: infoCol.descOpen ? 0 : 5; elide: infoCol.descOpen ? Text.ElideNone : Text.ElideRight
                        }
                        Text {
                            visible: detailPopup.groupDesc.length > 140
                            text: infoCol.descOpen ? i18n.t("read_less") : i18n.t("read_more")
                            color: theme.accent; font.pixelSize: 13; font.weight: Font.DemiBold
                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: infoCol.descOpen = !infoCol.descOpen }
                        }
                    }
                    InfoSep { visible: detailPopup.isGroup && detailPopup.groupDesc !== "" }

                    // Info bisnis (kontak): ContactProfile.svelte — .info-block lbl=business_info,
                    // baris .val 📍 alamat / ✉️ email; guard biz.isBiz && (address||email).
                    ColumnLayout {
                        property bool show: !detailPopup.isGroup && app.detail.isBiz === true
                                            && ((app.detail.address || "") !== "" || (app.detail.email || "") !== "")
                        Layout.fillWidth: true; Layout.leftMargin: 24; Layout.rightMargin: 24
                        Layout.topMargin: 14; Layout.bottomMargin: 14; spacing: 5
                        visible: show
                        Text { text: i18n.t("business_info"); color: theme.accent; font.pixelSize: 13 }
                        Text { visible: (app.detail.address || "") !== ""; Layout.fillWidth: true; wrapMode: Text.WordWrap
                            text: "📍 " + (app.detail.address || ""); color: theme.text; font.pixelSize: 15 }
                        Text { visible: (app.detail.email || "") !== ""; Layout.fillWidth: true; wrapMode: Text.WordWrap
                            text: "✉️ " + (app.detail.email || ""); color: theme.text; font.pixelSize: 15 }
                    }
                    InfoSep { visible: !detailPopup.isGroup && app.detail.isBiz === true
                                       && ((app.detail.address || "") !== "" || (app.detail.email || "") !== "") }

                    // About (kontak) — klem 5 baris + baca selengkapnya (>140 char), mirror desc grup.
                    ColumnLayout {
                        Layout.fillWidth: true; Layout.leftMargin: 24; Layout.rightMargin: 24
                        Layout.topMargin: 14; Layout.bottomMargin: 14; spacing: 5
                        visible: !detailPopup.isGroup && (app.detail.about || "") !== ""
                        Text { text: i18n.t("info_about"); color: theme.accent; font.pixelSize: 13 }
                        Text {
                            Layout.fillWidth: true; wrapMode: Text.WordWrap; text: app.detail.about || "—"
                            color: theme.text; font.pixelSize: 15
                            maximumLineCount: infoCol.aboutOpen ? 0 : 5; elide: infoCol.aboutOpen ? Text.ElideNone : Text.ElideRight
                        }
                        Text {
                            visible: (app.detail.about || "").length > 140
                            text: infoCol.aboutOpen ? i18n.t("read_less") : i18n.t("read_more")
                            color: theme.accent; font.pixelSize: 13; font.weight: Font.DemiBold
                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: infoCol.aboutOpen = !infoCol.aboutOpen }
                        }
                    }
                    InfoSep { visible: !detailPopup.isGroup && (app.detail.about || "") !== "" }

                    // Aksi admin grup (info-row): tambah anggota, link undangan, reset link.
                    ColumnLayout {
                        Layout.fillWidth: true; visible: detailPopup.isGroup && detailPopup.amAdmin; spacing: 0
                        Repeater {
                            model: [{ icon: "addmember", t: i18n.t("group_add_member"), a: "add" },
                                    { icon: "invitelink", t: i18n.t("invite_link"), a: "invite" },
                                    { icon: "resetlink", t: i18n.t("invite_reset"), a: "reset" }]
                            delegate: Rectangle {
                                Layout.fillWidth: true; implicitHeight: 50; color: gActHov.hovered ? theme.hover : "transparent"
                                RowLayout {
                                    anchors.fill: parent; anchors.leftMargin: 24; anchors.rightMargin: 24; spacing: 18
                                    Icon { Layout.preferredWidth: 22; Layout.preferredHeight: 22; svg: win.ico[modelData.icon]; color: theme.text2 }
                                    Text { Layout.fillWidth: true; text: modelData.t; color: theme.text; font.pixelSize: 15 }
                                }
                                HoverHandler { id: gActHov }
                                MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: {
                                    if (modelData.a === "add") win.prompt(i18n.t("group_add_member"), "62812", function(v){ var d = (v||"").replace(/[^0-9]/g, ""); if (d.length >= 6) app.act("UpdateGroupParticipants", [win.selectedChat.id, [d + "@s.whatsapp.net"], "add"]) })
                                    else if (modelData.a === "invite") app.fetchStr("GroupInviteLink", [win.selectedChat.id, false])
                                    else app.fetchStr("GroupInviteLink", [win.selectedChat.id, true]) } }
                            }
                        }
                    }
                    InfoSep { visible: detailPopup.isGroup && detailPopup.amAdmin }

                    // Pengaturan admin grup (info-block + .switch via Tog).
                    ColumnLayout {
                        Layout.fillWidth: true; visible: detailPopup.isGroup && detailPopup.amAdmin; spacing: 0
                        Text { Layout.leftMargin: 24; Layout.topMargin: 14; Layout.bottomMargin: 5
                            text: i18n.t("group_admin_settings"); color: theme.accent; font.pixelSize: 13 }
                        Repeater {
                            model: [{ k: "announce", a: "SetGroupAnnounce", t: i18n.t("group_announce") },
                                    { k: "locked", a: "SetGroupLocked", t: i18n.t("group_locked") },
                                    { k: "joinApproval", a: "SetGroupJoinApproval", t: i18n.t("group_join_approval") },
                                    { k: "adminAddOnly", a: "SetGroupAddMode", t: i18n.t("group_admin_add") }]
                            delegate: Rectangle {
                                id: setRow
                                Layout.fillWidth: true; implicitHeight: 46; color: rowHov.hovered ? theme.hover : "transparent"
                                // real engine: adminAddOnly. mock: adminAdd.
                                property bool on: app.detail[modelData.k] === true || (modelData.k === "adminAddOnly" && app.detail.adminAdd === true)
                                RowLayout {
                                    // .info-block pad 24 + .info-row pad 24 → toggle rows ter-inset 48 (label tetap 24).
                                    anchors.fill: parent; anchors.leftMargin: 48; anchors.rightMargin: 48; spacing: 18
                                    Text { Layout.fillWidth: true; text: modelData.t; color: theme.text; font.pixelSize: 15 }
                                    Tog { checked: setRow.on; onClicked: app.act(modelData.a, [win.selectedChat.id, !setRow.on]) }
                                }
                                HoverHandler { id: rowHov }
                                MouseArea { anchors.fill: parent; onClicked: app.act(modelData.a, [win.selectedChat.id, !setRow.on]) }
                            }
                        }
                    }
                    InfoSep { visible: detailPopup.isGroup && detailPopup.amAdmin }

                    // Daftar anggota (grup) + pencarian (>8 anggota).
                    ColumnLayout {
                        Layout.fillWidth: true; visible: detailPopup.isGroup; spacing: 0
                        Text { Layout.leftMargin: 24; Layout.topMargin: 14; Layout.bottomMargin: 5
                            text: detailPopup.memberList.length + " " + i18n.t("members_n"); color: theme.accent; font.pixelSize: 13 }
                        // .member-search: bg2, border line, radius 9, pad 7/11.
                        Rectangle {
                            visible: detailPopup.memberList.length > 8
                            Layout.fillWidth: true; Layout.leftMargin: 24; Layout.rightMargin: 24; Layout.topMargin: 6; Layout.bottomMargin: 8
                            implicitHeight: 34; radius: 9; color: theme.bg2; border.color: theme.line; border.width: 1
                            TextInput {
                                anchors.fill: parent; anchors.leftMargin: 11; anchors.rightMargin: 11
                                verticalAlignment: TextInput.AlignVCenter; color: theme.text; font.pixelSize: 14; clip: true
                                onTextChanged: infoCol.memberQ = text
                                Text { anchors.fill: parent; verticalAlignment: Text.AlignVCenter; visible: parent.text === ""
                                    text: i18n.t("search"); color: theme.text2; font.pixelSize: 14 }
                            }
                        }
                        Repeater {
                            model: detailPopup.memberList
                            delegate: Rectangle {
                                property bool isAdm: modelData.isAdmin === true || modelData.admin === true
                                visible: infoCol.memberQ.trim() === "" || (modelData.name || "").toLowerCase().indexOf(infoCol.memberQ.trim().toLowerCase()) >= 0
                                // .member: plain flex row, NO hover bg (hover hanya di .m-actions). .members gap:2px.
                                Layout.fillWidth: true; implicitHeight: visible ? 52 : 0; color: "transparent"
                                Layout.bottomMargin: visible ? 2 : 0
                                RowLayout {
                                    anchors.fill: parent; anchors.leftMargin: 24; anchors.rightMargin: 24; spacing: 12
                                    Avatar { Layout.preferredWidth: 40; Layout.preferredHeight: 40; fontSize: 16
                                        name: modelData.name || ""; accent: win.avatarColor(modelData.name || "?") }
                                    Text { Layout.fillWidth: true; text: modelData.name || ""; color: theme.text; font.pixelSize: 15; elide: Text.ElideRight }
                                    Rectangle { visible: parent.parent.isAdm; implicitWidth: adm.implicitWidth + 12; implicitHeight: 18
                                        radius: 8; color: "transparent"; border.width: 1; border.color: theme.accent
                                        Text { id: adm; anchors.centerIn: parent; text: i18n.t("member_admin"); color: theme.accent; font.pixelSize: 12 } }
                                    // .m-actions: aksi admin per-anggota (promote/demote/remove).
                                    // Tampil hanya bila amAdmin & bukan diri sendiri. Functional: UpdateGroupParticipants.
                                    RowLayout {
                                        id: mActions
                                        spacing: 4
                                        property bool memberIsAdmin: modelData.isAdmin === true || modelData.admin === true
                                        property bool isSelf: modelData.isSelf === true || modelData.isMe === true || modelData.self === true
                                        visible: detailPopup.amAdmin && !isSelf
                                        property string mjid: modelData.jid || modelData.id || ""
                                        // ▲ promote (jika belum admin) / ▼ demote (jika admin)
                                        Rectangle {
                                            implicitWidth: mPromote.implicitWidth + 12; implicitHeight: 22; radius: 6
                                            color: mPromHov.hovered ? theme.hover : "transparent"
                                            Text { id: mPromote; anchors.centerIn: parent
                                                text: mActions.memberIsAdmin ? "▼" : "▲"
                                                color: mPromHov.hovered ? theme.text : theme.text2; font.pixelSize: 13 }
                                            HoverHandler { id: mPromHov }
                                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                                                onClicked: app.act("UpdateGroupParticipants",
                                                    [win.selectedChat.id, [mActions.mjid],
                                                     mActions.memberIsAdmin ? "demote" : "promote"]) }
                                        }
                                        // ✕ remove (danger)
                                        Rectangle {
                                            implicitWidth: mRemove.implicitWidth + 12; implicitHeight: 22; radius: 6
                                            color: mRemHov.hovered ? theme.hover : "transparent"
                                            Text { id: mRemove; anchors.centerIn: parent; text: "✕"
                                                color: mRemHov.hovered ? "#e0463e" : theme.text2; font.pixelSize: 13 }
                                            HoverHandler { id: mRemHov }
                                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                                                onClicked: app.act("UpdateGroupParticipants", [win.selectedChat.id, [mActions.mjid], "remove"]) }
                                        }
                                    }
                                }
                            }
                        }
                        Item { Layout.preferredHeight: 8 }
                    }
                    InfoSep { visible: detailPopup.isGroup }

                    // Aksi kontak (info-row): Pesan, Simpan/Ganti nama, + Hapus label (danger) bila tersimpan.
                    ColumnLayout {
                        Layout.fillWidth: true; visible: !detailPopup.isGroup; spacing: 0
                        Repeater {
                            model: [{ icon: "message", t: i18n.t("message_action") },
                                    { icon: "addmember", t: app.detail.saved === true ? i18n.t("rename_contact") : i18n.t("save_contact") }]
                            delegate: Rectangle {
                                Layout.fillWidth: true; implicitHeight: 50; color: ctActHov.hovered ? theme.hover : "transparent"
                                RowLayout {
                                    anchors.fill: parent; anchors.leftMargin: 24; anchors.rightMargin: 24; spacing: 18
                                    Icon { Layout.preferredWidth: 22; Layout.preferredHeight: 22; svg: win.ico[modelData.icon]; color: theme.text2 }
                                    Text { Layout.fillWidth: true; text: modelData.t; color: theme.text; font.pixelSize: 15 }
                                }
                                HoverHandler { id: ctActHov }
                                MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: {
                                    if (index === 0) { app.openChat(win.selectedChat.id); detailPopup.close() }
                                    else win.prompt(i18n.t("save_contact"), app.detail.name || "", function(v){ if (v && v.trim()) app.act("SaveContactLabel", [win.selectedChat.id, v.trim()]) }) } }
                            }
                        }
                        // .info-row.danger 'remove_label' (#e35d6a) — hanya bila kontak tersimpan (prof.saved).
                        Rectangle {
                            visible: app.detail.saved === true
                            Layout.fillWidth: true; implicitHeight: 50; color: ctRmHov.hovered ? theme.hover : "transparent"
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 24; anchors.rightMargin: 24; spacing: 18
                                Icon { Layout.preferredWidth: 22; Layout.preferredHeight: 22; svg: win.ico["removelabel"]; color: "#e35d6a" }
                                Text { Layout.fillWidth: true; text: i18n.t("remove_label"); color: "#e35d6a"; font.pixelSize: 15 }
                            }
                            HoverHandler { id: ctRmHov }
                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                                onClicked: app.act("RemoveContactLabel", [win.selectedChat.id]) }
                        }
                    }
                    InfoSep { visible: !detailPopup.isGroup }

                    // Grup bersama (kontak): ContactProfile.svelte — .info-block lbl=common_groups (n),
                    // tiap baris button.info-row (ikon grup + nama) buka grup; guard commonGroups ada.
                    ColumnLayout {
                        property var groups: (app.detail.commonGroups instanceof Array) ? app.detail.commonGroups : []
                        Layout.fillWidth: true; visible: !detailPopup.isGroup && groups.length > 0; spacing: 0
                        Text { Layout.leftMargin: 24; Layout.topMargin: 14; Layout.bottomMargin: 6
                            text: i18n.t("common_groups") + " (" + groups.length + ")"; color: theme.accent; font.pixelSize: 13 }
                        Repeater {
                            model: parent.groups
                            delegate: Rectangle {
                                Layout.fillWidth: true; implicitHeight: 50; color: cgHov.hovered ? theme.hover : "transparent"
                                RowLayout {
                                    anchors.fill: parent; anchors.leftMargin: 24; anchors.rightMargin: 24; spacing: 18
                                    Icon { Layout.preferredWidth: 22; Layout.preferredHeight: 22; svg: win.ico["commongroup"]; color: theme.text2 }
                                    Text { Layout.fillWidth: true; text: modelData.name || (modelData.jid || "").split("@")[0] || ""
                                        color: theme.text; font.pixelSize: 15; elide: Text.ElideRight }
                                }
                                HoverHandler { id: cgHov }
                                MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                                    onClicked: { app.openChat(modelData.jid || ""); detailPopup.close() } }
                            }
                        }
                    }
                    InfoSep { visible: !detailPopup.isGroup && (app.detail.commonGroups instanceof Array) && app.detail.commonGroups.length > 0 }

                    // Galeri media bersama ("Media, links & docs"). InfoPanel.svelte .info-block + .media-grid.
                    // DATA-GUARDED: engine Qt belum punya endpoint shared-media → tampil hanya bila app.detail.media ada.
                    ColumnLayout {
                        id: mediaBlock
                        property var mediaList: (app.detail.media instanceof Array) ? app.detail.media : []
                        property bool mediaOpen: false   // .media-lbl toggle: default 9 cell, klik → semua.
                        Layout.fillWidth: true; visible: mediaList.length > 0; spacing: 0
                        // .media-lbl: button (flex, gap 6) dgn chevron (.chev 11px) toggle 9 ↔ semua.
                        RowLayout {
                            Layout.leftMargin: 24; Layout.rightMargin: 24; Layout.topMargin: 14; Layout.bottomMargin: 8
                            spacing: 6
                            Text {
                                text: i18n.t("info_media") + " (" + mediaBlock.mediaList.length + ")"
                                color: theme.text2; font.pixelSize: 13
                            }
                            Icon { Layout.preferredWidth: 11; Layout.preferredHeight: 11
                                visible: mediaBlock.mediaList.length > 9
                                svg: win.ico["chevdown"]; color: theme.text2
                                rotation: mediaBlock.mediaOpen ? 180 : 0
                                Behavior on rotation { NumberAnimation { duration: 120 } } }
                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                                enabled: mediaBlock.mediaList.length > 9
                                onClicked: mediaBlock.mediaOpen = !mediaBlock.mediaOpen }
                        }
                        // .media-grid: grid 3 kolom, gap 4, cell aspect-ratio 1, radius 8, bg2.
                        GridLayout {
                            Layout.fillWidth: true; Layout.leftMargin: 24; Layout.rightMargin: 24; Layout.bottomMargin: 8
                            columns: 3; columnSpacing: 4; rowSpacing: 4
                            Repeater {
                                model: mediaBlock.mediaOpen ? mediaBlock.mediaList : mediaBlock.mediaList.slice(0, 9)
                                delegate: Rectangle {
                                    Layout.fillWidth: true
                                    Layout.preferredHeight: width   // aspect-ratio: 1
                                    radius: 8; color: theme.bg2; clip: true
                                    Image {
                                        anchors.fill: parent; fillMode: Image.PreserveAspectCrop
                                        source: (typeof modelData === "string") ? modelData : (modelData.url || modelData.thumb || "")
                                        visible: status === Image.Ready
                                    }
                                    // .media-play overlay untuk video/gif.
                                    Rectangle {
                                        anchors.fill: parent; color: "#40000000"
                                        visible: modelData && (modelData.type === "video" || modelData.type === "gif")
                                        Text { anchors.centerIn: parent; text: "▶"; color: "#ffffff"; font.pixelSize: 18 }
                                    }
                                }
                            }
                        }
                    }
                    InfoSep { visible: mediaBlock.visible }

                    // Wallpaper per-chat (swatch). LOKAL/visual — engine Qt belum punya store wallpaper. NOTE.
                    ColumnLayout {
                        Layout.fillWidth: true; Layout.leftMargin: 24; Layout.rightMargin: 24
                        Layout.topMargin: 14; Layout.bottomMargin: 14; spacing: 0
                        RowLayout {
                            Layout.fillWidth: true; spacing: 18
                            Icon { Layout.preferredWidth: 22; Layout.preferredHeight: 22; Layout.alignment: Qt.AlignTop
                                svg: win.ico["wallpaperico"]; color: theme.text2 }
                            ColumnLayout {
                                Layout.fillWidth: true; spacing: 8
                                Text { text: i18n.t("wallpaper"); color: theme.text; font.pixelSize: 15 }
                                Flow {
                                    Layout.fillWidth: true; spacing: 6
                                    // .wp-sw.none (bg2, ✕) — terapkan & persist via win.setWallpaper.
                                    Rectangle {
                                        width: 26; height: 26; radius: 7; color: theme.bg2
                                        border.width: 2; border.color: win.chatWallpaper(win.selectedChat.id) === "" ? theme.accent : "transparent"
                                        Text { anchors.centerIn: parent; text: "✕"; color: theme.text2; font.pixelSize: 12 }
                                        MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: win.setWallpaper(win.selectedChat.id, "") }
                                    }
                                    Repeater {
                                        model: ["#0b141a", "#111b21", "#1d2b22", "#2a2233", "#11212b", "#e7ddd0", "#d9e4dd", "#efe7da"]
                                        delegate: Rectangle {
                                            width: 26; height: 26; radius: 7; color: modelData
                                            border.width: 2; border.color: win.chatWallpaper(win.selectedChat.id) === modelData ? theme.accent : "transparent"
                                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: win.setWallpaper(win.selectedChat.id, modelData) }
                                        }
                                    }
                                }
                            }
                        }
                    }
                    InfoSep {}

                    // Pesan sementara (disappearing) — select. NOTE: backing SetDisappearing.
                    Rectangle {
                        Layout.fillWidth: true; implicitHeight: 56; color: disHov.hovered ? theme.hover : "transparent"
                        RowLayout {
                            anchors.fill: parent; anchors.leftMargin: 24; anchors.rightMargin: 24; spacing: 18
                            Icon { Layout.preferredWidth: 22; Layout.preferredHeight: 22; svg: win.ico["clock"]; color: theme.text2 }
                            Text { Layout.fillWidth: true; text: i18n.t("disappearing_msg"); color: theme.text; font.pixelSize: 15 }
                            Combo {
                                implicitWidth: 120
                                textRole: "label"
                                model: [{ label: i18n.t("disappearing_off"), v: 0 },
                                        { label: i18n.t("disappearing_24h"), v: 86400 },
                                        { label: i18n.t("disappearing_7d"), v: 604800 },
                                        { label: i18n.t("disappearing_90d"), v: 7776000 }]
                                onActivated: app.act("SetDisappearing", [win.selectedChat.id, model[currentIndex].v])
                            }
                        }
                        HoverHandler { id: disHov }
                    }
                    InfoSep {}

                    // Enkripsi (info-row dengan sub).
                    Rectangle {
                        Layout.fillWidth: true; implicitHeight: encCol.implicitHeight + 28; color: "transparent"
                        RowLayout {
                            anchors.fill: parent; anchors.leftMargin: 24; anchors.rightMargin: 24; spacing: 18
                            Icon { Layout.preferredWidth: 22; Layout.preferredHeight: 22; Layout.alignment: Qt.AlignTop; svg: win.ico["lock"]; color: theme.text2 }
                            ColumnLayout {
                                id: encCol; Layout.fillWidth: true; Layout.alignment: Qt.AlignVCenter; spacing: 2
                                Text { text: i18n.t("info_enc"); color: theme.text; font.pixelSize: 15 }
                                Text { Layout.fillWidth: true; wrapMode: Text.WordWrap; text: i18n.t("info_enc_sub"); color: theme.text2; font.pixelSize: 13 }
                            }
                        }
                    }
                    InfoSep {}

                    // Grup aksi danger (export / clear / exit / block / report).
                    ColumnLayout {
                        Layout.fillWidth: true; spacing: 0
                        // Export chat (netral).
                        Rectangle {
                            Layout.fillWidth: true; implicitHeight: 50; color: expHov.hovered ? theme.hover : "transparent"
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 24; anchors.rightMargin: 24; spacing: 18
                                Icon { Layout.preferredWidth: 22; Layout.preferredHeight: 22; svg: win.ico["download"]; color: theme.text2 }
                                Text { Layout.fillWidth: true; text: i18n.t("export_chat"); color: theme.text; font.pixelSize: 15 }
                            }
                            HoverHandler { id: expHov }
                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: app.fetchStr("ExportChat", [win.selectedChat.id]) }
                        }
                        // Clear chat (danger).
                        Rectangle {
                            Layout.fillWidth: true; implicitHeight: 50; color: clrHov.hovered ? theme.hover : "transparent"
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 24; anchors.rightMargin: 24; spacing: 18
                                Icon { Layout.preferredWidth: 22; Layout.preferredHeight: 22; svg: win.ico["clearchat"]; color: "#e35d6a" }
                                Text { Layout.fillWidth: true; text: i18n.t("clear_chat"); color: "#e35d6a"; font.pixelSize: 15 }
                            }
                            HoverHandler { id: clrHov }
                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: app.act("ClearChatMessages", [win.selectedChat.id]) }
                        }
                        // Keluar grup (danger).
                        Rectangle {
                            Layout.fillWidth: true; implicitHeight: 50; visible: detailPopup.isGroup; color: leaveHov.hovered ? theme.hover : "transparent"
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 24; anchors.rightMargin: 24; spacing: 18
                                Icon { Layout.preferredWidth: 22; Layout.preferredHeight: 22; svg: win.ico["leavegroup"]; color: "#e35d6a" }
                                Text { Layout.fillWidth: true; text: i18n.t("leave_group"); color: "#e35d6a"; font.pixelSize: 15 }
                            }
                            HoverHandler { id: leaveHov }
                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: { app.act("LeaveGroup", [win.selectedChat.id]); detailPopup.close() } }
                        }
                        // Blokir kontak (danger).
                        Rectangle {
                            Layout.fillWidth: true; implicitHeight: 50; visible: !detailPopup.isGroup; color: blkHov.hovered ? theme.hover : "transparent"
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 24; anchors.rightMargin: 24; spacing: 18
                                Icon { Layout.preferredWidth: 22; Layout.preferredHeight: 22; svg: win.ico["block"]; color: "#e35d6a" }
                                Text { Layout.fillWidth: true; text: i18n.t("block"); color: "#e35d6a"; font.pixelSize: 15 }
                            }
                            HoverHandler { id: blkHov }
                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: app.act("Block", [win.selectedChat.id, true]) }
                        }
                        // Laporkan kontak (danger).
                        Rectangle {
                            Layout.fillWidth: true; implicitHeight: 50; visible: !detailPopup.isGroup; color: rptHov.hovered ? theme.hover : "transparent"
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 24; anchors.rightMargin: 24; spacing: 18
                                Icon { Layout.preferredWidth: 22; Layout.preferredHeight: 22; svg: win.ico["clearchat"]; color: "#e35d6a" }
                                Text { Layout.fillWidth: true; text: i18n.t("report"); color: "#e35d6a"; font.pixelSize: 15 }
                            }
                            HoverHandler { id: rptHov }
                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: { app.act("Block", [win.selectedChat.id, true]); detailPopup.close() } }
                        }
                    }
                    // .local-note (contact): 11px text2, padding 10/16, line-height 1.4.
                    Text {
                        visible: !detailPopup.isGroup
                        Layout.fillWidth: true; Layout.leftMargin: 16; Layout.rightMargin: 16
                        Layout.topMargin: 10; Layout.bottomMargin: 10
                        text: i18n.t("local_label_note"); color: theme.text2; font.pixelSize: 11
                        wrapMode: Text.WordWrap; lineHeight: 1.4
                    }
                }
            }
        }
    }

    // === Teruskan pesan (pilih chat tujuan) — .fwd-modal/.fwd-* (app.css) ===
    Popup {
        id: forwardPopup
        // .fwd-modal: width 380, max-height 70vh, sidebar-bg, radius 12.
        width: 380; height: Math.min(win.height * 0.70, 520); modal: true
        anchors.centerIn: Overlay.overlay; padding: 0
        onOpened: fwdSearch.text = ""
        // .modal-backdrop: rgba(0,0,0,.4)
        Overlay.modal: Rectangle { color: "#66000000" }
        background: Rectangle {
            color: theme.sidebarBg; radius: 12; border.width: 0
            // .fwd-modal box-shadow: 0 8px 30px rgba(0,0,0,.4)
            layer.enabled: true
            layer.effect: DropShadow {
                horizontalOffset: 0; verticalOffset: 8; radius: 30; samples: 41
                color: Qt.rgba(0, 0, 0, 0.4)
            }
        }
        ColumnLayout {
            anchors.fill: parent; spacing: 0
            // .fwd-head: padding 16, font 17, w600
            Text {
                Layout.fillWidth: true; leftPadding: 16; rightPadding: 16; topPadding: 16; bottomPadding: 16
                text: i18n.t("forward_action"); color: theme.text; font.pixelSize: 17; font.weight: Font.DemiBold
            }
            // .fwd-search: margin 0 12 10, padding 8/12, radius 8, search-bg
            Rectangle {
                Layout.fillWidth: true; Layout.leftMargin: 12; Layout.rightMargin: 12; Layout.bottomMargin: 10
                height: 36; radius: 8; color: theme.searchBg
                TextInput {
                    id: fwdSearch
                    anchors.fill: parent; anchors.leftMargin: 12; anchors.rightMargin: 12
                    verticalAlignment: TextInput.AlignVCenter; clip: true
                    font.pixelSize: 14; color: theme.text
                    Text {
                        anchors.verticalCenter: parent.verticalCenter
                        visible: !fwdSearch.text && !fwdSearch.activeFocus
                        text: i18n.t("search"); color: theme.text2; font.pixelSize: 14
                    }
                }
            }
            // .fwd-list
            ListView {
                Layout.fillWidth: true; Layout.fillHeight: true; clip: true; model: chatsModel
                delegate: ItemDelegate {
                    width: ListView.view.width; clip: true
                    // filter .fwd-row: case-insensitive name contains query
                    visible: !fwdSearch.text || (model.m.name || "").toLowerCase().indexOf(fwdSearch.text.toLowerCase()) >= 0
                    height: visible ? 54 : 0
                    onClicked: { app.forwardMsg(win.ctxMsg.id, model.m.id); forwardPopup.close() }
                    background: Rectangle { color: hovered ? theme.hover : "transparent" }
                    RowLayout {
                        anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 16; spacing: 12
                        Avatar { Layout.preferredWidth: 40; Layout.preferredHeight: 40; fontSize: 16
                            name: model.m.name || ""; jid: model.m.id; base: app.mediaBase
                            accent: win.avatarColor(model.m.name || "?"); group: model.m.group === true }
                        Text { Layout.fillWidth: true; text: model.m.name || ""; color: theme.text; font.pixelSize: 15; elide: Text.ElideRight }
                    }
                }
            }
        }
    }

    // === Lightbox media (Lightbox.svelte) — fullscreen gambar + simpan/tutup/caption ===
    Rectangle {
        id: lightbox
        anchors.fill: parent; z: 150; visible: lightboxSrc !== "" || lightboxItems.length > 0
        color: "#eb000000"   // rgba(0,0,0,.92)
        // Album: items[] → prev/next; selain itu gambar tunggal (lightboxSrc).
        readonly property int count: lightboxItems.length
        readonly property string curSrc: count > 0
            ? ((lightboxItems[Math.max(0, Math.min(lightboxIndex, count - 1))].url) || "")
            : lightboxSrc
        function close() { win.lightboxSrc = ""; win.lightboxCaption = ""; win.lightboxItems = []; win.lightboxIndex = 0 }
        function prev() { if (win.lightboxIndex > 0) win.lightboxIndex-- }
        function next() { if (win.lightboxIndex < count - 1) win.lightboxIndex++ }
        // Klik backdrop → tutup (.lb on:click|self)
        MouseArea { anchors.fill: parent; onClicked: lightbox.close() }
        // .lb-media: max 94vw/90vh, radius 6
        Image {
            id: lbImg
            anchors.centerIn: parent
            width: Math.min(implicitWidth, parent.width * 0.94)
            height: Math.min(implicitHeight, parent.height * 0.90)
            fillMode: Image.PreserveAspectFit; source: lightbox.curSrc; smooth: true
        }
        // Placeholder bila media tak tersedia dari engine (guarded).
        Text {
            anchors.centerIn: parent
            visible: lightbox.visible && lbImg.status !== Image.Ready
            text: "🖼️"; color: "#cccccc"; opacity: 0.5; font.pixelSize: 48
        }
        // .lb-save: lingkaran 38, rgba(255,255,255,.12), kiri tombol-tutup (right:70)
        Rectangle {
            anchors.top: parent.top; anchors.topMargin: 18
            anchors.right: parent.right; anchors.rightMargin: 70
            width: 38; height: 38; radius: 19
            color: saveMa.containsMouse ? "#38ffffff" : "#1fffffff"
            Icon { anchors.centerIn: parent; width: 20; height: 20; svg: win.ico["download"]; color: "#ffffff" }
            MouseArea { id: saveMa; anchors.fill: parent; hoverEnabled: true
                onClicked: if (typeof app.saveMedia === "function") app.saveMedia(lightbox.curSrc) }
        }
        // .lb-nav.prev "‹" — kiri 18, 46×46 rgba(255,255,255,.12), font 28; album >1.
        Rectangle {
            visible: lightbox.count > 1 && lightboxIndex > 0
            anchors.verticalCenter: parent.verticalCenter
            anchors.left: parent.left; anchors.leftMargin: 18
            width: 46; height: 46; radius: 23
            color: prevMa.containsMouse ? "#38ffffff" : "#1fffffff"
            Text { anchors.centerIn: parent; text: "‹"; color: "#ffffff"; font.pixelSize: 28 }
            MouseArea { id: prevMa; anchors.fill: parent; hoverEnabled: true; onClicked: lightbox.prev() }
        }
        // .lb-nav.next "›" — kanan 18.
        Rectangle {
            visible: lightbox.count > 1 && lightboxIndex < lightbox.count - 1
            anchors.verticalCenter: parent.verticalCenter
            anchors.right: parent.right; anchors.rightMargin: 18
            width: 46; height: 46; radius: 23
            color: nextMa.containsMouse ? "#38ffffff" : "#1fffffff"
            Text { anchors.centerIn: parent; text: "›"; color: "#ffffff"; font.pixelSize: 28 }
            MouseArea { id: nextMa; anchors.fill: parent; hoverEnabled: true; onClicked: lightbox.next() }
        }
        // .lb-count "i+1/N" — atas 22, terpusat, font 13, opacity .8; album >1.
        Text {
            visible: lightbox.count > 1
            anchors.top: parent.top; anchors.topMargin: 22
            anchors.left: parent.left; anchors.right: parent.right
            horizontalAlignment: Text.AlignHCenter
            text: (lightboxIndex + 1) + "/" + lightbox.count
            color: "#ffffff"; opacity: 0.8; font.pixelSize: 13
        }
        // .lb-x: lingkaran 38, rgba(255,255,255,.12), pojok kanan-atas (right:22)
        Rectangle {
            anchors.top: parent.top; anchors.topMargin: 18
            anchors.right: parent.right; anchors.rightMargin: 22
            width: 38; height: 38; radius: 19
            color: xMa.containsMouse ? "#38ffffff" : "#1fffffff"
            Icon { anchors.centerIn: parent; width: 18; height: 18; svg: win.ico["close"]; color: "#ffffff" }
            MouseArea { id: xMa; anchors.fill: parent; hoverEnabled: true; onClicked: lightbox.close() }
        }
        // .lb-cap: caption bawah, terpusat, #fff
        Text {
            visible: win.lightboxCaption !== ""
            anchors.bottom: parent.bottom; anchors.bottomMargin: 26
            anchors.left: parent.left; anchors.right: parent.right
            anchors.leftMargin: 24; anchors.rightMargin: 24
            horizontalAlignment: Text.AlignHCenter; wrapMode: Text.WordWrap
            text: win.lightboxCaption; color: "#ffffff"; font.pixelSize: 14
        }
    }

    // === Pratinjau media sebelum kirim (MediaPreviewModal.svelte / .mp-*) ===
    // CATATAN: engine Qt mengirim media langsung (sendMediaFile, tanpa caption/
    // view-once param yg sampai backend). Shell visual ini faithful; caption +
    // view-once + edit (rotate/crop) belum ter-wire ke backend → guarded/visual.
    Popup {
        id: mediaPreviewPopup
        parent: Overlay.overlay
        width: parent ? parent.width : 0; height: parent ? parent.height : 0
        modal: true; padding: 0; closePolicy: Popup.CloseOnEscape
        background: Rectangle { color: "#f70b141a" }   // rgba(11,20,26,.97)
        readonly property var items: (win.mediaDraft && win.mediaDraft.items) ? win.mediaDraft.items : []
        readonly property var cur: items.length > win.mediaDraftIdx ? items[win.mediaDraftIdx] : null
        function doSend() {
            var d = win.mediaDraft
            if (d && d.items) {
                for (var i = 0; i < d.items.length; i++) {
                    // Caption + view-once tak diteruskan engine (sendMediaFile param tetap).
                    app.sendMediaFile(d.items[i].kind, d.items[i].url)
                }
            }
            mpCaption.text = ""
            mediaPreviewPopup.close()
        }
        ColumnLayout {
            anchors.fill: parent; spacing: 0
            // Header: tombol ✕ kiri-atas (.mp-x)
            Item {
                Layout.fillWidth: true; Layout.preferredHeight: 0; z: 2
                Rectangle {
                    x: 18; y: 16; width: 30; height: 30; radius: 15; color: "transparent"
                    Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["close"]; color: "#ffffff" }
                    MouseArea { anchors.fill: parent; onClicked: mediaPreviewPopup.close() }
                }
            }
            // .mp-stage: media terpusat
            Item {
                Layout.fillWidth: true; Layout.fillHeight: true
                Image {
                    anchors.centerIn: parent
                    width: Math.min(implicitWidth, parent.width * 0.94)
                    height: Math.min(implicitHeight, parent.height * 0.80)
                    fillMode: Image.PreserveAspectFit; smooth: true
                    source: mediaPreviewPopup.cur && mediaPreviewPopup.cur.kind === "image" ? mediaPreviewPopup.cur.url : ""
                    visible: mediaPreviewPopup.cur && mediaPreviewPopup.cur.kind === "image"
                }
                // Video: placeholder (tak ada MediaPlayer di shell ini → guarded).
                ColumnLayout {
                    anchors.centerIn: parent; spacing: 8
                    visible: mediaPreviewPopup.cur && mediaPreviewPopup.cur.kind === "video"
                    Icon { Layout.alignment: Qt.AlignHCenter; width: 64; height: 64; svg: win.ico["play"]; color: "#ffffff" }
                    Text { Layout.alignment: Qt.AlignHCenter; text: mediaPreviewPopup.cur ? (mediaPreviewPopup.cur.name || "video") : ""
                        color: "#ffffff"; font.pixelSize: 14 }
                }
            }
            // .mp-edit: rotate/rotate/crop (hanya image). Visual; belum bake ke backend.
            RowLayout {
                Layout.alignment: Qt.AlignHCenter; Layout.topMargin: 6; spacing: 10
                visible: mediaPreviewPopup.cur && mediaPreviewPopup.cur.kind === "image"
                Repeater {
                    model: [ {ic: "rotate", tip: "rotate", flip: false},
                             {ic: "rotate", tip: "rotate", flip: true},
                             {ic: "crop", tip: "crop", flip: false} ]
                    delegate: Rectangle {
                        Layout.preferredWidth: 44; Layout.preferredHeight: 38; radius: 20
                        color: ebMa.containsMouse ? "#2effffff" : "#24ffffff"
                        Icon { anchors.centerIn: parent; width: 20; height: 20; svg: win.ico[modelData.ic]; color: "#ffffff"
                            transform: Scale { origin.x: 10; xScale: modelData.flip ? -1 : 1 } }
                        MouseArea { id: ebMa; anchors.fill: parent; hoverEnabled: true }
                    }
                }
            }
            // .mp-strip: thumbnail (hanya bila >1 item)
            RowLayout {
                Layout.alignment: Qt.AlignHCenter; Layout.topMargin: 6; Layout.bottomMargin: 6; spacing: 8
                visible: mediaPreviewPopup.items.length > 1
                Repeater {
                    model: mediaPreviewPopup.items
                    delegate: Rectangle {
                        // .mp-thumb: background none + border 2px transparent; .on → accent border.
                        Layout.preferredWidth: 54; Layout.preferredHeight: 54; radius: 8; clip: true
                        color: "transparent"
                        border.width: 2; border.color: index === win.mediaDraftIdx ? theme.accent : "transparent"
                        Image { anchors.fill: parent; anchors.margins: 2; fillMode: Image.PreserveAspectCrop
                            source: modelData.kind === "image" ? modelData.url : "" }
                        MouseArea { anchors.fill: parent; onClicked: win.mediaDraftIdx = index }
                    }
                }
            }
            // .mp-bar: caption + view-once + send
            RowLayout {
                Layout.fillWidth: true; Layout.maximumWidth: 760; Layout.alignment: Qt.AlignHCenter
                Layout.leftMargin: 18; Layout.rightMargin: 18; Layout.topMargin: 14; Layout.bottomMargin: 22
                spacing: 10
                // .mp-caption: pill bg2, radius 22
                Rectangle {
                    Layout.fillWidth: true; height: 46; radius: 22; color: theme.bg2
                    TextInput {
                        id: mpCaption
                        anchors.fill: parent; anchors.leftMargin: 18; anchors.rightMargin: 18
                        verticalAlignment: TextInput.AlignVCenter; clip: true
                        font.pixelSize: 14; color: theme.text
                        Text { anchors.verticalCenter: parent.verticalCenter
                            visible: !mpCaption.text && !mpCaption.activeFocus
                            text: i18n.t("add_caption"); color: theme.text2; font.pixelSize: 14 }
                    }
                }
                // .mp-once: lingkaran 48, toggle view-once
                Rectangle {
                    Layout.preferredWidth: 48; Layout.preferredHeight: 48; radius: 24
                    color: win.mediaDraftOnce ? theme.accent : "#24ffffff"
                    Icon { anchors.centerIn: parent; width: 24; height: 24; svg: win.ico["viewonce"]; color: "#ffffff" }
                    Text { anchors.centerIn: parent; text: "1"; color: "#ffffff"; font.pixelSize: 11; font.bold: true }
                    MouseArea { anchors.fill: parent; onClicked: win.mediaDraftOnce = !win.mediaDraftOnce }
                }
                // .mp-send: lingkaran 48 accent
                Rectangle {
                    Layout.preferredWidth: 48; Layout.preferredHeight: 48; radius: 24; color: theme.accent
                    Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["send"]; color: "#ffffff"; fill: "currentColor" }
                    Rectangle {
                        visible: mediaPreviewPopup.items.length > 1
                        anchors.top: parent.top; anchors.right: parent.right; anchors.topMargin: -4; anchors.rightMargin: -4
                        width: 18; height: 18; radius: 9; color: "#ffffff"
                        Text { anchors.centerIn: parent; text: mediaPreviewPopup.items.length; color: theme.accent; font.pixelSize: 11; font.bold: true }
                    }
                    MouseArea { anchors.fill: parent; onClicked: mediaPreviewPopup.doSend() }
                }
            }
        }
        // Esc / tutup → buang draf.
        onClosed: { win.mediaDraft = null; win.mediaDraftIdx = 0; win.mediaDraftOnce = false }
    }

    // === App-lock (PIN) ===
    Rectangle {
        anchors.fill: parent; z: 200; visible: locked; color: theme.bg
        ColumnLayout {
            anchors.centerIn: parent; spacing: 16; width: 260
            Text { Layout.alignment: Qt.AlignHCenter; text: "🔒"; font.pixelSize: 44 }
            Text { Layout.alignment: Qt.AlignHCenter; text: i18n.t("enter_pin"); color: theme.text; font.pixelSize: 16 }
            Rectangle {
                Layout.alignment: Qt.AlignHCenter; width: 160; height: 46; radius: 10
                color: theme.searchBg; border.color: theme.line
                TextInput {
                    id: pinInput; anchors.fill: parent; anchors.margins: 10
                    echoMode: TextInput.Password; horizontalAlignment: TextInput.AlignHCenter
                    font.pixelSize: 20; color: theme.text
                    onTextChanged: if (text === "1234") { win.locked = false; text = "" }
                }
            }
            Text { Layout.alignment: Qt.AlignHCenter; text: "demo: 1234"; color: theme.text2; font.pixelSize: 11 }
        }
    }

    // === Pemilih reaksi emoji (ReactionPicker.svelte / .rp-pop) ===
    // Svelte memakai <emoji-picker> 352×400 (grid emoji penuh) → di Qt direplikasi
    // sbg grid emoji bertema. Pilih → app.react(...). Bukan menu daftar siapa-react
    // (itu reactionDetailPopup). Quick-reaction row di atas + grid kategori di bawah.
    Popup {
        id: reactionPopup
        // .rp-pop: width min(352,92vw), height 400, radius 14, bg --bg
        width: Math.min(352, win.width * 0.92); height: 400; modal: true
        padding: 0
        // position:fixed di sebelah tombol asal (GAP=6), bukan center. Clamp ke dalam window.
        x: win.reactX < 0 ? (win.width - width) / 2
                          : Math.max(8, Math.min(win.reactX + 6, win.width - width - 8))
        y: win.reactY < 0 ? (win.height - height) / 2
                          : Math.max(8, Math.min(win.reactY, win.height - height - 8))
        Overlay.modal: Rectangle { color: "transparent" }  // .rp-backdrop transparan
        // .rp-pop: radius 14, box-shadow var(--shadow-lg), TANPA border (Svelte hanya shadow).
        background: Rectangle {
            color: theme.bg; radius: 14; clip: true
            layer.enabled: true
            layer.effect: DropShadow {
                horizontalOffset: 0; verticalOffset: 12; radius: 32; samples: 41
                color: Qt.rgba(0, 0, 0, theme.dark ? 0.55 : 0.16)
            }
        }
        // Daftar emoji ala WhatsApp quick-react + kategori umum.
        readonly property var quick: ["👍","❤️","😂","😮","😢","🙏"]
        readonly property var grid: [
            "👍","❤️","😂","😮","😢","🙏","🔥","🎉","👏","😍","😅","😎",
            "😭","😡","🥳","🤔","😴","🤗","😘","😜","🙄","😱","🤩","😏",
            "👌","✌️","🤝","💪","🙌","👀","💯","✨","⭐","🌟","💔","💕",
            "😊","😉","😋","😇","🤣","😆","🥰","😻","🤤","😬","😤","😩"
        ]
        // insertMode=true → sisip emoji ke composer (tombol emoji Composer.svelte),
        // selain itu → react ke pesan (ReactionPicker).
        property bool insertMode: false
        function send(em) {
            if (insertMode) { composerInput.text += em; reactionPopup.close(); return }
            app.react(win.ctxMsg.id, win.ctxMsg.senderId || "", win.ctxMsg.dir === "out", em)
            reactionPopup.close()
        }
        // ReactionPicker.svelte: HANYA <emoji-picker> mengisi .rp-pop — tanpa quick-row/'+'/divider.
        GridView {
            anchors.fill: parent; clip: true
            cellWidth: width / 6; cellHeight: 44; model: reactionPopup.grid
            delegate: ItemDelegate {
                width: GridView.view.cellWidth; height: 44
                background: Rectangle { color: hovered ? theme.hover : "transparent" }
                contentItem: Text { text: modelData; font.pixelSize: 22
                    horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter }
                onClicked: reactionPopup.send(modelData)
            }
            ScrollIndicator.vertical: ScrollIndicator {}
        }
    }

    // === Detail reaksi (siapa react apa) — daftar dari menu "Reaksi" ===
    Popup {
        id: reactionDetailPopup
        width: 300; height: 280; modal: true; anchors.centerIn: Overlay.overlay; padding: 12
        background: Rectangle { color: theme.bg; radius: 14; border.color: theme.line }
        ColumnLayout {
            anchors.fill: parent; spacing: 8
            Text { text: i18n.t("reactions"); color: theme.text; font.pixelSize: 16; font.bold: true }
            ListView {
                Layout.fillWidth: true; Layout.fillHeight: true; clip: true
                model: win.ctxMsg.reactions || []
                delegate: RowLayout {
                    width: ListView.view.width; height: 40; clip: true; spacing: 10
                    Text { text: modelData.emoji; font.pixelSize: 20 }
                    Text { Layout.fillWidth: true; text: (modelData.who || []).join(", "); color: theme.text; font.pixelSize: 14 }
                    Text { text: modelData.count; color: theme.text2; font.pixelSize: 13 }
                }
            }
        }
    }

    // === Info pesan (MessageInfoModal.svelte / .nc-card + .mi-*) ===
    // .nc-card: sidebar-bg, radius 14, padding 18, max-width 380.
    // Grid status/type/sent/from/ID: field2 tak ada di mock engine (hanya readBy/
    // deliveredTo) → dibungkus guard; muncul hanya bila app.detail menyediakannya.
    Popup {
        id: msgInfoPopup
        width: 380; height: Math.min(win.height * 0.84, 480); modal: true
        anchors.centerIn: Overlay.overlay; padding: 18
        Overlay.modal: Rectangle { color: "#66000000" }   // .nc-overlay rgba(0,0,0,.4)
        background: Rectangle {
            color: theme.sidebarBg; radius: 14; border.width: 0
            // .nc-card box-shadow var(--shadow-lg): 0 12px 32px rgba(0,0,0,.55) dark / .18 light
            layer.enabled: true
            layer.effect: DropShadow {
                horizontalOffset: 0; verticalOffset: 12; radius: 32; samples: 49
                color: Qt.rgba(0, 0, 0, theme.dark ? 0.55 : 0.18)
            }
        }
        // Peta status → dot color (app.css .mi-dot.delivered #3d8bd3 / .read accent).
        function statusLabel(s) {
            return s === "read" ? i18n.t("status_read")
                 : s === "delivered" ? i18n.t("status_delivered") : i18n.t("status_sent")
        }
        function typeLabel(t) {
            var m = { text:"t_text", image:"t_photo", video:"t_video", sticker:"t_sticker", voice:"t_voice" }
            return m[t] ? i18n.t(m[t]) : (t || "")
        }
        ColumnLayout {
            anchors.fill: parent; spacing: 0
            // h3 judul (margin 0 0 14)
            Text {
                Layout.fillWidth: true; bottomPadding: 14
                text: i18n.t("msg_info"); color: theme.text; font.pixelSize: 19; font.weight: Font.Bold
            }
            Flickable {
                Layout.fillWidth: true; Layout.fillHeight: true; clip: true
                contentHeight: miCol.implicitHeight; flickableDirection: Flickable.VerticalFlick
                ColumnLayout {
                    id: miCol; width: parent.width; spacing: 0
                    // .mi-grid (2 kolom auto/1fr, gap 8 16). Field di-guard pada app.detail.
                    GridLayout {
                        Layout.fillWidth: true; columns: 2; columnSpacing: 16; rowSpacing: 8
                        // Status (dengan dot)
                        Text { visible: !!app.detail.status; text: i18n.t("mi_status"); color: theme.text2; font.pixelSize: 13 }
                        RowLayout { visible: !!app.detail.status; spacing: 7
                            Rectangle { width: 8; height: 8; radius: 4
                                color: app.detail.status === "read" ? theme.accent
                                     : app.detail.status === "delivered" ? "#3d8bd3" : theme.text2 }
                            Text { text: msgInfoPopup.statusLabel(app.detail.status); color: theme.text; font.pixelSize: 14 }
                        }
                        // Tipe
                        Text { visible: !!app.detail.type; text: i18n.t("mi_type"); color: theme.text2; font.pixelSize: 13 }
                        Text { visible: !!app.detail.type; text: msgInfoPopup.typeLabel(app.detail.type); color: theme.text; font.pixelSize: 14 }
                        // Terkirim (waktu)
                        Text { visible: !!app.detail.sent; text: i18n.t("mi_sent"); color: theme.text2; font.pixelSize: 13 }
                        Text { visible: !!app.detail.sent; text: app.detail.sent || ""; color: theme.text; font.pixelSize: 14 }
                        // Dari (pengirim, bila bukan dari-saya)
                        Text { visible: !!app.detail.sender && !app.detail.fromMe; text: i18n.t("mi_from"); color: theme.text2; font.pixelSize: 13 }
                        Text { visible: !!app.detail.sender && !app.detail.fromMe; text: app.detail.sender || ""; color: theme.text; font.pixelSize: 14 }
                        // ID
                        Text { visible: !!app.detail.id; text: "ID"; color: theme.text2; font.pixelSize: 13 }
                        Text { visible: !!app.detail.id; Layout.fillWidth: true; text: app.detail.id || ""
                            color: theme.text; font.pixelSize: 11; font.family: "monospace"; wrapMode: Text.WrapAnywhere }
                    }
                    // .mi-sec Dibaca oleh (dot accent)
                    RowLayout {
                        visible: (app.detail.readBy || []).length > 0
                        Layout.topMargin: 14; Layout.bottomMargin: 6; spacing: 7
                        Rectangle { width: 8; height: 8; radius: 4; color: theme.accent }
                        Text { text: i18n.t("mi_read_by").toUpperCase(); color: theme.text2
                            font.pixelSize: 12; font.weight: Font.DemiBold; font.letterSpacing: 0.4 }
                    }
                    Repeater {
                        model: app.detail.readBy || []
                        delegate: RowLayout {
                            Layout.fillWidth: true; Layout.topMargin: 3; Layout.bottomMargin: 3
                            Text { Layout.fillWidth: true; text: modelData.name || ""; color: theme.text; font.pixelSize: 14 }
                            Text { text: modelData.time || ""; color: theme.text2; font.pixelSize: 12 }
                        }
                    }
                    // .mi-sec Terkirim ke (dot #3d8bd3)
                    RowLayout {
                        visible: (app.detail.deliveredTo || []).length > 0
                        Layout.topMargin: 14; Layout.bottomMargin: 6; spacing: 7
                        Rectangle { width: 8; height: 8; radius: 4; color: "#3d8bd3" }
                        Text { text: i18n.t("mi_delivered_to").toUpperCase(); color: theme.text2
                            font.pixelSize: 12; font.weight: Font.DemiBold; font.letterSpacing: 0.4 }
                    }
                    Repeater {
                        model: app.detail.deliveredTo || []
                        delegate: RowLayout {
                            Layout.fillWidth: true; Layout.topMargin: 3; Layout.bottomMargin: 3
                            Text { Layout.fillWidth: true; text: modelData.name || ""; color: theme.text; font.pixelSize: 14 }
                            Text { text: modelData.time || ""; color: theme.text2; font.pixelSize: 12 }
                        }
                    }
                    // .mi-note bila tak ada penerima yg baca/terkirim
                    Text {
                        visible: (app.detail.readBy || []).length === 0 && (app.detail.deliveredTo || []).length === 0
                        Layout.fillWidth: true; Layout.topMargin: 14
                        text: i18n.t("mi_note"); color: theme.text2; font.pixelSize: 12; wrapMode: Text.WordWrap; lineHeight: 1.5
                    }
                }
            }
            // .btn-accent close, justify flex-end, margin-top 16
            Btn { Layout.alignment: Qt.AlignRight; Layout.topMargin: 16; accent: true
                text: i18n.t("close"); onClicked: msgInfoPopup.close() }
        }
    }

    // === Privasi (PrivacyPane.svelte) — surface bg, pv-row + toggle + blocklist ===
    Popup {
        id: privacyPopup
        // Pane kiri full-height (PrivacyPane.svelte di .sidebar), bukan modal tengah.
        x: 0; y: 0; width: 400; height: win.height; modal: true; padding: 0
        background: Rectangle { color: theme.sidebarBg; border.width: 0; clip: true }
        // Saat dibuka: muat daftar diblokir ke searchModel (transient; tak aktif
        // saat popup terbuka). Engine kembalikan [] bila tak ada → empty state.
        onAboutToShow: app.loadInto("GetBlockedContacts", searchModel)

        // opsi visibilitas (nilai native engine: everyone/contacts/nobody).
        readonly property var pvVals: ["everyone", "contacts", "nobody"]
        readonly property var pvKeys: ["pv_everyone", "pv_contacts", "pv_nobody"]
        function pvLabels() { return [i18n.t("pv_everyone"), i18n.t("pv_contacts"), i18n.t("pv_nobody")] }

        ColumnLayout {
            anchors.fill: parent; spacing: 0
            // .pane-head (head-bg, 56px, gap 14) dgn tombol back + judul.
            Rectangle {
                Layout.fillWidth: true; Layout.preferredHeight: 56; color: theme.headBg
                radius: 14
                Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 14; color: theme.headBg } // ratakan sudut bawah
                RowLayout {
                    anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 16; spacing: 14
                    ItemDelegate {
                        Layout.preferredWidth: 36; Layout.preferredHeight: 36; hoverEnabled: true
                        background: Rectangle { radius: width / 2; color: parent.hovered ? theme.hover : "transparent" }
                        onClicked: { privacyPopup.close(); settingsPopup.open() }
                        Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["chevleft"]; color: theme.text }
                    }
                    Text { Layout.fillWidth: true; text: i18n.t("privacy_title"); color: theme.text
                        font.pixelSize: 17; font.weight: Font.DemiBold }
                }
            }

            Flickable {
                Layout.fillWidth: true; Layout.fillHeight: true; clip: true
                contentHeight: pvCol.implicitHeight; flickableDirection: Flickable.VerticalFlick
                ScrollIndicator.vertical: ScrollIndicator {}
                ColumnLayout {
                    id: pvCol; width: parent.width; spacing: 0

                    // 4 baris .pv-row dgn Combo bertema.
                    Repeater {
                        model: [
                            { key: "lastseen", label: "pv_last_seen" },
                            { key: "profile",  label: "pv_profile_photo" },
                            { key: "status",   label: "pv_status" },
                            { key: "groupadd", label: "pv_groups" }
                        ]
                        delegate: Rectangle {
                            Layout.fillWidth: true
                            implicitHeight: 50 // padding 13 + 24 control + 13
                            color: "transparent"
                            Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: theme.line }
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 16; spacing: 12
                                Text { Layout.fillWidth: true; text: i18n.t(modelData.label)
                                    color: theme.text; font.pixelSize: 15 } // 14.5 → 15 (int)
                                Combo {
                                    implicitWidth: 140
                                    model: privacyPopup.pvLabels()
                                    currentIndex: Math.max(0, privacyPopup.pvVals.indexOf(app.detail[modelData.key] || "everyone"))
                                    onActivated: app.setPrivacy(modelData.key, privacyPopup.pvVals[currentIndex])
                                }
                            }
                        }
                    }

                    // Baris .pv-row read-receipts dgn Tog (off bila "none"/"nobody").
                    Rectangle {
                        Layout.fillWidth: true; implicitHeight: 50; color: "transparent"
                        Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: theme.line }
                        RowLayout {
                            anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 16; spacing: 12
                            Text { Layout.fillWidth: true; text: i18n.t("pv_read_receipts")
                                color: theme.text; font.pixelSize: 15 }
                            Tog {
                                checked: (app.detail.readreceipts || "all") !== "none" && (app.detail.readreceipts || "all") !== "nobody"
                                onToggled: app.setPrivacy("readreceipts", checked ? "all" : "none")
                            }
                        }
                    }

                    // .list-label "Kontak diblokir (N)" — pad 8 16 4, 12.5px text2.
                    Text {
                        Layout.fillWidth: true; Layout.topMargin: 14
                        leftPadding: 16; rightPadding: 16; topPadding: 8; bottomPadding: 4
                        text: i18n.t("pv_blocked") + " (" + blockedRep.count + ")"
                        color: theme.text2; font.pixelSize: 13; font.weight: Font.DemiBold
                    }

                    // Daftar diblokir (.pv-blocked) — pad 9 16, av 38, btn-ghost unblock.
                    Repeater {
                        id: blockedRep
                        model: searchModel
                        delegate: Rectangle {
                            Layout.fillWidth: true; implicitHeight: 56; color: "transparent"
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 16; spacing: 12
                                Avatar {   // .pv-av: flat colored circle (colorFor(jid)) + initial, no photo
                                    Layout.preferredWidth: 38; Layout.preferredHeight: 38; fontSize: 15
                                    weight: Font.DemiBold   // .pv-av font-weight 600 (umum .avatar 700)
                                    name: model.m.name; jid: model.m.jid; base: ""
                                    accent: win.avatarColor(model.m.jid || model.m.name || "?")
                                }
                                Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1
                                    text: model.m.name || ""; color: theme.text; font.pixelSize: 15 }
                                Btn {
                                    text: i18n.t("pv_unblock")
                                    onClicked: { app.act("Block", [model.m.jid, false]); app.loadInto("GetBlockedContacts", searchModel); win.toast(i18n.t("toast_unblocked"), "ok") }
                                }
                            }
                        }
                    }

                    // Empty state (.empty-list) — pad 14 16, 13px text2.
                    Text {
                        visible: blockedRep.count === 0
                        Layout.fillWidth: true; leftPadding: 16; rightPadding: 16; topPadding: 14; bottomPadding: 14
                        text: i18n.t("pv_no_blocked"); color: theme.text2; font.pixelSize: 13
                    }
                    Item { Layout.preferredHeight: 12 }
                }
            }
        }
    }

    // === Penyimpanan (StoragePane.svelte) — ringkasan + bar per-jenis ===
    Popup {
        id: storagePopup
        // Pane kiri full-height (StoragePane.svelte di .sidebar), bukan modal tengah.
        x: 0; y: 0; width: 400; height: win.height; modal: true; padding: 0
        background: Rectangle { color: theme.sidebarBg; border.width: 0; clip: true }

        // fmt byte (StoragePane.svelte fmt()).
        function fmt(b) {
            if (!b) return "0 B"
            var k = 1024
            if (b < k) return b + " B"
            if (b < k * k) return Math.round(b / k) + " KB"
            if (b < k * k * k) return (b / k / k).toFixed(1) + " MB"
            return (b / k / k / k).toFixed(2) + " GB"
        }
        // label jenis (StoragePane.svelte label map).
        function kindName(kk) {
            var m = { text:"t_text", image:"t_photo", video:"t_video", sticker:"t_sticker",
                      voice:"t_voice", gif:"t_video", document:"attach_document" }
            return m[kk] ? i18n.t(m[kk]) : kk
        }
        // msgCount real engine; mock pakai "messages" → fallback.
        readonly property real total: (app.detail.dbBytes || 0) + (app.detail.mediaBytes || 0)
        readonly property int msgCount: (app.detail.msgCount !== undefined ? app.detail.msgCount
                                        : (app.detail.messages || 0))

        ColumnLayout {
            anchors.fill: parent; spacing: 0
            // .pane-head (gap 22 di Svelte).
            Rectangle {
                Layout.fillWidth: true; Layout.preferredHeight: 56; color: theme.headBg; radius: 14
                Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 14; color: theme.headBg }
                RowLayout {
                    anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 16; spacing: 22
                    ItemDelegate {
                        Layout.preferredWidth: 36; Layout.preferredHeight: 36; hoverEnabled: true
                        background: Rectangle { radius: width / 2; color: parent.hovered ? theme.hover : "transparent" }
                        onClicked: { storagePopup.close(); settingsPopup.open() }
                        Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["chevleft"]; color: theme.text }
                    }
                    Text { Layout.fillWidth: true; text: i18n.t("storage_title")
                        color: theme.text; font.pixelSize: 17; font.weight: Font.DemiBold }
                }
            }

            Flickable {
                Layout.fillWidth: true; Layout.fillHeight: true; clip: true
                contentHeight: suCol.implicitHeight; flickableDirection: Flickable.VerticalFlick
                ScrollIndicator.vertical: ScrollIndicator {}
                ColumnLayout {
                    id: suCol; width: parent.width; spacing: 0

                    // .su-totals (text-align center, pad 18 16)
                    ColumnLayout {
                        Layout.fillWidth: true; Layout.topMargin: 18; Layout.bottomMargin: 0; spacing: 0
                        // .su-total 30px bold
                        Text { Layout.alignment: Qt.AlignHCenter; text: storagePopup.fmt(storagePopup.total)
                            color: theme.text; font.pixelSize: 30; font.weight: Font.Bold }
                        Text { Layout.alignment: Qt.AlignHCenter; Layout.topMargin: 2; text: i18n.t("storage")
                            color: theme.text2; font.pixelSize: 13 }
                        // .su-split (gap 28, mt 14): DB / Media dlm aksen
                        RowLayout {
                            Layout.alignment: Qt.AlignHCenter; Layout.topMargin: 14; spacing: 28
                            ColumnLayout { spacing: 0
                                Text { Layout.alignment: Qt.AlignHCenter; text: storagePopup.fmt(app.detail.dbBytes || 0)
                                    color: theme.accent; font.pixelSize: 16; font.weight: Font.Bold }
                                Text { Layout.alignment: Qt.AlignHCenter; text: i18n.t("messages_label")
                                    color: theme.text2; font.pixelSize: 12 }
                            }
                            ColumnLayout { spacing: 0
                                Text { Layout.alignment: Qt.AlignHCenter; text: storagePopup.fmt(app.detail.mediaBytes || 0)
                                    color: theme.accent; font.pixelSize: 16; font.weight: Font.Bold }
                                Text { Layout.alignment: Qt.AlignHCenter; text: i18n.t("info_media")
                                    color: theme.text2; font.pixelSize: 12 }
                            }
                        }
                        // .su-count (mt 10, 12.5px)
                        Text { Layout.alignment: Qt.AlignHCenter; Layout.topMargin: 10
                            text: storagePopup.msgCount.toLocaleString(Qt.locale(), "f", 0) + " " + i18n.t("messages_label")
                            color: theme.text2; font.pixelSize: 13 }
                    }

                    // .su-list (pad 8 16) — baris per-jenis dgn bar proporsional.
                    ColumnLayout {
                        Layout.fillWidth: true; Layout.topMargin: 8; Layout.leftMargin: 16; Layout.rightMargin: 16; spacing: 0
                        Repeater {
                            model: app.detail.kinds || []
                            delegate: RowLayout {
                                Layout.fillWidth: true; Layout.topMargin: 8; Layout.bottomMargin: 8; spacing: 10
                                Text { Layout.preferredWidth: 90; text: storagePopup.kindName(modelData.kind)
                                    color: theme.text; font.pixelSize: 13; elide: Text.ElideRight }
                                Rectangle { // .su-bar track
                                    Layout.fillWidth: true; Layout.preferredHeight: 8; radius: 5; color: theme.bg2; clip: true
                                    Rectangle { // fill
                                        height: parent.height; radius: 5; color: theme.accent
                                        width: parent.width * (storagePopup.total
                                            ? Math.max(0.02, (modelData.bytes / storagePopup.total)) : 0)
                                    }
                                }
                                Text { text: storagePopup.fmt(modelData.bytes) + " · " + modelData.count
                                    color: theme.text2; font.pixelSize: 12 }
                            }
                        }
                    }
                    Item { Layout.preferredHeight: 12 }
                }
            }
        }
    }

    // === Pemilih folder per-chat (FolderPicker.svelte) — .nc-modal kartu ===
    Popup {
        id: folderPickerPopup
        modal: true; anchors.centerIn: Overlay.overlay; padding: 0
        width: 400; height: fpCol.implicitHeight
        Overlay.modal: Rectangle { color: "#66000000" } // .modal-backdrop rgba(0,0,0,.4)
        background: Rectangle { color: theme.sidebarBg; radius: 14; border.width: 0; clip: true }
        onClosed: win.folderPickFor = ""

        ColumnLayout {
            id: fpCol
            anchors.fill: parent; spacing: 0

            // .nc-head — judul "Folders · <chat>" + tombol tutup.
            RowLayout {
                Layout.fillWidth: true; Layout.leftMargin: 16; Layout.rightMargin: 16
                Layout.topMargin: 14; Layout.bottomMargin: 14; spacing: 10
                Text {
                    Layout.fillWidth: true; elide: Text.ElideRight
                    text: i18n.t("folders") + (win.folderPickName !== "" ? " · " + win.folderPickName : "")
                    color: theme.text; font.pixelSize: 17; font.weight: Font.DemiBold
                }
                ItemDelegate {
                    Layout.preferredWidth: 32; Layout.preferredHeight: 32; hoverEnabled: true
                    background: Rectangle { radius: width / 2; color: parent.hovered ? theme.hover : "transparent" }
                    onClicked: folderPickerPopup.close()
                    Icon { anchors.centerIn: parent; width: 20; height: 20; svg: win.ico["close"]; color: theme.text2 }
                }
            }

            // .fp-list (max-height 50vh, scroll). Baris .fp-row: checkbox + nama + ✕.
            Flickable {
                Layout.fillWidth: true
                Layout.leftMargin: 6; Layout.rightMargin: 6
                Layout.preferredHeight: Math.min(fpList.implicitHeight, win.height * 0.5)
                clip: true; contentHeight: fpList.implicitHeight
                flickableDirection: Flickable.VerticalFlick
                ScrollIndicator.vertical: ScrollIndicator {}
                ColumnLayout {
                    id: fpList; width: parent.width; spacing: 0
                    Repeater {
                        model: win.folders
                        delegate: ItemDelegate {
                            Layout.fillWidth: true; implicitHeight: 44; hoverEnabled: true
                            background: Rectangle { anchors.margins: 2; radius: 8; color: hovered ? theme.bg2 : "transparent" }
                            onClicked: win.toggleChatFolder(modelData.name, win.folderPickFor)
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 8; anchors.rightMargin: 8; spacing: 10
                                // checkbox (kotak; centang aksen bila chat ada di folder)
                                Rectangle {
                                    Layout.preferredWidth: 18; Layout.preferredHeight: 18; radius: 4
                                    border.width: 2
                                    property bool on: win.chatInFolder(modelData.name, win.folderPickFor)
                                    border.color: on ? theme.accent : theme.text2
                                    color: on ? theme.accent : "transparent"
                                    Icon { anchors.centerIn: parent; width: 12; height: 12; visible: parent.on
                                        svg: win.ico["check"]; color: "#ffffff"; vbox: "0 0 17 15" }
                                }
                                Text { Layout.fillWidth: true; text: modelData.name; color: theme.text
                                    font.pixelSize: 15; elide: Text.ElideRight }
                                // .fp-del: bare text2 ✕, background:none (no per-button hover circle)
                                ItemDelegate {
                                    Layout.preferredWidth: 28; Layout.preferredHeight: 28; hoverEnabled: true
                                    background: null
                                    onClicked: win.deleteFolder(modelData.name)
                                    Icon { anchors.centerIn: parent; width: 14; height: 14; svg: win.ico["close"]; color: theme.text2 }
                                }
                            }
                        }
                    }
                    // .fp-empty — "No folders yet"
                    Text {
                        visible: win.folders.length === 0
                        Layout.fillWidth: true; topPadding: 16; bottomPadding: 16
                        horizontalAlignment: Text.AlignHCenter
                        text: i18n.t("no_folders"); color: theme.text2; font.pixelSize: 13
                    }
                }
            }

            // .nc-create — "+ New folder": pill penuh (--r-pill 999), pad 11, margin 10 12, font 15/600.
            Rectangle {
                id: ncCreate
                Layout.fillWidth: true; Layout.topMargin: 10; Layout.bottomMargin: 10
                Layout.leftMargin: 12; Layout.rightMargin: 12
                implicitHeight: ncTxt.implicitHeight + 22   // 11 atas + 11 bawah
                radius: height / 2
                color: ncHov.hovered ? theme.accentDeep : theme.accent
                Text { id: ncTxt; anchors.centerIn: parent; text: "+ " + i18n.t("folder_new")
                    color: "#ffffff"; font.pixelSize: 15; font.weight: Font.DemiBold }
                HoverHandler { id: ncHov }
                MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                    onClicked: win.prompt(i18n.t("folder_new"), "", function(v){ if (v) win.addFolder(v) }) }
            }
        }
    }

    // === Kirim dokumen: pilih file → rename/cut → kirim ===
    FileDialog {
        id: docDialog
        title: "Pilih dokumen"
        onAccepted: { docName.text = selectedFile.toString().split("/").pop(); docPopup.open() }
        property url picked: selectedFile
    }
    Popup {
        id: docPopup
        width: 360; height: 200; modal: true; anchors.centerIn: Overlay.overlay; padding: 16
        background: Rectangle { color: theme.bg; radius: 14; border.color: theme.line }
        ColumnLayout {
            anchors.fill: parent; spacing: 12
            Text { text: i18n.t("send_document"); color: theme.text; font.pixelSize: 16; font.bold: true }
            Text { text: i18n.t("rename_optional"); color: theme.text2; font.pixelSize: 12 }
            Rectangle {
                Layout.fillWidth: true; height: 40; radius: 8; color: theme.searchBg; border.color: theme.line
                TextInput { id: docName; anchors.fill: parent; anchors.margins: 10; color: theme.text; font.pixelSize: 14; clip: true }
            }
            Item { Layout.fillHeight: true }
            RowLayout {
                Layout.alignment: Qt.AlignRight; spacing: 8
                Btn { text: i18n.t("cancel"); onClicked: docPopup.close() }
                Btn { accent: true;
                    text: i18n.t("send")
                    onClicked: { app.sendDocument(docDialog.picked, docName.text); docPopup.close() }
                }
            }
        }
    }

    // === Menu lampiran compose (.attach-menu) — paritas Composer.svelte: 8 item
    //     dgn line-SVG accent 20px + teks 14.5px, urutan Svelte:
    //     attach_media / attach_location / attach_poll / GIF / attach_sticker /
    //     attach_contact / quick_replies / attach_document.
    TMenu {
        id: attachMenu
        TMenuItem { iconName: "herophoto"; text: i18n.t("attach_media"); onTriggered: { mediaDialog.kind = "image"; mediaDialog.open() } }
        TMenuItem { iconName: "locpin"; text: i18n.t("attach_location"); onTriggered: win.prompt(i18n.t("attach_location"), "Jakarta", function(v){ app.act("SendLocation", [win.selectedChat.id, -6.2, 106.8, v]) }) }
        TMenuItem { iconName: "pollq"; text: i18n.t("attach_poll"); onTriggered: pollDialog.open() }
        TMenuItem { iconName: "gifb"; text: "GIF"; onTriggered: { app.loadGifs(); gifPopup.open() } }
        TMenuItem { iconName: "sticker"; text: i18n.t("attach_sticker"); onTriggered: { app.loadStickers(); stickerPopup.open() } }
        TMenuItem { iconName: "contacts"; text: i18n.t("attach_contact"); onTriggered: contactDialog.open() }
        TMenuItem { iconName: "zap"; text: i18n.t("quick_replies"); onTriggered: win.prompt(i18n.t("quick_replies"), "", function(v){ if (v) composerInput.text = composerInput.text ? composerInput.text + " " + v : v }) }
        TMenuItem { iconName: "document"; text: i18n.t("attach_document"); onTriggered: docDialog.open() }
    }

    // (Tombol emoji kini membuka reactionPopup penuh dlm mode-sisip — TMenu glyph dihapus.)

    // FileDialog media (gambar/video/audio/sticker/gif) → kirim file nyata.
    // Untuk gambar/video: tampilkan pratinjau (MediaPreviewModal) dulu; lainnya
    // langsung kirim (tak ada langkah caption di flow itu).
    FileDialog {
        id: mediaDialog
        property string kind: "image"
        onAccepted: {
            if (kind === "image" || kind === "video") {
                win.mediaDraftIdx = 0; win.mediaDraftOnce = false
                win.mediaDraft = { chatId: win.selectedChat.id || "",
                    items: [{ kind: kind, url: selectedFile.toString(), name: "" }] }
                mediaPreviewPopup.open()
            } else {
                app.sendMediaFile(kind, selectedFile)
            }
        }
    }

    // Dialog polling (pertanyaan + opsi).
    Popup {
        id: pollDialog
        width: 380; height: 330; modal: true; anchors.centerIn: Overlay.overlay; padding: 16
        background: Rectangle { color: theme.bg; radius: 14; border.color: theme.line }
        ColumnLayout {
            anchors.fill: parent; spacing: 10
            Text { text: i18n.t("send_poll"); color: theme.text; font.pixelSize: 16; font.bold: true }
            Repeater {
                id: pollFields
                model: [i18n.t("poll_question"), i18n.t("poll_option") + " 1", i18n.t("poll_option") + " 2", i18n.t("poll_option") + " 3"]
                delegate: Rectangle {
                    Layout.fillWidth: true; height: 40; radius: 8; color: theme.searchBg; border.color: theme.line
                    property alias value: pf.text
                    TextInput { id: pf; anchors.fill: parent; anchors.margins: 10; color: theme.text; font.pixelSize: 14; clip: true }
                    Text { visible: pf.text === ""; anchors.verticalCenter: parent.verticalCenter; anchors.left: parent.left; anchors.leftMargin: 10; text: modelData; color: theme.text2; font.pixelSize: 13 }
                }
            }
            Item { Layout.fillHeight: true }
            RowLayout {
                Layout.alignment: Qt.AlignRight; spacing: 8
                Btn { text: i18n.t("cancel"); onClicked: pollDialog.close() }
                Btn { accent: true;
                    text: i18n.t("send")
                    onClicked: {
                        var q = pollFields.itemAt(0).value
                        var opts = []
                        for (var i = 1; i < 4; i++) { var v = pollFields.itemAt(i).value; if (v && v.trim() !== "") opts.push(v) }
                        if (q.trim() !== "" && opts.length >= 2) app.act("SendPoll", [win.selectedChat.id, q, opts, 1])
                        pollDialog.close()
                    }
                }
            }
        }
    }

    // Dialog kirim kontak (nama + telepon).
    Popup {
        id: contactDialog
        width: 360; height: 220; modal: true; anchors.centerIn: Overlay.overlay; padding: 16
        background: Rectangle { color: theme.bg; radius: 14; border.color: theme.line }
        ColumnLayout {
            anchors.fill: parent; spacing: 10
            Text { text: i18n.t("send_contact"); color: theme.text; font.pixelSize: 16; font.bold: true }
            Rectangle { Layout.fillWidth: true; height: 40; radius: 8; color: theme.searchBg; border.color: theme.line
                TextInput { id: ctName; anchors.fill: parent; anchors.margins: 10; color: theme.text; font.pixelSize: 14; clip: true }
                Text { visible: ctName.text === ""; anchors.verticalCenter: parent.verticalCenter; anchors.left: parent.left; anchors.leftMargin: 10; text: i18n.t("contact_name"); color: theme.text2; font.pixelSize: 13 } }
            Rectangle { Layout.fillWidth: true; height: 40; radius: 8; color: theme.searchBg; border.color: theme.line
                TextInput { id: ctPhone; anchors.fill: parent; anchors.margins: 10; color: theme.text; font.pixelSize: 14; clip: true }
                Text { visible: ctPhone.text === ""; anchors.verticalCenter: parent.verticalCenter; anchors.left: parent.left; anchors.leftMargin: 10; text: i18n.t("contact_phone"); color: theme.text2; font.pixelSize: 13 } }
            Item { Layout.fillHeight: true }
            RowLayout {
                Layout.alignment: Qt.AlignRight; spacing: 8
                Btn { text: i18n.t("cancel"); onClicked: contactDialog.close() }
                Btn { accent: true; text: i18n.t("send"); onClicked: { if (ctName.text !== "" && ctPhone.text !== "") app.act("SendContact", [win.selectedChat.id, ctName.text, ctPhone.text]); contactDialog.close() } }
            }
        }
    }

    // Popup hasil (invite link / translate / link preview → app.lastResult).
    Popup {
        id: resultPopup
        width: 420; height: 170; modal: true; anchors.centerIn: Overlay.overlay; padding: 16
        background: Rectangle { color: theme.bg; radius: 14; border.color: theme.line }
        ColumnLayout {
            anchors.fill: parent; spacing: 12
            Text { text: i18n.t("result"); color: theme.text; font.pixelSize: 16; font.bold: true }
            TextEdit { Layout.fillWidth: true; Layout.fillHeight: true; readOnly: true; selectByMouse: true; wrapMode: TextEdit.Wrap; color: theme.text; font.pixelSize: 13; text: app.lastResult }
            Btn { Layout.alignment: Qt.AlignRight; text: i18n.t("close"); onClicked: resultPopup.close() }
        }
    }
    Connections {
        target: app
        function onLastResultChanged() { if (app.lastResult !== "") resultPopup.open() }
    }
    // Auto-pilih chat pertama saat daftar termuat → header conv terisi.
    Connections {
        target: chatsModel
        function onModelReset() {
            var c = chatsModel.get(0)
            if (c && c.id && (!win.selectedChat || !win.selectedChat.id)) win.selectedChat = c
        }
    }

    // === Posting status ===
    Popup {
        id: statusPostPopup
        width: 360; height: 230; modal: true; anchors.centerIn: Overlay.overlay; padding: 16
        background: Rectangle { color: theme.bg; radius: 14; border.color: theme.line }
        ColumnLayout {
            anchors.fill: parent; spacing: 12
            Text { text: i18n.t("create_status"); color: theme.text; font.pixelSize: 16; font.bold: true }
            Rectangle { Layout.fillWidth: true; Layout.fillHeight: true; radius: 8; color: theme.searchBg; border.color: theme.line
                TextInput { id: statusInput; anchors.fill: parent; anchors.margins: 10; color: theme.text; font.pixelSize: 14; wrapMode: TextInput.Wrap; clip: true } }
            RowLayout {
                Layout.alignment: Qt.AlignRight; spacing: 8
                Btn { text: i18n.t("photo_video"); onClicked: { app.act("PostMediaStatus", ["image", statusInput.text, ""]); statusPostPopup.close() } }
                Btn { accent: true; text: i18n.t("send_text"); onClicked: { app.act("PostTextStatus", [statusInput.text, 0, 0]); statusInput.text = ""; statusPostPopup.close() } }
            }
        }
    }

    // === Aksi channel (klik-kanan baris channel) ===
    TMenu {
        id: channelMenu
        TMenuItem { text: "➕  " + i18n.t("ch_follow"); onTriggered: app.act("FollowChannelByJID", [win.ctxChat.id || ""]) }
        TMenuItem { text: "➖  " + i18n.t("ch_unfollow"); onTriggered: app.act("UnfollowChannel", [win.ctxChat.id || ""]) }
        TMenuItem { text: "🔇  " + i18n.t("ch_mute"); onTriggered: app.act("MuteChannel", [win.ctxChat.id || "", true]) }
        TMenuItem { text: "📝  " + i18n.t("ch_post"); onTriggered: win.prompt(i18n.t("ch_post"), "", function(v){ app.act("PostChannel", [win.ctxChat.id || "", v]) }) }
        TMenuItem { text: "👍  " + i18n.t("ch_react"); onTriggered: app.act("ReactChannel", [win.ctxChat.id || "", "m", 0, "👍"]) }
        TMenuItem { text: "💬  " + i18n.t("ch_messages"); onTriggered: app.loadIntoA("GetChannelMessages", [win.ctxChat.id || ""], msgsModel) }
        TMenuItem { text: "🔎  " + i18n.t("ch_recommend"); onTriggered: app.loadIntoA("GetRecommendedChannels", [""], channelsModel) }
        TMenuItem { text: "✨  " + i18n.t("ch_create"); onTriggered: win.prompt(i18n.t("ch_create"), "", function(v){ app.act("CreateChannel", [v, ""]) }) }
    }

    // === Aksi komunitas (tombol keluar baris .comm-head) ===
    TMenu {
        id: commMenu
        TMenuItem { text: "⎋  " + i18n.t("comm_leave"); onTriggered: { app.act("LeaveCommunity", [win.ctxChat.id || ""]); app.loadInto("GetCommunities", communitiesModel) } }
    }

    // === Aksi kontak (klik-kanan baris kontak) ===
    TMenu {
        id: contactMenu
        TMenuItem { text: "🚫  " + i18n.t("ct_block"); onTriggered: app.act("Block", [win.ctxChat.id || "", true]) }
        TMenuItem { text: "✅  " + i18n.t("ct_unblock"); onTriggered: app.act("Block", [win.ctxChat.id || "", false]) }
        TMenuItem { text: "🏷️  " + i18n.t("ct_label"); onTriggered: win.prompt(i18n.t("ct_label"), win.ctxChat.name || "", function(v){ app.act("SaveContactLabel", [win.ctxChat.id || "", v]) }) }
        TMenuItem { text: "🧹  " + i18n.t("ct_unlabel"); onTriggered: app.act("RemoveContactLabel", [win.ctxChat.id || ""]) }
        TMenuItem { text: "ℹ️  " + i18n.t("ct_about"); onTriggered: app.fetchStr("GetContactAbout", [win.ctxChat.id || ""]) }
        TMenuItem { text: "💼  " + i18n.t("ct_business"); onTriggered: app.loadDetailA("GetBusinessProfile", [win.ctxChat.id || ""]) }
        TMenuItem { text: "👥  " + i18n.t("ct_common"); onTriggered: app.loadIntoA("GetCommonGroups", [win.ctxChat.id || ""], starredModel) }
        TMenuItem { text: "📵  " + i18n.t("ct_blocklist"); onTriggered: app.loadInto("GetBlockedContacts", contactsModel) }
        TMenuItem { text: "👁️  " + i18n.t("ct_presence"); onTriggered: app.act("SubscribePresence", [win.ctxChat.id || ""]) }
    }

    // === Aksi item terjadwal (klik-kanan) ===
    TMenu {
        id: schedMenu
        TMenuItem { text: "➕  " + i18n.t("s_schedule"); onTriggered: win.prompt(i18n.t("s_schedule"), "", function(v){ app.act("ScheduleMessage", [win.selectedChat.id || "", v, 0]) }) }
        TMenuItem { text: "❌  " + i18n.t("s_cancel"); onTriggered: { app.act("CancelScheduled", [win.ctxChat.id || ""]); app.loadInto("GetScheduled", scheduledModel) } }
        TMenuItem { text: "⏰  " + i18n.t("s_add_reminder"); onTriggered: app.act("AddReminder", [win.selectedChat.id || "", win.ctxMsg.id || "", "Ingat ini", 0]) }
        TMenuItem { text: "🗑️  " + i18n.t("s_del_reminder"); onTriggered: app.act("CancelReminder", [win.ctxChat.id || ""]) }
        TMenuItem { text: "📋  " + i18n.t("s_reminders"); onTriggered: app.loadInto("GetReminders", scheduledModel) }
    }

    // === Overflow header: utilitas (tutup permukaan method engine sisanya) ===
    TMenu {
        id: overflowMenu
        property string cid: win.selectedChat.id || ""
        TMenuItem { text: i18n.t("o_chat_media"); onTriggered: app.loadIntoA("GetChatMedia", [overflowMenu.cid], msgsModel) }
        TMenuItem { text: i18n.t("o_pinned"); onTriggered: app.loadIntoA("GetPinned", [overflowMenu.cid], msgsModel) }
        TMenuItem { text: i18n.t("o_poll_votes"); onTriggered: app.loadDetailA("GetPollVotes", [win.ctxMsg.id || "p"]) }
        TMenuItem { text: i18n.t("o_link_preview"); onTriggered: app.fetchStr("GetLinkPreview", ["https://example.com"]) }
        TMenuItem { text: i18n.t("o_fetch_media"); onTriggered: app.fetchStr("FetchRemoteMedia", ["https://example.com/a.jpg"]) }
        TMenuItem { text: i18n.t("o_check_wa"); onTriggered: app.act("IsOnWhatsApp", [["6281234567890"]]) }
        TMenuItem { text: i18n.t("o_search_stickers"); onTriggered: app.loadIntoA("SearchStickers", ["happy", ""], stickersModel) }
        TMenuItem { text: i18n.t("o_search_gifs"); onTriggered: app.loadIntoA("SearchGifs", ["happy", ""], gifsModel) }
        TMenuItem { text: i18n.t("o_open_chat"); onTriggered: app.act("OpenChat", [overflowMenu.cid]) }
        TMenuItem { text: i18n.t("o_load_old"); onTriggered: app.act("LoadOlderHistory", [overflowMenu.cid]) }
        TMenuItem { text: i18n.t("o_mark_unread"); onTriggered: app.act("MarkUnread", [overflowMenu.cid]) }
        TMenuItem { text: i18n.t("o_clear"); onTriggered: app.act("ClearChat", [overflowMenu.cid]) }
        TMenuItem { text: i18n.t("o_export"); onTriggered: app.fetchStr("ExportChat", [overflowMenu.cid]) }
        TMenuItem { text: i18n.t("o_disappearing"); onTriggered: app.act("SetDisappearing", [overflowMenu.cid, 604800]) }
        TMenuItem { text: i18n.t("o_profile"); onTriggered: { app.loadDetail("GetProfile", ""); detailPopup.open() } }
        TMenuItem { text: i18n.t("o_set_name"); onTriggered: win.prompt(i18n.t("o_set_name"), "", function(v){ app.act("SetMyName", [v]) }) }
        TMenuItem { text: i18n.t("o_set_about"); onTriggered: win.prompt(i18n.t("o_set_about"), "", function(v){ app.act("SetMyAbout", [v]) }) }
        TMenuItem { text: i18n.t("o_set_photo"); onTriggered: app.act("SetMyPhoto", ["", ""]) }
        TMenuItem { text: i18n.t("o_version"); onTriggered: app.fetchStr("Version", []) }
        TMenuItem { text: i18n.t("o_status_viewers"); onTriggered: app.loadIntoA("GetStatusViewers", ["st1"], starredModel) }
        TMenuItem { text: i18n.t("o_react_status"); onTriggered: app.act("ReactStatus", [overflowMenu.cid, "st1", "👍"]) }
        TMenuItem { text: i18n.t("o_reply_status"); onTriggered: app.act("ReplyStatus", [overflowMenu.cid, "st1", "teks", "balas"]) }
        TMenuItem { text: i18n.t("o_create_group"); onTriggered: win.prompt(i18n.t("o_create_group"), "", function(v){ app.act("CreateGroup", [v, []]) }) }
        TMenuItem { text: i18n.t("o_join_link"); onTriggered: win.prompt(i18n.t("o_join_link"), "https://chat.whatsapp.com/", function(v){ app.fetchStr("JoinGroupLink", [v]) }) }
        TMenuItem { text: i18n.t("o_preview_link"); onTriggered: win.prompt(i18n.t("o_preview_link"), "https://chat.whatsapp.com/", function(v){ app.fetchStr("PreviewGroupLink", [v]) }) }
        TMenuItem { text: i18n.t("o_follow_channel"); onTriggered: win.prompt(i18n.t("o_follow_channel"), "https://whatsapp.com/channel/", function(v){ app.loadDetailA("FollowChannel", [v]) }) }
        TMenuItem { text: i18n.t("o_leave_community"); onTriggered: app.act("LeaveCommunity", [overflowMenu.cid]) }
        TMenuItem { text: i18n.t("o_reject_call"); onTriggered: app.act("RejectCall", [overflowMenu.cid, "callid"]) }
        TMenuItem { text: i18n.t("o_post_status"); onTriggered: statusPostPopup.open() }
    }

    // === Dialog input teks reusable (dipakai aksi yang butuh input) ===
    Popup {
        id: promptDialog
        property string label: ""
        property var cb: null
        width: 380; height: 175; modal: true; anchors.centerIn: Overlay.overlay; padding: 16
        background: Rectangle { color: theme.bg; radius: 14; border.color: theme.line }
        onOpened: promptInput.forceActiveFocus()
        ColumnLayout {
            anchors.fill: parent; spacing: 12
            Text { text: promptDialog.label; color: theme.text; font.pixelSize: 16; font.bold: true }
            Rectangle {
                Layout.fillWidth: true; height: 40; radius: 8; color: theme.searchBg; border.color: theme.line
                TextInput {
                    id: promptInput; anchors.fill: parent; anchors.margins: 10
                    color: theme.text; font.pixelSize: 14; clip: true
                    Keys.onReturnPressed: { if (promptDialog.cb) promptDialog.cb(text); promptDialog.close() }
                }
            }
            RowLayout {
                Layout.alignment: Qt.AlignRight; spacing: 8
                Btn { text: i18n.t("cancel"); onClicked: promptDialog.close() }
                Btn { accent: true; text: i18n.t("save"); onClicked: { if (promptDialog.cb) promptDialog.cb(promptInput.text); promptDialog.close() } }
            }
        }
    }

    // === Gerbang login (Login.svelte) — tampil bila belum terhubung ===
    Rectangle {
        anchors.fill: parent
        z: 100
        visible: app.state !== "" && app.state !== "ready" && app.state !== "connected"
        // .login: bg head-bg, kolom, align tengah.
        color: theme.headBg
        // .login-bar: bar atas full-width 56px accent, "WhatsLite" 17 w600.
        Rectangle {
            anchors.top: parent.top; anchors.left: parent.left; anchors.right: parent.right
            height: 56; color: theme.accent
            Text {
                anchors.verticalCenter: parent.verticalCenter
                anchors.left: parent.left; anchors.leftMargin: 24
                text: "WhatsLite"; color: "#ffffff"; font.pixelSize: 17; font.weight: Font.DemiBold
            }
        }
        // .login-card: margin-top 56 (dari bar) → diposisikan di bawah bar, tengah.
        Rectangle {
            id: loginCard
            anchors.horizontalCenter: parent.horizontalCenter
            anchors.top: parent.top; anchors.topMargin: 56 + 56  // bar 56 + margin-top 56
            radius: 14; color: theme.sidebarBg
            implicitWidth: loginRow.implicitWidth + 88   // padding 44 kiri+kanan
            implicitHeight: loginRow.implicitHeight + 88
            layer.enabled: true
            layer.effect: DropShadow {
                horizontalOffset: 0; verticalOffset: 8; radius: 28; samples: 33
                color: Qt.rgba(0, 0, 0, 0.14)
            }
            RowLayout {
                id: loginRow
                anchors.centerIn: parent
                spacing: 56   // .login-card gap
                // .login-left: judul + langkah bernomor.
                ColumnLayout {
                    Layout.alignment: Qt.AlignVCenter
                    spacing: 0
                    Text {
                        text: i18n.t("login_title")
                        color: theme.text; font.pixelSize: 26; font.weight: Font.Medium
                        bottomPadding: 22   // .login-left h2 margin-bottom
                    }
                    // .login-steps: ol bernomor, gap 16, font 15, max-w 320.
                    ColumnLayout {
                        spacing: 16
                        Repeater {
                            model: [i18n.t("login_s1"), i18n.t("login_s2"), i18n.t("login_s3")]
                            delegate: RowLayout {
                                Layout.maximumWidth: 320
                                spacing: 8
                                Text { text: (index + 1) + "."; color: theme.text; font.pixelSize: 15 }
                                Text {
                                    Layout.fillWidth: true
                                    text: modelData; color: theme.text; font.pixelSize: 15
                                    wrapMode: Text.WordWrap; lineHeight: 1.45
                                }
                            }
                        }
                    }
                }
                // .login-right: QR + waiting + ganti ke nomor telepon.
                ColumnLayout {
                    Layout.alignment: Qt.AlignVCenter
                    spacing: 12
                    // .qr / .qr-img: 220×220, bg putih, radius 8, pad 8.
                    Rectangle {
                        Layout.alignment: Qt.AlignHCenter
                        width: 220; height: 220; radius: 8; color: "#ffffff"
                        Image {
                            anchors.fill: parent; anchors.margins: 8; fillMode: Image.PreserveAspectFit
                            source: app.qr; visible: app.qr !== ""
                        }
                        Text {
                            // QR placeholder: use theme token instead of off-palette #555.
                            anchors.centerIn: parent; visible: app.qr === ""
                            text: i18n.t("login_waiting"); color: theme.text2; font.pixelSize: 13
                        }
                        // Klik QR → mulai sambungan (parity tombol "connect" lama).
                        MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: app.doConnect() }
                    }
                    // .login-waiting
                    Text {
                        Layout.alignment: Qt.AlignHCenter
                        text: i18n.t("login_waiting"); color: theme.text2; font.pixelSize: 13
                    }
                    // .pl-switch: tautan accent (ganti ke nomor telepon).
                    Text {
                        Layout.alignment: Qt.AlignHCenter
                        text: i18n.t("login_use_phone"); color: theme.accent; font.pixelSize: 13
                        MouseArea {
                            id: phoneSwitchMa; anchors.fill: parent; hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onClicked: app.fetchStr("LinkWithPhone", ["6281234567890"])
                        }
                        // hover sendiri (bukan bersama) — perbaikan bug shared-hover.
                        font.underline: phoneSwitchMa.containsMouse
                    }
                }
            }
        }
        // .login-hint: di bawah kartu (margin-top 30), text2 14.
        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            anchors.top: loginCard.bottom; anchors.topMargin: 30
            text: i18n.t("login_hint"); color: theme.text2; font.pixelSize: 14
        }
    }

    // === Toast — pil bawah-tengah (Toast.svelte / .toast app.css) ===
    Timer { id: toastTimer; interval: 2500; repeat: false; onTriggered: win.toastText = "" }
    Item {
        anchors.fill: parent
        z: 200
        visible: win.toastText !== ""
        // .toast-stack: bottom 22px, tengah horizontal.
        Rectangle {
            id: toastPill
            anchors.horizontalCenter: parent.horizontalCenter
            anchors.bottom: parent.bottom; anchors.bottomMargin: 22
            // .toast: pad 10px 16px, radius 999, max-width 70vw, nowrap ellipsis.
            implicitWidth: Math.min(toastLbl.implicitWidth + 32, parent.width * 0.7)
            implicitHeight: toastLbl.implicitHeight + 20
            radius: 999
            color: win.toastType === "error" ? "#c0392b"
                 : win.toastType === "ok" ? "#2e7d52" : "#2a2f36"
            layer.enabled: true
            layer.effect: DropShadow {
                horizontalOffset: 0; verticalOffset: 6; radius: 20; samples: 25
                color: Qt.rgba(0, 0, 0, 0.3)
            }
            Text {
                id: toastLbl
                anchors.centerIn: parent
                width: Math.min(implicitWidth, toastPill.parent.width * 0.7 - 32)
                text: win.toastText
                color: "#ffffff"; font.pixelSize: 14
                elide: Text.ElideRight; maximumLineCount: 1
            }
        }
    }

    // Auto-buka panel (uji/screenshot) bila diminta via env WALITE_OPEN.
    Timer {
        running: (typeof openPanel !== "undefined") && openPanel !== ""
        interval: 1500; repeat: false
        onTriggered: {
            if (openPanel === "sticker") { app.loadStickers(); stickerPopup.open() }
            else if (openPanel === "stkonline") { stickerPopup.tab = "online"; app.searchOnline("SearchStickers", "", onlineStkModel); stickerPopup.open() }
            else if (openPanel === "gifonline") { gifPopup.tab = "online"; app.searchOnline("SearchGifs", "", onlineGifModel); gifPopup.open() }
            else if (openPanel === "gif") { app.loadGifs(); gifPopup.open() }
            else if (openPanel === "settings") settingsPopup.open()
            else if (openPanel === "search") { searchInput.text = "rapat"; app.search("rapat", searchModel) }
            else if (openPanel === "detail") { app.loadDetail("GetGroupInfo", "g"); detailPopup.open() }
            else if (openPanel === "detailc") { app.loadDetail("GetContactProfile", "c"); detailPopup.open() }
            else if (openPanel === "forward") { win.ctxMsg = { id: "m1" }; forwardPopup.open() }
            else if (openPanel === "privacy") { app.loadDetail("GetPrivacy", ""); privacyPopup.open() }
            else if (openPanel === "storage") { app.loadDetail("GetStorageUsage", ""); storagePopup.open() }
            else if (openPanel === "folder") { win.folders = [{ name: "Kerja", jids: ["c"] }, { name: "Keluarga", jids: [] }]; win.openFolderPicker("c", "Alice") }
            else if (openPanel === "msginfo") { app.loadDetailA("GetMessageInfo", ["c", "m1"]); msgInfoPopup.open() }
            else if (openPanel === "reaction") { win.ctxMsg = { id: "m1", senderId: "", dir: "in" }; win.reactX = 80; win.reactY = 120; reactionPopup.insertMode = false; reactionPopup.open() }
            else if (openPanel === "reactiondetail") { win.ctxMsg = { reactions: [{ emoji: "👍", count: 2, who: ["Alice", "Bob"] }, { emoji: "❤️", count: 1, who: ["Citra"] }] }; reactionDetailPopup.open() }
            else if (openPanel === "mediapreview") { win.mediaDraftIdx = 0; win.mediaDraftOnce = false; win.mediaDraft = { chatId: "c", items: [{ kind: "image", url: (app.mediaBase || "") + "/media/c/m1", name: "" }] }; mediaPreviewPopup.open() }
            else if (openPanel === "lightbox") {
                win.lightboxCaption = "Sunset di pantai 🌅"
                // Album 3 item → verifikasi nav prev/next + counter (.lb-nav/.lb-count).
                var base = (app.mediaBase || "")
                win.lightboxItems = [{ url: base + "/media/c/m1" }, { url: base + "/media/c/m2" }, { url: base + "/media/c/m3" }]
                win.lightboxIndex = 0
            }
            else if (openPanel === "poll") pollDialog.open()
            else if (openPanel === "contact") contactDialog.open()
            else if (openPanel === "toast") win.toast(i18n.t("toast_copied"))
            else { activeView = openPanel; win.loadView(openPanel) } // calls/starred/status/contacts/channels/communities/archived/scheduled
        }
    }
}
