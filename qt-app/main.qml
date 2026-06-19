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
    property string lightboxSrc: ""  // media fullscreen (kosong = tutup)
    property bool locked: (typeof startLock !== "undefined") && startLock // app-lock PIN
    property var replyTo: null        // pesan yang sedang dibalas (banner composer)
    property var ctxChat: ({})        // chat target context-menu baris

    // --- Token tema (light + dark) — cocok dgn app.css [data-theme] ---
    QtObject {
        id: theme
        property bool dark: (typeof startDark !== "undefined") ? startDark : false
        readonly property color railBg: dark ? "#11161d" : "#f4f6fa"
        readonly property color railIco: dark ? "#8a97a3" : "#6b7785"
        readonly property color accent: dark ? "#06c98c" : "#06b67f"
        readonly property color sidebarBg: dark ? "#0e1318" : "#ffffff"
        readonly property color bg: dark ? "#1a232a" : "#ffffff"
        readonly property color bg2: dark ? "#222e35" : "#f0f2f5"
        readonly property color line: dark ? "#2a3942" : "#e4e8ee"
        readonly property color searchBg: dark ? "#1b232b" : "#eef1f6"
        readonly property color wallpaper: dark ? "#0a0f14" : "#eef1f6"
        readonly property color inBg: dark ? "#1d262e" : "#ffffff"
        readonly property color outBg: dark ? "#114b39" : "#d6f3c4"
        readonly property color text: dark ? "#e7ecf0" : "#0f1722"
        readonly property color text2: dark ? "#8a97a3" : "#6b7785"
        readonly property color hover: dark ? "#161d24" : "#f2f4f8"
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
                        color: (activeView === modelData.view) ? Qt.rgba(0.02, 0.71, 0.5, 0.14) : "transparent"
                        Text { anchors.centerIn: parent; text: modelData.icon; font.pixelSize: 20 }
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
                // Header
                Rectangle {
                    Layout.fillWidth: true; Layout.preferredHeight: 60
                    color: theme.sidebarBg
                    Text {
                        anchors.verticalCenter: parent.verticalCenter
                        anchors.left: parent.left; anchors.leftMargin: 16
                        text: "WhatsLite"; font.pixelSize: 22; font.bold: true; color: theme.text
                    }
                }
                // Search (FTS pesan)
                Rectangle {
                    Layout.fillWidth: true; Layout.preferredHeight: 44
                    Layout.margins: 8; radius: 10; color: theme.searchBg
                    TextInput {
                        id: searchInput
                        anchors.fill: parent; anchors.leftMargin: 14; anchors.rightMargin: 14
                        verticalAlignment: TextInput.AlignVCenter
                        color: theme.text; font.pixelSize: 14; clip: true
                        onTextChanged: app.search(text, searchModel)
                    }
                    Text {
                        visible: searchInput.text === ""
                        anchors.verticalCenter: parent.verticalCenter
                        anchors.left: parent.left; anchors.leftMargin: 14
                        text: i18n.t("search"); color: theme.text2; font.pixelSize: 14
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
                        delegate: ItemDelegate {
                            width: chatList.width; height: 68
                            onClicked: { win.selectedChat = model.m; app.openChat(model.m.id) }
                            background: Rectangle { color: hovered ? theme.hover : "transparent" }
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 12; anchors.rightMargin: 12; spacing: 12
                                Rectangle {
                                    width: 49; height: 49; radius: 24.5; color: theme.accent
                                    Text {
                                        anchors.centerIn: parent; color: "white"; font.pixelSize: 18; font.bold: true
                                        text: (model.m.name || "?").charAt(0).toUpperCase()
                                    }
                                }
                                ColumnLayout {
                                    Layout.fillWidth: true; spacing: 2
                                    Text {
                                        Layout.fillWidth: true; elide: Text.ElideRight
                                        text: model.m.name || model.m.id || ""
                                        font.pixelSize: 16; font.weight: Font.Medium; color: theme.text
                                    }
                                    Text {
                                        Layout.fillWidth: true; elide: Text.ElideRight
                                        text: model.m.preview || ""; font.pixelSize: 13; color: theme.text2
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
                            width: ListView.view.width; height: 64
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12; spacing: 12
                                Text { text: model.m.video ? "📹" : "📞"; font.pixelSize: 20 }
                                ColumnLayout {
                                    Layout.fillWidth: true; spacing: 2
                                    Text { text: model.m.name || ""; color: theme.text; font.pixelSize: 15; font.weight: Font.Medium }
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
                            width: ListView.view.width; height: 62
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
                            width: ListView.view.width; height: 64
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12; spacing: 12
                                Rectangle {
                                    width: 46; height: 46; radius: 23
                                    color: "transparent"; border.width: 2
                                    border.color: model.m.seen ? theme.text2 : theme.accent
                                    Rectangle {
                                        anchors.centerIn: parent; width: 40; height: 40; radius: 20; color: theme.accent
                                        Text { anchors.centerIn: parent; color: "white"; font.pixelSize: 16; font.bold: true
                                            text: (model.m.name || "?").charAt(0).toUpperCase() }
                                    }
                                }
                                ColumnLayout {
                                    Layout.fillWidth: true; spacing: 2
                                    Text { text: model.m.name || ""; color: theme.text; font.pixelSize: 15; font.weight: Font.Medium }
                                    Text { text: (model.m.count || 0) + " pembaruan · " + (model.m.time || "")
                                        color: theme.text2; font.pixelSize: 12 }
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
                            width: ListView.view.width; height: 60
                            onClicked: { win.selectedChat = { name: model.m.name, id: model.m.jid }; activeView = "chats"; app.openChat(model.m.jid) }
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12; spacing: 12
                                Rectangle {
                                    width: 42; height: 42; radius: 21; color: theme.accent
                                    Text { anchors.centerIn: parent; color: "white"; font.pixelSize: 16; font.bold: true
                                        text: (model.m.name || "?").charAt(0).toUpperCase() }
                                }
                                ColumnLayout {
                                    Layout.fillWidth: true; spacing: 2
                                    Text { text: model.m.name || ""; color: theme.text; font.pixelSize: 15; font.weight: Font.Medium }
                                    Text { Layout.fillWidth: true; elide: Text.ElideRight
                                        text: model.m.about || ""; color: theme.text2; font.pixelSize: 12 }
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
                            width: ListView.view.width; height: 64
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12; spacing: 12
                                Text { text: "📢"; font.pixelSize: 22 }
                                ColumnLayout {
                                    Layout.fillWidth: true; spacing: 2
                                    Text { text: model.m.name || ""; color: theme.text; font.pixelSize: 15; font.weight: Font.Medium }
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
                            width: ListView.view.width; height: 64
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12; spacing: 12
                                Rectangle { width: 42; height: 42; radius: 10; color: theme.accent
                                    Text { anchors.centerIn: parent; text: "👥"; font.pixelSize: 18 } }
                                ColumnLayout {
                                    Layout.fillWidth: true; spacing: 2
                                    Text { text: model.m.name || ""; color: theme.text; font.pixelSize: 15; font.weight: Font.Medium }
                                    Text { text: model.m.subtitle || ""; color: theme.text2; font.pixelSize: 12 }
                                }
                            }
                        }
                    }
                    ListView {
                        anchors.fill: parent; visible: activeView === "archived" && searchInput.text === ""
                        clip: true; model: archivedModel
                        delegate: Item {
                            width: ListView.view.width; height: 64
                            RowLayout {
                                anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 12; spacing: 12
                                Rectangle { width: 42; height: 42; radius: 21; color: theme.text2
                                    Text { anchors.centerIn: parent; color: "white"; font.pixelSize: 16; font.bold: true
                                        text: (model.m.name || "?").charAt(0).toUpperCase() } }
                                ColumnLayout {
                                    Layout.fillWidth: true; spacing: 2
                                    Text { text: model.m.name || ""; color: theme.text; font.pixelSize: 15; font.weight: Font.Medium }
                                    Text { Layout.fillWidth: true; elide: Text.ElideRight; text: model.m.preview || ""; color: theme.text2; font.pixelSize: 12 }
                                }
                            }
                        }
                    }
                    ListView {
                        anchors.fill: parent; visible: activeView === "scheduled" && searchInput.text === ""
                        clip: true; model: scheduledModel
                        delegate: Item {
                            width: ListView.view.width; height: 64
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
                            width: ListView.view.width; height: 62
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
                color: theme.bg; border.color: theme.line
                Text {
                    anchors.verticalCenter: parent.verticalCenter
                    anchors.left: parent.left; anchors.leftMargin: 16
                    text: win.selectedChat.name || i18n.t("pick_conversation"); font.pixelSize: 16; font.bold: true; color: theme.text
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
                // Overflow ⋮ — utilitas (media/pin/poll/profil/grup/status/dll).
                Rectangle {
                    anchors.right: parent.right; anchors.rightMargin: 12
                    anchors.verticalCenter: parent.verticalCenter
                    width: 36; height: 36; radius: 18; color: "transparent"
                    Text { anchors.centerIn: parent; text: "⋮"; color: theme.text; font.pixelSize: 22 }
                    MouseArea { anchors.fill: parent; onClicked: overflowMenu.popup() }
                }
            }
            // Timeline — pola tervalidasi (ListView + reuseItems), bubble in/out
            Rectangle {
                Layout.fillWidth: true; Layout.fillHeight: true
                color: theme.wallpaper
                ListView {
                    id: timeline
                    anchors.fill: parent
                    anchors.margins: 12
                    clip: true
                    model: msgsModel
                    reuseItems: true
                    spacing: 6
                    header: Item {
                        width: timeline.width; height: 40
                        Button {
                            anchors.centerIn: parent; flat: true; text: "↑ " + i18n.t("load_older")
                            visible: timeline.count > 0
                            onClicked: app.loadOlder()
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
                            radius: 8
                            color: parent.out ? theme.outBg : theme.inBg
                            border.color: theme.line
                            ColumnLayout {
                                id: content
                                anchors.left: parent.left
                                anchors.top: parent.top
                                anchors.margins: 8
                                spacing: 3
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
                                // Teks biasa
                                Text {
                                    visible: model.m.type !== "document" && model.m.type !== "sticker" && model.m.type !== "gif"
                                    text: model.m.text || ""
                                    wrapMode: Text.WordWrap; color: theme.text; font.pixelSize: 15
                                    Layout.maximumWidth: timeline.width * 0.66
                                }
                                Text {
                                    Layout.alignment: Qt.AlignRight
                                    text: model.m.time || ""
                                    color: theme.text2; font.pixelSize: 11
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
                    anchors.fill: parent; anchors.margins: 8; spacing: 8
                    // Menu lampiran (lokasi/polling/kontak/mention).
                    Rectangle {
                        width: 40; height: 40; radius: 20; color: "transparent"
                        Text { anchors.centerIn: parent; text: "➕"; font.pixelSize: 18 }
                        MouseArea { anchors.fill: parent; onClicked: attachMenu.popup() }
                    }
                    // Lampirkan dokumen → pilih file → rename/cut → kirim.
                    Rectangle {
                        width: 40; height: 40; radius: 20; color: "transparent"
                        Text { anchors.centerIn: parent; text: "📎"; font.pixelSize: 19 }
                        MouseArea { anchors.fill: parent; onClicked: docDialog.open() }
                    }
                    // Tombol stiker → picker koleksi (fitur #1).
                    Rectangle {
                        width: 40; height: 40; radius: 20; color: "transparent"
                        Text { anchors.centerIn: parent; text: "🏷️"; font.pixelSize: 20 }
                        MouseArea { anchors.fill: parent; onClicked: { app.loadStickers(); stickerPopup.open() } }
                    }
                    // Tombol GIF → picker koleksi (fitur #3).
                    Rectangle {
                        width: 40; height: 40; radius: 20; color: "transparent"
                        Text { anchors.centerIn: parent; text: "🎬"; font.pixelSize: 18 }
                        MouseArea { anchors.fill: parent; onClicked: { app.loadGifs(); gifPopup.open() } }
                    }
                    Rectangle {
                        Layout.fillWidth: true; Layout.fillHeight: true
                        radius: 18; color: theme.bg; border.color: theme.line
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
                    Rectangle {
                        id: sendBtn
                        width: 40; height: 40; radius: 20; color: theme.accent
                        Text { anchors.centerIn: parent; text: "➤"; color: "white"; font.pixelSize: 16 }
                        MouseArea { anchors.fill: parent; onClicked: composerInput.parent.send() }
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
            visible: win.ctxMsg.reactions !== undefined && win.ctxMsg.reactions.length > 0
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
                Text { Layout.fillWidth: true; text: "Proxy"; color: theme.text; font.pixelSize: 14 }
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
                visible: app.detail.members !== undefined
                Layout.leftMargin: 16; text: (app.detail.count || 0) + " anggota"
                color: theme.text; font.pixelSize: 13; font.bold: true
            }
            ListView {
                Layout.fillWidth: true; Layout.fillHeight: true; Layout.margins: 8; clip: true
                visible: app.detail.members !== undefined
                model: app.detail.members || []
                delegate: RowLayout {
                    width: ListView.view.width; height: 44; spacing: 10
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
                visible: app.detail.members !== undefined
                Layout.fillWidth: true; Layout.leftMargin: 12; Layout.rightMargin: 12; spacing: 6
                Button { text: "Rename"; onClicked: app.act("SetGroupSubject", [win.selectedChat.id, "Grup Baru"]) }
                Button { text: "Deskripsi"; onClicked: app.act("SetGroupDescription", [win.selectedChat.id, "Deskripsi baru"]) }
                Button { text: "Foto"; onClicked: app.act("SetGroupPhoto", [win.selectedChat.id, ""]) }
                Button { text: "Announce"; onClicked: app.act("SetGroupAnnounce", [win.selectedChat.id, true]) }
                Button { text: "Kunci"; onClicked: app.act("SetGroupLocked", [win.selectedChat.id, true]) }
                Button { text: "Approval"; onClicked: app.act("SetGroupJoinApproval", [win.selectedChat.id, true]) }
                Button { text: "AddMode"; onClicked: app.act("SetGroupAddMode", [win.selectedChat.id, true]) }
                Button { text: "Tambah anggota"; onClicked: app.act("UpdateGroupParticipants", [win.selectedChat.id, [], "add"]) }
                Button { text: "Link undangan"; onClicked: app.fetchStr("GroupInviteLink", [win.selectedChat.id, false]) }
                Button { text: "Permintaan"; onClicked: app.loadIntoA("GetGroupRequests", [win.selectedChat.id], starredModel) }
                Button { text: "Setujui"; onClicked: app.act("UpdateGroupRequest", [win.selectedChat.id, [], true]) }
                Button { text: "Keluar"; onClicked: { app.act("LeaveGroup", [win.selectedChat.id]); detailPopup.close() } }
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
                    width: ListView.view.width; height: 56
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
                    width: ListView.view.width; height: 40; spacing: 10
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
                    text: "Kirim"
                    onClicked: { app.sendDocument(docDialog.picked, docName.text); docPopup.close() }
                }
            }
        }
    }

    // === Menu lampiran compose (lokasi/polling/kontak/mention/poll-vote) ===
    Menu {
        id: attachMenu
        MenuItem { text: "📍  Lokasi"; onTriggered: app.act("SendLocation", [win.selectedChat.id, -6.2, 106.8, "Jakarta"]) }
        MenuItem { text: "📊  Polling"; onTriggered: app.act("SendPoll", [win.selectedChat.id, "Pilih opsi?", ["Opsi A", "Opsi B"], 1]) }
        MenuItem { text: "👤  Kontak"; onTriggered: app.act("SendContact", [win.selectedChat.id, "Kontak Baru", "+6281234567890"]) }
        MenuItem { text: "@  Mention semua"; onTriggered: app.act("SendTextMentioned", [win.selectedChat.id, "halo semua", []]) }
        MenuItem { text: "🗳️  Vote polling (contoh)"; onTriggered: app.act("VotePoll", [win.selectedChat.id, win.selectedChat.id, win.ctxMsg.id || "p", ["Opsi A"]]) }
        MenuItem { text: "🖼️  Kirim stiker (file)"; onTriggered: app.act("SendSticker", [win.selectedChat.id, ""]) }
        MenuItem { text: "🎞️  Kirim GIF (file)"; onTriggered: app.act("SendGif", [win.selectedChat.id, ""]) }
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
        MenuItem { text: "➕  Ikuti"; onTriggered: app.act("FollowChannelByJID", [win.ctxChat.id || ""]) }
        MenuItem { text: "➖  Berhenti ikuti"; onTriggered: app.act("UnfollowChannel", [win.ctxChat.id || ""]) }
        MenuItem { text: "🔇  Bisukan"; onTriggered: app.act("MuteChannel", [win.ctxChat.id || "", true]) }
        MenuItem { text: "📝  Posting"; onTriggered: app.act("PostChannel", [win.ctxChat.id || "", "Halo pengikut"]) }
        MenuItem { text: "👍  Reaksi"; onTriggered: app.act("ReactChannel", [win.ctxChat.id || "", "m", 0, "👍"]) }
        MenuItem { text: "💬  Lihat pesan"; onTriggered: app.loadIntoA("GetChannelMessages", [win.ctxChat.id || ""], msgsModel) }
        MenuItem { text: "🔎  Rekomendasi"; onTriggered: app.loadIntoA("GetRecommendedChannels", [""], channelsModel) }
        MenuItem { text: "✨  Buat channel"; onTriggered: app.act("CreateChannel", ["Channel Baru", "deskripsi"]) }
    }

    // === Aksi kontak (klik-kanan baris kontak) ===
    Menu {
        id: contactMenu
        MenuItem { text: "🚫  Blokir"; onTriggered: app.act("Block", [win.ctxChat.id || "", true]) }
        MenuItem { text: "✅  Buka blokir"; onTriggered: app.act("Block", [win.ctxChat.id || "", false]) }
        MenuItem { text: "🏷️  Beri label"; onTriggered: app.act("SaveContactLabel", [win.ctxChat.id || "", "Penting"]) }
        MenuItem { text: "🧹  Hapus label"; onTriggered: app.act("RemoveContactLabel", [win.ctxChat.id || ""]) }
        MenuItem { text: "ℹ️  Tentang"; onTriggered: app.fetchStr("GetContactAbout", [win.ctxChat.id || ""]) }
        MenuItem { text: "💼  Profil bisnis"; onTriggered: app.loadDetailA("GetBusinessProfile", [win.ctxChat.id || ""]) }
        MenuItem { text: "👥  Grup bersama"; onTriggered: app.loadIntoA("GetCommonGroups", [win.ctxChat.id || ""], starredModel) }
        MenuItem { text: "📵  Daftar blokir"; onTriggered: app.loadInto("GetBlockedContacts", contactsModel) }
        MenuItem { text: "👁️  Langganan presence"; onTriggered: app.act("SubscribePresence", [win.ctxChat.id || ""]) }
    }

    // === Aksi item terjadwal (klik-kanan) ===
    Menu {
        id: schedMenu
        MenuItem { text: "➕  Jadwalkan pesan"; onTriggered: app.act("ScheduleMessage", [win.selectedChat.id || "", "Pesan terjadwal", 0]) }
        MenuItem { text: "❌  Batalkan jadwal"; onTriggered: { app.act("CancelScheduled", [win.ctxChat.id || ""]); app.loadInto("GetScheduled", scheduledModel) } }
        MenuItem { text: "⏰  Tambah pengingat"; onTriggered: app.act("AddReminder", [win.selectedChat.id || "", win.ctxMsg.id || "", "Ingat ini", 0]) }
        MenuItem { text: "🗑️  Hapus pengingat"; onTriggered: app.act("CancelReminder", [win.ctxChat.id || ""]) }
        MenuItem { text: "📋  Daftar pengingat"; onTriggered: app.loadInto("GetReminders", scheduledModel) }
    }

    // === Overflow header: utilitas (tutup permukaan method engine sisanya) ===
    Menu {
        id: overflowMenu
        property string cid: win.selectedChat.id || ""
        MenuItem { text: "Media chat"; onTriggered: app.loadIntoA("GetChatMedia", [overflowMenu.cid], msgsModel) }
        MenuItem { text: "Pesan disematkan"; onTriggered: app.loadIntoA("GetPinned", [overflowMenu.cid], msgsModel) }
        MenuItem { text: "Hasil polling"; onTriggered: app.loadDetailA("GetPollVotes", [win.ctxMsg.id || "p"]) }
        MenuItem { text: "Pratinjau tautan"; onTriggered: app.fetchStr("GetLinkPreview", ["https://example.com"]) }
        MenuItem { text: "Ambil media URL"; onTriggered: app.fetchStr("FetchRemoteMedia", ["https://example.com/a.jpg"]) }
        MenuItem { text: "Cek nomor di WA"; onTriggered: app.act("IsOnWhatsApp", [["6281234567890"]]) }
        MenuItem { text: "Cari stiker online"; onTriggered: app.loadIntoA("SearchStickers", ["happy", ""], stickersModel) }
        MenuItem { text: "Cari GIF online"; onTriggered: app.loadIntoA("SearchGifs", ["happy", ""], gifsModel) }
        MenuItem { text: "Buka chat (server)"; onTriggered: app.act("OpenChat", [overflowMenu.cid]) }
        MenuItem { text: "Muat riwayat lama"; onTriggered: app.act("LoadOlderHistory", [overflowMenu.cid]) }
        MenuItem { text: "Tandai belum dibaca"; onTriggered: app.act("MarkUnread", [overflowMenu.cid]) }
        MenuItem { text: "Bersihkan chat"; onTriggered: app.act("ClearChat", [overflowMenu.cid]) }
        MenuItem { text: "Ekspor chat"; onTriggered: app.fetchStr("ExportChat", [overflowMenu.cid]) }
        MenuItem { text: "Pesan sementara 7h"; onTriggered: app.act("SetDisappearing", [overflowMenu.cid, 604800]) }
        MenuItem { text: "Profil saya"; onTriggered: { app.loadDetail("GetProfile", ""); detailPopup.open() } }
        MenuItem { text: "Ubah nama saya"; onTriggered: app.act("SetMyName", ["Nama Saya"]) }
        MenuItem { text: "Ubah about"; onTriggered: app.act("SetMyAbout", ["Tentang saya"]) }
        MenuItem { text: "Ubah foto saya"; onTriggered: app.act("SetMyPhoto", ["", ""]) }
        MenuItem { text: "Versi app"; onTriggered: app.fetchStr("Version", []) }
        MenuItem { text: "Penonton status"; onTriggered: app.loadIntoA("GetStatusViewers", ["st1"], starredModel) }
        MenuItem { text: "Reaksi status"; onTriggered: app.act("ReactStatus", [overflowMenu.cid, "st1", "👍"]) }
        MenuItem { text: "Balas status"; onTriggered: app.act("ReplyStatus", [overflowMenu.cid, "st1", "teks", "balas"]) }
        MenuItem { text: "Buat grup"; onTriggered: app.act("CreateGroup", ["Grup Baru", []]) }
        MenuItem { text: "Gabung grup via link"; onTriggered: app.fetchStr("JoinGroupLink", ["https://chat.whatsapp.com/xxx"]) }
        MenuItem { text: "Pratinjau link grup"; onTriggered: app.fetchStr("PreviewGroupLink", ["https://chat.whatsapp.com/xxx"]) }
        MenuItem { text: "Ikuti channel via link"; onTriggered: app.loadDetailA("FollowChannel", ["https://whatsapp.com/channel/xxx"]) }
        MenuItem { text: "Keluar komunitas"; onTriggered: app.act("LeaveCommunity", [overflowMenu.cid]) }
        MenuItem { text: "Tolak panggilan"; onTriggered: app.act("RejectCall", [overflowMenu.cid, "callid"]) }
        MenuItem { text: "Posting status…"; onTriggered: statusPostPopup.open() }
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
            else { activeView = openPanel; win.loadView(openPanel) } // calls/starred/status/contacts/channels/communities/archived/scheduled
        }
    }
}
