// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// sidepanesview.go — sidebar pane CALLS (paritas frontend/src/lib/sidebar/
// CallsPane.svelte + app.css): .sidebar-head 60px ("Panggilan" 23/Bold), lalu
// daftar baris panggilan — avatar 49 + nama + sub-baris (panah arah + label,
// merah utk tak terjawab) + ikon panggil accent kanan. Fungsi murni, data demo.
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

// spCall = satu baris demo panggilan.
type spCall struct {
	name   string
	time   string
	video  bool
	missed bool
}

// SidePanesView menggambar sidebar 380px (t.SidebarBg) berisi pane CALLS:
// header .pane-head + 4 baris panggilan demo. Fungsi murni, mandiri (standalone).
func SidePanesView(gtx layout.Context, th *material.Theme, t Theme, calls []spCall) layout.Dimensions {
	w := gtx.Dp(468)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	if calls == nil { // data demo (render standalone / gio-shot)
		calls = []spCall{
			{name: "Andi Pratama", time: "19.08", video: true, missed: true},
			{name: "Keluarga", time: "18.41", video: false, missed: false},
			{name: "Sarah", time: "17.55", video: false, missed: true},
			{name: "Tim Proyek X", time: "16.20", video: true, missed: false},
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return spPaneHead(gtx, th, t, w, "Panggilan")
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			// .chat-list { padding: 4px 8px; }
			return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				children := make([]layout.FlexChild, 0, len(calls))
				for i := range calls {
					c := calls[i]
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return spCallRow(gtx, th, t, c)
					}))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
			})
		}),
	)
}

// spPaneHead — .sidebar-head { height: 60px; padding: 0 18px; background: head-bg;
// border-bottom: 1px solid divider } ; h1 23/Bold (letter-spacing -.3px).
func spPaneHead(gtx layout.Context, th *material.Theme, t Theme, w int, title string) layout.Dimensions {
	h := gtx.Dp(60)
	sz := image.Pt(w, h)
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Rect{Max: sz}.Op())
	// border-bottom: 1px solid var(--divider)
	bh := gtx.Dp(1)
	paint.FillShape(gtx.Ops, t.Divider, clip.Rect{Min: image.Pt(0, h-bh), Max: image.Pt(w, h)}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Inset{Left: unit.Dp(18), Right: unit.Dp(18)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 23, title) // .sidebar-head h1 23px/700
			lbl.Color = t.Text
			lbl.Font.Weight = font.Bold
			return lbl.Layout(gtx)
		})
	})
	return layout.Dimensions{Size: sz}
}

// spCallRow — .chat-row { padding: 10px 12px; gap: 13px } : avatar 49 + kolom
// (nama 16.5/Medium + sub-baris panah+label) + ikon panggil accent kanan.
func spCallRow(gtx layout.Context, th *material.Theme, t Theme, c spCall) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return spAvatar(gtx, th, t, c.name, 49)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(13)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					// .row-top : nama (flex 1) + waktu (text2 12)
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(th, 16.5, c.name) // .row-name 16.5px/500
								lbl.Color = t.Text
								lbl.MaxLines = 1
								lbl.Font.Weight = font.Medium
								return lbl.Layout(gtx)
							}),
							// .row-time { margin-left: 8px }
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(th, 12, c.time) // .row-time 12px
								lbl.Color = t.Text2
								return lbl.Layout(gtx)
							}),
						)
					}),
					// .row-bottom { margin-top: 2px }
					layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return spCallLine(gtx, th, t, c)
					}),
				)
			}),
			// ikon panggil accent kanan (telepon / kamera tergantung tipe)
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return spCallIcon(gtx, t, c.video)
			}),
		)
	})
}

// spCallLine — .call-line { gap: 6px } : panah arah kecil + label.
// Tak terjawab → merah #e35d6a; selain itu → text2.
func spCallLine(gtx layout.Context, th *material.Theme, t Theme, c spCall) layout.Dimensions {
	col := t.Text2
	if c.missed {
		col = color.NRGBA{R: 0xef, G: 0x53, B: 0x50, A: 0xff} // .call-line.missed #ef5350
	}
	var label string
	if c.video {
		label = "Panggilan video · "
	} else {
		label = "Panggilan suara · "
	}
	if c.missed {
		label += "Tak terjawab"
	} else {
		label += "Masuk"
	}
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return spArrow(gtx, col, c.missed)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 14, label)
			lbl.Color = col
			lbl.MaxLines = 1
			return lbl.Layout(gtx)
		}),
	)
}

// spArrow — .call-ico 15x15 : panah arah panggilan (callArrowOut, paritas SVG
// path M7 17L17 7M17 7H9M17 7v8). Warna mengikuti garis (currentColor / merah
// utk tak terjawab).
func spArrow(gtx layout.Context, col color.NRGBA, missed bool) layout.Dimensions {
	_ = missed
	return icon(gtx, "callArrowOut", 15, col)
}

// spCallIcon — ikon panggil accent di kanan baris (.icon-btn ~40, glyph accent).
// Pakai ikon native "calls" (gagang telepon WhatsApp) ber-tint accent, 20dp glyph
// di tengah kotak 40dp.
func spCallIcon(gtx layout.Context, t Theme, video bool) layout.Dimensions {
	_ = video
	box := gtx.Dp(40)
	sz := image.Pt(box, box)
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return icon(gtx, "calls", 20, t.Accent)
	})
	return layout.Dimensions{Size: sz}
}

// spAvatar — .avatar 49 : lingkaran avatarColor(name) + inisial putih (paritas
// u.avatar di ui.go: font 0.4*d, Bold, putih, di tengah).
func spAvatar(gtx layout.Context, th *material.Theme, t Theme, name string, dp int) layout.Dimensions {
	_ = t
	d := gtx.Dp(unit.Dp(dp))
	sz := image.Pt(d, d)
	col := avatarColor(name)
	// rekam latar lingkaran lalu inisial di tengah (tanpa op import bila tak perlu).
	paint.FillShape(gtx.Ops, col, clip.Ellipse{Max: sz}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, unit.Sp(float32(dp)*0.4), initial(name))
		lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		lbl.Font.Weight = font.Bold
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}
