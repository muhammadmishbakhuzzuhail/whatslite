// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// statuspaneview.go — sidebar pane STATUS (paritas frontend/src/lib/sidebar/
// StatusPane.svelte + app.css): .pane-head 56px ("Status" 19/SemiBold); baris
// "My status" (avatar 48 + badge "+" accent kanan-bawah, nama 15/SemiBold +
// hint text2); label .ct-letter "TERKINI" (accent 12); lalu 3 baris status
// terkini (avatar dgn cincin accent utk belum dilihat + nama + waktu text2).
// Fungsi murni, data demo inline (standalone render).
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

// stpItem = satu baris status terkini (.status-row di .ct-letter section).
type stpItem struct {
	name string
	time string
	seen bool // true → cincin abu (var--line), false → cincin accent (belum dilihat)
}

// StatusPaneView menggambar sidebar 380px (t.SidebarBg) berisi pane STATUS:
// header .pane-head + baris "My status" + label "TERKINI" + 3 baris status.
// Fungsi murni, mandiri (standalone render).
func StatusPaneView(gtx layout.Context, th *material.Theme, t Theme, items []stpItem) layout.Dimensions {
	w := gtx.Dp(380)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	if items == nil { // data demo (render standalone / gio-shot)
		items = []stpItem{
			{name: "Andi Pratama", time: "2 menit lalu", seen: false},
			{name: "Sarah Wijaya", time: "15 menit lalu", seen: false},
			{name: "Tim Proyek X", time: "1 jam lalu", seen: true},
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// .pane-head 56px — "Status" 19/SemiBold.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return stpPaneHead(gtx, th, t, w, "Status")
		}),
		// Baris "My status" (avatar + badge "+").
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return stpMyStatusRow(gtx, th, t)
		}),
		// .ct-letter label "TERKINI" (accent 12/Bold).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return stpSectionLabel(gtx, th, t, "TERKINI")
		}),
		// daftar baris status terkini.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, len(items))
			for i := range items {
				it := items[i]
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return stpStatusRow(gtx, th, t, it)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)
}

// stpPaneHead — .pane-head { height: 56px; padding: 0 16px; background: head-bg }
// h2 19/SemiBold.
func stpPaneHead(gtx layout.Context, th *material.Theme, t Theme, w int, title string) layout.Dimensions {
	h := gtx.Dp(56)
	sz := image.Pt(w, h)
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Rect{Max: sz}.Op())
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

// stpMyStatusRow — .status-row { padding: 10px 14px; gap: 14px } : avatar 48 dgn
// badge "+" accent kanan-bawah, lalu kolom (nama 15/SemiBold "Status saya" +
// hint 12.5 text2 "Ketuk untuk menambahkan").
func stpMyStatusRow(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return stpMyAvatar(gtx, th, t)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, 15, "Status saya")
						lbl.Color = t.Text
						lbl.MaxLines = 1
						lbl.Font.Weight = font.SemiBold
						return lbl.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(12.5), "Ketuk untuk menambahkan")
						lbl.Color = t.Text2
						lbl.MaxLines = 1
						return lbl.Layout(gtx)
					}),
				)
			}),
		)
	})
}

// stpMyAvatar — avatar 48 (.status-av) + badge "+" 18px accent di kanan-bawah
// (.status-add: right:-2 bottom:-2, border 2px var--bg). Inisial "?".
func stpMyAvatar(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	d := gtx.Dp(48)
	badge := gtx.Dp(18)
	bd := gtx.Dp(2)    // border badge (2px bg)
	off := gtx.Dp(2)   // right/bottom: -2px → tonjol keluar 2px
	sz := image.Pt(d+off, d+off)

	// lingkaran avatar (avatarColor utk "me") + inisial "?" putih.
	paint.FillShape(gtx.Ops, avatarColor("me"), clip.Ellipse{Max: image.Pt(d, d)}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = image.Pt(d, d), image.Pt(d, d)
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, unit.Sp(18), initial("?"))
		lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		lbl.Font.Weight = font.SemiBold
		return lbl.Layout(gtx)
	})

	// badge "+": lingkaran bg (border) lalu lingkaran accent di dalam, "+" putih.
	bx := d + off - badge // pojok kanan-bawah, tonjol 2px
	by := d + off - badge
	paint.FillShape(gtx.Ops, t.Bg, clip.Ellipse{Min: image.Pt(bx, by), Max: image.Pt(bx+badge, by+badge)}.Op(gtx.Ops))
	paint.FillShape(gtx.Ops, t.Accent, clip.Ellipse{Min: image.Pt(bx+bd, by+bd), Max: image.Pt(bx+badge-bd, by+badge-bd)}.Op(gtx.Ops))
	// glyph "+": dua batang putih di tengah badge.
	cx := bx + badge/2
	cy := by + badge/2
	arm := gtx.Dp(5)
	sw := gtx.Dp(2)
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	paint.FillShape(gtx.Ops, white, clip.Rect{Min: image.Pt(cx-arm/2, cy-sw/2), Max: image.Pt(cx-arm/2+arm, cy-sw/2+sw)}.Op()) // horizontal
	paint.FillShape(gtx.Ops, white, clip.Rect{Min: image.Pt(cx-sw/2, cy-arm/2), Max: image.Pt(cx-sw/2+sw, cy-arm/2+arm)}.Op()) // vertical

	return layout.Dimensions{Size: sz}
}

// stpSectionLabel — .ct-letter { padding: 5px 16px; color: accent; font 12/Bold }.
func stpSectionLabel(gtx layout.Context, th *material.Theme, t Theme, txt string) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, 12, txt)
		lbl.Color = t.Accent
		lbl.MaxLines = 1
		lbl.Font.Weight = font.Bold
		return lbl.Layout(gtx)
	})
}

// stpStatusRow — .status-row { padding: 10px 14px; gap: 14px } : avatar 48 dgn
// cincin (.ring accent utk belum dilihat, var--line utk sudah) + kolom nama
// 15/SemiBold + waktu 12.5 text2.
func stpStatusRow(gtx layout.Context, th *material.Theme, t Theme, it stpItem) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return stpRingAvatar(gtx, th, t, it)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, 15, it.name)
						lbl.Color = t.Text
						lbl.MaxLines = 1
						lbl.Font.Weight = font.SemiBold
						return lbl.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(12.5), it.time)
						lbl.Color = t.Text2
						lbl.MaxLines = 1
						return lbl.Layout(gtx)
					}),
				)
			}),
		)
	})
}

// stpRingAvatar — .ring { padding: 2.5px } : cincin (accent jika belum dilihat,
// var--line jika sudah) langsung mengelilingi avatar 48 (.status-av), tanpa
// celah dalam — padding 2.5px = ketebalan cincin.
func stpRingAvatar(gtx layout.Context, th *material.Theme, t Theme, it stpItem) layout.Dimensions {
	av := gtx.Dp(48)
	ringW := gtx.Dp(unit.Dp(2.5)) // ketebalan cincin (.ring padding 2.5px)
	pad := ringW
	full := av + pad*2
	sz := image.Pt(full, full)

	col := t.Accent
	if it.seen {
		col = t.Line
	}
	// cincin penuh accent/line (padding 2.5px mengelilingi avatar).
	paint.FillShape(gtx.Ops, col, clip.Ellipse{Max: image.Pt(full, full)}.Op(gtx.Ops))
	// avatar di tengah, langsung di dalam cincin.
	paint.FillShape(gtx.Ops, avatarColor(it.name), clip.Ellipse{Min: image.Pt(pad, pad), Max: image.Pt(pad+av, pad+av)}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, unit.Sp(18), initial(it.name))
		lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		lbl.Font.Weight = font.SemiBold
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}
