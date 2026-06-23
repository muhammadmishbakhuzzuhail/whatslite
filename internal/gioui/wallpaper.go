// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// wallpaper.go — latar percakapan PERSIS WhatsApp: pakai aset doodle PNG yang SAMA
// dgn frontend Svelte (doodle-dark/-light.png, 540×960), di-tile 412px + wash warna
// wallpaper semi-transparan di atasnya (resep app.css: background-image linear-
// gradient(wash) + doodle, size cover,412px; dark wash .84 / light .5). Aset di-decode
// sekali (cache). Ini menggantikan motif gambar-tangan yg tak pernah sama persis.
package gioui

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/png"
	"sync"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

//go:embed doodle-dark.png
var doodleDarkPNG []byte

//go:embed doodle-light.png
var doodleLightPNG []byte

var (
	doodleOnce  sync.Once
	doodleDark  paint.ImageOp
	doodleLight paint.ImageOp
	doodleOK    bool
)

func loadDoodles() {
	dec := func(b []byte) (paint.ImageOp, bool) {
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			return paint.ImageOp{}, false
		}
		return paint.NewImageOp(img), true
	}
	d, okd := dec(doodleDarkPNG)
	l, okl := dec(doodleLightPNG)
	doodleDark, doodleLight = d, l
	doodleOK = okd && okl
}

// isDarkColor — true bila warna gelap (luminance rendah) → pilih doodle gelap + wash gelap.
func isDarkColor(c color.NRGBA) bool {
	return (int(c.R)*299+int(c.G)*587+int(c.B)*114)/1000 < 128
}

// drawWallpaper — warna wallpaper + doodle PNG WhatsApp di-tile (lebar 412dp) + wash
// warna wallpaper semi-transparan (dark .84 / light .5) → doodle samar persis WA.
func drawWallpaper(gtx layout.Context, t Theme) {
	paint.FillShape(gtx.Ops, t.Wallpaper, clip.Rect{Max: gtx.Constraints.Max}.Op())
	doodleOnce.Do(loadDoodles)

	dark := isDarkColor(t.Wallpaper)
	dop := doodleLight
	if dark {
		dop = doodleDark
	}
	if doodleOK {
		src := dop.Size()
		if src.X > 0 && src.Y > 0 {
			tileW := gtx.Dp(412) // background-size: …, 412px auto
			s := float32(tileW) / float32(src.X)
			tileH := int(float32(src.Y) * s)
			if tileH < 1 {
				tileH = 1
			}
			area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
			for ty := 0; ty < gtx.Constraints.Max.Y; ty += tileH {
				for tx := 0; tx < gtx.Constraints.Max.X; tx += tileW {
					o := op.Offset(image.Pt(tx, ty)).Push(gtx.Ops)
					aff := op.Affine(f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(s, s))).Push(gtx.Ops)
					dop.Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)
					aff.Pop()
					o.Pop()
				}
			}
			area.Pop()
		}
	}

	// wash: warna wallpaper di atas doodle → kurangi kontras. HANYA mode gelap:
	// doodle-light.png sudah sangat tipis (alpha ≤30 garis gelap) — wash apa pun
	// membuatnya tak terlihat di beige. Mode terang → tanpa wash.
	if dark {
		wash := t.Wallpaper
		wash.A = 214 // ≈ .84
		paint.FillShape(gtx.Ops, wash, clip.Rect{Max: gtx.Constraints.Max}.Op())
	}
}
