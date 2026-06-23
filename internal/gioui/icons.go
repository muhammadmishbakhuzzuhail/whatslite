// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// icons.go — ikon garis WhatsApp (SVG path, sama dgn Svelte/Qt) di-raster via
// oksvg/rasterx → paint.ImageOp ber-tint. Pure-Go. Ganti glyph emoji yg dipakai
// sementara → ikon native ala WhatsApp. Cache per (nama,ukuran,warna).
package gioui

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"

	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// iconPaths — isi <svg> ikon (viewBox 0 0 24 24), disalin dari frontend Svelte /
// Qt win.ico. Stroke-only (fill none) kecuali yang disebut di iconSolid.
var iconPaths = map[string]string{
	"chats":     `<path d="M12 3C6.5 3 2 6.8 2 11.5c0 2.3 1.1 4.4 2.9 5.9-.1 1.2-.6 2.6-1.4 3.6 1.6-.2 3.2-.8 4.4-1.6 1.2.4 2.6.6 4.1.6 5.5 0 10-3.8 10-8.5S17.5 3 12 3z"/>`,
	"status":    `<circle cx="12" cy="12" r="9" stroke-dasharray="3 3"/>`,
	"channels":  `<path d="M4 9.5v5h3.5L13 18V6L7.5 9.5H4z"/><path d="M16.5 8.5a4.5 4.5 0 0 1 0 7"/><path d="M19 6a8 8 0 0 1 0 12"/>`,
	"calls":     `<path d="M5 4h3l2 5-2.5 1.5a11 11 0 0 0 5 5L15 13l5 2v3a2 2 0 0 1-2 2A16 16 0 0 1 3 6a2 2 0 0 1 2-2z"/>`,
	"contacts":  `<circle cx="12" cy="8" r="4"/><path d="M4 20c0-4 3.6-6.5 8-6.5s8 2.5 8 6.5"/>`,
	"settings":  `<circle cx="12" cy="12" r="3"/><path d="M19.4 13.5a7.5 7.5 0 0 0 0-3l1.9-1.5-1.9-3.3-2.3 1a7.5 7.5 0 0 0-2.6-1.5L14.1 2H9.9l-.4 2.7a7.5 7.5 0 0 0-2.6 1.5l-2.3-1L2.7 8.5l1.9 1.5a7.5 7.5 0 0 0 0 3l-1.9 1.5 1.9 3.3 2.3-1a7.5 7.5 0 0 0 2.6 1.5l.4 2.7h4.2l.4-2.7a7.5 7.5 0 0 0 2.6-1.5l2.3 1 1.9-3.3-1.9-1.5z"/>`,
	"emoji":     `<circle cx="12" cy="12" r="9"/><circle cx="9" cy="10" r="1"/><circle cx="15" cy="10" r="1"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/>`,
	"plus":      `<path d="M12 5v14M5 12h14"/>`,
	"mic":       `<rect x="9" y="2.5" width="6" height="10.5" rx="3"/><path d="M5.5 11a6.5 6.5 0 0 0 13 0"/><path d="M12 17.5V21"/><path d="M8.5 21h7"/>`,
	"send":      `<path d="M3 11l18-8-8 18-2-7-8-3z"/>`,
	"pin":       `<path d="M12 17v5M7 4h10l-1 6 3 3H5l3-3-1-6z"/>`,
	"mute":      `<path d="M5 9v6h3l4 4V5L8 9H5z"/><path d="M16 8a5 5 0 0 1 0 8"/><path d="M3 3l18 18"/>`,
	"search":    `<circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/>`,
	"overflow":  `<circle cx="12" cy="5" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="12" cy="19" r="1.6"/>`,
	"newchat":   `<path d="M12 5H7a3 3 0 0 0-3 3v9a3 3 0 0 0 3 3h9a3 3 0 0 0 3-3v-5"/><path d="M18 3l3 3-9 9H9v-3z"/>`,
	"reply":     `<path d="M10 17l-5-5 5-5M5 12h10a4 4 0 0 1 4 4v3"/>`,
	"forward":   `<path d="M14 7l5 5-5 5M19 12H9a4 4 0 0 0-4 4v3"/>`,
	"star":      `<path d="M12 3l2.6 5.5 6 .8-4.4 4.2 1.1 6L12 16.8 6.7 19.5l1.1-6L3.4 9.3l6-.8z"/>`,
	"info":      `<circle cx="12" cy="12" r="9"/><path d="M12 16v-5M12 8h.01"/>`,
	"trash":     `<path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M6 6l1 14a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-14"/>`,
	"check":     `<path d="M3 7.5l3.5 3.5L14 4"/>`,
	"checks":    `<path d="M1 7.5l3.5 3.5L12 4"/><path d="M7 11l3.5 3.5L18 7"/>`,
	"back":      `<path d="M15 5l-7 7 7 7"/>`,
	"close":     `<path d="M6 6l12 12M18 6L6 18"/>`,
	// --- ikon komponen (settings/info/bubble/picker) ---
	"theme":        `<path d="M21 13A9 9 0 1 1 11 3a7 7 0 0 0 10 10z"/>`,
	"globe":        `<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.5 2.5 2.5 15 0 18M12 3C9.5 5.5 9.5 18.5 12 21"/>`,
	"globe2":       `<path d="M4 12h16M12 4a15 15 0 0 1 0 16M12 4a15 15 0 0 0 0 16"/><circle cx="12" cy="12" r="9"/>`,
	"disk":         `<rect x="4" y="4" width="16" height="16" rx="2"/><path d="M8 4v6h8V4M8 16h.01"/>`,
	"lock":         `<rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/>`,
	"clock":        `<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 3"/>`,
	"window":       `<rect x="3" y="4" width="18" height="14" rx="2"/><path d="M8 21h8M12 18v3"/>`,
	"bell":         `<path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.7 21a2 2 0 0 1-3.4 0"/>`,
	"editpen":      `<path d="M4 20h4L18 10l-4-4L4 16z"/><path d="M14 6l4 4"/>`,
	"addmember":    `<circle cx="9" cy="8" r="4"/><path d="M2 20c0-3.5 3-6 7-6M18 11v6M15 14h6"/>`,
	"invitelink":   `<path d="M9 15l6-6M8 13l-2 2a3 3 0 0 0 4 4l2-2M16 11l2-2a3 3 0 0 0-4-4l-2 2"/>`,
	"resetlink":    `<path d="M4 12a8 8 0 0 1 14-5l2 2M20 12a8 8 0 0 1-14 5l-2-2M18 4v5h-5M6 20v-5h5"/>`,
	"clearchat":    `<path d="M10 3h4l1 4h5v3H4V7h5z"/><path d="M6 10v9a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2v-9"/>`,
	"block":        `<circle cx="12" cy="12" r="9"/><path d="M5.5 5.5l13 13"/>`,
	"message":      `<path d="M4 5h16v11H8l-4 4z"/>`,
	"leavegroup":   `<path d="M15 4h3a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-3"/><path d="M10 17l-5-5 5-5M5 12h11"/>`,
	"docfile":      `<path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5"/>`,
	"play":         `<path d="M8 5v14l11-7z"/>`,
	"locpin":       `<path d="M12 21s7-6 7-11a7 7 0 0 0-14 0c0 5 7 11 7 11z"/><circle cx="12" cy="10" r="2.5"/>`,
	"download":     `<path d="M12 4v11M7 11l5 5 5-5M5 20h14"/>`,
	"chevrondown":  `<path d="M6 9l6 6 6-6"/>`,
	"chevronup":    `<path d="M6 15l6-6 6 6"/>`,
	"camera":       `<path d="M3 8a2 2 0 0 1 2-2h2l1.5-2h7L19 6h0a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><circle cx="12" cy="12.5" r="3.5"/>`,
	"rotate":       `<path d="M21 12a9 9 0 1 1-2.6-6.4"/><path d="M21 3v5h-5"/>`,
	"sticker":      `<path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h8l6-6V5a2 2 0 0 0-2-2z"/><path d="M14 21v-4a2 2 0 0 1 2-2h4"/>`,
	"gifb":         `<rect x="3" y="5" width="18" height="14" rx="2"/><path d="M8 9v6M11 9v6h2M16 9h-2v6M16 12h-1"/>`,
	"pollq":        `<path d="M5 5h14M5 12h9M5 19h5"/>`,
	"chevleft":     `<path d="M15 5l-7 7 7 7"/>`,
	"chevdown":     `<path d="M6 9l6 6 6-6"/>`,
	"speaker":      `<path d="M11 5L6 9H2v6h4l5 4zM15 9a3 3 0 0 1 0 6M18 6a7 7 0 0 1 0 12"/>`,
	"hamburger":    `<path d="M4 6h16M4 12h16M4 18h16"/>`,
	"eyeoff":       `<circle cx="12" cy="12" r="9"/><path d="M5.6 5.6l12.8 12.8"/>`,
	"moon":         `<path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/>`,
	"emojiface":    `<circle cx="12" cy="12" r="9"/><path d="M8 14s1.5 2 4 2 4-2 4-2M9 9h.01M15 9h.01"/>`,
	"zoom":         `<path d="M7 8V5h3M17 8V5h-3M7 16v3h3M17 16v3h-3"/>`,
	"power":        `<path d="M18.36 6.64a9 9 0 1 1-12.73 0M12 2v10"/>`,
	"verif":        `<path d="M5 12l4 4 10-10"/>`,
	"callArrowOut": `<path d="M7 17L17 7M17 7H9M17 7v8"/>`,
	"wallpaperico": `<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 15l5-4 4 3 5-5 4 4"/>`,
	"archive":      `<rect x="3" y="4" width="18" height="5" rx="1"/><path d="M5 9v10a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V9M9 13h6"/>`,
	"communities":  `<circle cx="9" cy="8.5" r="3"/><path d="M3.5 18.5c0-3 2.4-4.5 5.5-4.5s5.5 1.5 5.5 4.5"/><circle cx="16.5" cy="9.5" r="2.3"/><path d="M16 14.2c2.6.1 4.5 1.5 4.5 4.3"/>`,
}

type iconKey struct {
	name string
	size int
	col  color.NRGBA
}

var (
	iconMu    sync.Mutex
	iconCache = map[iconKey]paint.ImageOp{}
)

// iconOp: raster ikon → ImageOp ber-warna `col`, ukuran piksel `size`. Cache.
func iconOp(name string, size int, col color.NRGBA) (paint.ImageOp, bool) {
	p, ok := iconPaths[name]
	if !ok || size <= 0 {
		return paint.ImageOp{}, false
	}
	key := iconKey{name, size, col}
	iconMu.Lock()
	defer iconMu.Unlock()
	if op, ok := iconCache[key]; ok {
		return op, true
	}
	hex := fmt.Sprintf("#%02x%02x%02x", col.R, col.G, col.B)
	svg := `<svg viewBox="0 0 24 24" fill="none" stroke="` + hex +
		`" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">` + p + `</svg>`
	icon, err := oksvg.ReadIconStream(strings.NewReader(svg))
	if err != nil {
		return paint.ImageOp{}, false
	}
	icon.SetTarget(0, 0, float64(size), float64(size))
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, rgba, rgba.Bounds())
	raster := rasterx.NewDasher(size, size, scanner)
	icon.Draw(raster, 1.0)
	op := paint.NewImageOp(rgba)
	iconCache[key] = op
	return op, true
}

// icon: gambar ikon WhatsApp dp×dp warna col (di tengah gtx).
func icon(gtx layout.Context, name string, dp int, col color.NRGBA) layout.Dimensions {
	d := gtx.Dp(unit.Dp(dp))
	op, ok := iconOp(name, d, col)
	if !ok {
		return layout.Dimensions{Size: image.Pt(d, d)}
	}
	op.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: image.Pt(d, d)}
}
