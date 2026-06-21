// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// sidepanesview.go — sidebar pane CALLS (paritas frontend/src/lib/sidebar/
// CallsPane.svelte + app.css): .pane-head 56px ("Panggilan" 19/SemiBold), lalu
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
func SidePanesView(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	w := gtx.Dp(380)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	calls := []spCall{
		{name: "Andi Pratama", time: "19.08", video: true, missed: true},
		{name: "Keluarga", time: "18.41", video: false, missed: false},
		{name: "Sarah", time: "17.55", video: false, missed: true},
		{name: "Tim Proyek X", time: "16.20", video: true, missed: false},
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

// spPaneHead — .pane-head { height: 56px; padding: 0 16px; background: head-bg }
// h2 19/SemiBold.
func spPaneHead(gtx layout.Context, th *material.Theme, t Theme, w int, title string) layout.Dimensions {
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

// spCallRow — .chat-row { padding: 8px 12px; gap: 13px } : avatar 49 + kolom
// (nama 16/Medium + sub-baris panah+label) + ikon panggil accent kanan.
func spCallRow(gtx layout.Context, th *material.Theme, t Theme, c spCall) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
								lbl := material.Label(th, 16, c.name)
								lbl.Color = t.Text
								lbl.MaxLines = 1
								lbl.Font.Weight = font.Medium
								return lbl.Layout(gtx)
							}),
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
		col = color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff} // #e35d6a
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

// spArrow — .call-ico 15x15 : panah diagonal (stroke 2). Masuk → panah turun-kiri
// (incoming), tak terjawab/keluar → panah naik-kanan. Digambar via batang +
// kepala panah (clip.Rect tipis), warna mengikuti garis (currentColor).
func spArrow(gtx layout.Context, col color.NRGBA, missed bool) layout.Dimensions {
	d := gtx.Dp(15)
	sz := image.Pt(d, d)
	sw := gtx.Dp(2) // stroke-width: 2
	pad := gtx.Dp(2)
	lo := pad
	hi := d - pad

	// batang diagonal: deret kotak sw x sw sepanjang diagonal.
	// incoming (masuk) = arah kiri-bawah (dari kanan-atas ke kiri-bawah).
	// missed/outgoing = arah kanan-atas (dari kiri-bawah ke kanan-atas).
	n := hi - lo
	if n < 1 {
		n = 1
	}
	for i := 0; i <= n; i++ {
		var x, y int
		if missed { // ke kanan-atas
			x = lo + i
			y = hi - i
		} else { // ke kiri-bawah
			x = hi - i
			y = hi - i
		}
		paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(x, y), Max: image.Pt(x+sw, y+sw)}.Op())
	}

	// kepala panah di ujung (dua batang pendek membentuk sudut).
	head := gtx.Dp(5)
	if missed { // ujung kanan-atas → kepala buka ke kiri & ke bawah
		tx, ty := hi, lo
		// batang horizontal (ke kiri)
		paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(tx-head, ty), Max: image.Pt(tx+sw, ty+sw)}.Op())
		// batang vertikal (ke bawah)
		paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(tx, ty), Max: image.Pt(tx+sw, ty+head)}.Op())
	} else { // ujung kiri-bawah → kepala buka ke kanan & ke atas
		tx, ty := lo, hi
		// batang horizontal (ke kanan)
		paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(tx, ty), Max: image.Pt(tx+head, ty+sw)}.Op())
		// batang vertikal (ke atas)
		paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(tx, ty-head), Max: image.Pt(tx+sw, ty+sw)}.Op())
	}
	return layout.Dimensions{Size: sz}
}

// spCallIcon — ikon panggil accent di kanan baris (.icon-btn ~40, glyph accent).
// video → kotak kamera; suara → gagang telepon. Disederhanakan dgn clip yg
// dipakai ui.go (Rect/RRect/Ellipse).
func spCallIcon(gtx layout.Context, t Theme, video bool) layout.Dimensions {
	box := gtx.Dp(40)
	sz := image.Pt(box, box)
	gw := gtx.Dp(20) // area glyph 20x20 di tengah
	ox := (box - gw) / 2
	oy := (box - gw) / 2

	if video {
		// kamera: badan rounded + corong segitiga kanan.
		bw := gtx.Dp(13)
		bh := gtx.Dp(13)
		bx := ox
		by := oy + (gw-bh)/2
		r := gtx.Dp(3)
		paint.FillShape(gtx.Ops, t.Accent, clip.RRect{Rect: image.Rectangle{Min: image.Pt(bx, by), Max: image.Pt(bx+bw, by+bh)}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		// corong: segitiga via deret kotak menyempit.
		lh := bh
		cx := bx + bw
		for i := 0; i < lh; i++ {
			dist := i
			if i > lh/2 {
				dist = lh - i
			}
			lw := dist * gtx.Dp(7) / (lh / 2)
			if lw <= 0 {
				continue
			}
			y := by + i
			paint.FillShape(gtx.Ops, t.Accent, clip.Rect{Min: image.Pt(cx, y), Max: image.Pt(cx+lw, y+1)}.Op())
		}
	} else {
		// telepon: gagang diagonal sederhana (dua bulatan + batang).
		ed := gtx.Dp(7)
		paint.FillShape(gtx.Ops, t.Accent, clip.Ellipse{Min: image.Pt(ox, oy), Max: image.Pt(ox+ed, oy+ed)}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, t.Accent, clip.Ellipse{Min: image.Pt(ox+gw-ed, oy+gw-ed), Max: image.Pt(ox+gw, oy+gw)}.Op(gtx.Ops))
		sw := gtx.Dp(3)
		lo := ox + ed/2
		hi := ox + gw - ed/2
		n := hi - lo
		if n < 1 {
			n = 1
		}
		for i := 0; i <= n; i++ {
			x := lo + i
			y := oy + ed/2 + i*(gw-ed)/n
			paint.FillShape(gtx.Ops, t.Accent, clip.Rect{Min: image.Pt(x, y), Max: image.Pt(x+sw, y+sw)}.Op())
		}
	}
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
