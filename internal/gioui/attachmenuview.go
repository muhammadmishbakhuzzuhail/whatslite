// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// attachmenuview.go — menu lampiran (tombol "+" composer): Foto & Video, Dokumen,
// Kontak, Lokasi, Polling. Tiap baris ikon-bulat accent + label (paritas
// AttachMenu WhatsApp). Fungsi murni; ctl nil → render statis (gio-shot).
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

// attachItems — baris menu lampiran (ikon, label, kategori utk OnAttach).
var attachItems = []struct{ icon, label, category string }{
	{"wallpaperico", "Foto & Video", "media"},
	{"docfile", "Dokumen", "document"},
	{"contacts", "Kontak", "contact"},
	{"locpin", "Lokasi", "location"},
	{"pollq", "Polling", "poll"},
}

// AttachCtl = state interaktif menu lampiran. nil → statis. Clicks: 1 per baris.
type AttachCtl struct {
	Clicks []widget.Clickable
}

// AttachCategory mengembalikan kategori baris ke-i (utk handler UI).
func AttachCategory(i int) string {
	if i < 0 || i >= len(attachItems) {
		return ""
	}
	return attachItems[i].category
}

// AttachCount = jumlah baris menu lampiran.
func AttachCount() int { return len(attachItems) }

// AttachMenuView — kartu menu lampiran terpusat di atas backdrop redup.
func AttachMenuView(gtx layout.Context, th *material.Theme, t Theme, ctl *AttachCtl) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 90}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return amCard(gtx, th, t, ctl)
	})
}

func amCard(gtx layout.Context, th *material.Theme, t Theme, ctl *AttachCtl) layout.Dimensions {
	w := gtx.Dp(260)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	macro := op.Record(gtx.Ops)
	children := make([]layout.FlexChild, 0, len(attachItems))
	for i := range attachItems {
		idx := i
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			row := func(gtx layout.Context) layout.Dimensions { return amRow(gtx, th, t, idx) }
			if ctl != nil && idx < len(ctl.Clicks) {
				return ctl.Clicks[idx].Layout(gtx, row)
			}
			return row(gtx)
		}))
	}
	dims := layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
	call := macro.Stop()
	r := gtx.Dp(14)
	paint.FillShape(gtx.Ops, t.Bg2, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// amRow — satu baris: lingkaran accent ber-ikon putih + label.
func amRow(gtx layout.Context, th *material.Theme, t Theme, i int) layout.Dimensions {
	it := attachItems[i]
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	return layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(18), Right: unit.Dp(18)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				d := gtx.Dp(40)
				sz := image.Pt(d, d)
				paint.FillShape(gtx.Ops, t.Accent, clip.Ellipse{Max: sz}.Op(gtx.Ops))
				gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
				layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return icon(gtx, it.icon, 20, white)
				})
				return layout.Dimensions{Size: sz}
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 15, it.label)
				lbl.Color = t.Text
				return lbl.Layout(gtx)
			}),
		)
	})
}
