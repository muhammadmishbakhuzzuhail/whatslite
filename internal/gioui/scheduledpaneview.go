// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// scheduledpaneview.go — sidebar pane TERJADWAL (paritas frontend/src/lib/sidebar/
// ScheduledPane.svelte + app.css): .pane-head 56px (chevron kembali + "Terjadwal"
// 17), section header uppercase "TERJADWAL (2)" (12/SemiBold text2), lalu .sc-row:
// nama 14/SemiBold + teks 13 text2 + baris waktu (⏰ + "besok 08.00") 12 accent +
// tombol ✕ bundar 30px (bg2). Fungsi murni, data demo inline (standalone render).
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

// scItem = satu baris .sc-row terjadwal (nama + teks + waktu).
type scItem struct {
	name string
	text string
	when string
}

// ScheduledPaneView menggambar sidebar 380px (t.SidebarBg) berisi pane TERJADWAL:
// .pane-head + section header + 2 baris demo. Fungsi murni, mandiri (standalone).
func ScheduledPaneView(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	w := gtx.Dp(468)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	items := []scItem{
		{name: "Andi Pratama", text: "Selamat pagi, jangan lupa rapat ya", when: "besok 08.00"},
		{name: "Tim Proyek X", text: "Reminder: deadline laporan akhir", when: "Sen 09.30"},
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// .pane-head { height:56px; padding:0 16px; gap:22px } : chevron + h2 17.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return scPaneHead(gtx, th, t, w, "Terjadwal")
		}),
		// .sc-sec uppercase header 12/SemiBold text2.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return scSection(gtx, th, t, "TERJADWAL (2)")
		}),
		// daftar .sc-row.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, len(items))
			for i := range items {
				it := items[i]
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return scRow(gtx, th, t, it)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)
}

// scPaneHead — .pane-head { height:56px; padding:0 16px; gap:22px; background:head-bg }
// : chevron kembali (M15 5l-7 7 7 7) + h2 17/SemiBold.
func scPaneHead(gtx layout.Context, th *material.Theme, t Theme, w int, title string) layout.Dimensions {
	h := gtx.Dp(56)
	sz := image.Pt(w, h)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return scChevron(gtx, t.Text2)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(22)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, 17, title)
					lbl.Color = t.Text
					lbl.Font.Weight = font.SemiBold
					return lbl.Layout(gtx)
				}),
			)
		})
	})
	return layout.Dimensions{Size: sz}
}

// scChevron — ikon kembali .icon-btn svg (M15 5l-7 7 7 7): ikon "chevleft" 24x24,
// warna text2.
func scChevron(gtx layout.Context, col color.NRGBA) layout.Dimensions {
	return icon(gtx, "chevleft", 24, col)
}

// scSection — .sc-sec { font-size:12; font-weight:700; uppercase; letter-spacing:.4px;
// color:text2; padding:14px 16px 6px }.
func scSection(gtx layout.Context, th *material.Theme, t Theme, title string) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(14), Bottom: unit.Dp(6), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, 12, title)
		lbl.Color = t.Text2
		lbl.Font.Weight = font.SemiBold
		return lbl.Layout(gtx)
	})
}

// scRow — .sc-row { display:flex; align-items:center; gap:8px; padding:6px 12px } :
// .sc-main (kolom: nama + teks + waktu) + .sc-x (tombol ✕ bundar 30).
func scRow(gtx layout.Context, th *material.Theme, t Theme, it scItem) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			// .sc-main { flex:1; padding:6px 4px } kolom kiri.
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						// .sc-name 14/SemiBold text.
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, 14, it.name)
							lbl.Color = t.Text
							lbl.MaxLines = 1
							lbl.Font.Weight = font.SemiBold
							return lbl.Layout(gtx)
						}),
						// .sc-text 13 text2 (ellipsis).
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, 13, it.text)
							lbl.Color = t.Text2
							lbl.MaxLines = 1
							return lbl.Layout(gtx)
						}),
						// .sc-when { margin-top:2px } : ⏰ (ikon clock) + waktu, 12 accent.
						layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return icon(gtx, "clock", 14, t.Accent)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									lbl := material.Label(th, 12, it.when)
									lbl.Color = t.Accent
									lbl.MaxLines = 1
									return lbl.Layout(gtx)
								}),
							)
						}),
					)
				})
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			// .sc-x tombol batal bundar 30 bg2.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return scCancel(gtx, th, t)
			}),
		)
	})
}

// scCancel — .sc-x { background:bg2; color:text2; width:30; height:30;
// border-radius:50% } berisi glyph ✕ di tengah.
func scCancel(gtx layout.Context, _ *material.Theme, t Theme) layout.Dimensions {
	d := gtx.Dp(30)
	sz := image.Pt(d, d)
	paint.FillShape(gtx.Ops, t.Bg2, clip.Ellipse{Max: sz}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return icon(gtx, "close", 14, t.Text2)
	})
	return layout.Dimensions{Size: sz}
}
