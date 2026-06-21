// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// theme.go — token warna WhatsLite (disalin dari frontend/src/styles/app.css)
// untuk paritas visual. Dark + light.
package main

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
	Selected  color.NRGBA
}

func newTheme(dark bool) Theme {
	if dark {
		return Theme{
			Dark: true, RailBg: hex("#11161d"), RailIco: hex("#8a97a3"),
			SidebarBg: hex("#0e1318"), Bg: hex("#1a232a"), Bg2: hex("#222e35"),
			HeadBg: hex("#11171e"), Line: hex("#2a3942"), Divider: hex("#1c252d"),
			SearchBg: hex("#1b232b"), Wallpaper: hex("#0a0f14"), InBg: hex("#1f2c33"),
			OutBg: hex("#144d37"), Text: hex("#e7ecf0"), Text2: hex("#8a97a3"),
			Hover: hex("#161d24"), Accent: hex("#06c98c"), Tick: hex("#34b7f1"),
			Selected: hex("#12302a"),
		}
	}
	return Theme{
		Dark: false, RailBg: hex("#f4f6fa"), RailIco: hex("#6b7785"),
		SidebarBg: hex("#ffffff"), Bg: hex("#eef1f6"), Bg2: hex("#f0f2f5"),
		HeadBg: hex("#ffffff"), Line: hex("#e4e8ee"), Divider: hex("#eceff3"),
		SearchBg: hex("#eef1f6"), Wallpaper: hex("#eef1f6"), InBg: hex("#ffffff"),
		OutBg: hex("#d9fdd3"), Text: hex("#111b21"), Text2: hex("#667781"),
		Hover: hex("#f2f4f8"), Accent: hex("#06b67f"), Tick: hex("#53bdeb"),
		Selected: hex("#e7f6ef"),
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
