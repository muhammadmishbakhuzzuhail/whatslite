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

	"math"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// stpItem = satu baris status terkini (.status-row di .ct-letter section).
type stpItem struct {
	name      string
	time      string
	jid       string // utk muat foto profil asli (fallback inisial bila kosong/gagal)
	seen      bool   // semua dilihat (cincin abu penuh)
	count     int    // jumlah unggahan (segmen cincin); <=1 → cincin utuh
	seenCount int    // segmen sudah dilihat (abu); sisanya accent
}

// StatusPaneView menggambar sidebar 380px (t.SidebarBg) berisi pane STATUS:
// header .pane-head + baris "My status" + label "TERKINI" + 3 baris status.
// Fungsi murni, mandiri (standalone render).
func StatusPaneView(gtx layout.Context, th *material.Theme, t Theme, items []stpItem, clicks []widget.Clickable, avFn cpAvatarFn, selfName, selfJID string) layout.Dimensions {
	w := gtx.Dp(468)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	if items == nil { // data demo (render standalone / gio-shot)
		items = []stpItem{
			{name: "Andi Pratama", time: "2 menit lalu", count: 5, seenCount: 2}, // 5 segmen, 2 dilihat
			{name: "Sarah Wijaya", time: "15 menit lalu", count: 3, seenCount: 0},
			{name: "Tim Proyek X", time: "1 jam lalu", seen: true, count: 4, seenCount: 4}, // semua abu
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// .pane-head 56px — "Status" 19/SemiBold.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return stpPaneHead(gtx, th, t, w, "Status")
		}),
		// Baris "My status" (avatar + badge "+").
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return stpMyStatusRow(gtx, th, t, avFn, selfName, selfJID)
		}),
		// .ct-letter label "TERKINI" (accent 12/Bold).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return stpSectionLabel(gtx, th, t, "TERKINI")
		}),
		// daftar baris status terkini.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, len(items))
			for i := range items {
				it, idx := items[i], i
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					row := func(gtx layout.Context) layout.Dimensions { return stpStatusRow(gtx, th, t, it, avFn) }
					if idx < len(clicks) {
						return clicks[idx].Layout(gtx, row)
					}
					return row(gtx)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)
}

// stpPaneHead — .pane-head { height: 56px; padding: 0 16px; background: head-bg }
// h2 19/SemiBold.
func stpPaneHead(gtx layout.Context, th *material.Theme, t Theme, w int, title string) layout.Dimensions {
	return paneHead(gtx, th, t, w, title)
}

// stpMyStatusRow — .status-row { padding: 10px 14px; gap: 14px } : avatar 48 dgn
// badge "+" accent kanan-bawah, lalu kolom (nama 15/SemiBold "Status saya" +
// hint 12.5 text2 "Ketuk untuk menambahkan").
func stpMyStatusRow(gtx layout.Context, th *material.Theme, t Theme, avFn cpAvatarFn, selfName, selfJID string) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return stpMyAvatar(gtx, th, t, avFn, selfName, selfJID)
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
func stpMyAvatar(gtx layout.Context, th *material.Theme, t Theme, avFn cpAvatarFn, selfName, selfJID string) layout.Dimensions {
	d := gtx.Dp(48)
	badge := gtx.Dp(18)
	bd := gtx.Dp(2)  // border badge (2px bg)
	off := gtx.Dp(2) // right/bottom: -2px → tonjol keluar 2px
	sz := image.Pt(d+off, d+off)

	// avatar diri: foto profil asli (avFn) bila ada, else lingkaran warna + inisial.
	if avFn != nil && selfJID != "" {
		nm := selfName
		if nm == "" {
			nm = "Saya"
		}
		cg := gtx
		cg.Constraints.Min, cg.Constraints.Max = image.Pt(d, d), image.Pt(d, d)
		avFn(cg, nm, selfJID, 48)
	} else {
		paint.FillShape(gtx.Ops, avatarColor("me"), clip.Ellipse{Max: image.Pt(d, d)}.Op(gtx.Ops))
		gtx.Constraints.Min, gtx.Constraints.Max = image.Pt(d, d), image.Pt(d, d)
		layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(18), initial("?"))
			lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			lbl.Font.Weight = font.SemiBold
			return lbl.Layout(gtx)
		})
	}

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
func stpStatusRow(gtx layout.Context, th *material.Theme, t Theme, it stpItem, avFn cpAvatarFn) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return stpRingAvatar(gtx, th, t, it, avFn)
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
func stpRingAvatar(gtx layout.Context, th *material.Theme, t Theme, it stpItem, avFn cpAvatarFn) layout.Dimensions {
	av := gtx.Dp(48)
	ringW := gtx.Dp(unit.Dp(2.5)) // ketebalan cincin (.ring padding 2.5px)
	pad := ringW
	full := av + pad*2
	sz := image.Pt(full, full)

	// avatar di tengah cincin: foto asli (avFn) bila ada, else inisial berwarna.
	stpInnerAvatar(gtx, th, t, it.name, it.jid, av, pad, avFn)
	// cincin TERSEGMEN (ala WhatsApp): N busur sama panjang dgn celah; segmen
	// dilihat = abu (t.Line), belum = accent. count<=1 → cincin utuh.
	n := it.count
	if n < 1 {
		n = 1
	}
	stpStatusRing(gtx, full, av, pad, n, it.seenCount, t.Accent, withAlpha(t.Text2, 0x88))
	return layout.Dimensions{Size: sz}
}

// stpStatusRing — gambar cincin status sbg N busur (stroke) di tepi luar: segmen
// indeks < seen = abu, sisanya accent. n==1 → lingkaran penuh tanpa celah.
func stpStatusRing(gtx layout.Context, full, av, pad, n, seen int, accent, gray color.NRGBA) {
	c := f32.Pt(float32(full)/2, float32(full)/2)
	r := float32(av+pad) / 2 // garis-tengah stroke (mengisi anulus av..full)
	strokeW := float32(pad)
	if strokeW < float32(gtx.Dp(2)) {
		strokeW = float32(gtx.Dp(2))
	}
	gap := float32(0) // celah antar-segmen (rad)
	if n > 1 {
		gap = 0.20
	}
	seg := float32(2*math.Pi) / float32(n)
	start := float32(-math.Pi / 2) // mulai dari atas
	for i := 0; i < n; i++ {
		a0 := start + float32(i)*seg + gap/2
		sweep := seg - gap
		col := accent
		if i < seen {
			col = gray
		}
		p0 := f32.Pt(c.X+r*float32(math.Cos(float64(a0))), c.Y+r*float32(math.Sin(float64(a0))))
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(p0)
		foc := f32.Pt(c.X-p0.X, c.Y-p0.Y) // pusat lingkaran relatif thd pena
		p.Arc(foc, foc, sweep)
		paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: strokeW}.Op())
	}
}

// stpInnerAvatar — gambar avatar bulat (foto asli via avFn / fallback inisial)
// ukuran avPx, di-offset (pad,pad) supaya pas di dalam cincin/badge.
func stpInnerAvatar(gtx layout.Context, th *material.Theme, t Theme, name, jid string, avPx, pad int, avFn cpAvatarFn) {
	if avFn != nil {
		off := op.Offset(image.Pt(pad, pad)).Push(gtx.Ops)
		cg := gtx
		cg.Constraints.Min, cg.Constraints.Max = image.Pt(avPx, avPx), image.Pt(avPx, avPx)
		avFn(cg, name, jid, 48) // u.avatar: foto profil (mask bulat) / inisial
		off.Pop()
		return
	}
	// fallback (standalone render): lingkaran warna + inisial putih.
	paint.FillShape(gtx.Ops, avatarColor(name), clip.Ellipse{Min: image.Pt(pad, pad), Max: image.Pt(pad+avPx, pad+avPx)}.Op(gtx.Ops))
	off := op.Offset(image.Pt(pad, pad)).Push(gtx.Ops)
	cg := gtx
	cg.Constraints.Min, cg.Constraints.Max = image.Pt(avPx, avPx), image.Pt(avPx, avPx)
	layout.Center.Layout(cg, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, unit.Sp(18), initial(name))
		lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		lbl.Font.Weight = font.SemiBold
		return lbl.Layout(gtx)
	})
	off.Pop()
}
