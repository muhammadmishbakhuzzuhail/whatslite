// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// convheaderview.go — bar header percakapan (paritas frontend/src/lib/chat/
// ConvHeader.svelte + app.css .conv-head). Tinggi 60, head-bg, divider bawah 1px,
// padding kiri 18; avatar 40 + nama 16/Medium + status "online" 13; di kanan dua
// tombol ikon bundar 40 (search + overflow) digambar sbg glyph text2. Fungsi
// murni, data demo inline (standalone render).
package gioui

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// ConvHeaderView menggambar bar header percakapan 60px di atas gtx; sisa area
// diisi t.Bg. Self-contained: avatar/nama/status + dua tombol ikon demo.
func ConvHeaderView(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	// latar penuh: t.Bg (sisa di bawah header)
	paint.FillShape(gtx.Ops, t.Bg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	// .conv-head: height 60, head-bg, border-bottom 1px divider.
	h := gtx.Dp(60)
	sz := image.Pt(gtx.Constraints.Max.X, h)
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Rect{Max: sz}.Op())
	paint.FillShape(gtx.Ops, t.Divider, clip.Rect{Min: image.Pt(0, h-1), Max: sz}.Op())

	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	// padding kiri 18, kanan 12 (per kontrak render); align center vertikal.
	layout.Inset{Left: unit.Dp(18), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			// .conv-peer: avatar 40 + gap 13 + meta (flex:1).
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return chvAvatar(gtx, th, "Andi Pratama", 40)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(13)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							// .conv-name: 16/500
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(th, 16, "Andi Pratama")
								lbl.Color = t.Text
								lbl.MaxLines = 1
								lbl.Font.Weight = font.Medium
								return lbl.Layout(gtx)
							}),
							// .conv-status: 13/text2
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(th, 13, "online")
								lbl.Color = t.Text2
								lbl.MaxLines = 1
								return lbl.Layout(gtx)
							}),
						)
					}),
				)
			}),
			// .conv-actions: gap 2, flex-shrink:0 — search + overflow.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return chvIconBtn(gtx, t, chvSearchGlyph)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return chvIconBtn(gtx, t, chvOverflowGlyph)
					}),
				)
			}),
		)
	})
	return layout.Dimensions{Size: sz}
}

// chvAvatar: lingkaran 40 warna avatarColor(name) + inisial putih di tengah.
func chvAvatar(gtx layout.Context, th *material.Theme, name string, dp int) layout.Dimensions {
	d := gtx.Dp(unit.Dp(dp))
	sz := image.Pt(d, d)
	paint.FillShape(gtx.Ops, avatarColor(name), clip.Ellipse{Max: sz}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, unit.Sp(float32(dp)*0.4), initial(name))
		lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		lbl.Font.Weight = font.Bold
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}

// chvIconBtn: .icon-btn — kotak 40x40 (lingkaran transparan), glyph text2 di tengah.
func chvIconBtn(gtx layout.Context, t Theme, glyph func(gtx layout.Context, t Theme, cx, cy int) layout.Context) layout.Dimensions {
	d := gtx.Dp(40)
	sz := image.Pt(d, d)
	// bg transparan (.icon-btn background: transparent) — tidak digambar.
	glyph(gtx, t, d/2, d/2)
	return layout.Dimensions{Size: sz}
}

// chvSearchGlyph: ikon kaca pembesar (.icon-btn svg ~18px) — cincin lingkaran +
// gagang diagonal, warna text2. Disusun dari ellipse + rect (API ui.go).
func chvSearchGlyph(gtx layout.Context, t Theme, cx, cy int) layout.Context {
	// (helper murni efek-samping menggambar; tipe balik diabaikan)
	r := gtx.Dp(7)  // jari-jari cincin (viewBox r=7)
	bw := gtx.Dp(2) // tebal garis (stroke)
	ringCX := cx - gtx.Dp(2)
	ringCY := cy - gtx.Dp(2)
	// cincin: ellipse penuh text2, lalu lubang dgn warna head-bg di dalam.
	outer := image.Rectangle{Min: image.Pt(ringCX-r, ringCY-r), Max: image.Pt(ringCX+r, ringCY+r)}
	paint.FillShape(gtx.Ops, t.Text2, clip.Ellipse{Min: outer.Min, Max: outer.Max}.Op(gtx.Ops))
	inner := image.Rectangle{Min: image.Pt(ringCX-r+bw, ringCY-r+bw), Max: image.Pt(ringCX+r-bw, ringCY+r-bw)}
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Ellipse{Min: inner.Min, Max: inner.Max}.Op(gtx.Ops))
	// gagang diagonal: deret kotak kecil dari tepi cincin ke kanan-bawah.
	hx := ringCX + gtx.Dp(5)
	hy := ringCY + gtx.Dp(5)
	for i := 0; i < gtx.Dp(5); i++ {
		paint.FillShape(gtx.Ops, t.Text2, clip.Rect{Min: image.Pt(hx+i, hy+i), Max: image.Pt(hx+i+bw, hy+i+bw)}.Op())
	}
	return gtx
}

// chvOverflowGlyph: ikon "kebab" tiga titik vertikal (overflow menu), warna text2.
func chvOverflowGlyph(gtx layout.Context, t Theme, cx, cy int) layout.Context {
	dotD := gtx.Dp(3) // jari-jari ~1.6 → diameter ~3px
	gap := gtx.Dp(5)  // jarak antar pusat titik (cy 5/12/19)
	for i := -1; i <= 1; i++ {
		oy := cy + i*gap
		min := image.Pt(cx-dotD, oy-dotD)
		max := image.Pt(cx+dotD, oy+dotD)
		paint.FillShape(gtx.Ops, t.Text2, clip.Ellipse{Min: min, Max: max}.Op(gtx.Ops))
	}
	return gtx
}
