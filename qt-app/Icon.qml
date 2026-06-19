// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// Icon — render ikon garis WhatsApp dari potongan SVG (path/circle/rect) milik
// komponen Svelte, diwarnai sesuai tema. Pakai qt6-svg via data-URI. Sama persis
// gaya app.css: fill none, stroke currentColor, stroke-width 1.8, linecap round.
import QtQuick

Image {
    id: ic
    property string svg: ""        // isi <svg> (mis. '<path d="…"/><circle …/>')
    property color color: "#000000"
    property int box: 24
    property string vbox: "0 0 " + box + " " + box  // viewBox custom (mis. ticks "0 0 18 14")
    property string fill: "none"   // "currentColor" untuk ikon solid (avatar grup)
    sourceSize.width: width
    sourceSize.height: height
    fillMode: Image.PreserveAspectFit
    source: "data:image/svg+xml;utf8," + encodeURIComponent(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="' + vbox +
        '" fill="' + (fill === "currentColor" ? color : fill) + '" stroke="' + (fill === "none" ? color : "none") +
        '" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">' +
        svg + '</svg>')
}
