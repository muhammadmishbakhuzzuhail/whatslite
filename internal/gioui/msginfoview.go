// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// msginfoview.go — modal "Info pesan" di atas backdrop redup (paritas
// frontend/src/lib/chat/MessageInfoModal.svelte + .nc-card app.css): kartu
// sidebarBg radius 14, judul 19/Bold, lalu seksi "DIBACA OLEH" (dot accent) +
// "TERKIRIM KE" (dot #3d8bd3) berisi baris penerima (avatar + nama + waktu),
// diakhiri tombol "Tutup" accent. Fungsi murni, data demo inline (standalone).
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

// miRcpt = satu baris penerima (.mi-rcpt): avatar + nama + waktu kanan.
type miRcpt struct {
	name string
	time string
}

// MsgInfoView menggambar backdrop redup penuh lalu kartu info-pesan terpusat.
func MsgInfoView(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	// latar app (var(--bg)) lalu backdrop rgba(0,0,0,.5) penuh.
	paint.FillShape(gtx.Ops, t.Bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 128}, clip.Rect{Max: gtx.Constraints.Max}.Op())

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return miCard(gtx, th, t)
	})
}

// miCard — .nc-card: sidebarBg, radius 14, lebar ~360, padding 18, kolom isi.
func miCard(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	w := gtx.Dp(360)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w

	readBy := []miRcpt{
		{name: "Andi Pratama", time: "19.08"},
		{name: "Sarah", time: "19.05"},
	}
	deliveredTo := []miRcpt{
		{name: "Budi Santoso", time: "18.59"},
		{name: "Citra Dewi", time: "18.57"},
	}

	blue := color.NRGBA{R: 0x3d, G: 0x8b, B: 0xd3, A: 0xff} // #3d8bd3

	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// judul h3 19/Bold, margin-bottom 14.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 19, "Info pesan")
				lbl.Color = t.Text
				lbl.Font.Weight = font.Bold
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),

			// .mi-sec "DIBACA OLEH" (dot accent) + baris-baris.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return miSec(gtx, th, t, "DIBACA OLEH", t.Accent)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return miList(gtx, th, t, readBy)
			}),

			// .mi-sec "TERKIRIM KE" (dot #3d8bd3) + baris-baris.
			layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return miSec(gtx, th, t, "TERKIRIM KE", blue)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return miList(gtx, th, t, deliveredTo)
			}),

			// tombol "Tutup" accent rata-kanan, margin-top 16.
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return miBtn(gtx, th, t.Accent, white, "Tutup")
					}),
				)
			}),
		)
	})
	call := macro.Stop()
	r := gtx.Dp(14)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// miSec — .mi-sec: dot 8px + label 12/SemiBold uppercase text2 (gap 7).
func miSec(gtx layout.Context, th *material.Theme, t Theme, label string, dot color.NRGBA) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return miDot(gtx, dot)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(7)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 12, label)
			lbl.Color = t.Text2
			lbl.Font.Weight = font.SemiBold
			return lbl.Layout(gtx)
		}),
	)
}

// miDot — .mi-dot: lingkaran 8x8.
func miDot(gtx layout.Context, col color.NRGBA) layout.Dimensions {
	d := gtx.Dp(8)
	sz := image.Pt(d, d)
	paint.FillShape(gtx.Ops, col, clip.Ellipse{Max: sz}.Op(gtx.Ops))
	return layout.Dimensions{Size: sz}
}

// miList — kolom baris penerima.
func miList(gtx layout.Context, th *material.Theme, t Theme, rows []miRcpt) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(rows))
	for i := range rows {
		r := rows[i]
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return miRcptRow(gtx, th, t, r)
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// miRcptRow — .mi-rcpt: padding 3/0, avatar 34 + nama 13.5 + waktu 12 text2 kanan.
func miRcptRow(gtx layout.Context, th *material.Theme, t Theme, r miRcpt) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return miAvatar(gtx, th, r.name, 34)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(11)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 13.5, r.name)
				lbl.Color = t.Text
				lbl.MaxLines = 1
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 12, r.time)
				lbl.Color = t.Text2
				return lbl.Layout(gtx)
			}),
		)
	})
}

// miAvatar — lingkaran avatarColor(name) + inisial putih (paritas u.avatar).
func miAvatar(gtx layout.Context, th *material.Theme, name string, dp int) layout.Dimensions {
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

// miBtn — .btn-accent: radius 10, padding 9/16, font 14/SemiBold.
func miBtn(gtx layout.Context, th *material.Theme, bg, fg color.NRGBA, txt string) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, 14, txt)
		lbl.Color = fg
		lbl.Font.Weight = font.SemiBold
		return lbl.Layout(gtx)
	})
	call := macro.Stop()
	r := gtx.Dp(10)
	paint.FillShape(gtx.Ops, bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}
