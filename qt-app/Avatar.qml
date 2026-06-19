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
    property real fontSize: 18

    implicitWidth: 44
    implicitHeight: 44
    radius: width / 2
    color: accent

    Text {
        anchors.centerIn: parent
        visible: img.status !== Image.Ready
        color: "white"; font.pixelSize: root.fontSize; font.bold: true
        text: (root.name || "?").charAt(0).toUpperCase()
    }
    Image {
        id: img
        anchors.fill: parent
        fillMode: Image.PreserveAspectCrop
        source: (root.base && root.jid) ? (root.base + "/avatar/" + root.jid) : ""
        visible: status === Image.Ready
        layer.enabled: true
        layer.effect: OpacityMask {
            maskSource: Rectangle { width: img.width; height: img.height; radius: width / 2 }
        }
    }
}
