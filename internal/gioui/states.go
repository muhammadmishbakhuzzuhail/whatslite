// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// states.go — DEMO standalone "chrome" percakapan: day-chip + unread-divider
// pill di atas, lalu splash percakapan kosong (.conv-splash) di tengah. Nilai px
// /warna persis dari frontend/src/styles/app.css. Fungsi murni, data dibakar.
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

// StatesView mengisi seluruh area gtx dengan latar wallpaper, lalu menumpuk
// pill day-chip + unread-divider di atas dan splash kosong di tengah.
func StatesView(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	drawWallpaper(gtx, t)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// .day-chip span — margin 8px 0
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return statesPill(gtx, th, t, "HARI INI", 13, unit.Dp(6), unit.Dp(12))
				})
			})
		}),
		// .unread-divider span — margin 6px 0
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return statesPill(gtx, th, t, "2 PESAN BELUM DIBACA", 12, unit.Dp(5), unit.Dp(14))
				})
			})
		}),
		// .conv-splash — flex:1, terpusat
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return statesSplash(gtx, th, t)
			})
		}),
	)
}

// statesPill — pill berlatar in-bg, radius 8, teks text2. (day-chip/unread).
func statesPill(gtx layout.Context, th *material.Theme, t Theme, txt string, sp unit.Sp, padV, padH unit.Dp) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: padV, Bottom: padV, Left: padH, Right: padH}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, sp, txt)
				lbl.Color = t.Text2
				lbl.Font.Weight = font.Medium
				lbl.MaxLines = 1
				return lbl.Layout(gtx)
			})
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			// digambar setelah ukuran konten diketahui via Stack: latar di belakang.
			r := gtx.Dp(8)
			sz := gtx.Constraints.Min
			paint.FillShape(gtx.Ops, t.InBg, clip.RRect{Rect: image.Rectangle{Max: sz}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
			return layout.Dimensions{Size: sz}
		}),
	)
}

// statesSplash — .conv-splash: lingkaran head-bg 200px + bubble 96px (text2 @.45),
// heading 28px/Light, subtitle 14px text2, baris lock + "Terenkripsi end-to-end".
func statesSplash(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(24), Bottom: unit.Dp(24), Left: unit.Dp(24), Right: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			// .splash-logo (200px lingkaran head-bg) + bubble 96px di tengah
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return statesLogo(gtx, t)
				})
			}),
			// h2 — 28px font-weight 300, margin-bottom 8px
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, 28, "WhatsLite untuk Desktop")
					lbl.Color = t.Text
					lbl.Font.Weight = font.Light
					lbl.MaxLines = 1
					lbl.Alignment = 2 // text.Middle
					return lbl.Layout(gtx)
				})
			}),
			// p — 14px text2
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 14, "Kirim dan terima pesan tanpa membuka ponsel.")
				lbl.Color = t.Text2
				lbl.Alignment = 2 // text.Middle
				return lbl.Layout(gtx)
			}),
			// .splash-enc — margin-top 34px, gap 6px, 13px text2
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(34)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return statesLock(gtx, t.Text2)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, 13, "Terenkripsi end-to-end")
							lbl.Color = t.Text2
							lbl.MaxLines = 1
							return lbl.Layout(gtx)
						}),
					)
				})
			}),
		)
	})
}

// statesLogo — lingkaran 200px head-bg dengan glyph chat WhatsApp 92px di tengah
// (ikon SVG "chats", text2 @ .5) — lebih rapi dari bentuk gambar-tangan lama.
func statesLogo(gtx layout.Context, t Theme) layout.Dimensions {
	d := gtx.Dp(200)
	sz := image.Pt(d, d)
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Ellipse{Max: sz}.Op(gtx.Ops))

	ic := t.Text2
	ic.A = uint8(float32(ic.A) * 0.5)

	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return icon(gtx, "chats", 92, ic)
	})
	return layout.Dimensions{Size: sz}
}

// statesLock — gembok kecil 14px (badan kotak + gagang) untuk baris enkripsi.
func statesLock(gtx layout.Context, col color.NRGBA) layout.Dimensions {
	w := gtx.Dp(14)
	sz := image.Pt(w, w)
	// badan kunci
	bodyTop := gtx.Dp(6)
	body := image.Rectangle{Min: image.Pt(0, bodyTop), Max: image.Pt(w, w)}
	rr := gtx.Dp(2)
	paint.FillShape(gtx.Ops, col, clip.RRect{Rect: body, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
	// gagang (shackle): cincin tipis di atas badan via dua RRect (outer - inner).
	shW := gtx.Dp(8)
	shH := gtx.Dp(6)
	shX := (w - shW) / 2
	outer := image.Rectangle{Min: image.Pt(shX, 0), Max: image.Pt(shX+shW, bodyTop)}
	or := gtx.Dp(4)
	paint.FillShape(gtx.Ops, col, clip.RRect{Rect: outer, NW: or, NE: or}.Op(gtx.Ops))
	// lubang gagang: timpa dengan transparan tak mungkin (tanpa alpha-cut), maka
	// cukup gambar potongan dalam pakai warna latar — abaikan, siluet padat cukup.
	_ = shH
	return layout.Dimensions{Size: sz}
}
