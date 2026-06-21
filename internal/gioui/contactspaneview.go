// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// contactspaneview.go — sidebar pane KONTAK (paritas frontend/src/lib/sidebar/
// ContactsPane.svelte + app.css): .pane-head 56px ("Kontak" 19/SemiBold), .ct-top
// (pil pencarian searchBg + tombol accent "Kontak baru"), lalu daftar .ct-list
// dgn pemisah huruf .ct-letter (accent 12/Bold, pad 5/16, bg --bg) + baris .ct-row
// (avatar 40 + nama 15/Normal + about/status 12.5 text2 + ikon info "i" text2 kanan).
// Fungsi murni, data demo inline (standalone render).
package gioui

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// cpContact = satu kontak demo (.ct-row).
type cpContact struct {
	name   string
	about  string
	online bool // .ct-dot (titik hijau online)
}

// cpGroup = satu kelompok huruf (.ct-letter + items).
type cpGroup struct {
	letter string
	items  []cpContact
}

// ContactsPaneView menggambar sidebar 380px (t.SidebarBg) berisi pane KONTAK.
// Fungsi murni, mandiri (standalone render).
func ContactsPaneView(gtx layout.Context, th *material.Theme, t Theme, groups []cpGroup) layout.Dimensions {
	w := gtx.Dp(380)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	if groups == nil { // data demo (render standalone / gio-shot)
		groups = []cpGroup{
			{letter: "A", items: []cpContact{{name: "Alice", about: "Tersedia", online: true}}},
			{letter: "B", items: []cpContact{{name: "Bob", about: "Di tempat kerja"}}},
			{letter: "C", items: []cpContact{{name: "Carol", about: "Sibuk · jangan ganggu"}}},
		}
	}

	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// .pane-head 56px "Kontak" (19/SemiBold).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return cpPaneHead(gtx, th, t, w, "Kontak")
		}),
		// .ct-top { gap: 8px; padding: 6px 12px 10px } : pil cari + tombol baru.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return cpTop(gtx, th, t)
		}),
		// .ct-list : pemisah huruf + baris kontak.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, len(groups)*2)
			for _, g := range groups {
				gg := g
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return cpLetter(gtx, th, t, gg.letter)
				}))
				for _, c := range gg.items {
					cc := c
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return cpRow(gtx, th, t, cc)
					}))
				}
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)
	return layout.Dimensions{Size: sz}
}

// cpPaneHead — .pane-head { height: 56px; padding: 0 16px; background: head-bg }
// h2 19/SemiBold.
func cpPaneHead(gtx layout.Context, th *material.Theme, t Theme, w int, title string) layout.Dimensions {
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

// cpTop — .ct-top { display:flex; gap:8px; padding:6px 12px 10px } : pil pencarian
// (searchBg, r-pill, magnifier + placeholder) + tombol accent "Kontak baru".
func cpTop(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return cpSearchPill(gtx, th, t)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout), // gap: 8px
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return cpNewBtn(gtx, th, t)
			}),
		)
	})
}

// cpSearchPill — .ct-search { background: var(--bg2); border-radius: 10px;
// padding: 9px 14px } : magnifier 18 text2 + placeholder "Cari".
func cpSearchPill(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon(gtx, "search", 18, t.Text2)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(14.5), "Cari")
				lbl.Color = t.Text2
				lbl.MaxLines = 1
				return lbl.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	r := gtx.Dp(10) // border-radius: 10px
	paint.FillShape(gtx.Ops, t.Bg2, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// cpNewBtn — .ct-new { background: accent; color: #fff; border-radius: 10px;
// padding: 8px 11px; gap: 6px; font-size: 13px } : ikon tambah-kontak + "Kontak baru".
func cpNewBtn(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(11), Right: unit.Dp(11)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon(gtx, "addmember", 17, white)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout), // gap: 6px
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 13, "Kontak baru")
				lbl.Color = white
				lbl.MaxLines = 1
				return lbl.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	r := gtx.Dp(10) // border-radius: 10px
	paint.FillShape(gtx.Ops, t.Accent, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// cpLetter — .ct-letter { background: var(--bg); color: var(--accent);
// font-size: 12px; font-weight: 700; padding: 5px 16px } pemisah huruf alfabet.
func cpLetter(gtx layout.Context, th *material.Theme, t Theme, letter string) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		lbl := material.Label(th, 12, letter)
		lbl.Color = t.Accent
		lbl.MaxLines = 1
		lbl.Font.Weight = font.Bold
		return lbl.Layout(gtx)
	})
	call := macro.Stop()
	paint.FillShape(gtx.Ops, t.Bg, clip.Rect{Max: dims.Size}.Op()) // background: var(--bg)
	call.Add(gtx.Ops)
	return dims
}

// cpRow — .ct-row { gap: 12px; padding: 8px 14px } : avatar 40 (.avatar.sm) +
// .ct-av (titik online .ct-dot) + kolom .ct-meta (.ct-name 15/Normal text +
// .ct-sub 12.5 text2) + ikon info "i" .ct-info text2 kanan.
func cpRow(gtx layout.Context, th *material.Theme, t Theme, c cpContact) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return cpAvatarDot(gtx, th, t, c)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout), // gap: 12px
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, 15, c.name) // .ct-name 15px
						lbl.Color = t.Text
						lbl.MaxLines = 1
						lbl.Font.Weight = font.Normal
						return lbl.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(12.5), c.about) // .ct-sub 12.5px
						lbl.Color = t.Text2
						lbl.MaxLines = 1
						return lbl.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon(gtx, "info", 20, t.Text2)
			}),
		)
	})
}

// cpAvatarDot — .ct-av { position: relative } : avatar 40 (.avatar.sm) + titik
// online .ct-dot { right:-1; bottom:-1; 12x12; bg:#28c840; border:2px var(--bg) }.
func cpAvatarDot(gtx layout.Context, th *material.Theme, t Theme, c cpContact) layout.Dimensions {
	av := cpAvatar(gtx, th, t, c.name, 40)
	if c.online {
		dot := gtx.Dp(12)
		bw := gtx.Dp(2)
		off := gtx.Dp(1) // right:-1px; bottom:-1px
		x := av.Size.X - dot + off
		y := av.Size.Y - dot + off
		green := color.NRGBA{R: 0x28, G: 0xc8, B: 0x40, A: 0xff} // #28c840
		// border 2px var(--bg).
		paint.FillShape(gtx.Ops, t.Bg, clip.Ellipse{Min: image.Pt(x, y), Max: image.Pt(x+dot, y+dot)}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, green, clip.Ellipse{Min: image.Pt(x+bw, y+bw), Max: image.Pt(x+dot-bw, y+dot-bw)}.Op(gtx.Ops))
	}
	return av
}

// cpAvatar — .ct-av lingkaran avatarColor(name) + inisial putih (paritas
// u.avatar: font 0.4*d, Bold, putih, di tengah).
func cpAvatar(gtx layout.Context, th *material.Theme, t Theme, name string, dp int) layout.Dimensions {
	_ = t
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
