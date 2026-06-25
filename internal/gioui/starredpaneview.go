// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// starredpaneview.go — panel "Pesan berbintang" (lintas chat). Header (← + judul
// + glyph bintang) lalu daftar hit (.hit-row: avatar 40 + nama 15/Medium + teks
// 13.5 + jam 12) — sama gaya SearchView. Buka chat saat baris diketuk. Fungsi
// murni; data demo inline saat ctl nil (standalone render gio-shot).
package gioui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// StarredCtl = state panel berbintang. nil → demo. Hit nyata (GetStarred) +
// clickable per-hit (buka chat) + tombol kembali.
type StarredCtl struct {
	Hits      []svHit
	HitClicks []widget.Clickable
	Back      *widget.Clickable
}

// StarredPaneView menggambar sidebar 380 berisi daftar pesan berbintang.
func StarredPaneView(gtx layout.Context, th *material.Theme, t Theme, ctl *StarredCtl) layout.Dimensions {
	w := gtx.Dp(468)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	hits := []svHit{
		{name: "Andi Pratama", text: "Lokasi rapat: Jl. Merdeka 17, lantai 3", time: "19.08"},
		{name: "Keluarga", text: "Ibu: nomor rekening tabungan haji 1234567890", time: "18.41"},
		{name: "Tim Proyek X", text: "Budi: deadline rilis fitur diundur ke Jumat", time: "16.20"},
	}
	if ctl != nil {
		hits = ctl.Hits
	}

	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return stHeader(gtx, th, t, ctl)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return svListLabel(gtx, th, t, "PESAN BERBINTANG")
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(hits) == 0 {
				return layout.Inset{Top: unit.Dp(40)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(th, 14, "Belum ada pesan berbintang")
						l.Color = t.Text2
						return l.Layout(gtx)
					})
				})
			}
			children := make([]layout.FlexChild, 0, len(hits))
			for i := range hits {
				hh, idx := hits[i], i
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					row := func(gtx layout.Context) layout.Dimensions { return svHitRow(gtx, th, t, hh) }
					if ctl != nil && idx < len(ctl.HitClicks) {
						return ctl.HitClicks[idx].Layout(gtx, row)
					}
					return row(gtx)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)
	return layout.Dimensions{Size: sz}
}

// stHeader — header panel: tombol kembali (←) + judul "Pesan berbintang" 17/Medium
// + glyph bintang accent kanan.
func stHeader(gtx layout.Context, th *material.Theme, t Theme, ctl *StarredCtl) layout.Dimensions {
	var back *widget.Clickable
	if ctl != nil {
		back = ctl.Back
	}
	return subPaneHead(gtx, th, t, gtx.Constraints.Max.X, "Pesan berbintang", back, func(gtx layout.Context) layout.Dimensions {
		return icon(gtx, "star", 20, t.Accent)
	})
}
