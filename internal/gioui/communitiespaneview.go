// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// communitiespaneview.go — sidebar pane KOMUNITAS: .pane-head + daftar kartu
// komunitas (ikon communities + nama + "N grup" + ringkas sub-grup). Fungsi murni,
// data demo inline (standalone render). nil → demo.
package gioui

import (
	"image"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// comItem = satu komunitas di daftar (nama + sub-baris jumlah/nama grup).
type comItem struct {
	name string
	sub  string // "N grup · Grup A, Grup B, …"
}

// CommunitiesPaneView — sidebar 380px (t.SidebarBg) berisi pane KOMUNITAS.
func CommunitiesPaneView(gtx layout.Context, th *material.Theme, t Theme, items []comItem) layout.Dimensions {
	w := gtx.Dp(380)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	if items == nil { // data demo (render standalone / gio-shot)
		items = []comItem{
			{name: "Tim Kantor", sub: "3 grup · Umum, Proyek X, Acara"},
			{name: "Keluarga Besar", sub: "2 grup · Umum, Liburan"},
		}
	}

	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return comPaneHead(gtx, th, t, w, "Komunitas")
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(items) == 0 {
				return layout.Inset{Top: unit.Dp(40)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(th, 14, "Belum ada komunitas")
						l.Color = t.Text2
						return l.Layout(gtx)
					})
				})
			}
			children := make([]layout.FlexChild, 0, len(items))
			for i := range items {
				it := items[i]
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return comRow(gtx, th, t, it)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)
}

func comPaneHead(gtx layout.Context, th *material.Theme, t Theme, w int, title string) layout.Dimensions {
	h := gtx.Dp(56)
	sz := image.Pt(w, h)
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Rect{Max: sz}.Op())
	bh := gtx.Dp(1)
	paint.FillShape(gtx.Ops, t.Divider, clip.Rect{Min: image.Pt(0, h-bh), Max: image.Pt(w, h)}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 19, title)
			lbl.Color = t.Text
			lbl.Font.Weight = font.SemiBold
			return lbl.Layout(gtx)
		})
	})
	return layout.Dimensions{Size: sz}
}

// comRow — kartu komunitas: ikon communities (kotak membulat) + nama + sub.
func comRow(gtx layout.Context, th *material.Theme, t Theme, it comItem) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				d := gtx.Dp(46)
				bsz := image.Pt(d, d)
				r := gtx.Dp(12)
				paint.FillShape(gtx.Ops, t.Bg2, clip.RRect{Rect: image.Rectangle{Max: bsz}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
				gtx.Constraints.Min, gtx.Constraints.Max = bsz, bsz
				layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return icon(gtx, "communities", 24, t.Accent)
				})
				return layout.Dimensions{Size: bsz}
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(13)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(th, 16, it.name)
						l.Color, l.MaxLines, l.Font.Weight = t.Text, 1, font.Medium
						return l.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(th, 13, it.sub)
						l.Color, l.MaxLines = t.Text2, 1
						return l.Layout(gtx)
					}),
				)
			}),
		)
	})
}
