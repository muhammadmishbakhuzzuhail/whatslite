// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// theme.go — token warna WhatsApp Web ASLI (palet resmi web.whatsapp.com).
// Dark + light. (Sebelumnya pakai app.css WhatsLite; user pilih WA-Web persis.)
package gioui

import "image/color"

func hex(s string) color.NRGBA {
	// "#rrggbb"
	var r, g, b uint8
	_, _ = sscanHex(s, &r, &g, &b)
	return color.NRGBA{R: r, G: g, B: b, A: 0xff}
}

func sscanHex(s string, r, g, b *uint8) (int, error) {
	if len(s) == 7 && s[0] == '#' {
		*r = hb(s[1])<<4 | hb(s[2])
		*g = hb(s[3])<<4 | hb(s[4])
		*b = hb(s[5])<<4 | hb(s[6])
		return 3, nil
	}
	return 0, nil
}
func hb(c byte) uint8 {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

// Theme = palet aktif (app.css variabel). Nilai persis dari :root / [data-theme=dark].
type Theme struct {
	Dark      bool
	RailBg    color.NRGBA
	RailIco   color.NRGBA
	SidebarBg color.NRGBA
	Bg        color.NRGBA
	Bg2       color.NRGBA
	HeadBg    color.NRGBA
	Line      color.NRGBA
	Divider   color.NRGBA
	SearchBg  color.NRGBA
	Wallpaper color.NRGBA
	InBg      color.NRGBA
	OutBg     color.NRGBA
	Text      color.NRGBA
	Text2     color.NRGBA
	Hover     color.NRGBA
	Accent    color.NRGBA
	Tick      color.NRGBA
	Link      color.NRGBA // warna tautan (biru) di teks bubble
	Selected  color.NRGBA
}

// DarkTheme/LightTheme: ekspor utk render-tool (gio-shot).
func DarkTheme() Theme  { return newTheme(true) }
func LightTheme() Theme { return newTheme(false) }

func newTheme(dark bool) Theme {
	if dark {
		// WhatsApp Web dark (resmi): panel #111b21, header/bubble-in #202c33,
		// bubble-out #005c4b, wallpaper #0b141a, accent #00a884, border #2a3942.
		return Theme{
			Dark: true, RailBg: hex("#202c33"), RailIco: hex("#aebac1"),
			SidebarBg: hex("#0b141a"), Bg: hex("#0b141a"), Bg2: hex("#202c33"),
			HeadBg: hex("#202c33"), Line: hex("#2a3942"), Divider: hex("#222d34"),
			SearchBg: hex("#202c33"), Wallpaper: hex("#0b141a"), InBg: hex("#202c33"),
			OutBg: hex("#005c4b"), Text: hex("#e9edef"), Text2: hex("#8696a0"),
			Hover: hex("#2a3942"), Accent: hex("#00a884"), Tick: hex("#53bdeb"),
			Link: hex("#53bdeb"), Selected: hex("#2a3942"),
		}
	}
	// Light MODERN (palet netral ala Tailwind slate + aksen WhatsApp green):
	// sidebar putih, area chat slate-50, border slate-200, teks slate-900/500,
	// bubble-in putih + bubble-out green-100, link biru-600. Lebih bersih &
	// kontemporer daripada beige WA-Web lama.
	return Theme{
		Dark: false, RailBg: hex("#f1f5f9"), RailIco: hex("#475569"),
		SidebarBg: hex("#ffffff"), Bg: hex("#f8fafc"), Bg2: hex("#f1f5f9"),
		HeadBg: hex("#ffffff"), Line: hex("#e2e8f0"), Divider: hex("#eef2f6"),
		SearchBg: hex("#f1f5f9"), Wallpaper: hex("#f8fafc"), InBg: hex("#ffffff"),
		OutBg: hex("#dcfce7"), Text: hex("#0f172a"), Text2: hex("#64748b"),
		Hover: hex("#f1f5f9"), Accent: hex("#00a884"), Tick: hex("#10b981"),
		Link: hex("#2563eb"), Selected: hex("#eef2f6"),
	}
}

// avatarColor: warna avatar deterministik dari nama (paritas win.avatarColor).
func avatarColor(name string) color.NRGBA {
	pal := []string{"#e5614e", "#5b9e3d", "#5b6ef5", "#9b59b6", "#e9418a",
		"#f2a33c", "#06b67f", "#3d8bd3", "#d9534f", "#16a085"}
	var h uint32 = 2166136261
	for i := 0; i < len(name); i++ {
		h ^= uint32(name[i])
		h *= 16777619
	}
	return hex(pal[int(h)%len(pal)])
}
