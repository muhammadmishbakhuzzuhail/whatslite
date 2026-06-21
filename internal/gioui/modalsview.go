// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// modalsview.go — modal "Teruskan" (forward) di atas backdrop redup (paritas
// .modal-backdrop + .fwd-modal app.css). Kartu sidebarBg radius 12 berisi judul,
// pil pencarian, daftar chat (avatar 40 + nama + sub), lalu tombol Kirim/Batal.
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
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// mvRow = satu baris chat dalam daftar teruskan (avatar + nama + sub).
type mvRow struct {
	name string
	sub  string
}

// FwdCtl = state interaktif modal teruskan. nil → daftar demo statis (gio-shot).
// Tiap baris Rows[i] dapat diklik (Clicks[i]) → teruskan ke chat itu lalu tutup.
type FwdCtl struct {
	Rows   []mvRow
	Clicks []widget.Clickable
	Cancel *widget.Clickable
}

// ModalsView menggambar backdrop redup penuh lalu kartu forward terpusat.
func ModalsView(gtx layout.Context, th *material.Theme, t Theme, ctl *FwdCtl) layout.Dimensions {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}

	// latar app (var(--bg)) lalu backdrop rgba(0,0,0,.4) penuh.
	paint.FillShape(gtx.Ops, t.Bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 102}, clip.Rect{Max: gtx.Constraints.Max}.Op())

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return mvCard(gtx, th, t, white, ctl)
	})
}

// mvCard — .fwd-modal: sidebarBg, radius 12, lebar 380, kolom isi.
func mvCard(gtx layout.Context, th *material.Theme, t Theme, white color.NRGBA, ctl *FwdCtl) layout.Dimensions {
	w := gtx.Dp(380)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w

	rows := []mvRow{
		{name: "Andi Pratama", sub: "Terakhir dilihat hari ini 19.08"},
		{name: "Keluarga", sub: "Ibu, Ayah, Dimas, +4"},
		{name: "Sarah", sub: "Online"},
		{name: "Tim Proyek X", sub: "Budi, Citra, Eka, +9"},
	}
	if ctl != nil {
		rows = ctl.Rows
	}

	macro := op.Record(gtx.Ops)
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// .fwd-head — padding 16, judul 17/SemiBold.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 17, "Teruskan")
				lbl.Color = t.Text
				lbl.Font.Weight = font.SemiBold
				return lbl.Layout(gtx)
			})
		}),
		// .fwd-search — margin 0 12 10, pil searchBg radius 8, padding 8/12.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return mvSearch(gtx, th, t)
		}),
		// .fwd-list — baris-baris chat (klik = teruskan bila interaktif).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, len(rows))
			for i := range rows {
				rr, idx := rows[i], i
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					row := func(gtx layout.Context) layout.Dimensions { return mvChatRow(gtx, th, t, rr) }
					if ctl != nil && idx < len(ctl.Clicks) {
						return ctl.Clicks[idx].Layout(gtx, row)
					}
					return row(gtx)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
		// baris tombol bawah: Batal (+ Kirim pada mode statis demo).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return mvFooter(gtx, th, t, white, ctl)
		}),
	)
	call := macro.Stop()
	r := gtx.Dp(12)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// mvSearch — pil .fwd-search: searchBg, radius 8, padding 8/12, teks "Cari pesan" text2.
func mvSearch(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			lbl := material.Label(th, 14, "Cari pesan")
			lbl.Color = t.Text2
			return lbl.Layout(gtx)
		})
		call := macro.Stop()
		r := gtx.Dp(8)
		paint.FillShape(gtx.Ops, t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// mvChatRow — .fwd-row: padding 8/16, gap 12, avatar 40 + nama 15/Medium + sub 13 text2.
func mvChatRow(gtx layout.Context, th *material.Theme, t Theme, r mvRow) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return mvAvatar(gtx, th, r.name, 40)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, 15, r.name)
						lbl.Color = t.Text
						lbl.MaxLines = 1
						return lbl.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, 13, r.sub)
						lbl.Color = t.Text2
						lbl.MaxLines = 1
						return lbl.Layout(gtx)
					}),
				)
			}),
		)
	})
}

// mvAvatar — lingkaran 40 warna avatarColor(name) + inisial putih (paritas u.avatar).
func mvAvatar(gtx layout.Context, th *material.Theme, name string, dp int) layout.Dimensions {
	d := gtx.Dp(unit.Dp(dp))
	sz := image.Pt(d, d)
	paint.FillShape(gtx.Ops, avatarColor(name), clip.Ellipse{Max: sz}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, 16, initial(name))
		lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		lbl.Font.Weight = font.Bold
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}

// mvFooter — baris tombol: Kirim (.btn-accent radius 10, putih) + Batal (.btn-ghost bg2).
func mvFooter(gtx layout.Context, th *material.Theme, t Theme, white color.NRGBA, ctl *FwdCtl) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(16), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		batal := func(gtx layout.Context) layout.Dimensions {
			return mvBtn(gtx, th, t.Bg2, t.Text, "Batal", font.SemiBold)
		}
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				// petunjuk: ketuk chat utk meneruskan (mode interaktif).
				if ctl == nil {
					return layout.Dimensions{}
				}
				lbl := material.Label(th, 13, "Ketuk chat untuk meneruskan")
				lbl.Color = t.Text2
				return layout.W.Layout(gtx, lbl.Layout)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if ctl != nil && ctl.Cancel != nil {
					return ctl.Cancel.Layout(gtx, batal)
				}
				return batal(gtx)
			}),
			// tombol "Kirim" hanya pada render statis demo (mode interaktif = ketuk baris).
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if ctl != nil {
					return layout.Dimensions{}
				}
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return mvBtn(gtx, th, t.Accent, white, "Kirim", font.SemiBold)
					}),
				)
			}),
		)
	})
}

// mvBtn — tombol generik (.btn-accent/.btn-ghost): radius 10, padding 9/16, font 14.5/600 (body inherit).
func mvBtn(gtx layout.Context, th *material.Theme, bg, fg color.NRGBA, txt string, wt font.Weight) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, 14.5, txt)
		lbl.Color = fg
		lbl.Font.Weight = wt
		return lbl.Layout(gtx)
	})
	call := macro.Stop()
	r := gtx.Dp(10)
	paint.FillShape(gtx.Ops, bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}
