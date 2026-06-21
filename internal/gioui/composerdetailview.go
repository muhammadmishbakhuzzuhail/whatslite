// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// composerdetailview.go — bar composer di dasar layar (paritas Composer.svelte +
// app.css .composer): bar head-bg min-height 64, divider 1px atas, padding 9/16,
// gap 10. Berisi tombol-ikon emoji (.icon-btn 40px, svg wajah 22 text2), tombol
// attach "+" (40px, svg plus 22 text2), pil input flex-1 (search-bg radius 22,
// "Ketik pesan" text2 15, pad 9/16), dan tombol mic (40px, svg mic 22 text2).
// Fungsi murni, data demo inline (standalone render).
package gioui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// ComposerDetailView menggambar wallpaper penuh lalu bar composer di dasar gtx.
func ComposerDetailView(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	// latar wallpaper penuh; bar composer ditempel di dasar via Flexed pengisi.
	paint.FillShape(gtx.Ops, t.Wallpaper, clip.Rect{Max: gtx.Constraints.Max}.Op())

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)}
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return cdvBar(gtx, th, t)
		}),
	)
}

// cdvBar — .composer: min-height 64, head-bg, divider 1px atas, padding 9/16, gap 10.
func cdvBar(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	w := gtx.Constraints.Max.X

	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = w - gtx.Dp(32)
		// min-height 64: isi minimal = 64 - 18 (pad atas+bawah 9/9) = 46.
		if gtx.Constraints.Min.Y < gtx.Dp(46) {
			gtx.Constraints.Min.Y = gtx.Dp(46)
		}
		gap := layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout)
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return cdvEmojiBtn(gtx, t)
			}),
			gap,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return cdvPlusBtn(gtx, t)
			}),
			gap,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return cdvInput(gtx, th, t)
			}),
			gap,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return cdvMicBtn(gtx, t)
			}),
		)
	})
	call := macro.Stop()

	// latar head-bg di belakang isi + divider 1px di tepi atas.
	bar := image.Rectangle{Max: image.Pt(w, dims.Size.Y)}
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Rect{Max: bar.Max}.Op())
	paint.FillShape(gtx.Ops, t.Divider, clip.Rect{Max: image.Pt(w, gtx.Dp(1))}.Op())
	call.Add(gtx.Ops)
	return layout.Dimensions{Size: image.Pt(w, dims.Size.Y)}
}

// cdvSvgScale: peta satuan viewBox 24 → svg 22px (app.css `svg { width:22px }`).
func cdvSvgScale(gtx layout.Context, v float32) int { return gtx.Dp(unit.Dp(v * 22.0 / 24.0)) }

// cdvStroke: lebar stroke svg (app.css `stroke-width: 1.8`) diskala ke 22px, min 1px.
func cdvStroke(gtx layout.Context) int {
	w := cdvSvgScale(gtx, 1.8)
	if w < 1 {
		w = 1
	}
	return w
}

// cdvDisc: lingkaran penuh radius r warna c berpusat di ctr.
func cdvDisc(gtx layout.Context, c color.NRGBA, ctr image.Point, r int) {
	if r < 1 {
		r = 1
	}
	rect := image.Rectangle{Min: image.Pt(ctr.X-r, ctr.Y-r), Max: image.Pt(ctr.X+r, ctr.Y+r)}
	paint.FillShape(gtx.Ops, c, clip.RRect{Rect: rect, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
}

// cdvRing: cincin (outline lingkaran) tebal sw warna c di atas latar bg.
func cdvRing(gtx layout.Context, c, bg color.NRGBA, ctr image.Point, r, sw int) {
	cdvDisc(gtx, c, ctr, r)
	cdvDisc(gtx, bg, ctr, r-sw)
}

// cdvEmojiBtn — .icon-btn 40px transparan, svg wajah text2 22 (viewBox 24): muka
// cincin r=9, dua mata di (9,10)/(15,10), busur mulut M8.5 14.5a4 4 0 0 0 7 0.
func cdvEmojiBtn(gtx layout.Context, t Theme) layout.Dimensions {
	d := gtx.Dp(40)
	sz := image.Pt(d, d)
	off := image.Pt((d-gtx.Dp(22))/2, (d-gtx.Dp(22))/2)
	at := func(x, y float32) image.Point { return off.Add(image.Pt(cdvSvgScale(gtx, x), cdvSvgScale(gtx, y))) }
	sw := cdvStroke(gtx)
	// muka: cincin r=9.
	cdvRing(gtx, t.Text2, t.HeadBg, at(12, 12), cdvSvgScale(gtx, 9), sw)
	// mata: dua titik r≈1.
	cdvDisc(gtx, t.Text2, at(9, 10), cdvSvgScale(gtx, 1))
	cdvDisc(gtx, t.Text2, at(15, 10), cdvSvgScale(gtx, 1))
	// mulut: busur senyum, didekati dgn batang horizontal tebal sw di y≈15.
	mw := cdvSvgScale(gtx, 7)
	m := at(12, 15)
	paint.FillShape(gtx.Ops, t.Text2, clip.RRect{Rect: image.Rectangle{Min: image.Pt(m.X-mw/2, m.Y), Max: image.Pt(m.X+mw/2, m.Y+sw)}, NW: 0, NE: 0, SE: sw / 2, SW: sw / 2}.Op(gtx.Ops))
	return layout.Dimensions{Size: sz}
}

// cdvPlusBtn — .icon-btn 40px, svg plus text2 22 (viewBox 24): M12 5v14 M5 12h14,
// stroke 1.8. Palang penuh 14 unit (5..19) diskala ke 22px.
func cdvPlusBtn(gtx layout.Context, t Theme) layout.Dimensions {
	d := gtx.Dp(40)
	sz := image.Pt(d, d)
	off := image.Pt((d-gtx.Dp(22))/2, (d-gtx.Dp(22))/2)
	at := func(x, y float32) image.Point { return off.Add(image.Pt(cdvSvgScale(gtx, x), cdvSvgScale(gtx, y))) }
	sw := cdvStroke(gtx)
	c := at(12, 12)
	half := cdvSvgScale(gtx, 7) // dari 12 ke 5/19 = 7 unit
	// horizontal (M5 12h14)
	paint.FillShape(gtx.Ops, t.Text2, clip.Rect{Min: image.Pt(c.X-half, c.Y-sw/2), Max: image.Pt(c.X+half, c.Y-sw/2+sw)}.Op())
	// vertikal (M12 5v14)
	paint.FillShape(gtx.Ops, t.Text2, clip.Rect{Min: image.Pt(c.X-sw/2, c.Y-half), Max: image.Pt(c.X-sw/2+sw, c.Y+half)}.Op())
	return layout.Dimensions{Size: sz}
}

// cdvInput — .composer .input: flex-1, search-bg, radius 22, padding 9/16,
// teks placeholder "Ketik pesan" text2 15 rata kiri.
func cdvInput(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		lbl := material.Label(th, 15, "Ketik pesan")
		lbl.Color = t.Text2
		lbl.MaxLines = 1
		return lbl.Layout(gtx)
	})
	call := macro.Stop()
	rr := gtx.Dp(22)
	paint.FillShape(gtx.Ops, t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// cdvMicBtn — .icon-btn 40px, svg mic text2 22 (viewBox 24): rect x9 y3 w6 h11 rx3
// (kapsul), path M5 11a7 7 0 0 0 14 0 (busur dudukan), M12 18v3 (tiang).
func cdvMicBtn(gtx layout.Context, t Theme) layout.Dimensions {
	d := gtx.Dp(40)
	sz := image.Pt(d, d)
	off := image.Pt((d-gtx.Dp(22))/2, (d-gtx.Dp(22))/2)
	at := func(x, y float32) image.Point { return off.Add(image.Pt(cdvSvgScale(gtx, x), cdvSvgScale(gtx, y))) }
	sw := cdvStroke(gtx)
	cx := off.X + cdvSvgScale(gtx, 12)

	// kapsul mic (rect x=9 y=3 w=6 h=11 rx=3) → skala ke 22px.
	capW := cdvSvgScale(gtx, 6)
	capTop := off.Y + cdvSvgScale(gtx, 3)
	capH := cdvSvgScale(gtx, 11)
	rr := capW / 2
	capsule := image.Rectangle{Min: image.Pt(cx-capW/2, capTop), Max: image.Pt(cx+capW/2, capTop+capH)}
	paint.FillShape(gtx.Ops, t.Text2, clip.RRect{Rect: capsule, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))

	// dudukan: busur U (M5 11a7 7 0 0 0 14 0) — cincin bawah tebal sw, separuh atas ditutup.
	cradleW := cdvSvgScale(gtx, 14) // dari x=5 ke x=19 = 14 unit
	cradleTop := off.Y + cdvSvgScale(gtx, 11)
	r2 := cradleW / 2
	cr := image.Pt(cx, cradleTop)
	cdvRing(gtx, t.Text2, t.HeadBg, cr, r2, sw)
	// tutup separuh atas cincin agar terlihat seperti U.
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Rect{Min: image.Pt(cr.X-r2-sw, cr.Y-r2-sw), Max: image.Pt(cr.X+r2+sw, cr.Y)}.Op())
	// gambar ulang kapsul agar tetap di atas tutup.
	paint.FillShape(gtx.Ops, t.Text2, clip.RRect{Rect: capsule, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))

	// tiang bawah (M12 18v3): garis vertikal pendek di tengah.
	stand := at(12, 18)
	standH := cdvSvgScale(gtx, 3) // 18→21 = 3 unit
	paint.FillShape(gtx.Ops, t.Text2, clip.Rect{Min: image.Pt(stand.X-sw/2, stand.Y), Max: image.Pt(stand.X-sw/2+sw, stand.Y+standH)}.Op())

	return layout.Dimensions{Size: sz}
}
