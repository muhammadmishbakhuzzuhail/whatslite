// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// wallpaper.go — latar percakapan ala WhatsApp: warna dasar + pola "doodle" tipis
// (ikon-ikon WhatsApp tersebar, alpha sangat rendah) yang di-tile menutup area.
// Tile diraster sekali per warna (cache) → blit murni saat menggambar.
package gioui

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

const doodleTileSize = 380

var (
	doodleMu    sync.Mutex
	doodleCache = map[color.NRGBA]paint.ImageOp{}
)

// rasterIcon — raster satu glyph SVG ke RGBA size×size warna col (tanpa cache;
// dipakai menyusun tile doodle). nil bila nama tak dikenal.
func rasterIcon(name string, size int, col color.NRGBA) *image.RGBA {
	p, ok := iconPaths[name]
	if !ok || size <= 0 {
		return nil
	}
	hex := fmt.Sprintf("#%02x%02x%02x", col.R, col.G, col.B)
	svg := `<svg viewBox="0 0 24 24" fill="none" stroke="` + hex +
		`" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">` + p + `</svg>`
	ic, err := oksvg.ReadIconStream(strings.NewReader(svg))
	if err != nil {
		return nil
	}
	ic.SetTarget(0, 0, float64(size), float64(size))
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, rgba, rgba.Bounds())
	ic.Draw(rasterx.NewDasher(size, size, scanner), 1.0)
	return rgba
}

// doodleTile — tile pola doodle (transparan + glyph tersebar @alpha rendah), cache
// per-warna. di-tile menutup wallpaper.
func doodleTile(col color.NRGBA) paint.ImageOp {
	doodleMu.Lock()
	defer doodleMu.Unlock()
	if op, ok := doodleCache[col]; ok {
		return op
	}
	tile := image.NewRGBA(image.Rect(0, 0, doodleTileSize, doodleTileSize))
	// motif tersebar (nama ikon, x, y, ukuran) — ditata agar terasa acak namun
	// menyambung mulus saat di-tile (jangan menyentuh tepi berlebihan).
	type place struct {
		name    string
		x, y, s int
	}
	// motif kecil & rapat (mirip doodle WhatsApp) — hindari dashed-circle yg aneh
	// saat di-tile; ukuran 20-26, tersebar agar tak ada celah besar.
	places := []place{
		{"chats", 10, 16, 24}, {"locpin", 96, 40, 22}, {"mic", 176, 18, 20},
		{"emoji", 246, 44, 24}, {"sticker", 322, 20, 22}, {"send", 60, 100, 22},
		{"pollq", 150, 110, 24}, {"calls", 232, 120, 20}, {"docfile", 318, 110, 22},
		{"star", 24, 188, 20}, {"contacts", 110, 198, 24}, {"emoji", 210, 196, 20},
		{"locpin", 300, 200, 22}, {"chats", 40, 286, 22}, {"mic", 130, 300, 20},
		{"pollq", 224, 292, 24}, {"sticker", 312, 300, 22}, {"send", 178, 350, 20},
		{"contacts", 350, 348, 22}, {"calls", 14, 348, 20},
	}
	const alpha = 0.024
	for _, pl := range places {
		sub := rasterIcon(pl.name, pl.s, col)
		if sub == nil {
			continue
		}
		for yy := 0; yy < pl.s; yy++ {
			for xx := 0; xx < pl.s; xx++ {
				c := sub.RGBAAt(xx, yy)
				if c.A == 0 {
					continue
				}
				dx, dy := pl.x+xx, pl.y+yy
				if dx < 0 || dy < 0 || dx >= doodleTileSize || dy >= doodleTileSize {
					continue
				}
				c.A = uint8(float64(c.A) * alpha)
				tile.SetRGBA(dx, dy, c)
			}
		}
	}
	imgOp := paint.NewImageOp(tile)
	doodleCache[col] = imgOp
	return imgOp
}

// drawWallpaper — isi area dengan warna wallpaper lalu tile pola doodle di atasnya.
func drawWallpaper(gtx layout.Context, t Theme) {
	paint.FillShape(gtx.Ops, t.Wallpaper, clip.Rect{Max: gtx.Constraints.Max}.Op())
	tileOp := doodleTile(t.Text2)
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	for ty := 0; ty < gtx.Constraints.Max.Y; ty += doodleTileSize {
		for tx := 0; tx < gtx.Constraints.Max.X; tx += doodleTileSize {
			off := op.Offset(image.Pt(tx, ty)).Push(gtx.Ops)
			tileOp.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			off.Pop()
		}
	}
}
