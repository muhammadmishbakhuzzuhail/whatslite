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
	"channels":  `<path d="M4 9v6h4l5 4V5L8 9H4z"/><path d="M16 8a5 5 0 0 1 0 8"/>`,
	"calls":     `<path d="M5 4h3l2 5-2.5 1.5a11 11 0 0 0 5 5L15 13l5 2v3a2 2 0 0 1-2 2A16 16 0 0 1 3 6a2 2 0 0 1 2-2z"/>`,
	"contacts":  `<circle cx="12" cy="8" r="4"/><path d="M4 20c0-4 3.6-6.5 8-6.5s8 2.5 8 6.5"/>`,
	"settings":  `<circle cx="12" cy="12" r="3"/><path d="M12 2v3M12 19v3M2 12h3M19 12h3M5 5l2 2M17 17l2 2M19 5l-2 2M7 17l-2 2"/>`,
	"emoji":     `<circle cx="12" cy="12" r="9"/><circle cx="9" cy="10" r="1"/><circle cx="15" cy="10" r="1"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/>`,
	"plus":      `<path d="M12 5v14M5 12h14"/>`,
	"mic":       `<rect x="9" y="3" width="6" height="11" rx="3"/><path d="M5 11a7 7 0 0 0 14 0M12 18v3"/>`,
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
