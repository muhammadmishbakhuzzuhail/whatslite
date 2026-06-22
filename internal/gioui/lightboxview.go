// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// lightboxview.go — lightbox gambar layar penuh (paritas frontend/src/lib/chat/
// Lightbox.svelte + app.css): backdrop rgba(0,0,0,.92), foto terpusat (aproksimasi
// gradien sunset oranye→ungu via dua isi bertumpuk) maks 94vw/90vh radius 6, dua
// tombol bulat 38 rgba(255,255,255,.12) kanan-atas (unduh right 70 + ✕ right 22,
// top 18), keterangan teks putih bawah-tengah (tanpa pil). Fungsi murni, data demo
// inline (standalone render).
package gioui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// LbCtl = state lightbox nyata. nil → foto demo (gradien) + caption demo.
type LbCtl struct {
	Img     paint.ImageOp
	Has     bool
	Caption string
	Close   *widget.Clickable
	Save    *widget.Clickable
}

// LightboxView menggambar backdrop redup penuh lalu foto terpusat, tombol unduh/
// tutup kanan-atas, dan keterangan teks bawah-tengah.
func LightboxView(gtx layout.Context, th *material.Theme, t Theme, ctl *LbCtl) layout.Dimensions {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}

	// .lb — backdrop rgba(0,0,0,.92) penuh. Pemanggil (overlay app) sudah mengisi
	// backdrop case-level; isi lagi di sini agar render mandiri (gio-shot) tetap redup.
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 235}, clip.Rect{Max: gtx.Constraints.Max}.Op())

	// foto terpusat (.lb-media maks 94vw/90vh).
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return lbPhoto(gtx, ctl)
	})

	// dua tombol bulat 38 kanan-atas: unduh (right 70) + ✕ (right 22), top 18.
	lbTopButtons(gtx, th, white, ctl)

	// keterangan bawah-tengah (.lb-cap) — teks putih saja, tanpa latar pil.
	cap := "Sunset di pantai 🌅"
	if ctl != nil {
		cap = ctl.Caption
	}
	if cap != "" {
		lbCaption(gtx, th, white, cap)
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}
}

// lbPhoto — foto nyata (ctl.Img, aspek-fit maks 94%/90%) bila ada; jika tidak,
// aproksimasi gradien sunset (oranye atas → ungu bawah) + pita tengah.
func lbPhoto(gtx layout.Context, ctl *LbCtl) layout.Dimensions {
	maxW := gtx.Constraints.Max.X * 94 / 100
	maxH := gtx.Constraints.Max.Y * 90 / 100
	r := gtx.Dp(6)

	// foto nyata: aspek-fit ke maks 94vw/90vh, klip RRect radius 6.
	if ctl != nil && ctl.Has {
		is := ctl.Img.Size()
		w, h := is.X, is.Y
		if w <= 0 || h <= 0 {
			w, h = maxW, maxH
		}
		if w > maxW {
			h = h * maxW / w
			w = maxW
		}
		if h > maxH {
			w = w * maxH / h
			h = maxH
		}
		sz := image.Pt(w, h)
		defer clip.RRect{Rect: image.Rectangle{Max: sz}, NW: r, NE: r, SE: r, SW: r}.Push(gtx.Ops).Pop()
		drawImageRect(gtx.Ops, ctl.Img, sz)
		return layout.Dimensions{Size: sz}
	}

	// demo: gradien sunset ~600x400 dibatasi 94%/90%.
	w := gtx.Dp(600)
	h := gtx.Dp(400)
	if w > maxW {
		w = maxW
	}
	if h > maxH {
		h = maxH
	}
	sz := image.Pt(w, h)

	macro := op.Record(gtx.Ops)
	// lapisan dasar: oranye (langit/matahari).
	orange := color.NRGBA{R: 0xf2, G: 0x8b, B: 0x3c, A: 0xff}
	purple := color.NRGBA{R: 0x4b, G: 0x2c, B: 0x6e, A: 0xff}
	paint.FillShape(gtx.Ops, orange, clip.Rect{Max: sz}.Op())
	// paruh bawah ungu (laut/senja) — isi kedua bertumpuk utk kesan gradien.
	half := h / 2
	paint.FillShape(gtx.Ops, purple, clip.Rect{Min: image.Pt(0, half), Max: sz}.Op())
	// pita transisi tipis di tengah utk melembutkan batas oranye→ungu.
	band := color.NRGBA{R: 0xc8, G: 0x5a, B: 0x55, A: 0xff}
	bandH := gtx.Dp(48)
	paint.FillShape(gtx.Ops, band, clip.Rect{Min: image.Pt(0, half-bandH/2), Max: image.Pt(w, half+bandH/2)}.Op())
	// "matahari": lingkaran kuning lembut di sepertiga atas.
	sd := gtx.Dp(90)
	sx := (w - sd) / 2
	sy := h/3 - sd/2
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0xff, G: 0xd4, B: 0x6e, A: 0xff}, clip.Ellipse{Min: image.Pt(sx, sy), Max: image.Pt(sx+sd, sy+sd)}.Op(gtx.Ops))
	call := macro.Stop()

	// klip seluruh foto ke RRect radius 6.
	defer clip.RRect{Rect: image.Rectangle{Max: sz}, NW: r, NE: r, SE: r, SW: r}.Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return layout.Dimensions{Size: sz}
}

// lbTopButtons — dua tombol bulat 38 (rgba(255,255,255,.12)) di kanan-atas:
// glyph unduh lalu ✕ tutup; jarak top 18, ✕ right 22, unduh right 70.
func lbTopButtons(gtx layout.Context, th *material.Theme, white color.NRGBA, ctl *LbCtl) {
	d := gtx.Dp(38)
	top := gtx.Dp(18)
	xRight := gtx.Dp(22)
	saveRight := gtx.Dp(70)

	// posisi dihitung dari lebar penuh DULU (jgn pakai gtx.Constraints setelah
	// area-klik mengubahnya — itu yg sempat menyembunyikan tombol unduh).
	xX := gtx.Constraints.Max.X - xRight - d   // ✕ tutup — .lb-x right 22.
	dlX := gtx.Constraints.Max.X - saveRight - d // unduh — .lb-save right 70.

	lbCircleAt(gtx, xX, top, d)
	lbGlyphX(gtx, th, xX, top, d, white)
	lbCircleAt(gtx, dlX, top, d)
	lbDownloadGlyph(gtx, dlX, top, d, white)

	// area klik d×d (transparan; visual sudah digambar di atas). gtx lokal agar
	// mutasi Constraints tak bocor ke perhitungan lain.
	hit := func(c *widget.Clickable, x int) {
		if c == nil {
			return
		}
		cgtx := gtx
		off := op.Offset(image.Pt(x, top)).Push(cgtx.Ops)
		cgtx.Constraints.Min = image.Pt(d, d)
		cgtx.Constraints.Max = image.Pt(d, d)
		c.Layout(cgtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(d, d)}
		})
		off.Pop()
	}
	if ctl != nil {
		hit(ctl.Close, xX)
		hit(ctl.Save, dlX)
	}
}

// lbCircleAt — lingkaran 38 rgba(255,255,255,.12) pada offset (x,y).
func lbCircleAt(gtx layout.Context, x, y, d int) {
	bg := color.NRGBA{R: 255, G: 255, B: 255, A: 31} // .12 ≈ 31/255
	paint.FillShape(gtx.Ops, bg, clip.Ellipse{Min: image.Pt(x, y), Max: image.Pt(x+d, y+d)}.Op(gtx.Ops))
}

// lbGlyphX — label "✕" putih terpusat dalam tombol di (x,y).
func lbGlyphX(gtx layout.Context, th *material.Theme, x, y, d int, white color.NRGBA) {
	off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
	gtx.Constraints.Min = image.Pt(d, d)
	gtx.Constraints.Max = image.Pt(d, d)
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, 18, "✕")
		lbl.Color = white
		return lbl.Layout(gtx)
	})
	off.Pop()
}

// lbDownloadGlyph — glyph unduh (panah ke bawah + alas) putih, garis 2px,
// digambar dari rect tipis (tangkai + dua sisi mata panah + alas) di (x,y).
func lbDownloadGlyph(gtx layout.Context, x, y, d int, white color.NRGBA) {
	// pusat tombol.
	cx := x + d/2
	cy := y + d/2
	w := gtx.Dp(2) // tebal garis
	half := gtx.Dp(7)

	// tangkai vertikal panah.
	paint.FillShape(gtx.Ops, white, clip.Rect{Min: image.Pt(cx-w/2, cy-half), Max: image.Pt(cx+w/2, cy+half-gtx.Dp(1))}.Op())
	// mata panah (V): dua batang miring diaproksimasi sebagai tangga rect kecil.
	steps := gtx.Dp(6)
	for i := 0; i < steps; i++ {
		yy := cy + half - gtx.Dp(1) - i
		paint.FillShape(gtx.Ops, white, clip.Rect{Min: image.Pt(cx-i-w/2, yy), Max: image.Pt(cx-i+w/2, yy+w)}.Op())
		paint.FillShape(gtx.Ops, white, clip.Rect{Min: image.Pt(cx+i-w/2, yy), Max: image.Pt(cx+i+w/2, yy+w)}.Op())
	}
	// alas (garis bawah / "tray").
	base := gtx.Dp(8)
	by := cy + half + gtx.Dp(3)
	paint.FillShape(gtx.Ops, white, clip.Rect{Min: image.Pt(cx-base, by), Max: image.Pt(cx+base, by+w)}.Op())
}

// lbCaption — keterangan bawah-tengah (.lb-cap): teks putih saja "Sunset di pantai 🌅",
// font-size 14, tanpa latar pil (text-only), text-shadow 0 1px 4px rgba(0,0,0,.7),
// terpusat horizontal, jarak bottom 26 dari bawah, padding 0 24 kiri/kanan.
func lbCaption(gtx layout.Context, th *material.Theme, white color.NRGBA, cap string) {
	bottom := gtx.Dp(26)
	shadow := color.NRGBA{R: 0, G: 0, B: 0, A: 179} // text-shadow .7 ≈ 179/255

	// render teks (padding 0 24 kiri/kanan) ke dalam makro utk mengukur lalu posisi.
	macro := op.Record(gtx.Ops)
	gtx.Constraints.Min.X = 0
	dims := layout.Inset{Left: unit.Dp(24), Right: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, 14, cap)
		lbl.Color = white
		lbl.MaxLines = 1
		return lbl.Layout(gtx)
	})
	call := macro.Stop()

	// posisi: terpusat horizontal, jarak `bottom` dari bawah. Teks saja, tanpa latar.
	px := (gtx.Constraints.Max.X - dims.Size.X) / 2
	py := gtx.Constraints.Max.Y - bottom - dims.Size.Y

	// text-shadow 0 1px 4px rgba(0,0,0,.7): bayangan tipis 1px ke bawah.
	soff := op.Offset(image.Pt(px, py+gtx.Dp(1))).Push(gtx.Ops)
	layout.Inset{Left: unit.Dp(24), Right: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, 14, cap)
		lbl.Color = shadow
		lbl.MaxLines = 1
		return lbl.Layout(gtx)
	})
	soff.Pop()

	// teks putih di atas bayangan.
	off := op.Offset(image.Pt(px, py)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
}
