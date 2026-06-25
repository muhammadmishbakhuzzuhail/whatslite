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
	"image/draw"
	"image/png"
	"sync"

	xdraw "golang.org/x/image/draw"

	"gioui.org/layout"
	"gioui.org/op/paint"
)

//go:embed doodle-dark.png
var doodleDarkPNG []byte

//go:embed doodle-light.png
var doodleLightPNG []byte

var (
	doodleOnce     sync.Once
	doodleDarkImg  image.Image
	doodleLightImg image.Image
	doodleOK       bool

	// cache wallpaper PRE-RENDER: bg+tile doodle+wash di-komposit SEKALI ke satu
	// image → di-redraw 1 op/frame (bukan tiling puluhan op tiap frame). Regenerasi
	// hanya saat tema/ukuran berubah. ImageOp boleh dipakai lintas-frame.
	wpOp          paint.ImageOp
	wpHas, wpDark bool
	wpW, wpH      int
	wpWall        color.NRGBA
)

func loadDoodles() {
	dec := func(b []byte) image.Image {
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			return nil
		}
		return img
	}
	doodleDarkImg, doodleLightImg = dec(doodleDarkPNG), dec(doodleLightPNG)
	doodleOK = doodleDarkImg != nil && doodleLightImg != nil
}

// isDarkColor — true bila warna gelap (luminance rendah) → pilih doodle gelap + wash gelap.
func isDarkColor(c color.NRGBA) bool {
	return (int(c.R)*299+int(c.G)*587+int(c.B)*114)/1000 < 128
}

// drawWallpaper — warna wallpaper + doodle PNG WhatsApp di-tile (lebar 412dp) + wash
// warna wallpaper semi-transparan (dark .84 / light .5) → doodle samar persis WA.
func drawWallpaper(gtx layout.Context, t Theme) {
	doodleOnce.Do(loadDoodles)
	w, h := gtx.Constraints.Max.X, gtx.Constraints.Max.Y
	if w < 1 || h < 1 {
		return
	}
	dark := isDarkColor(t.Wallpaper)
	if !wpHas || wpDark != dark || wpW != w || wpH != h || wpWall != t.Wallpaper {
		wpOp = buildWallpaper(t, dark, w, h, gtx.Dp(412))
		wpHas, wpDark, wpW, wpH, wpWall = true, dark, w, h, t.Wallpaper
	}
	wpOp.Add(gtx.Ops) // 1 op/frame (bukan tiling)
	paint.PaintOp{}.Add(gtx.Ops)
}

// buildWallpaper — komposit CPU SEKALI: warna + tile doodle (lebar tileW) + wash
// gelap → satu image. Dipanggil hanya saat tema/ukuran berubah.
func buildWallpaper(t Theme, dark bool, w, h, tileW int) paint.ImageOp {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{t.Wallpaper}, image.Point{}, draw.Src) // warna dasar
	src := doodleLightImg
	if dark {
		src = doodleDarkImg
	}
	if doodleOK && src != nil && tileW > 0 {
		sb := src.Bounds()
		s := float64(tileW) / float64(sb.Dx())
		tileH := int(float64(sb.Dy()) * s)
		if tileH < 1 {
			tileH = 1
		}
		tile := image.NewRGBA(image.Rect(0, 0, tileW, tileH))
		xdraw.ApproxBiLinear.Scale(tile, tile.Bounds(), src, sb, xdraw.Over, nil)
		for ty := 0; ty < h; ty += tileH {
			for tx := 0; tx < w; tx += tileW {
				draw.Draw(dst, image.Rect(tx, ty, tx+tileW, ty+tileH), tile, image.Point{}, draw.Over)
			}
		}
	}
	if dark { // wash gelap di atas doodle (mode terang tak perlu)
		wash := t.Wallpaper
		wash.A = 214 // ≈ .84
		draw.Draw(dst, dst.Bounds(), &image.Uniform{wash}, image.Point{}, draw.Over)
	}
	return paint.NewImageOp(dst)
}
