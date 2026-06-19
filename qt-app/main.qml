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

ApplicationWindow {
    id: win
    width: 1100; height: 740; visible: true
    title: "WhatsLite (QML)"

    property var ctxMsg: ({})        // pesan target context-menu
    property var selectedChat: ({})  // chat aktif (utk header)
    property string activeView: "chats" // chats | calls | starred
    property string chatFilter: "Semua" // filter chip aktif
    property string lightboxSrc: ""  // media fullscreen (kosong = tutup)
    property bool locked: (typeof startLock !== "undefined") && startLock // app-lock PIN
    property var replyTo: null        // pesan yang sedang dibalas (banner composer)
    property var ctxChat: ({})        // chat target context-menu baris
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
        "settings": '<circle cx="12" cy="12" r="3"/><path d="M12 2v3M12 19v3M2 12h3M19 12h3M5 5l2 2M17 17l2 2M19 5l-2 2M7 17l-2 2"/>',
        "search": '<circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/>',
        "plus": '<path d="M12 5v14M5 12h14"/>',
        "send": '<path d="M3 11l18-8-8 18-2-7-8-3z"/>',
        "emoji": '<circle cx="12" cy="12" r="9"/><circle cx="9" cy="10" r="1"/><circle cx="15" cy="10" r="1"/><path d="M8.5 14.5a4 4 0 0 0 7 0"/>',
        "mic": '<rect x="9" y="3" width="6" height="11" rx="3"/><path d="M5 11a7 7 0 0 0 14 0M12 18v3"/>',
        "pollq": '<path d="M5 5h14M5 12h9M5 19h5"/>',
        "sticker": '<path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h8l6-6V5a2 2 0 0 0-2-2z"/><path d="M14 21v-4a2 2 0 0 1 2-2h4"/>',
        "gifb": '<rect x="3" y="5" width="18" height="14" rx="2"/><path d="M8 9v6M11 9v6h2M16 9h-2v6M16 12h-1"/>',
        "document": '<path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><path d="M14 3v6h6"/>',
        "overflow": '<circle cx="12" cy="5" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="12" cy="19" r="1.6"/>',
        "newchat": '<path d="M12 5H7a3 3 0 0 0-3 3v9a3 3 0 0 0 3 3h9a3 3 0 0 0 3-3v-5"/><path d="M18.5 3.5a2.1 2.1 0 0 1 3 3L13 15l-4 1 1-4 8.5-8.5z"/>',
        "pin": '<path d="M12 17v5M7 4h10l-1 6 3 3H5l3-3-1-6z"/>',
        "mute": '<path d="M5 9v6h3l4 4V5L8 9H5z"/><path d="M16 8a5 5 0 0 1 0 8"/><path d="M3 3l18 18"/>',
        "check": '<path d="M3 7.5l3.5 3.5L14 4"/>',
        "checks": '<path d="M1 7.5l3.2 3.2L10 4"/><path d="M7 10.7L12.8 4"/>'
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

    // --- Token tema (light + dark) — cocok dgn app.css [data-theme] ---
    QtObject {
        id: theme
        property bool dark: (typeof startDark !== "undefined") ? startDark : false
        // Token DISALIN PERSIS dari frontend/src/styles/app.css (desain app
        // existing) — sumber valid in-repo, bukan tebakan eksternal.
        readonly property color railBg: dark ? "#11161d" : "#f4f6fa"
        readonly property color railIco: dark ? "#8a97a3" : "#6b7785"
        readonly property color accent: dark ? "#06c98c" : "#06b67f"
        readonly property color accentDeep: dark ? "#06b67f" : "#048a60"
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
                anchors.topMargin: 12
                spacing: 6
                Repeater {
                    model: [
                        { icon: "💬", view: "chats" },
                        { icon: "📷", view: "status" },
                        { icon: "📢", view: "channels" },
                        { icon: "👥", view: "communities" },
                        { icon: "⭐", view: "starred" },
                        { icon: "📞", view: "calls" },
                        { icon: "👤", view: "contacts" },
                        { icon: "🗄️", view: "archived" },
                        { icon: "⏰", view: "scheduled" },
                        { icon: "⚙️", view: "settings" }
                    ]
                    delegate: Rectangle {
                        Layout.alignment: Qt.AlignHCenter
                        width: 44; height: 44; radius: 22
                        color: (activeView === modelData.view) ? theme.selected : "transparent"
                        Icon {
                            anchors.centerIn: parent; width: 24; height: 24
                            svg: win.ico[modelData.view] || ""
                            color: (activeView === modelData.view) ? theme.accent : theme.railIco
                        }
                        MouseArea {
                            anchors.fill: parent
                            onClicked: {
                                if (modelData.view === "settings") { settingsPopup.open(); return }
                                activeView = modelData.view
                                win.loadView(modelData.view)
                            }
                        }
                    }
                }
                Item { Layout.fillHeight: true }
                // Toggle tema light/dark (bawah rail).
                Rectangle {
                    Layout.alignment: Qt.AlignHCenter
                    Layout.bottomMargin: 12
                    width: 44; height: 44; radius: 22; color: "transparent"
                    Text { anchors.centerIn: parent; text: theme.dark ? "☀️" : "🌙"; font.pixelSize: 18 }
                    MouseArea { anchors.fill: parent; onClicked: theme.dark = !theme.dark }
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
                // Header: judul view + aksi (chat baru, menu) — ala WhatsApp.
                Rectangle {
                    Layout.fillWidth: true; Layout.preferredHeight: 60
                    color: theme.headBg
                    RowLayout {
                        anchors.fill: parent; anchors.leftMargin: 18; anchors.rightMargin: 10; spacing: 4
                        Text {
                            Layout.fillWidth: true
                            text: ({ chats: "Chat", status: "Status", channels: "Channels", communities: "Communities", calls: "Panggilan", contacts: "Kontak", starred: "Berbintang", archived: "Arsip", scheduled: "Terjadwal" }[activeView] || "Chat")
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
                // Search (FTS pesan)
                Rectangle {
                    Layout.fillWidth: true; Layout.preferredHeight: 44
                    Layout.margins: 8; radius: 22; color: theme.searchBg
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
                        text: i18n.t("search"); color: theme.text2; font.pixelSize: 14
                    }
                }
                // Filter chips (Semua / Belum dibaca N / Favorit / Grup N / +) — ala WhatsApp.
                Flow {
                    Layout.fillWidth: true; Layout.leftMargin: 14; Layout.rightMargin: 14; Layout.bottomMargin: 6; spacing: 8
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
                            radius: 16; height: 30; implicitWidth: crow.implicitWidth + 26
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
                        radius: 16; height: 30; width: 34; color: theme.searchBg
                        Text { anchors.centerIn: parent; text: "+"; font.pixelSize: 17; color: theme.text2 }
                        MouseArea { anchors.fill: parent
                            onClicked: win.prompt(i18n.t("folder_new"), "", function(v){ if (v) app.act("AddFolder", [v]) }) }
                    }
                }
                // Daftar (swap per activeView): chats / calls / starred
                Item {
                    Layout.fillWidth: true; Layout.fillHeight: true
                    // --- Chats ---
                    ListView {
                        id: chatList
                        anchors.fill: parent
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
                            width: chatList.width; height: 54
                            onClicked: { activeView = "archived"; app.loadInto("GetArchivedChats", archivedModel) }
                            background: Rectangle { anchors.margins: 3; radius: theme.r; color: hovered ? theme.hover : "transparent" }
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 22; anchors.rightMargin: 16; spacing: 16
                                Icon { Layout.preferredWidth: 22; Layout.preferredHeight: 22; svg: win.ico["archived"]; color: theme.accent }
                                Text { Layout.fillWidth: true; text: "Diarsipkan"; color: theme.text; font.pixelSize: 15 }
                            }
                        }
                        delegate: ItemDelegate {
                            width: chatList.width; height: 68; clip: true
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
                                    Layout.fillWidth: true; Layout.alignment: Qt.AlignVCenter; spacing: 3
                                    // Baris 1: nama (kiri) + waktu (kanan)
                                    RowLayout {
                                        Layout.fillWidth: true; spacing: 6
                                        Text {
                                            Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1; wrapMode: Text.NoWrap
                                            text: model.m.name || model.m.id || ""
                                            font.pixelSize: 16; color: theme.text
                                            font.weight: (model.m.unread || (model.m.badge || 0) > 0) ? Font.Bold : Font.Medium
                                        }
                                        Text {
                                            text: model.m.time || ""
                                            color: (model.m.badge > 0) ? theme.accent : theme.text2; font.pixelSize: 12
                                        }
                                    }
                                    // Baris 2: preview (kiri) + badge unread (kanan)
                                    RowLayout {
                                        Layout.fillWidth: true; spacing: 4
                                        // Ticks preview (pesan terakhir keluar).
                                        Icon {
                                            visible: model.m.sent === true
                                            Layout.preferredWidth: 16; Layout.preferredHeight: 12; Layout.alignment: Qt.AlignVCenter
                                            vbox: "0 0 18 14"; svg: win.ico["checks"]
                                            color: model.m.status === "read" ? theme.tick : theme.text2
                                        }
                                        Text {
                                            Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1; wrapMode: Text.NoWrap
                                            text: model.m.preview || ""; font.pixelSize: 14
                                            // Unread → preview lebih terang + medium (app.css .chat-row.unread .row-preview).
                                            color: model.m.unread ? theme.text : theme.text2
                                            font.weight: model.m.unread ? Font.Medium : Font.Normal
                                        }
                                        Rectangle {
                                            visible: (model.m.badge || 0) > 0
                                            radius: 10; color: theme.accent
                                            implicitWidth: Math.max(20, bdg.implicitWidth + 12); implicitHeight: 20
                                            Text { id: bdg; anchors.centerIn: parent; color: "white"; font.pixelSize: 12; font.bold: true
                                                text: model.m.badge > 99 ? "99+" : (model.m.badge || 0) }
                                        }
                                        // Pin/mute (saat tak ada badge) — ala WhatsApp.
                                        Row {
                                            visible: !((model.m.badge || 0) > 0) && (model.m.pinned === true || model.m.muted === true)
                                            spacing: 4
                                            Icon { visible: model.m.muted === true; width: 16; height: 16; svg: win.ico["mute"]; color: theme.text2 }
                                            Icon { visible: model.m.pinned === true; width: 15; height: 15; rotation: 45; svg: win.ico["pin"]; color: theme.text2 }
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
                    // --- Riwayat panggilan ---
                    ListView {
                        anchors.fill: parent
                        visible: activeView === "calls" && searchInput.text === ""
                        clip: true; model: callsModel
                        delegate: Item {
                            width: ListView.view.width; height: 64; clip: true
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12; spacing: 12
                                Text { text: model.m.video ? "📹" : "📞"; font.pixelSize: 20 }
                                ColumnLayout {
                                    Layout.fillWidth: true; spacing: 2
                                    Text { text: model.m.name || ""; color: theme.text; font.pixelSize: 16; font.weight: Font.Medium }
                                    Text {
                                        text: (model.m.status === "missed" ? "Tak terjawab" : "Ditolak") + " · " + (model.m.time || "")
                                        color: model.m.status === "missed" ? "#e0533d" : theme.text2; font.pixelSize: 12
                                    }
                                }
                            }
                        }
                    }
                    // --- Pesan berbintang ---
                    ListView {
                        anchors.fill: parent
                        visible: activeView === "starred" && searchInput.text === ""
                        clip: true; model: starredModel
                        delegate: Item {
                            width: ListView.view.width; height: 62; clip: true
                            ColumnLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12
                                anchors.topMargin: 8; anchors.bottomMargin: 8; spacing: 2
                                Text { text: "⭐ " + (model.m.chatName || ""); color: theme.text; font.pixelSize: 13; font.weight: Font.Medium }
                                Text {
                                    Layout.fillWidth: true; elide: Text.ElideRight
                                    text: model.m.text || ""; color: theme.text2; font.pixelSize: 13
                                }
                            }
                        }
                    }
                    // --- Status (cerita) ---
                    ListView {
                        anchors.fill: parent
                        visible: activeView === "status" && searchInput.text === ""
                        clip: true; model: statusModel
                        delegate: Item {
                            width: ListView.view.width; height: 64; clip: true
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12; spacing: 12
                                Rectangle {
                                    width: 46; height: 46; radius: 23
                                    color: "transparent"; border.width: 2
                                    border.color: model.m.seen ? theme.text2 : theme.accent
                                    Avatar {
                                        anchors.centerIn: parent; width: 40; height: 40; fontSize: 16
                                        name: model.m.name; jid: model.m.id; base: app.mediaBase; accent: win.avatarColor(model.m.name)
                                    }
                                }
                                ColumnLayout {
                                    Layout.fillWidth: true; spacing: 2
                                    Text { text: model.m.name || ""; color: theme.text; font.pixelSize: 16; font.weight: Font.Medium }
                                    Text { text: (model.m.count || 0) + " pembaruan · " + (model.m.time || "")
                                        color: theme.text2; font.pixelSize: 14 }
                                }
                            }
                        }
                    }
                    // --- Kontak ---
                    ListView {
                        anchors.fill: parent
                        visible: activeView === "contacts" && searchInput.text === ""
                        clip: true; model: contactsModel
                        delegate: ItemDelegate {
                            width: ListView.view.width; height: 60; clip: true
                            onClicked: { win.selectedChat = { name: model.m.name, id: model.m.jid }; activeView = "chats"; app.openChat(model.m.jid) }
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12; spacing: 12
                                Avatar {
                                    Layout.preferredWidth: 42; Layout.preferredHeight: 42; fontSize: 16
                                    name: model.m.name; jid: model.m.jid; base: app.mediaBase; accent: win.avatarColor(model.m.name)
                                }
                                ColumnLayout {
                                    Layout.fillWidth: true; spacing: 2
                                    Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1
                                        text: model.m.name || ""; color: theme.text; font.pixelSize: 16; font.weight: Font.Medium }
                                    Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1
                                        text: model.m.about || ""; color: theme.text2; font.pixelSize: 14 }
                                }
                            }
                            MouseArea { anchors.fill: parent; acceptedButtons: Qt.RightButton
                                onClicked: { win.ctxChat = { id: model.m.jid, name: model.m.name }; contactMenu.popup() } }
                        }
                    }
                    // --- Channels / Communities / Archived / Scheduled (pola sama) ---
                    ListView {
                        anchors.fill: parent; visible: activeView === "channels" && searchInput.text === ""
                        clip: true; model: channelsModel
                        delegate: Item {
                            width: ListView.view.width; height: 64; clip: true
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12; spacing: 12
                                Text { text: "📢"; font.pixelSize: 22 }
                                ColumnLayout {
                                    Layout.fillWidth: true; spacing: 2
                                    Text { text: model.m.name || ""; color: theme.text; font.pixelSize: 16; font.weight: Font.Medium }
                                    Text { Layout.fillWidth: true; elide: Text.ElideRight; text: model.m.preview || ""; color: theme.text2; font.pixelSize: 12 }
                                }
                            }
                            MouseArea { anchors.fill: parent; acceptedButtons: Qt.RightButton
                                onClicked: { win.ctxChat = { id: model.m.jid || model.m.id || "", name: model.m.name }; channelMenu.popup() } }
                        }
                    }
                    ListView {
                        anchors.fill: parent; visible: activeView === "communities" && searchInput.text === ""
                        clip: true; model: communitiesModel
                        delegate: Item {
                            width: ListView.view.width; height: 64; clip: true
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12; spacing: 12
                                Rectangle { width: 42; height: 42; radius: 10; color: theme.accent
                                    Text { anchors.centerIn: parent; text: "👥"; font.pixelSize: 18 } }
                                ColumnLayout {
                                    Layout.fillWidth: true; spacing: 2
                                    Text { text: model.m.name || ""; color: theme.text; font.pixelSize: 16; font.weight: Font.Medium }
                                    Text { text: model.m.subtitle || ""; color: theme.text2; font.pixelSize: 12 }
                                }
                            }
                        }
                    }
                    ListView {
                        anchors.fill: parent; visible: activeView === "archived" && searchInput.text === ""
                        clip: true; model: archivedModel
                        delegate: Item {
                            width: ListView.view.width; height: 64; clip: true
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12; spacing: 12
                                Avatar { Layout.preferredWidth: 42; Layout.preferredHeight: 42; fontSize: 16
                                    name: model.m.name; jid: model.m.id; base: app.mediaBase; accent: theme.text2 }
                                ColumnLayout {
                                    Layout.fillWidth: true; spacing: 2
                                    Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1
                                        text: model.m.name || ""; color: theme.text; font.pixelSize: 16; font.weight: Font.Medium }
                                    Text { Layout.fillWidth: true; elide: Text.ElideRight; maximumLineCount: 1; text: model.m.preview || ""; color: theme.text2; font.pixelSize: 14 }
                                }
                            }
                        }
                    }
                    ListView {
                        anchors.fill: parent; visible: activeView === "scheduled" && searchInput.text === ""
                        clip: true; model: scheduledModel
                        delegate: Item {
                            width: ListView.view.width; height: 64; clip: true
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12; spacing: 12
                                Text { text: "⏰"; font.pixelSize: 20 }
                                ColumnLayout {
                                    Layout.fillWidth: true; spacing: 2
                                    Text { text: (model.m.chatName || "") + " · " + (model.m.time || ""); color: theme.text; font.pixelSize: 14; font.weight: Font.Medium }
                                    Text { Layout.fillWidth: true; elide: Text.ElideRight; text: model.m.text || ""; color: theme.text2; font.pixelSize: 12 }
                                }
                            }
                            MouseArea { anchors.fill: parent; acceptedButtons: Qt.RightButton
                                onClicked: { win.ctxChat = { id: model.m.id || "", name: model.m.chatName }; schedMenu.popup() } }
                        }
                    }
                    // --- Hasil pencarian (override semua view saat mengetik) ---
                    ListView {
                        anchors.fill: parent
                        visible: searchInput.text !== ""
                        clip: true; model: searchModel
                        delegate: Item {
                            width: ListView.view.width; height: 62; clip: true
                            ColumnLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12
                                anchors.topMargin: 8; anchors.bottomMargin: 8; spacing: 2
                                Text { text: "🔍 " + (model.m.chatName || ""); color: theme.text; font.pixelSize: 13; font.weight: Font.Medium }
                                Text { Layout.fillWidth: true; elide: Text.ElideRight
                                    text: model.m.text || ""; color: theme.text2; font.pixelSize: 13 }
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
            // Header conv
            Rectangle {
                Layout.fillWidth: true; Layout.preferredHeight: 60
                color: theme.headBg; border.color: theme.line
                RowLayout {
                    anchors.left: parent.left; anchors.leftMargin: 16; anchors.right: parent.right; anchors.rightMargin: 54
                    anchors.verticalCenter: parent.verticalCenter; spacing: 12
                    Avatar {
                        visible: win.selectedChat.id !== undefined
                        Layout.preferredWidth: 40; Layout.preferredHeight: 40; fontSize: 16
                        name: win.selectedChat.name || ""; jid: win.selectedChat.id || ""
                        base: app.mediaBase; accent: win.avatarColor(win.selectedChat.name || "?")
                        group: win.selectedChat.group === true
                    }
                    ColumnLayout {
                        Layout.fillWidth: true; spacing: 0
                        Text { Layout.fillWidth: true; elide: Text.ElideRight
                            text: win.selectedChat.name || i18n.t("pick_conversation"); font.pixelSize: 16; font.bold: true; color: theme.text }
                        Text { visible: win.selectedChat.id !== undefined
                            text: app.typing ? i18n.t("typing") : (win.selectedChat.status || (win.selectedChat.group ? "klik utk info grup" : "online"))
                            color: app.typing ? theme.accent : theme.text2; font.pixelSize: 12 }
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
                    width: 36; height: 36; radius: 18; color: searchHov.hovered ? theme.hover : "transparent"
                    visible: win.selectedChat.id !== undefined
                    Icon { anchors.centerIn: parent; width: 20; height: 20; svg: win.ico["search"]; color: theme.railIco }
                    HoverHandler { id: searchHov }
                    MouseArea { anchors.fill: parent; onClicked: { activeView = "chats"; searchInput.forceActiveFocus() } }
                }
                // Overflow ⋮ — utilitas (media/pin/poll/profil/grup/status/dll).
                Rectangle {
                    id: convOverflow
                    anchors.right: parent.right; anchors.rightMargin: 12
                    anchors.verticalCenter: parent.verticalCenter
                    width: 36; height: 36; radius: 18; color: "transparent"
                    Icon { anchors.centerIn: parent; width: 22; height: 22; svg: win.ico["overflow"]; color: theme.railIco }
                    MouseArea { anchors.fill: parent; onClicked: overflowMenu.popup() }
                }
            }
            // Timeline — pola tervalidasi (ListView + reuseItems), bubble in/out
            Rectangle {
                Layout.fillWidth: true; Layout.fillHeight: true
                color: theme.wallpaper
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
                        Button {
                            Layout.alignment: Qt.AlignHCenter; flat: true; text: "↑ " + i18n.t("load_older")
                            visible: timeline.count > 0
                            onClicked: app.loadOlder()
                        }
                        // Separator tanggal (ala WhatsApp) — pill terpusat.
                        Rectangle {
                            Layout.alignment: Qt.AlignHCenter
                            visible: timeline.count > 0
                            radius: 8; color: theme.dark ? "#182229" : "#ffffff"
                            implicitWidth: dlbl.implicitWidth + 22; implicitHeight: 26
                            Text { id: dlbl; anchors.centerIn: parent; text: "HARI INI"; color: theme.text2; font.pixelSize: 12; font.weight: Font.Medium }
                        }
                        // Pembatas "belum dibaca" (ala WhatsApp).
                        Rectangle {
                            Layout.fillWidth: true; Layout.bottomMargin: 6
                            visible: (win.selectedChat.badge || 0) > 0
                            color: theme.dark ? "#182229" : "#ffffff"; implicitHeight: 28
                            Text { anchors.centerIn: parent; color: theme.accent; font.pixelSize: 12; font.weight: Font.Medium
                                text: (win.selectedChat.badge || 0) + " PESAN BELUM DIBACA" }
                        }
                    }
                    delegate: Item {
                        width: timeline.width
                        implicitHeight: bubble.implicitHeight + 4
                        property bool out: (model.m.dir === "out")
                        Rectangle {
                            id: bubble
                            x: parent.out ? parent.width - width - 4 : 4
                            width: content.implicitWidth + 16
                            implicitHeight: content.implicitHeight + 16
                            // Tail ala WhatsApp (app.css): sudut atas dekat pengirim 6px, lainnya r.
                            radius: theme.r
                            topLeftRadius: parent.out ? theme.r : 6
                            topRightRadius: parent.out ? 6 : theme.r
                            color: parent.out ? theme.outBg : theme.inBg
                            border.color: theme.line
                            ColumnLayout {
                                id: content
                                property var pmsg: model.m // tangkap pesan (hindari shadowing Repeater)
                                anchors.left: parent.left
                                anchors.top: parent.top
                                anchors.margins: 8
                                spacing: 3
                                // Nama pengirim (grup, pesan masuk) — warna per-pengirim.
                                Text {
                                    visible: win.selectedChat.group === true && content.pmsg.dir === "in" && (content.pmsg.sender || "") !== ""
                                    text: content.pmsg.sender || ""
                                    color: win.avatarColor(content.pmsg.sender || ""); font.pixelSize: 13; font.weight: Font.DemiBold
                                }
                                // Blok kutipan balasan (bar warna + nama + teks).
                                // .quote: bar 4px --quote-bar, bg --quote-bg, radius 4, padding 5/9, mb 5.
                                Rectangle {
                                    visible: (content.pmsg.quoteId || "") !== ""
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
                                // Document → ikon + nama + (PDF · ukuran · halaman)
                                RowLayout {
                                    visible: model.m.type === "document"
                                    spacing: 8
                                    Text { text: "📄"; font.pixelSize: 26 }
                                    ColumnLayout {
                                        spacing: 1
                                        Text {
                                            text: model.m.text || "Dokumen"
                                            color: theme.text; font.pixelSize: 14; font.weight: Font.Medium
                                        }
                                        Text {
                                            text: win.fmtDoc(model.m)
                                            color: theme.text2; font.pixelSize: 12
                                        }
                                    }
                                }
                                // Stiker / GIF placeholder
                                Text {
                                    visible: model.m.type === "sticker" || model.m.type === "gif"
                                    text: model.m.type === "sticker" ? "🏷️  Stiker" : "🎬  GIF"
                                    color: theme.text2; font.pixelSize: 14
                                }
                                // Polling: pertanyaan + opsi (klik = vote → VotePoll). app.css .poll-card.
                                ColumnLayout {
                                    visible: model.m.type === "poll"
                                    spacing: 6
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
                                            Layout.fillWidth: true; Layout.minimumWidth: 214; implicitHeight: 38
                                            radius: 10; color: theme.bg; border.width: 1; border.color: pollHov.hovered ? theme.accent : theme.line
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
                                // Thumbnail gambar/video (data-URI di thumb)
                                Image {
                                    visible: (model.m.type === "image" || model.m.type === "video") && (model.m.thumb || "").indexOf("data:") === 0
                                    source: visible ? model.m.thumb : ""
                                    Layout.preferredWidth: Math.min(timeline.width * 0.45, 240)
                                    Layout.preferredHeight: Layout.preferredWidth * 0.62
                                    fillMode: Image.PreserveAspectCrop; clip: true
                                    Text { visible: model.m.type === "video"; anchors.centerIn: parent; text: "▶"; color: "white"; font.pixelSize: 30 }
                                }
                                // Voice note
                                RowLayout {
                                    visible: model.m.type === "voice"; spacing: 8
                                    Text { text: "🎤"; font.pixelSize: 20 }
                                    Text { text: i18n.t("voice") + " · " + (content.pmsg.text || ""); color: theme.text; font.pixelSize: 14 }
                                }
                                // Teks biasa
                                Text {
                                    visible: ["document", "sticker", "gif", "poll", "voice"].indexOf(model.m.type) < 0
                                    text: model.m.text || ""
                                    wrapMode: Text.WordWrap; color: theme.text; font.pixelSize: 15
                                    Layout.maximumWidth: timeline.width * 0.66
                                }
                                // Waktu + ticks di pojok kanan-bawah bubble (ala WhatsApp).
                                RowLayout {
                                    Layout.alignment: Qt.AlignRight; spacing: 4
                                    Text { text: model.m.time || ""; color: theme.text2; font.pixelSize: 11 }
                                    Icon {
                                        visible: content.pmsg.dir === "out"
                                        vbox: "0 0 18 14"; width: 16; height: 12
                                        svg: win.ico["checks"]
                                        color: content.pmsg.status === "read" ? theme.tick : theme.text2
                                    }
                                }
                                // Chip reaksi (emoji + jumlah)
                                Flow {
                                    visible: content.pmsg.reactions && content.pmsg.reactions.length > 0
                                    Layout.fillWidth: true; spacing: 4
                                    Repeater {
                                        model: content.pmsg.reactions || []
                                        delegate: Rectangle {
                                            radius: 10; color: theme.bg2; border.color: theme.line
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
                                    if (mouse.button === Qt.RightButton) { win.ctxMsg = model.m; msgMenu.popup() }
                                    else if (["image", "sticker", "gif"].indexOf(model.m.type) >= 0)
                                        win.lightboxSrc = (app.mediaBase || "") + "/media/" + (win.selectedChat.id || "") + "/" + model.m.id
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
            // Composer
            Rectangle {
                Layout.fillWidth: true; Layout.preferredHeight: 56
                color: theme.bg2
                RowLayout {
                    anchors.fill: parent; anchors.margins: 8; spacing: 6
                    // Emoji (placeholder picker) — kiri, ala Composer.svelte.
                    Rectangle {
                        width: 40; height: 40; radius: 20; color: emojiHov.hovered ? theme.hover : "transparent"
                        Icon { anchors.centerIn: parent; width: 24; height: 24; svg: win.ico["emoji"]; color: theme.railIco }
                        HoverHandler { id: emojiHov }
                        MouseArea { anchors.fill: parent; onClicked: emojiMenu.popup() }
                    }
                    // Lampiran (+) → menu: dokumen/stiker/gif/gambar/video/lokasi/polling/kontak/mention.
                    Rectangle {
                        width: 40; height: 40; radius: 20; color: attachHov.hovered ? theme.hover : "transparent"
                        Icon { anchors.centerIn: parent; width: 26; height: 26; svg: win.ico["plus"]; color: theme.railIco }
                        HoverHandler { id: attachHov }
                        MouseArea { anchors.fill: parent; onClicked: attachMenu.popup() }
                    }
                    Rectangle {
                        Layout.fillWidth: true; Layout.fillHeight: true
                        radius: 18; color: theme.headBg; border.color: theme.line
                        function send() {
                            if (composerInput.text.trim() === "") return
                            if (win.replyTo && win.replyTo.id)
                                app.replyText(win.replyTo.id, win.replyTo.senderId || "", win.replyTo.text || "", composerInput.text)
                            else
                                app.sendText(composerInput.text)
                            composerInput.text = ""
                            win.replyTo = null
                        }
                        TextInput {
                            id: composerInput
                            anchors.fill: parent; anchors.leftMargin: 14; anchors.rightMargin: 14
                            verticalAlignment: TextInput.AlignVCenter
                            color: theme.text; font.pixelSize: 14; clip: true
                            onTextChanged: app.sendTyping(text.length > 0)
                            Keys.onReturnPressed: parent.send()
                            Keys.onEnterPressed: parent.send()
                        }
                        Text {
                            visible: composerInput.text === ""
                            anchors.verticalCenter: parent.verticalCenter
                            anchors.left: parent.left; anchors.leftMargin: 14
                            text: i18n.t("type_message"); color: theme.text2; font.pixelSize: 14
                        }
                    }
                    // Kosong → mic (transparan); ada teks → tombol kirim (accent).
                    Rectangle {
                        id: sendBtn
                        property bool hasText: composerInput.text.trim() !== ""
                        width: 40; height: 40; radius: 20
                        color: hasText ? theme.accent : "transparent"
                        Icon { anchors.centerIn: parent; width: 22; height: 22
                            svg: sendBtn.hasText ? win.ico["send"] : win.ico["mic"]
                            color: sendBtn.hasText ? "white" : theme.railIco }
                        MouseArea { anchors.fill: parent; onClicked: if (sendBtn.hasText) composerInput.parent.send() }
                    }
                }
            }
        }
    }

    // === Picker stiker (koleksi tersimpan, fitur CRUD #1) ===
    Popup {
        id: stickerPopup
        width: 324; height: 360
        x: win.width - width - 24
        y: win.height - height - 70
        padding: 8
        background: Rectangle { color: theme.bg; radius: 12; border.color: theme.line }
        GridView {
            id: stickerGrid
            anchors.fill: parent
            cellWidth: 100; cellHeight: 100; clip: true
            model: stickersModel
            delegate: Item {
                width: 100; height: 100
                Rectangle {
                    anchors.fill: parent; anchors.margins: 6; radius: 10
                    color: theme.searchBg; border.color: theme.line
                    Image {
                        id: stkImg
                        anchors.fill: parent; anchors.margins: 8
                        fillMode: Image.PreserveAspectFit
                        source: app.mediaBase ? (app.mediaBase + "/sticker/" + model.m.hash) : ""
                        visible: status === Image.Ready
                    }
                    // Fallback (mock tanpa byte / belum termuat): label tipe.
                    ColumnLayout {
                        anchors.centerIn: parent
                        visible: stkImg.status !== Image.Ready
                        Text { Layout.alignment: Qt.AlignHCenter; text: "🏷️"; font.pixelSize: 30 }
                        Text {
                            Layout.alignment: Qt.AlignHCenter
                            text: model.m.animated ? "animasi" : "statis"
                            color: theme.text2; font.pixelSize: 10
                        }
                    }
                    MouseArea {
                        anchors.fill: parent
                        onClicked: { app.sendSticker(model.m.hash); stickerPopup.close() }
                    }
                }
            }
        }
    }

    // === Picker GIF (fitur #3) ===
    Popup {
        id: gifPopup
        width: 324; height: 320
        x: win.width - width - 24
        y: win.height - height - 70
        padding: 8
        background: Rectangle { color: theme.bg; radius: 12; border.color: theme.line }
        GridView {
            anchors.fill: parent
            cellWidth: 150; cellHeight: 100; clip: true
            model: gifsModel
            delegate: Item {
                width: 150; height: 100
                Rectangle {
                    anchors.fill: parent; anchors.margins: 6; radius: 10
                    color: theme.searchBg; border.color: theme.line
                    Image {
                        id: gifImg
                        anchors.fill: parent; anchors.margins: 8; fillMode: Image.PreserveAspectFit
                        source: app.mediaBase ? (app.mediaBase + "/savedgif/" + model.m.hash) : ""
                        visible: status === Image.Ready
                    }
                    Text {
                        anchors.centerIn: parent; visible: gifImg.status !== Image.Ready
                        text: "🎬 GIF"; color: theme.text2; font.pixelSize: 14
                    }
                    MouseArea { anchors.fill: parent; onClicked: { app.sendGif(model.m.hash); gifPopup.close() } }
                }
            }
        }
    }

    // === Menu aksi pesan (klik-kanan bubble) ===
    Menu {
        id: msgMenu
        MenuItem { text: "👍  " + i18n.t("m_like"); onTriggered: app.react(win.ctxMsg.id, win.ctxMsg.senderId || "", win.ctxMsg.dir === "out", "👍") }
        MenuItem { text: "↩️  " + i18n.t("m_reply"); onTriggered: win.replyTo = win.ctxMsg }
        MenuItem {
            text: "✏️  " + i18n.t("m_edit"); visible: win.ctxMsg.dir === "out"; height: visible ? implicitHeight : 0
            onTriggered: { editInput.text = win.ctxMsg.text || ""; editPopup.open() }
        }
        MenuItem { text: "📌  " + i18n.t("m_pin"); onTriggered: app.pinMessage(win.ctxMsg.id, win.ctxMsg.senderId || "", win.ctxMsg.dir === "out", true) }
        MenuItem { text: "⭐  " + i18n.t("m_star"); onTriggered: app.star(win.ctxMsg.id, win.ctxMsg.senderId || "", win.ctxMsg.dir === "out", true) }
        MenuItem {
            text: "💾  " + i18n.t("m_save_sticker"); visible: win.ctxMsg.type === "sticker"; height: visible ? implicitHeight : 0
            onTriggered: app.saveStickerFromMsg(win.ctxMsg.id)
        }
        MenuItem {
            text: "🎬  " + i18n.t("m_save_gif"); visible: win.ctxMsg.type === "gif"; height: visible ? implicitHeight : 0
            onTriggered: app.saveGifFromMsg(win.ctxMsg.id)
        }
        MenuItem { text: "↪️  " + i18n.t("m_forward"); onTriggered: forwardPopup.open() }
        MenuItem {
            text: "😀  " + i18n.t("m_reactions")
            visible: win.ctxMsg.reactions && win.ctxMsg.reactions.length > 0
            height: visible ? implicitHeight : 0
            onTriggered: reactionPopup.open()
        }
        MenuItem {
            text: "ℹ️  " + i18n.t("m_info")
            onTriggered: { app.loadDetailA("GetMessageInfo", [win.selectedChat.id || "", win.ctxMsg.id]); msgInfoPopup.open() }
        }
        MenuItem { text: "🗑️  " + i18n.t("m_delete_all"); onTriggered: app.deleteMsg(win.ctxMsg.id, win.ctxMsg.senderId || "", win.ctxMsg.dir === "out", true) }
    }

    // === Menu kelola chat (klik-kanan baris) ===
    Menu {
        id: chatMenu
        MenuItem { text: "✓  " + i18n.t("c_mark_read"); onTriggered: app.markRead(win.ctxChat.id) }
        MenuItem { text: "📌  " + (win.ctxChat.pinned ? i18n.t("c_unpin") : i18n.t("c_pin")); onTriggered: app.pinChat(win.ctxChat.id, !win.ctxChat.pinned) }
        MenuItem { text: (win.ctxChat.muted ? "🔔  " + i18n.t("c_unmute") : "🔇  " + i18n.t("c_mute")); onTriggered: app.muteChat(win.ctxChat.id, !win.ctxChat.muted) }
        MenuItem { text: "🗄️  " + i18n.t("c_archive"); onTriggered: app.archiveChat(win.ctxChat.id, true) }
        MenuItem { text: "🗑️  " + i18n.t("c_delete"); onTriggered: app.deleteChat(win.ctxChat.id) }
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
                Button { text: i18n.t("cancel"); onClicked: editPopup.close() }
                Button { text: i18n.t("save"); onClicked: { app.editMessage(win.ctxMsg.id, editInput.text); editPopup.close() } }
            }
        }
    }

    // === Setelan (gear rail) — anti-delete (fitur #2) ===
    Popup {
        id: settingsPopup
        width: 380; height: 560; modal: true
        anchors.centerIn: Overlay.overlay
        padding: 16
        onOpened: { app.act("GetProxy", []); app.act("GetRetention", []); app.act("GetBackgroundClose", []) }
        background: Rectangle { color: theme.bg; radius: 14; border.color: theme.line }
        ColumnLayout {
            anchors.fill: parent; spacing: 14
            Text { text: i18n.t("settings"); font.pixelSize: 18; font.bold: true; color: theme.text }
            RowLayout {
                Layout.fillWidth: true; spacing: 8
                Text { Layout.fillWidth: true; text: i18n.t("language"); color: theme.text; font.pixelSize: 14 }
                ComboBox {
                    implicitWidth: 150
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
            RowLayout {
                Layout.fillWidth: true; spacing: 10
                ColumnLayout {
                    Layout.fillWidth: true; spacing: 2
                    Text { text: i18n.t("keep_deleted"); color: theme.text; font.pixelSize: 14 }
                    Text { text: i18n.t("keep_deleted_sub"); color: theme.text2; font.pixelSize: 11 }
                }
                Switch { checked: app.keepDeleted; onToggled: app.setKeepDeleted(checked) }
            }
            RowLayout {
                Layout.fillWidth: true; spacing: 8
                Text { Layout.fillWidth: true; text: i18n.t("proxy"); color: theme.text; font.pixelSize: 14 }
                Rectangle { width: 150; height: 34; radius: 8; color: theme.searchBg; border.color: theme.line
                    TextInput { id: proxyInput; anchors.fill: parent; anchors.margins: 8; color: theme.text; font.pixelSize: 13; clip: true } }
                Button { text: i18n.t("set"); onClicked: app.act("SetProxy", [proxyInput.text]) }
            }
            RowLayout {
                Layout.fillWidth: true; spacing: 8
                Text { Layout.fillWidth: true; text: i18n.t("retention"); color: theme.text; font.pixelSize: 14 }
                SpinBox { id: retSpin; from: 0; to: 3650; value: 90; editable: true }
                Button { text: i18n.t("set"); onClicked: app.act("SetRetention", [retSpin.value]) }
            }
            RowLayout {
                Layout.fillWidth: true; spacing: 8
                Text { Layout.fillWidth: true; text: i18n.t("bg_close"); color: theme.text; font.pixelSize: 14 }
                Switch { onToggled: app.act("SetBackgroundClose", [checked]) }
            }
            Button { Layout.fillWidth: true; text: i18n.t("disappearing_7d"); onClicked: app.act("SetDefaultDisappearing", [604800]) }
            Button { Layout.fillWidth: true; text: i18n.t("storage"); onClicked: { app.loadDetail("GetStorageUsage", ""); settingsPopup.close(); detailPopup.open() } }
            Button { Layout.fillWidth: true; text: i18n.t("translate_example"); onClicked: app.fetchStr("Translate", ["Hello world", "id"]) }
            Button {
                Layout.fillWidth: true; text: i18n.t("privacy")
                onClicked: { app.loadDetail("GetPrivacy", ""); settingsPopup.close(); privacyPopup.open() }
            }
            Button {
                Layout.fillWidth: true; text: i18n.t("logout")
                onClicked: { app.logout(); settingsPopup.close() }
            }
            Item { Layout.fillHeight: true }
            Button { Layout.alignment: Qt.AlignRight; text: i18n.t("close"); onClicked: settingsPopup.close() }
        }
    }

    // === Detail grup / profil kontak (klik header) ===
    Popup {
        id: detailPopup
        width: 380; height: 470; modal: true
        anchors.centerIn: Overlay.overlay; padding: 0
        background: Rectangle { color: theme.bg; radius: 14; border.color: theme.line }
        ColumnLayout {
            anchors.fill: parent; spacing: 0
            Rectangle {
                Layout.fillWidth: true; Layout.preferredHeight: 130; color: theme.accent
                ColumnLayout {
                    anchors.centerIn: parent; spacing: 6
                    Rectangle {
                        Layout.alignment: Qt.AlignHCenter; width: 64; height: 64; radius: 32; color: "white"
                        Text { anchors.centerIn: parent; color: theme.accent; font.pixelSize: 26; font.bold: true
                            text: (app.detail.name || "?").charAt(0).toUpperCase() }
                    }
                    Text { Layout.alignment: Qt.AlignHCenter; text: app.detail.name || ""; color: "white"; font.pixelSize: 18; font.bold: true }
                }
            }
            Text {
                Layout.fillWidth: true; Layout.margins: 16; wrapMode: Text.WordWrap; color: theme.text2; font.pixelSize: 13
                text: app.detail.desc || app.detail.about || app.detail.phone || ""
            }
            Text {
                visible: !!app.detail.members
                Layout.leftMargin: 16; text: (app.detail.count || 0) + " " + i18n.t("members")
                color: theme.text; font.pixelSize: 13; font.bold: true
            }
            ListView {
                Layout.fillWidth: true; Layout.fillHeight: true; Layout.margins: 8; clip: true
                visible: !!app.detail.members
                model: app.detail.members || []
                delegate: RowLayout {
                    width: ListView.view.width; height: 44; clip: true; spacing: 10
                    Rectangle {
                        Layout.leftMargin: 8; width: 32; height: 32; radius: 16; color: theme.accent
                        Text { anchors.centerIn: parent; color: "white"; font.pixelSize: 13; font.bold: true
                            text: (modelData.name || "?").charAt(0).toUpperCase() }
                    }
                    Text { Layout.fillWidth: true; text: modelData.name || ""; color: theme.text; font.pixelSize: 14 }
                    Text { visible: modelData.admin === true; text: "admin"; color: theme.accent; font.pixelSize: 11 }
                }
            }
            // Admin grup (tampil saat detail grup)
            Flow {
                visible: !!app.detail.members
                Layout.fillWidth: true; Layout.leftMargin: 12; Layout.rightMargin: 12; spacing: 6
                Button { text: i18n.t("g_rename"); onClicked: win.prompt(i18n.t("g_rename"), win.selectedChat.name || "", function(v){ app.act("SetGroupSubject", [win.selectedChat.id, v]) }) }
                Button { text: i18n.t("g_desc"); onClicked: win.prompt(i18n.t("g_desc"), app.detail.desc || "", function(v){ app.act("SetGroupDescription", [win.selectedChat.id, v]) }) }
                Button { text: i18n.t("g_photo"); onClicked: app.act("SetGroupPhoto", [win.selectedChat.id, ""]) }
                Button { text: i18n.t("g_announce"); onClicked: app.act("SetGroupAnnounce", [win.selectedChat.id, true]) }
                Button { text: i18n.t("g_lock"); onClicked: app.act("SetGroupLocked", [win.selectedChat.id, true]) }
                Button { text: i18n.t("g_approval"); onClicked: app.act("SetGroupJoinApproval", [win.selectedChat.id, true]) }
                Button { text: i18n.t("g_addmode"); onClicked: app.act("SetGroupAddMode", [win.selectedChat.id, true]) }
                Button { text: i18n.t("g_add_member"); onClicked: app.act("UpdateGroupParticipants", [win.selectedChat.id, [], "add"]) }
                Button { text: i18n.t("g_invite"); onClicked: app.fetchStr("GroupInviteLink", [win.selectedChat.id, false]) }
                Button { text: i18n.t("g_requests"); onClicked: app.loadIntoA("GetGroupRequests", [win.selectedChat.id], starredModel) }
                Button { text: i18n.t("g_approve"); onClicked: app.act("UpdateGroupRequest", [win.selectedChat.id, [], true]) }
                Button { text: i18n.t("g_leave"); onClicked: { app.act("LeaveGroup", [win.selectedChat.id]); detailPopup.close() } }
            }
            Button { Layout.alignment: Qt.AlignRight; Layout.margins: 12; text: i18n.t("close"); onClicked: detailPopup.close() }
        }
    }

    // === Teruskan pesan (pilih chat tujuan) ===
    Popup {
        id: forwardPopup
        width: 360; height: 440; modal: true
        anchors.centerIn: Overlay.overlay; padding: 12
        background: Rectangle { color: theme.bg; radius: 14; border.color: theme.line }
        ColumnLayout {
            anchors.fill: parent; spacing: 8
            Text { text: i18n.t("forward_to"); color: theme.text; font.pixelSize: 16; font.bold: true }
            ListView {
                Layout.fillWidth: true; Layout.fillHeight: true; clip: true; model: chatsModel
                delegate: ItemDelegate {
                    width: ListView.view.width; height: 56; clip: true
                    onClicked: { app.forwardMsg(win.ctxMsg.id, model.m.id); forwardPopup.close() }
                    RowLayout {
                        anchors.fill: parent; anchors.leftMargin: 8; spacing: 10
                        Rectangle { width: 38; height: 38; radius: 19; color: theme.accent
                            Text { anchors.centerIn: parent; color: "white"; font.bold: true
                                text: (model.m.name || "?").charAt(0).toUpperCase() } }
                        Text { Layout.fillWidth: true; text: model.m.name || ""; color: theme.text; font.pixelSize: 15 }
                    }
                }
            }
        }
    }

    // === Lightbox media (klik foto/stiker) ===
    Rectangle {
        anchors.fill: parent; z: 150; visible: lightboxSrc !== ""; color: "#e6000000"
        Image {
            anchors.centerIn: parent; width: parent.width * 0.8; height: parent.height * 0.8
            fillMode: Image.PreserveAspectFit; source: lightboxSrc
        }
        Text { anchors.centerIn: parent; visible: parent.visible; text: "🖼️ (media dimuat dari engine)"; color: "#cccccc"; opacity: 0.5 }
        MouseArea { anchors.fill: parent; onClicked: win.lightboxSrc = "" }
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

    // === Detail reaksi (siapa react apa) ===
    Popup {
        id: reactionPopup
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

    // === Info pesan (tanda terima per-penerima) ===
    Popup {
        id: msgInfoPopup
        width: 330; height: 340; modal: true; anchors.centerIn: Overlay.overlay; padding: 14
        background: Rectangle { color: theme.bg; radius: 14; border.color: theme.line }
        ColumnLayout {
            anchors.fill: parent; spacing: 6
            Text { text: i18n.t("msg_info"); color: theme.text; font.pixelSize: 16; font.bold: true }
            Text { text: i18n.t("read") + " ✓✓"; color: theme.accent; font.pixelSize: 13; font.bold: true }
            Repeater {
                model: app.detail.readBy || []
                delegate: RowLayout {
                    Layout.fillWidth: true
                    Text { Layout.fillWidth: true; text: modelData.name || ""; color: theme.text; font.pixelSize: 13 }
                    Text { text: modelData.time || ""; color: theme.text2; font.pixelSize: 12 }
                }
            }
            Text { text: i18n.t("delivered") + " ✓"; color: theme.text2; font.pixelSize: 13; font.bold: true }
            Repeater {
                model: app.detail.deliveredTo || []
                delegate: RowLayout {
                    Layout.fillWidth: true
                    Text { Layout.fillWidth: true; text: modelData.name || ""; color: theme.text; font.pixelSize: 13 }
                    Text { text: modelData.time || ""; color: theme.text2; font.pixelSize: 12 }
                }
            }
            Item { Layout.fillHeight: true }
            Button { Layout.alignment: Qt.AlignRight; text: i18n.t("close"); onClicked: msgInfoPopup.close() }
        }
    }

    // === Privasi (dari settings) ===
    Popup {
        id: privacyPopup
        width: 380; height: 420; modal: true; anchors.centerIn: Overlay.overlay; padding: 16
        background: Rectangle { color: theme.bg; radius: 14; border.color: theme.line }
        ColumnLayout {
            anchors.fill: parent; spacing: 10
            Text { text: i18n.t("privacy_title"); color: theme.text; font.pixelSize: 18; font.bold: true }
            Repeater {
                model: [
                    { key: "lastseen", label: "Terakhir dilihat" },
                    { key: "profile", label: "Foto profil" },
                    { key: "status", label: "Status" },
                    { key: "readreceipts", label: "Laporan dibaca" },
                    { key: "groupadd", label: "Grup" },
                    { key: "online", label: "Sedang online" }
                ]
                delegate: RowLayout {
                    Layout.fillWidth: true; spacing: 10
                    Text { Layout.fillWidth: true; text: modelData.label; color: theme.text; font.pixelSize: 14 }
                    ComboBox {
                        implicitWidth: 130
                        model: ["everyone", "contacts", "nobody"]
                        currentIndex: Math.max(0, model.indexOf(app.detail[modelData.key] || "everyone"))
                        onActivated: app.setPrivacy(modelData.key, currentText)
                    }
                }
            }
            Item { Layout.fillHeight: true }
            Button { Layout.alignment: Qt.AlignRight; text: i18n.t("close"); onClicked: privacyPopup.close() }
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
                Button { text: i18n.t("cancel"); onClicked: docPopup.close() }
                Button {
                    text: i18n.t("send")
                    onClicked: { app.sendDocument(docDialog.picked, docName.text); docPopup.close() }
                }
            }
        }
    }

    // === Menu lampiran compose (lokasi/polling/kontak/mention/poll-vote) ===
    Menu {
        id: attachMenu
        MenuItem { text: "📄  " + i18n.t("a_document"); onTriggered: docDialog.open() }
        MenuItem { text: "🏷️  " + i18n.t("a_stickers"); onTriggered: { app.loadStickers(); stickerPopup.open() } }
        MenuItem { text: "🎬  " + i18n.t("a_gifs"); onTriggered: { app.loadGifs(); gifPopup.open() } }
        MenuItem { text: "🖼️  " + i18n.t("a_image"); onTriggered: { mediaDialog.kind = "image"; mediaDialog.open() } }
        MenuItem { text: "🎬  " + i18n.t("a_video"); onTriggered: { mediaDialog.kind = "video"; mediaDialog.open() } }
        MenuItem { text: "🎵  " + i18n.t("a_audio"); onTriggered: { mediaDialog.kind = "audio"; mediaDialog.open() } }
        MenuItem { text: "📍  " + i18n.t("a_location"); onTriggered: win.prompt(i18n.t("a_location"), "Jakarta", function(v){ app.act("SendLocation", [win.selectedChat.id, -6.2, 106.8, v]) }) }
        MenuItem { text: "📊  " + i18n.t("a_poll"); onTriggered: pollDialog.open() }
        MenuItem { text: "👤  " + i18n.t("a_contact"); onTriggered: contactDialog.open() }
        MenuItem { text: "@  " + i18n.t("a_mention"); onTriggered: win.prompt(i18n.t("a_mention"), "", function(v){ app.act("SendTextMentioned", [win.selectedChat.id, v, []]) }) }
        MenuItem { text: "🏷️  " + i18n.t("a_send_sticker"); onTriggered: { mediaDialog.kind = "sticker"; mediaDialog.open() } }
        MenuItem { text: "🎞️  " + i18n.t("a_send_gif"); onTriggered: { mediaDialog.kind = "gif"; mediaDialog.open() } }
        MenuItem { text: "🗳️  " + i18n.t("a_vote"); onTriggered: app.act("VotePoll", [win.selectedChat.id, win.selectedChat.id, win.ctxMsg.id || "p", ["Opsi A"]]) }
    }

    // Emoji picker ringkas (sisip ke composer) — paritas tombol emoji Composer.svelte.
    Menu {
        id: emojiMenu
        Repeater {
            model: ["😀", "😂", "😍", "👍", "🙏", "🎉", "❤️", "🔥", "😢", "😮"]
            delegate: MenuItem { text: modelData; onTriggered: composerInput.text += modelData }
        }
    }

    // FileDialog media (gambar/video/audio/sticker/gif) → kirim file nyata.
    FileDialog {
        id: mediaDialog
        property string kind: "image"
        onAccepted: app.sendMediaFile(kind, selectedFile)
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
                Button { text: i18n.t("cancel"); onClicked: pollDialog.close() }
                Button {
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
                Button { text: i18n.t("cancel"); onClicked: contactDialog.close() }
                Button { text: i18n.t("send"); onClicked: { if (ctName.text !== "" && ctPhone.text !== "") app.act("SendContact", [win.selectedChat.id, ctName.text, ctPhone.text]); contactDialog.close() } }
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
            Button { Layout.alignment: Qt.AlignRight; text: i18n.t("close"); onClicked: resultPopup.close() }
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
                Button { text: i18n.t("photo_video"); onClicked: { app.act("PostMediaStatus", ["image", statusInput.text, ""]); statusPostPopup.close() } }
                Button { text: i18n.t("send_text"); onClicked: { app.act("PostTextStatus", [statusInput.text, 0, 0]); statusInput.text = ""; statusPostPopup.close() } }
            }
        }
    }

    // === Aksi channel (klik-kanan baris channel) ===
    Menu {
        id: channelMenu
        MenuItem { text: "➕  " + i18n.t("ch_follow"); onTriggered: app.act("FollowChannelByJID", [win.ctxChat.id || ""]) }
        MenuItem { text: "➖  " + i18n.t("ch_unfollow"); onTriggered: app.act("UnfollowChannel", [win.ctxChat.id || ""]) }
        MenuItem { text: "🔇  " + i18n.t("ch_mute"); onTriggered: app.act("MuteChannel", [win.ctxChat.id || "", true]) }
        MenuItem { text: "📝  " + i18n.t("ch_post"); onTriggered: win.prompt(i18n.t("ch_post"), "", function(v){ app.act("PostChannel", [win.ctxChat.id || "", v]) }) }
        MenuItem { text: "👍  " + i18n.t("ch_react"); onTriggered: app.act("ReactChannel", [win.ctxChat.id || "", "m", 0, "👍"]) }
        MenuItem { text: "💬  " + i18n.t("ch_messages"); onTriggered: app.loadIntoA("GetChannelMessages", [win.ctxChat.id || ""], msgsModel) }
        MenuItem { text: "🔎  " + i18n.t("ch_recommend"); onTriggered: app.loadIntoA("GetRecommendedChannels", [""], channelsModel) }
        MenuItem { text: "✨  " + i18n.t("ch_create"); onTriggered: win.prompt(i18n.t("ch_create"), "", function(v){ app.act("CreateChannel", [v, ""]) }) }
    }

    // === Aksi kontak (klik-kanan baris kontak) ===
    Menu {
        id: contactMenu
        MenuItem { text: "🚫  " + i18n.t("ct_block"); onTriggered: app.act("Block", [win.ctxChat.id || "", true]) }
        MenuItem { text: "✅  " + i18n.t("ct_unblock"); onTriggered: app.act("Block", [win.ctxChat.id || "", false]) }
        MenuItem { text: "🏷️  " + i18n.t("ct_label"); onTriggered: win.prompt(i18n.t("ct_label"), win.ctxChat.name || "", function(v){ app.act("SaveContactLabel", [win.ctxChat.id || "", v]) }) }
        MenuItem { text: "🧹  " + i18n.t("ct_unlabel"); onTriggered: app.act("RemoveContactLabel", [win.ctxChat.id || ""]) }
        MenuItem { text: "ℹ️  " + i18n.t("ct_about"); onTriggered: app.fetchStr("GetContactAbout", [win.ctxChat.id || ""]) }
        MenuItem { text: "💼  " + i18n.t("ct_business"); onTriggered: app.loadDetailA("GetBusinessProfile", [win.ctxChat.id || ""]) }
        MenuItem { text: "👥  " + i18n.t("ct_common"); onTriggered: app.loadIntoA("GetCommonGroups", [win.ctxChat.id || ""], starredModel) }
        MenuItem { text: "📵  " + i18n.t("ct_blocklist"); onTriggered: app.loadInto("GetBlockedContacts", contactsModel) }
        MenuItem { text: "👁️  " + i18n.t("ct_presence"); onTriggered: app.act("SubscribePresence", [win.ctxChat.id || ""]) }
    }

    // === Aksi item terjadwal (klik-kanan) ===
    Menu {
        id: schedMenu
        MenuItem { text: "➕  " + i18n.t("s_schedule"); onTriggered: win.prompt(i18n.t("s_schedule"), "", function(v){ app.act("ScheduleMessage", [win.selectedChat.id || "", v, 0]) }) }
        MenuItem { text: "❌  " + i18n.t("s_cancel"); onTriggered: { app.act("CancelScheduled", [win.ctxChat.id || ""]); app.loadInto("GetScheduled", scheduledModel) } }
        MenuItem { text: "⏰  " + i18n.t("s_add_reminder"); onTriggered: app.act("AddReminder", [win.selectedChat.id || "", win.ctxMsg.id || "", "Ingat ini", 0]) }
        MenuItem { text: "🗑️  " + i18n.t("s_del_reminder"); onTriggered: app.act("CancelReminder", [win.ctxChat.id || ""]) }
        MenuItem { text: "📋  " + i18n.t("s_reminders"); onTriggered: app.loadInto("GetReminders", scheduledModel) }
    }

    // === Overflow header: utilitas (tutup permukaan method engine sisanya) ===
    Menu {
        id: overflowMenu
        property string cid: win.selectedChat.id || ""
        MenuItem { text: i18n.t("o_chat_media"); onTriggered: app.loadIntoA("GetChatMedia", [overflowMenu.cid], msgsModel) }
        MenuItem { text: i18n.t("o_pinned"); onTriggered: app.loadIntoA("GetPinned", [overflowMenu.cid], msgsModel) }
        MenuItem { text: i18n.t("o_poll_votes"); onTriggered: app.loadDetailA("GetPollVotes", [win.ctxMsg.id || "p"]) }
        MenuItem { text: i18n.t("o_link_preview"); onTriggered: app.fetchStr("GetLinkPreview", ["https://example.com"]) }
        MenuItem { text: i18n.t("o_fetch_media"); onTriggered: app.fetchStr("FetchRemoteMedia", ["https://example.com/a.jpg"]) }
        MenuItem { text: i18n.t("o_check_wa"); onTriggered: app.act("IsOnWhatsApp", [["6281234567890"]]) }
        MenuItem { text: i18n.t("o_search_stickers"); onTriggered: app.loadIntoA("SearchStickers", ["happy", ""], stickersModel) }
        MenuItem { text: i18n.t("o_search_gifs"); onTriggered: app.loadIntoA("SearchGifs", ["happy", ""], gifsModel) }
        MenuItem { text: i18n.t("o_open_chat"); onTriggered: app.act("OpenChat", [overflowMenu.cid]) }
        MenuItem { text: i18n.t("o_load_old"); onTriggered: app.act("LoadOlderHistory", [overflowMenu.cid]) }
        MenuItem { text: i18n.t("o_mark_unread"); onTriggered: app.act("MarkUnread", [overflowMenu.cid]) }
        MenuItem { text: i18n.t("o_clear"); onTriggered: app.act("ClearChat", [overflowMenu.cid]) }
        MenuItem { text: i18n.t("o_export"); onTriggered: app.fetchStr("ExportChat", [overflowMenu.cid]) }
        MenuItem { text: i18n.t("o_disappearing"); onTriggered: app.act("SetDisappearing", [overflowMenu.cid, 604800]) }
        MenuItem { text: i18n.t("o_profile"); onTriggered: { app.loadDetail("GetProfile", ""); detailPopup.open() } }
        MenuItem { text: i18n.t("o_set_name"); onTriggered: win.prompt(i18n.t("o_set_name"), "", function(v){ app.act("SetMyName", [v]) }) }
        MenuItem { text: i18n.t("o_set_about"); onTriggered: win.prompt(i18n.t("o_set_about"), "", function(v){ app.act("SetMyAbout", [v]) }) }
        MenuItem { text: i18n.t("o_set_photo"); onTriggered: app.act("SetMyPhoto", ["", ""]) }
        MenuItem { text: i18n.t("o_version"); onTriggered: app.fetchStr("Version", []) }
        MenuItem { text: i18n.t("o_status_viewers"); onTriggered: app.loadIntoA("GetStatusViewers", ["st1"], starredModel) }
        MenuItem { text: i18n.t("o_react_status"); onTriggered: app.act("ReactStatus", [overflowMenu.cid, "st1", "👍"]) }
        MenuItem { text: i18n.t("o_reply_status"); onTriggered: app.act("ReplyStatus", [overflowMenu.cid, "st1", "teks", "balas"]) }
        MenuItem { text: i18n.t("o_create_group"); onTriggered: win.prompt(i18n.t("o_create_group"), "", function(v){ app.act("CreateGroup", [v, []]) }) }
        MenuItem { text: i18n.t("o_join_link"); onTriggered: win.prompt(i18n.t("o_join_link"), "https://chat.whatsapp.com/", function(v){ app.fetchStr("JoinGroupLink", [v]) }) }
        MenuItem { text: i18n.t("o_preview_link"); onTriggered: win.prompt(i18n.t("o_preview_link"), "https://chat.whatsapp.com/", function(v){ app.fetchStr("PreviewGroupLink", [v]) }) }
        MenuItem { text: i18n.t("o_follow_channel"); onTriggered: win.prompt(i18n.t("o_follow_channel"), "https://whatsapp.com/channel/", function(v){ app.loadDetailA("FollowChannel", [v]) }) }
        MenuItem { text: i18n.t("o_leave_community"); onTriggered: app.act("LeaveCommunity", [overflowMenu.cid]) }
        MenuItem { text: i18n.t("o_reject_call"); onTriggered: app.act("RejectCall", [overflowMenu.cid, "callid"]) }
        MenuItem { text: i18n.t("o_post_status"); onTriggered: statusPostPopup.open() }
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
                Button { text: i18n.t("cancel"); onClicked: promptDialog.close() }
                Button { text: i18n.t("save"); onClicked: { if (promptDialog.cb) promptDialog.cb(promptInput.text); promptDialog.close() } }
            }
        }
    }

    // === Gerbang login QR — tampil bila belum terhubung ===
    Rectangle {
        anchors.fill: parent
        z: 100
        visible: app.state !== "" && app.state !== "ready" && app.state !== "connected"
        color: theme.bg
        ColumnLayout {
            anchors.centerIn: parent; spacing: 18; width: 340
            Text { Layout.alignment: Qt.AlignHCenter; text: "WhatsLite"; font.pixelSize: 28; font.bold: true; color: theme.accent }
            Text {
                Layout.fillWidth: true; horizontalAlignment: Text.AlignHCenter; wrapMode: Text.WordWrap
                text: i18n.t("link_hint")
                color: theme.text2; font.pixelSize: 13
            }
            Rectangle {
                Layout.alignment: Qt.AlignHCenter; width: 240; height: 240; radius: 12
                color: "white"; border.color: theme.line
                Image {
                    anchors.fill: parent; anchors.margins: 10; fillMode: Image.PreserveAspectFit
                    source: app.qr; visible: app.qr !== ""
                }
                Text { anchors.centerIn: parent; visible: app.qr === ""; text: app.state || "menghubungkan…"; color: "#555" }
            }
            Button { Layout.alignment: Qt.AlignHCenter; text: i18n.t("connect"); onClicked: app.doConnect() }
            Button { Layout.alignment: Qt.AlignHCenter; text: i18n.t("link_code"); onClicked: app.fetchStr("AddViaQR", [""]) }
            Button { Layout.alignment: Qt.AlignHCenter; text: i18n.t("link_phone"); onClicked: app.fetchStr("LinkWithPhone", ["6281234567890"]) }
}
    }

    // Auto-buka panel (uji/screenshot) bila diminta via env WALITE_OPEN.
    Timer {
        running: (typeof openPanel !== "undefined") && openPanel !== ""
        interval: 1500; repeat: false
        onTriggered: {
            if (openPanel === "sticker") { app.loadStickers(); stickerPopup.open() }
            else if (openPanel === "gif") { app.loadGifs(); gifPopup.open() }
            else if (openPanel === "settings") settingsPopup.open()
            else if (openPanel === "search") { searchInput.text = "rapat"; app.search("rapat", searchModel) }
            else if (openPanel === "detail") { app.loadDetail("GetGroupInfo", "g"); detailPopup.open() }
            else if (openPanel === "forward") { win.ctxMsg = { id: "m1" }; forwardPopup.open() }
            else if (openPanel === "privacy") { app.loadDetail("GetPrivacy", ""); privacyPopup.open() }
            else if (openPanel === "msginfo") { app.loadDetailA("GetMessageInfo", ["c", "m1"]); msgInfoPopup.open() }
            else if (openPanel === "reaction") { win.ctxMsg = { reactions: [{ emoji: "👍", count: 2, who: ["Alice", "Bob"] }, { emoji: "❤️", count: 1, who: ["Citra"] }] }; reactionPopup.open() }
            else if (openPanel === "poll") pollDialog.open()
            else if (openPanel === "contact") contactDialog.open()
            else { activeView = openPanel; win.loadView(openPanel) } // calls/starred/status/contacts/channels/communities/archived/scheduled
        }
    }
}
