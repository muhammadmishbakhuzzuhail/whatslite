// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// PoC timeline: ListView ter-virtualisasi + delegate recycling (reuseItems).
// Validasi otomatis: scroll seluruh 10k baris, ukur JUMLAH delegate hidup
// maksimum. Bila recycling jalan → tetap kecil (sebatas terlihat + cacheBuffer),
// BUKAN 10k. Itu bukti titik-lemah "list panjang tinggi-variabel" teratasi.
import QtQuick
import QtQuick.Controls

ApplicationWindow {
    id: win
    width: 420; height: 720; visible: true
    title: "WhatsLite QML PoC"

    property int liveDelegates: 0
    property int maxLiveDelegates: 0

    ListView {
        id: list
        anchors.fill: parent
        model: messageModel
        reuseItems: true          // recycle delegate (Qt 5.15+)
        cacheBuffer: 800
        spacing: 4

        delegate: Item {
            width: ListView.view.width
            implicitHeight: bubble.implicitHeight + 6
            // Hitung delegate yang BENAR-BENAR ada (instansiasi vs hancur/recycle).
            Component.onCompleted: { win.liveDelegates++; if (win.liveDelegates > win.maxLiveDelegates) win.maxLiveDelegates = win.liveDelegates }
            Component.onDestruction: { win.liveDelegates-- }

            Rectangle {
                id: bubble
                x: mout ? parent.width - width - 8 : 8
                width: Math.min(parent.width * 0.78, txt.implicitWidth + 20)
                implicitHeight: txt.implicitHeight + 12
                radius: 8
                color: mout ? "#dcf8c6" : "#ffffff"
                border.color: "#e0e0e0"
                Text {
                    id: txt
                    anchors.centerIn: parent
                    width: Math.min(parent.parent.width * 0.78 - 20, implicitWidth)
                    wrapMode: Text.WordWrap
                    text: mtext
                }
            }
        }
    }

    // Sapu seluruh list (lompat 137 baris/tick) → paksa layout + recycling.
    Timer {
        interval: 1; running: true; repeat: true
        property int i: 0
        onTriggered: {
            list.positionViewAtIndex(i, ListView.Beginning)
            i += 137
            if (i >= list.count) {
                console.log("RESULT rows=" + list.count
                    + " contentHeight=" + Math.round(list.contentHeight)
                    + " maxLiveDelegates=" + win.maxLiveDelegates)
                Qt.quit()
            }
        }
    }
}
