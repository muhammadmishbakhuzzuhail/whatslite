// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// fonts.go — shaper teks dgn fallback emoji warna. gofont (UI) + NotoColorEmoji
// (emoji) → bubble/preview tak lagi tofu (□). go-text Gio render glyph bitmap
// warna sejak 2023.
package gioui

import (
	"os"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/text"
	"golang.org/x/image/font/gofont/gomono"
)

// emojiPaths = lokasi umum NotoColorEmoji per-distro (fallback bila tak ada).
var emojiPaths = []string{
	"/usr/share/fonts/noto/NotoColorEmoji.ttf",
	"/usr/share/fonts/truetype/noto/NotoColorEmoji.ttf",
	"/usr/share/fonts/google-noto-emoji/NotoColorEmoji.ttf",
	"/usr/share/fonts/NotoColorEmoji.ttf",
}

// NewShaper: gofont + emoji warna (bila font emoji ada di sistem).
func NewShaper() *text.Shaper {
	col := gofont.Collection()
	// Go Mono → typeface "Go Mono" (utk teks ```monospace``` ala WhatsApp).
	if face, err := opentype.Parse(gomono.TTF); err == nil {
		col = append(col, font.FontFace{Font: font.Font{Typeface: "Go Mono"}, Face: face})
	}
	for _, p := range emojiPaths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		faces, err := opentype.ParseCollection(b)
		if err != nil {
			continue
		}
		col = append(col, faces...)
		break
	}
	return text.NewShaper(text.WithCollection(col))
}

var _ = font.FontFace{}
