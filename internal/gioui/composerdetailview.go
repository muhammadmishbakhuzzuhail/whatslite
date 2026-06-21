// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// composerdetailview.go — bar composer di dasar layar (paritas Composer.svelte +
// app.css .composer): bar head-bg min-height 64, divider 1px atas, padding 9/16,
// gap 10. Berisi tombol-ikon emoji (😊 40px), tombol attach "+" (40px), pil input
// flex-1 (search-bg radius 22, "Ketik pesan" text2 15, pad kiri 16), dan tombol
// mic (40px, bentuk mic text2). Fungsi murni, data demo inline (standalone render).
package gioui

import (
	"image"

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
				return cdvEmojiBtn(gtx, th)
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

// cdvEmojiBtn — .icon-btn 40px lingkaran transparan, glyph 😊 ukuran 22 di tengah.
func cdvEmojiBtn(gtx layout.Context, th *material.Theme) layout.Dimensions {
	d := gtx.Dp(40)
	sz := image.Pt(d, d)
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, 22, "😊")
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}

// cdvPlusBtn — .icon-btn 40px, ikon "+" (dua palang text2) di tengah lingkaran transparan.
func cdvPlusBtn(gtx layout.Context, t Theme) layout.Dimensions {
	d := gtx.Dp(40)
	sz := image.Pt(d, d)
	// palang horizontal + vertikal 18px, tebal 2px, warna text2 (paritas svg +).
	arm := gtx.Dp(18)
	th2 := gtx.Dp(2)
	cx, cy := d/2, d/2
	// horizontal
	paint.FillShape(gtx.Ops, t.Text2, clip.Rect{Min: image.Pt(cx-arm/2, cy-th2/2), Max: image.Pt(cx+arm/2, cy-th2/2+th2)}.Op())
	// vertikal
	paint.FillShape(gtx.Ops, t.Text2, clip.Rect{Min: image.Pt(cx-th2/2, cy-arm/2), Max: image.Pt(cx-th2/2+th2, cy+arm/2)}.Op())
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

// cdvMicBtn — .icon-btn 40px, bentuk mic text2: kapsul rounded + busur dudukan + tiang.
func cdvMicBtn(gtx layout.Context, t Theme) layout.Dimensions {
	d := gtx.Dp(40)
	sz := image.Pt(d, d)
	cx := d / 2

	// kapsul mic (rect x=9 w=6 h=11 rx=3 dalam 24 viewBox) → skala ke 40.
	capW := gtx.Dp(7)
	capH := gtx.Dp(13)
	capTop := gtx.Dp(9)
	rr := capW / 2
	cap := image.Rectangle{Min: image.Pt(cx-capW/2, capTop), Max: image.Pt(cx+capW/2, capTop+capH)}
	paint.FillShape(gtx.Ops, t.Text2, clip.RRect{Rect: cap, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))

	// dudukan: busur U (path M5 11a7 7 0 0 0 14 0) — dekati dgn cincin bawah tipis.
	bw := gtx.Dp(2)
	cradleW := gtx.Dp(18)
	cradleTop := capTop + capH - gtx.Dp(2)
	cradleH := gtx.Dp(7)
	outer := image.Rectangle{Min: image.Pt(cx-cradleW/2, cradleTop), Max: image.Pt(cx+cradleW/2, cradleTop+cradleH*2)}
	r2 := cradleW / 2
	paint.FillShape(gtx.Ops, t.Text2, clip.RRect{Rect: outer, NW: r2, NE: r2, SE: r2, SW: r2}.Op(gtx.Ops))
	inner := image.Rectangle{Min: image.Pt(outer.Min.X + bw, outer.Min.Y - bw), Max: image.Pt(outer.Max.X - bw, outer.Max.Y - bw)}
	ri := r2 - bw
	paint.FillShape(gtx.Ops, t.HeadBg, clip.RRect{Rect: inner, NW: ri, NE: ri, SE: ri, SW: ri}.Op(gtx.Ops))
	// tutup separuh atas busur agar terlihat seperti U (gambar HeadBg di atas).
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Rect{Min: image.Pt(outer.Min.X, outer.Min.Y), Max: image.Pt(outer.Max.X, cradleTop + cradleH)}.Op())
	// gambar ulang kapsul agar tetap di atas tutup.
	paint.FillShape(gtx.Ops, t.Text2, clip.RRect{Rect: cap, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))

	// tiang bawah (M12 18v3): garis vertikal pendek di tengah.
	standTop := cradleTop + cradleH
	standH := gtx.Dp(4)
	paint.FillShape(gtx.Ops, t.Text2, clip.Rect{Min: image.Pt(cx-bw/2, standTop), Max: image.Pt(cx-bw/2+bw, standTop+standH)}.Op())

	return layout.Dimensions{Size: sz}
}
