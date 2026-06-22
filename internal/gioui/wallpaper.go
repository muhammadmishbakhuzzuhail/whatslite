// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// wallpaper.go — latar percakapan ala WhatsApp: warna dasar + pola "doodle" tipis.
// Doodle = motif objek kecil line-art (hati, kamera, kado, balon, cangkir, not,
// payung, awan, kue, pesawat, dst) — bukan ikon UI — tersebar rapat dgn rotasi
// beragam, alpha rendah. Tile diraster sekali per warna (cache) → blit murni.
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

const doodleTileSize = 360

// doodlePaths — motif doodle (inner SVG, viewBox 0 0 24 24, stroke). Objek
// playful ala WhatsApp, bukan ikon aplikasi.
var doodlePaths = map[string]string{
	"heart":    `<path d="M12 20C12 20 4 15.5 4 9.8 4 6.9 6.3 5.2 8.7 6.2 10.2 6.8 11.3 8 12 9 12.7 8 13.8 6.8 15.3 6.2 17.7 5.2 20 6.9 20 9.8 20 15.5 12 20 12 20Z"/>`,
	"camera":   `<rect x="3" y="7" width="18" height="12" rx="2"/><circle cx="12" cy="13" r="3.2"/><path d="M8.5 7l1.3-2h4.4l1.3 2"/>`,
	"gift":     `<rect x="4.5" y="9" width="15" height="10.5" rx="1"/><path d="M4.5 13h15M12 9v10.5"/><path d="M12 9C12 9 9.5 9.2 8.4 7.6 7.7 6.4 9 5.2 10.1 6 11 6.7 12 9 12 9zM12 9C12 9 14.5 9.2 15.6 7.6 16.3 6.4 15 5.2 13.9 6 13 6.7 12 9 12 9z"/>`,
	"balloon":  `<path d="M12 3.2C9 3.2 7.2 5.6 7.2 8.2 7.2 11 9.6 13 12 13 14.4 13 16.8 11 16.8 8.2 16.8 5.6 15 3.2 12 3.2z"/><path d="M12 13l-.6 1.6h1.2zM11.7 14.6c-.6 1.4 1.2 2.4.3 3.8"/>`,
	"mug":      `<path d="M5 8h11v7.5a3 3 0 0 1-3 3H8a3 3 0 0 1-3-3z"/><path d="M16 10h2.4a2 2 0 0 1 0 4H16"/><path d="M7.5 4.5c-.6.8.6 1.4 0 2.2M10.5 4.5c-.6.8.6 1.4 0 2.2"/>`,
	"music":    `<path d="M9 17.5V6.2l8-1.8v9.2"/><ellipse cx="6.7" cy="17.6" rx="2.3" ry="2"/><ellipse cx="15.3" cy="15.8" rx="2.3" ry="2"/>`,
	"sun":      `<circle cx="12" cy="12" r="4"/><path d="M12 3v2.5M12 18.5V21M3 12h2.5M18.5 12H21M5.6 5.6l1.8 1.8M16.6 16.6l1.8 1.8M18.4 5.6l-1.8 1.8M7.4 16.6l-1.8 1.8"/>`,
	"cloud":    `<path d="M7.5 18h9a3.7 3.7 0 0 0 .3-7.4 5.2 5.2 0 0 0-10-1A3.6 3.6 0 0 0 7.5 18z"/>`,
	"umbrella": `<path d="M12 3.5C7 3.5 3 7.5 3 12.5h18C21 7.5 17 3.5 12 3.5z"/><path d="M12 12.5v6a2 2 0 0 0 3.8.8"/>`,
	"cake":     `<path d="M4.5 13.5h15v6h-15z"/><path d="M4.5 13.5c1.6-2 3.4-2 5 0 1.6-2 3.4-2 5 0 1.6-2 3.4-2 5 0"/><path d="M12 13.5V8.5M12 6.5l1.2-2-1.2-1.2-1.2 1.2z"/>`,
	"plane":    `<path d="M3 11.5 21 4l-7 17-3.2-6.8z"/><path d="M10.8 14.2 21 4"/>`,
	"smiley":   `<circle cx="12" cy="12" r="8.5"/><path d="M8.5 14.5s1.3 2 3.5 2 3.5-2 3.5-2"/><circle cx="9" cy="10" r=".6"/><circle cx="15" cy="10" r=".6"/>`,
	"leaf":     `<path d="M5 19C5 11.5 11 5 19 5 19 13 12.5 19 5 19z"/><path d="M6 18C9.5 14 13.5 10 17 7"/>`,
	"bell":     `<path d="M6.5 16.5V11a5.5 5.5 0 0 1 11 0v5.5l1.8 2H4.7z"/><path d="M10 19.5a2 2 0 0 0 4 0"/>`,
	"star":     `<path d="M12 4l2.3 4.9 5.2.7-3.8 3.7 1 5.3L12 16.7 7.3 18.6l1-5.3L4.5 9.6l5.2-.7z"/>`,
	"anchor":   `<circle cx="12" cy="5" r="2"/><path d="M12 7v12M6 12c0 4 3 6 6 6s6-2 6-6M8.5 11.5h7"/>`,
}

var (
	doodleMu    sync.Mutex
	doodleCache = map[color.NRGBA]paint.ImageOp{}
)

// rasterPath — raster inner SVG (path doodle) ke RGBA size×size warna col, dirotasi
// rot derajat di sekitar pusat (12,12). nil bila gagal.
func rasterPath(inner string, size int, col color.NRGBA, rot int) *image.RGBA {
	if size <= 0 {
		return nil
	}
	hex := fmt.Sprintf("#%02x%02x%02x", col.R, col.G, col.B)
	body := inner
	if rot != 0 {
		body = fmt.Sprintf(`<g transform="rotate(%d 12 12)">%s</g>`, rot, inner)
	}
	svg := `<svg viewBox="0 0 24 24" fill="none" stroke="` + hex +
		`" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`
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

// doodleTile — tile pola doodle (transparan + motif tersebar @alpha rendah), cache
// per-warna, di-tile menutup wallpaper.
func doodleTile(col color.NRGBA) paint.ImageOp {
	doodleMu.Lock()
	defer doodleMu.Unlock()
	if op, ok := doodleCache[col]; ok {
		return op
	}
	tile := image.NewRGBA(image.Rect(0, 0, doodleTileSize, doodleTileSize))
	type place struct {
		name    string
		x, y, s int
		rot     int
	}
	// tata rapat & beragam (objek + rotasi); hindari celah besar, sambung saat tile.
	places := []place{
		{"heart", 6, 8, 26, -12}, {"camera", 92, 4, 28, 8}, {"music", 180, 10, 26, -6},
		{"balloon", 256, 6, 24, 10}, {"sun", 322, 14, 26, 0},
		{"gift", 40, 70, 26, -8}, {"cloud", 124, 64, 28, 0}, {"mug", 210, 70, 26, 6},
		{"star", 296, 76, 22, 14}, {"plane", 6, 130, 26, -18},
		{"smiley", 86, 128, 26, 0}, {"umbrella", 168, 124, 26, 8}, {"cake", 250, 130, 26, -6},
		{"leaf", 326, 132, 26, 20}, {"bell", 44, 196, 24, -10},
		{"anchor", 126, 200, 26, 6}, {"heart", 210, 198, 22, 16}, {"camera", 290, 196, 26, -8},
		{"music", 10, 262, 26, 12}, {"gift", 92, 268, 26, 0}, {"balloon", 176, 264, 24, -10},
		{"cloud", 252, 270, 28, 6}, {"sun", 330, 268, 24, 0},
		{"star", 50, 326, 22, -14}, {"mug", 132, 322, 26, 8}, {"smiley", 214, 326, 24, 0},
		{"plane", 300, 324, 26, 14},
	}
	const alpha = 0.05
	for _, pl := range places {
		inner, ok := doodlePaths[pl.name]
		if !ok {
			continue
		}
		sub := rasterPath(inner, pl.s, col, pl.rot)
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
