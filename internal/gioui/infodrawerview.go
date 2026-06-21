// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// infodrawerview.go — laci info grup kanan (paritas frontend/src/lib/chat/
// InfoPanel.svelte + app.css): header 56, hero avatar 200, bar pemisah
// wallpaper 6px, blok Deskripsi, lalu baris aksi (tambah/undangan/keluar).
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

// InfoDrawerView menggambar laci info grup 400px di sisi kanan (sidebarBg).
func InfoDrawerView(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	// .info-panel { width: 400px; background: var(--sidebar-bg); }
	w := gtx.Dp(400)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	const groupName = "Grup Kerja"
	dangerCol := color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff} // #e35d6a

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// .info-head — height 56, head-bg, title 17/500.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return infoDrawerHead(gtx, th, t, w)
		}),
		// .info-hero — pad 28/24, avatar 200, nama 19/500, "4 anggota" 14.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return infoDrawerHero(gtx, th, t, groupName)
		}),
		// pemisah 6px var(--wallpaper) (border-bottom .info-hero).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return infoDrawerSep(gtx, t, w)
		}),
		// .info-block — Deskripsi: lbl accent 13 + val text 15.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return infoDrawerBlock(gtx, th, t, "Deskripsi", "Koordinasi tim proyek")
		}),
		// pemisah 6px var(--wallpaper) (border-bottom .info-block).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return infoDrawerSep(gtx, t, w)
		}),
		// baris aksi (.info-row): ikon 22 + label 15.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return infoDrawerRow(gtx, th, t, infoDrawerAddIcon, "Tambah anggota", t.Text2, t.Text)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return infoDrawerRow(gtx, th, t, infoDrawerLinkIcon, "Link undangan", t.Text2, t.Text)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return infoDrawerRow(gtx, th, t, infoDrawerLeaveIcon, "Keluar grup", dangerCol, dangerCol)
		}),
	)
}

// infoDrawerHead: .info-head — tinggi 56, latar head-bg, pad 0 16, title 17/500.
func infoDrawerHead(gtx layout.Context, th *material.Theme, t Theme, w int) layout.Dimensions {
	h := gtx.Dp(56)
	sz := image.Pt(w, h)
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Rect{Max: sz}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 17, "Info grup")
			lbl.Color = t.Text
			lbl.Font.Weight = font.Medium
			return lbl.Layout(gtx)
		})
	})
	return layout.Dimensions{Size: sz}
}

// infoDrawerHero: .info-hero — pad 28/24, avatar 200 di tengah + nama + jumlah.
func infoDrawerHero(gtx layout.Context, th *material.Theme, t Theme, name string) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(28), Bottom: unit.Dp(28), Left: unit.Dp(24), Right: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			// .avatar.big — 200x200, lingkaran avatarColor + inisial putih 80.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return infoDrawerAvatar(gtx, th, name, 200)
			}),
			// margin: 0 auto 16px.
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			// .iname — 19/500 (mengikuti spesifikasi laci).
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 19, name)
				lbl.Color = t.Text
				lbl.Font.Weight = font.SemiBold
				return lbl.Layout(gtx)
			}),
			// .iphone — margin-top 4, 14, text2.
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 14, "4 anggota")
				lbl.Color = t.Text2
				return lbl.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: dims.Size}.Op())
	call.Add(gtx.Ops)
	return dims
}

// infoDrawerAvatar: lingkaran avatarColor(name) + inisial putih, ukuran dp.
func infoDrawerAvatar(gtx layout.Context, th *material.Theme, name string, dp int) layout.Dimensions {
	d := gtx.Dp(unit.Dp(dp))
	sz := image.Pt(d, d)
	paint.FillShape(gtx.Ops, avatarColor(name), clip.Ellipse{Max: sz}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// font-size: 80px untuk .avatar.big.
		lbl := material.Label(th, 80, initial(name))
		lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		lbl.Font.Weight = font.SemiBold
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}

// infoDrawerSep: border-bottom 6px var(--wallpaper) sebagai bar penuh-lebar.
func infoDrawerSep(gtx layout.Context, t Theme, w int) layout.Dimensions {
	h := gtx.Dp(6)
	sz := image.Pt(w, h)
	paint.FillShape(gtx.Ops, t.Wallpaper, clip.Rect{Max: sz}.Op())
	return layout.Dimensions{Size: sz}
}

// infoDrawerBlock: .info-block — pad 14/24, lbl accent 13 (mb 5) + val 15 text.
func infoDrawerBlock(gtx layout.Context, th *material.Theme, t Theme, label, value string) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(14), Bottom: unit.Dp(14), Left: unit.Dp(24), Right: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 13, label)
				lbl.Color = t.Accent
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 15, value)
				lbl.Color = t.Text
				return lbl.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: dims.Size}.Op())
	call.Add(gtx.Ops)
	return dims
}

// infoDrawerRow: .info-row — pad 14/24, gap 18, ikon 22 + label 15.
func infoDrawerRow(gtx layout.Context, th *material.Theme, t Theme, icon func(layout.Context, color.NRGBA), label string, iconCol, textCol color.NRGBA) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(14), Bottom: unit.Dp(14), Left: unit.Dp(24), Right: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return infoDrawerIconBox(gtx, icon, iconCol)
			}),
			// gap: 18px.
			layout.Rigid(layout.Spacer{Width: unit.Dp(18)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 15, label)
				lbl.Color = textCol
				lbl.MaxLines = 1
				return lbl.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: dims.Size}.Op())
	call.Add(gtx.Ops)
	return dims
}

// infoDrawerIconBox: kotak ikon 22x22 (.info-row svg) — gambar glyph via fn.
func infoDrawerIconBox(gtx layout.Context, icon func(layout.Context, color.NRGBA), col color.NRGBA) layout.Dimensions {
	d := gtx.Dp(22)
	icon(gtx, col)
	return layout.Dimensions{Size: image.Pt(d, d)}
}

// ---- ikon-placeholder 22 (bentuk sederhana via clip, warna col) ----

// infoDrawerAddIcon: tambah anggota — lingkaran kepala + bahu + plus.
func infoDrawerAddIcon(gtx layout.Context, col color.NRGBA) {
	d := gtx.Dp(22)
	bw := gtx.Dp(2)
	// kepala: cincin lingkaran kiri-atas.
	hd := gtx.Dp(8)
	hx := gtx.Dp(2)
	infoDrawerRing(gtx, col, image.Pt(hx, gtx.Dp(1)), hd, bw)
	// bahu: bar mendatar di bawah kepala.
	paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(hx, gtx.Dp(13)), Max: image.Pt(hx+hd, gtx.Dp(13)+bw)}.Op())
	// plus di kanan.
	px := gtx.Dp(15)
	py := gtx.Dp(10)
	pl := gtx.Dp(7)
	paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(px, py+(pl-bw)/2), Max: image.Pt(px+pl, py+(pl+bw)/2)}.Op())
	paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(px+(pl-bw)/2, py), Max: image.Pt(px+(pl+bw)/2, py+pl)}.Op())
	_ = d
}

// infoDrawerLinkIcon: link undangan — dua batang diagonal (rantai).
func infoDrawerLinkIcon(gtx layout.Context, col color.NRGBA) {
	bw := gtx.Dp(2)
	// dua kapsul diagonal sebagai mata rantai (didekati dgn rect membulat).
	rr := gtx.Dp(3)
	a := image.Rectangle{Min: image.Pt(gtx.Dp(3), gtx.Dp(9)), Max: image.Pt(gtx.Dp(12), gtx.Dp(9)+bw+gtx.Dp(3))}
	paint.FillShape(gtx.Ops, col, clip.RRect{Rect: a, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
	b := image.Rectangle{Min: image.Pt(gtx.Dp(10), gtx.Dp(11)), Max: image.Pt(gtx.Dp(19), gtx.Dp(11)+bw+gtx.Dp(3))}
	paint.FillShape(gtx.Ops, col, clip.RRect{Rect: b, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
}

// infoDrawerLeaveIcon: keluar grup — pintu + panah keluar.
func infoDrawerLeaveIcon(gtx layout.Context, col color.NRGBA) {
	bw := gtx.Dp(2)
	// bingkai pintu (kotak kanan, terbuka kiri).
	dx := gtx.Dp(11)
	top := gtx.Dp(3)
	bot := gtx.Dp(19)
	right := gtx.Dp(19)
	paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(right-bw, top), Max: image.Pt(right, bot)}.Op())              // sisi kanan
	paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(dx, top), Max: image.Pt(right, top+bw)}.Op())                 // atas
	paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(dx, bot-bw), Max: image.Pt(right, bot)}.Op())                 // bawah
	// panah keluar: batang mendatar + kepala.
	ay := gtx.Dp(11)
	paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(gtx.Dp(3), ay), Max: image.Pt(gtx.Dp(13), ay+bw)}.Op())      // batang
	// kepala panah (dua bar miring didekati dgn rect kecil).
	paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(gtx.Dp(5), ay-gtx.Dp(3)), Max: image.Pt(gtx.Dp(5)+bw, ay+bw+gtx.Dp(3))}.Op())
	paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(gtx.Dp(6), ay-gtx.Dp(2)), Max: image.Pt(gtx.Dp(6)+bw, ay)}.Op())
	paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(gtx.Dp(6), ay+bw), Max: image.Pt(gtx.Dp(6)+bw, ay+bw+gtx.Dp(2))}.Op())
}

// infoDrawerRing: cincin (lingkaran luar col, lubang dalam sidebarBg) — kepala ikon.
func infoDrawerRing(gtx layout.Context, col color.NRGBA, min image.Point, d, bw int) {
	outer := image.Rectangle{Min: min, Max: image.Pt(min.X+d, min.Y+d)}
	paint.FillShape(gtx.Ops, col, clip.Ellipse{Min: outer.Min, Max: outer.Max}.Op(gtx.Ops))
	inner := image.Rectangle{Min: image.Pt(min.X+bw, min.Y+bw), Max: image.Pt(min.X+d-bw, min.Y+d-bw)}
	// lubang: pakai warna sidebar agar tampak cincin di atas latar baris.
	hole := color.NRGBA{R: 0x0e, G: 0x13, B: 0x18, A: 0xff} // var(--sidebar-bg) gelap
	paint.FillShape(gtx.Ops, hole, clip.Ellipse{Min: inner.Min, Max: inner.Max}.Op(gtx.Ops))
	_ = col
}
