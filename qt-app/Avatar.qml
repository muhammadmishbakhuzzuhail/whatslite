// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// Avatar — lingkaran foto profil (dimuat via <base>/avatar/<jid>) dengan
// fallback inisial nama. Foto di-mask bundar (OpacityMask).
import QtQuick
import Qt5Compat.GraphicalEffects

Rectangle {
    id: root
    property string name: ""
    property string jid: ""
    property string base: ""   // mediaBase engine
    property color accent: "#06b67f"
    property real fontSize: 19
    property bool group: false // grup → siluet orang-banyak (bukan inisial)
    property int weight: Font.Bold // app.css .avatar 700; .pv-av (blokir) 600

    implicitWidth: 44
    implicitHeight: 44
    radius: width / 2
    color: accent

    Text {
        anchors.centerIn: parent
        visible: img.status !== Image.Ready && !root.group
        color: "white"; font.pixelSize: root.fontSize; font.weight: root.weight
        text: (root.name || "?").charAt(0).toUpperCase()
    }
    // Siluet grup (default WhatsApp) bila grup tanpa foto.
    Image {
        anchors.centerIn: parent; visible: img.status !== Image.Ready && root.group
        width: parent.width * 0.6; height: parent.height * 0.6
        source: "data:image/svg+xml;utf8," + encodeURIComponent(
            '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="white"><path d="M16 11c1.66 0 3-1.34 3-3s-1.34-3-3-3-3 1.34-3 3 1.34 3 3 3zm-8 0c1.66 0 3-1.34 3-3S9.66 5 8 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V18h14v-1.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.99 1.97 3.45V18h6v-1.5c0-2.33-4.67-3.5-7-3.5z"/></svg>')
    }
    Image {
        id: img
        anchors.fill: parent
        fillMode: Image.PreserveAspectCrop
        // Svelte avatarUrl(): /avatar/<encodeURIComponent(jid)>. jid berisi '@',':' → WAJIB encode.
        source: (root.base && root.jid) ? (root.base + "/avatar/" + encodeURIComponent(root.jid)) : ""
        visible: status === Image.Ready
        layer.enabled: true
        layer.effect: OpacityMask {
            maskSource: Rectangle { width: img.width; height: img.height; radius: width / 2 }
        }
    }
}
