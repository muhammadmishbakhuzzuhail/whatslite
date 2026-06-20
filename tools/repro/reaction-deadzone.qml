import QtQuick
import QtQuick.Window
import QtQuick.Layouts
import QtQuick.Controls


ApplicationWindow {
    id: win
    width: 1100; height: 740; visible: true
    color: "#0a0f14"

    Item { id: root; anchors.fill: parent
    RowLayout {
        anchors.fill: parent; spacing: 0
        Rectangle { Layout.preferredWidth: 64; Layout.fillHeight: true; color: "#11161d" }   // rail
        Rectangle { Layout.preferredWidth: 400; Layout.fillHeight: true; color: "#0e1318" }  // sidebar
        // conversation
        ColumnLayout {
            Layout.fillWidth: true; Layout.fillHeight: true; spacing: 0
            Rectangle { Layout.fillWidth: true; Layout.preferredHeight: 60; color: "#11171e" } // conv header
            Rectangle {
                Layout.fillWidth: true; Layout.fillHeight: true; color: "#0a0f14"
                Image { anchors.fill: parent; fillMode: Image.Tile; opacity: 0.5
                    source: "file:///home/zuhail/Documents/Workspace/whatslite/qt-app/assets/doodle-dark.png" }
                Rectangle { anchors.fill: parent; color: "#0a0f14"; opacity: 0.84 }
                ListView {
                    id: lv
                    anchors.fill: parent; anchors.margins: 12; clip: true
                    model: ListModel {
                        ListElement { dir: "in" } ListElement { dir: "out" } ListElement { dir: "in" }
                        ListElement { dir: "out" } ListElement { dir: "in" } ListElement { dir: "out" }
                    }
                    reuseItems: true; spacing: 6
                    delegate: Item {
                        id: msgItem
                        width: ListView.view.width
                        property bool out: model.dir === "out"
                        implicitHeight: bubble.implicitHeight + 4
                        Rectangle {
                            id: bubble
                            x: out ? parent.width - width - 4 : 4
                            width: content.implicitWidth + 26
                            implicitHeight: content.implicitHeight + 16
                            radius: 18; topLeftRadius: out ? 18 : 6; topRightRadius: out ? 6 : 18
                            color: out ? "#114b39" : "#1d262e"
                            ColumnLayout { id: content
                                anchors.left: parent.left; anchors.top: parent.top
                                anchors.leftMargin: 13; anchors.rightMargin: 13; anchors.topMargin: 8; anchors.bottomMargin: 8
                                Text { text: (msgItem.out ? "OUT " : "IN ") + index + " some message text"; color: "white" } }
                            MouseArea { anchors.fill: parent }
                        }
                        Rectangle {
                            width: 54; height: 20; radius: 10; color: "red"
                            y: bubble.y + bubble.height + 2
                            x: out ? (bubble.x + bubble.width - width - 8) : (bubble.x + 8)
                            Text { anchors.centerIn: parent; text: "RX"; color: "white"; font.pixelSize: 11 }
                        }
                    }
                }
            }
            Rectangle { Layout.fillWidth: true; Layout.preferredHeight: 56; color: "#222e35" } // composer
        }
    }
    }
    Timer { running: true; interval: 1800; repeat: false
        onTriggered: root.grabToImage(function(res) { res.saveToFile("/tmp/repro9.png"); Qt.callLater(Qt.quit) }) }
}
