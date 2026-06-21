// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// bubbletypes.go — galeri tipe bubble non-teks (paritas Bubble.svelte + app.css):
// document, voice, location, contact, poll. Tiap sampel sebagai bubble masuk
// (in-bg, RRect 18 + ekor 6px kiri-atas, padding 8/13). Data demo inline.
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

// BubbleTypesView menggambar kolom vertikal berisi satu sampel tiap tipe bubble
// non-teks, di atas latar wallpaper. Fungsi murni, mandiri (standalone render).
func BubbleTypesView(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	paint.FillShape(gtx.Ops, t.Wallpaper, clip.Rect{Max: gtx.Constraints.Max}.Op())

	gap := layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout)

	return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return btInBubble(gtx, th, t, btDocument)
			}),
			gap,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return btInBubble(gtx, th, t, btVoice)
			}),
			gap,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return btInBubble(gtx, th, t, btLocation)
			}),
			gap,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return btInBubble(gtx, th, t, btContact)
			}),
			gap,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return btInBubble(gtx, th, t, btPoll)
			}),
		)
	})
}

// btInBubble membungkus konten dalam bubble masuk: latar t.InBg, RRect radius 18
// dengan ekor 6px kiri-atas, padding 8/13, rata kiri (W).
func btInBubble(gtx layout.Context, th *material.Theme, t Theme, body func(layout.Context, *material.Theme, Theme) layout.Dimensions) layout.Dimensions {
	return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		content := func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(13), Right: unit.Dp(13)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return body(gtx, th, t)
			})
		}
		macro := op.Record(gtx.Ops)
		dims := content(gtx)
		call := macro.Stop()
		r := gtx.Dp(18)
		tl := gtx.Dp(6) // ekor kiri-atas utk bubble masuk
		paint.FillShape(gtx.Ops, t.InBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: tl, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// ---- (1) DOCUMENT (.doc-card: ikon 40 accent-tint rounded + nama + meta) ----
func btDocument(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return btDocIcon(gtx, t)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(11)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, 14, "Laporan_Tahunan_2026.pdf")
					lbl.Color = t.Text
					lbl.MaxLines = 1
					return lbl.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(3)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, 12, "PDF · 512 KB · 12 hal")
					lbl.Color = t.Text2
					lbl.MaxLines = 1
					return lbl.Layout(gtx)
				}),
			)
		}),
	)
}

// btDocIcon: kotak 40x40, radius 9, latar tint accent ~16% (color-mix accent/transparan).
func btDocIcon(gtx layout.Context, t Theme) layout.Dimensions {
	d := gtx.Dp(40)
	sz := image.Pt(d, d)
	r := gtx.Dp(9)
	tint := color.NRGBA{R: t.Accent.R, G: t.Accent.G, B: t.Accent.B, A: 41} // 16% ≈ 41/255
	paint.FillShape(gtx.Ops, tint, clip.RRect{Rect: image.Rectangle{Max: sz}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	// ikon dokumen sederhana: lembaran accent dgn sudut terlipat, 22x22 di tengah.
	iw := gtx.Dp(15)
	ih := gtx.Dp(20)
	ox := (d - iw) / 2
	oy := (d - ih) / 2
	paint.FillShape(gtx.Ops, t.Accent, clip.Rect{Min: image.Pt(ox, oy), Max: image.Pt(ox+iw, oy+ih)}.Op())
	// sudut terlipat: potong segitiga kanan-atas dgn warna kotak (tint) — tiru lipatan.
	fold := gtx.Dp(6)
	paint.FillShape(gtx.Ops, tint, clip.Rect{Min: image.Pt(ox+iw-fold, oy), Max: image.Pt(ox+iw, oy+fold)}.Op())
	return layout.Dimensions{Size: sz}
}

// ---- (2) VOICE (.bubble.voice min-width 230: play 34 + wave 22 bar + 0:12) ----
func btVoice(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	minW := gtx.Dp(230)
	if gtx.Constraints.Min.X < minW {
		gtx.Constraints.Min.X = minW
	}
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return btPlayCircle(gtx, t)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(9)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return btWave(gtx, t)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(9)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 12, "0:12")
			lbl.Color = t.Text2
			return lbl.Layout(gtx)
		}),
	)
}

// btPlayCircle: tombol play 34 (.play) — segitiga text2 di dalam lingkaran transparan.
func btPlayCircle(gtx layout.Context, t Theme) layout.Dimensions {
	d := gtx.Dp(34)
	sz := image.Pt(d, d)
	// glyph play: segitiga kasar dari beberapa baris horizontal (clip.Rect) supaya
	// tetap pakai API yg ada di ui.go. Warna text2.
	cx := gtx.Dp(13)
	cy := d / 2
	hh := gtx.Dp(7) // setengah tinggi segitiga
	for i := 0; i < hh*2; i++ {
		y := cy - hh + i
		// lebar mengecil dari pangkal (atas+bawah) ke ujung (tengah): bentuk ▶
		dist := i
		if i > hh {
			dist = hh*2 - i
		}
		w := dist * gtx.Dp(14) / hh
		if w <= 0 {
			continue
		}
		paint.FillShape(gtx.Ops, t.Text2, clip.Rect{Min: image.Pt(cx, y), Max: image.Pt(cx+w, y+1)}.Op())
	}
	return layout.Dimensions{Size: sz}
}

// btWave: 22 bar (.wave span) — rect tipis rounded, tinggi berselang, text2 @ .55.
func btWave(gtx layout.Context, t Theme) layout.Dimensions {
	h := gtx.Dp(26)
	barCol := color.NRGBA{R: t.Text2.R, G: t.Text2.G, B: t.Text2.B, A: 140} // ~.55 opacity
	bw := gtx.Dp(3)
	g := gtx.Dp(3)
	n := 22
	total := n*bw + (n-1)*g
	if total > gtx.Constraints.Max.X {
		total = gtx.Constraints.Max.X
	}
	rr := gtx.Dp(2)
	x := 0
	for i := 0; i < n; i++ {
		// pola tinggi: ganjil 60%, genap 95%, kelipatan 3 → 40% (paritas nth-child).
		frac := 60
		if (i+1)%2 == 0 {
			frac = 95
		}
		if (i+1)%3 == 0 {
			frac = 40
		}
		bh := h * frac / 100
		oy := (h - bh) / 2
		rect := image.Rectangle{Min: image.Pt(x, oy), Max: image.Pt(x+bw, oy+bh)}
		paint.FillShape(gtx.Ops, barCol, clip.RRect{Rect: rect, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		x += bw + g
	}
	return layout.Dimensions{Size: image.Pt(total, h)}
}

// ---- (3) LOCATION (.loc-card 280: map 140 high + pin, lalu baris label) ----
func btLocation(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	w := gtx.Dp(280)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	r := gtx.Dp(12)
	mapH := gtx.Dp(140)

	macro := op.Record(gtx.Ops)
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			// area peta: Wallpaper, tinggi 140, pin accent di tengah.
			sz := image.Pt(w, mapH)
			paint.FillShape(gtx.Ops, t.Wallpaper, clip.Rect{Max: sz}.Op())
			// pin: lingkaran accent + titik di tengah peta.
			pd := gtx.Dp(14)
			px := (w - pd) / 2
			py := (mapH - pd) / 2
			paint.FillShape(gtx.Ops, t.Accent, clip.Ellipse{Min: image.Pt(px, py), Max: image.Pt(px+pd, py+pd)}.Op(gtx.Ops))
			return layout.Dimensions{Size: sz}
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			// baris label .loc-lbl: latar Bg2, padding 8/10, ikon accent + teks.
			return btLocLabel(gtx, th, t, w)
		}),
	)
	call := macro.Stop()
	// latar kartu membulat (.loc-card border-radius 12) digambar di belakang isi;
	// sudut isi sendiri persegi (clip non-trivial dihindari demi API yg dipakai ui.go).
	paint.FillShape(gtx.Ops, t.Bg2, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

func btLocLabel(gtx layout.Context, th *material.Theme, t Theme, w int) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = w - gtx.Dp(20)
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				// ikon pin kecil 18: lingkaran accent.
				d := gtx.Dp(12)
				oy := gtx.Dp(3)
				paint.FillShape(gtx.Ops, t.Accent, clip.Ellipse{Min: image.Pt(0, oy), Max: image.Pt(d, oy+d)}.Op(gtx.Ops))
				return layout.Dimensions{Size: image.Pt(gtx.Dp(18), gtx.Dp(18))}
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 13, "Jl. Sudirman No. 12, Jakarta")
				lbl.Color = t.Text
				lbl.MaxLines = 1
				return lbl.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	paint.FillShape(gtx.Ops, t.Bg2, clip.Rect{Max: dims.Size}.Op())
	call.Add(gtx.Ops)
	return dims
}

// ---- (4) CONTACT (.ctc-card: avatar 40 + nama + nomor + "Simpan") ----
func btContact(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	minW := gtx.Dp(200)
	if gtx.Constraints.Min.X < minW {
		gtx.Constraints.Min.X = minW
	}
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			// .ctc-av: lingkaran 40 accent + inisial putih.
			d := gtx.Dp(40)
			sz := image.Pt(d, d)
			paint.FillShape(gtx.Ops, t.Accent, clip.Ellipse{Max: sz}.Op(gtx.Ops))
			gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
			layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 16, initial("Dewi Anggraini"))
				lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
				lbl.Font.Weight = font.SemiBold
				return lbl.Layout(gtx)
			})
			return layout.Dimensions{Size: sz}
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(11)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, 14, "Dewi Anggraini")
					lbl.Color = t.Text
					lbl.MaxLines = 1
					lbl.Font.Weight = font.SemiBold
					return lbl.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, 12, "+62 812-3456-7890")
					lbl.Color = t.Text2
					lbl.MaxLines = 1
					return lbl.Layout(gtx)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 13, "Simpan")
			lbl.Color = t.Accent
			lbl.Font.Weight = font.SemiBold
			return lbl.Layout(gtx)
		}),
	)
}

// ---- (5) POLL (.poll-card: pertanyaan + 3 opsi bordered + radio) ----
func btPoll(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	minW := gtx.Dp(230)
	if gtx.Constraints.Min.X < minW {
		gtx.Constraints.Min.X = minW
	}
	opts := []string{"Pantai", "Gunung", "Kota"}
	rowGap := layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 14, "Liburan ke mana minggu depan?")
			lbl.Color = t.Text
			lbl.Font.Weight = font.SemiBold
			return lbl.Layout(gtx)
		}),
		rowGap,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return btPollOpt(gtx, th, t, opts[0]) }),
		rowGap,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return btPollOpt(gtx, th, t, opts[1]) }),
		rowGap,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return btPollOpt(gtx, th, t, opts[2]) }),
	)
}

// btPollOpt: .poll-opt — rect membulat radius 10, border Line, latar Bg, radio + teks.
func btPollOpt(gtx layout.Context, th *material.Theme, t Theme, txt string) layout.Dimensions {
	r := gtx.Dp(10)
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(11), Right: unit.Dp(11)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return btRadio(gtx, t)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(9)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 13, txt)
				lbl.Color = t.Text
				lbl.MaxLines = 1
				return lbl.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	// latar Bg + border Line (border = isi 1px frame: gambar Line penuh lalu Bg inset).
	bw := gtx.Dp(1)
	full := image.Rectangle{Max: dims.Size}
	paint.FillShape(gtx.Ops, t.Line, clip.RRect{Rect: full, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	inner := image.Rectangle{Min: image.Pt(bw, bw), Max: image.Pt(dims.Size.X-bw, dims.Size.Y-bw)}
	paint.FillShape(gtx.Ops, t.Bg, clip.RRect{Rect: inner, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// btRadio: .poll-radio 16x16 lingkaran, border 2px text2 (cincin via 2 ellipse).
func btRadio(gtx layout.Context, t Theme) layout.Dimensions {
	d := gtx.Dp(16)
	bw := gtx.Dp(2)
	sz := image.Pt(d, d)
	paint.FillShape(gtx.Ops, t.Text2, clip.Ellipse{Max: sz}.Op(gtx.Ops))
	inner := image.Rectangle{Min: image.Pt(bw, bw), Max: image.Pt(d-bw, d-bw)}
	paint.FillShape(gtx.Ops, t.InBg, clip.Ellipse{Min: inner.Min, Max: inner.Max}.Op(gtx.Ops))
	return layout.Dimensions{Size: sz}
}
