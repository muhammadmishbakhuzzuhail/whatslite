// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// archivedpaneview.go — sidebar pane ARSIP (paritas frontend/src/lib/sidebar/
// ArchivedPane.svelte + app.css): .pane-head 56px (chevron back ‹ text2 +
// judul "Arsip" 17), lalu .chat-list berisi baris chat normal (avatar 49 +
// nama 16.5/Medium + waktu + preview 14 text2). Fungsi murni, data demo inline.
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

// avChat = satu baris demo chat terarsip.
type avChat struct {
	name    string
	time    string
	preview string
}

// ArchivedPaneView menggambar sidebar 380px (t.SidebarBg) berisi pane ARSIP:
// .pane-head (back chevron + "Arsip") + daftar 3 chat terarsip. Fungsi murni,
// mandiri (standalone render).
func ArchivedPaneView(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	w := gtx.Dp(380)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	chats := []avChat{
		{name: "Andi Pratama", time: "19.08", preview: "Mantap! Sampai nanti malam 🙌"},
		{name: "Promo Belanja", time: "Kemarin", preview: "Diskon 50% khusus hari ini!"},
		{name: "Grup Alumni", time: "Senin", preview: "Budi: jadi reuni bulan depan?"},
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return avPaneHead(gtx, th, t, w)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			// .chat-list { padding: 4px 8px; }
			return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				children := make([]layout.FlexChild, 0, len(chats))
				for i := range chats {
					c := chats[i]
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return avChatRow(gtx, th, t, c)
					}))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
			})
		}),
	)
}

// avPaneHead — .pane-head { height: 56px; padding: 0 16px; background: head-bg;
// gap: 14px } : icon-btn back (chevron ‹ text2) + h2 "Arsip" 17px.
func avPaneHead(gtx layout.Context, th *material.Theme, t Theme, w int) layout.Dimensions {
	h := gtx.Dp(56)
	sz := image.Pt(w, h)
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Rect{Max: sz}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			// .icon-btn 40x40 berisi chevron back (path M15 5l-7 7 7 7).
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return avBackBtn(gtx, t)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 17, "Arsip")
				lbl.Color = t.Text
				lbl.Font.Weight = font.SemiBold
				return lbl.Layout(gtx)
			}),
		)
	})
	return layout.Dimensions{Size: sz}
}

// avBackBtn — .icon-btn 40x40 transparan berisi chevron ‹ (back), warna text2.
// Glyph 24dp di tengah (SVG path M15 5l-7 7 7 7 → ikon "chevleft").
func avBackBtn(gtx layout.Context, t Theme) layout.Dimensions {
	box := gtx.Dp(40)
	sz := image.Pt(box, box)
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return icon(gtx, "chevleft", 24, t.Text2)
	})
	return layout.Dimensions{Size: sz}
}

// avChatRow — .chat-row { padding: 10px 12px; gap: 13px; border-radius: 14px } :
// avatar 49 + kolom (nama 16.5/Medium + waktu | preview 14 text2).
func avChatRow(gtx layout.Context, th *material.Theme, t Theme, c avChat) layout.Dimensions {
	// .chat-row latar transparan (hover/active diabaikan utk render statis).
	return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return avAvatar(gtx, th, c.name, 49)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(13)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					// .row-top : nama (flex 1, 16.5/500) + waktu (text2 12).
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(th, 16.5, c.name)
								lbl.Color = t.Text
								lbl.MaxLines = 1
								lbl.Font.Weight = font.Medium
								return lbl.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(th, 12, c.time)
								lbl.Color = t.Text2
								return lbl.Layout(gtx)
							}),
						)
					}),
					// .row-bottom { margin-top: 2px }
					layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, 14, c.preview)
						lbl.Color = t.Text2
						lbl.MaxLines = 1
						return lbl.Layout(gtx)
					}),
				)
			}),
		)
	})
}

// avAvatar — .avatar 49 : lingkaran avatarColor(name) + inisial putih 0.4*d Bold
// (paritas u.avatar di ui.go).
func avAvatar(gtx layout.Context, th *material.Theme, name string, dp int) layout.Dimensions {
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
