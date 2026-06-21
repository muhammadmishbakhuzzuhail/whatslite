// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// bubbleextrasview.go — galeri "extras" pada bubble (paritas Bubble.svelte +
// app.css): (1) bubble masuk dgn blok KUTIPAN balasan (.quote: bar accent kiri +
// nama accent 13 + teks 13 text2 di area quote-bg membulat) lalu teks balasan;
// (2) bubble keluar dgn CHIP REAKSI di bawah (.reaction: pil out-bg border divider
// r-12, emoji+jumlah) rata kanan; (3) bubble keluar dgn centang BACA ganda (dua centang
// accent/tick) di samping jam; (4) bubble masuk dgn MENTION (@Budi accent) inline.
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

// bxReaction = satu chip reaksi (.reaction): emoji + jumlah.
type bxReaction struct {
	emoji string
	count int
}

// BubbleExtrasView menggambar kolom vertikal bubble teks masuk/keluar yg
// memperagakan extras (kutipan, reaksi, centang baca, mention) di atas wallpaper.
// Fungsi murni, mandiri (standalone render).
func BubbleExtrasView(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	paint.FillShape(gtx.Ops, t.Wallpaper, clip.Rect{Max: gtx.Constraints.Max}.Op())

	gap := layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout)

	return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// (1) masuk + kutipan balasan di atas teks.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return bxQuoteBubble(gtx, th, t)
			}),
			gap,
			// (2) keluar + chip reaksi di bawah, rata kanan.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return bxReactionBubble(gtx, th, t)
			}),
			gap,
			// (3) keluar + centang baca ganda di samping jam.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return bxReadBubble(gtx, th, t)
			}),
			gap,
			// (4) masuk + mention @Budi accent inline.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return bxMentionBubble(gtx, th, t)
			}),
		)
	})
}

// bxQuoteBg / bxQuoteBar — token --quote-bg / --quote-bar (app.css). Diturunkan
// dari accent (quote-bar ≈ accent; quote-bg = accent ~9-12% opacity).
func bxQuoteBar(t Theme) color.NRGBA { return t.Accent }
func bxQuoteBg(t Theme) color.NRGBA {
	a := uint8(31) // ~12%
	if !t.Dark {
		a = 23 // ~9%
	}
	return color.NRGBA{R: t.Accent.R, G: t.Accent.G, B: t.Accent.B, A: a}
}

// bxBubble — bungkus konten dlm bubble: latar in/out, RRect 18 + ekor 6px (kiri-
// atas utk masuk, kanan-atas utk keluar), padding 8/13, rata kiri/kanan. Sama
// dengan u.bubble di ui.go.
func bxBubble(gtx layout.Context, t Theme, out bool, body func(gtx layout.Context) layout.Dimensions) layout.Dimensions {
	bg := t.InBg
	align := layout.W
	if out {
		bg = t.OutBg
		align = layout.E
	}
	maxW := gtx.Constraints.Max.X * 66 / 100
	return align.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		content := func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = maxW
			return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(13), Right: unit.Dp(13)}.Layout(gtx, body)
		}
		macro := op.Record(gtx.Ops)
		dims := content(gtx)
		call := macro.Stop()
		r := gtx.Dp(18)
		tl, tr := r, r
		if out {
			tr = gtx.Dp(6)
		} else {
			tl = gtx.Dp(6)
		}
		paint.FillShape(gtx.Ops, bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: tl, NE: tr, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// bxMeta — baris jam (.meta) rata kanan: jam 11 text2, opsional centang baca.
func bxMeta(gtx layout.Context, th *material.Theme, t Theme, time string, ticks bool) layout.Dimensions {
	return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 11, time)
				lbl.Color = t.Text2
				return lbl.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !ticks {
					return layout.Dimensions{}
				}
				return layout.Inset{Left: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return bxTicks(gtx, t)
				})
			}),
		)
	})
}

// ---- (1) KUTIPAN balasan (.quote: bar 4px quote-bar + nama + teks, di quote-bg) ----
func bxQuoteBubble(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	return bxBubble(gtx, t, false, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// blok kutipan.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return bxQuoteBlock(gtx, th, t, "Andi Pratama", "Jadi nanti malam jadi ngumpul kan?")
			}),
			// .quote { margin-bottom: 5px }
			layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
			// teks balasan.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 14.5, "Jadi dong! Jam 8 di tempat biasa ya 👍")
				lbl.Color = t.Text
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return bxMeta(gtx, th, t, "19.05", false)
			}),
		)
	})
}

// bxQuoteBlock — .quote: border-left 4px quote-bar, latar quote-bg, radius 4,
// padding 5/9. Isi: .quote-name (13/600 quote-bar) + .quote-text (13 text2).
func bxQuoteBlock(gtx layout.Context, th *material.Theme, t Theme, name, text string) layout.Dimensions {
	bar := gtx.Dp(4) // border-left: 4px
	r := gtx.Dp(4)   // border-radius: 4px
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(9), Right: unit.Dp(9)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 13, name)
				lbl.Color = bxQuoteBar(t)
				lbl.MaxLines = 1
				lbl.Font.Weight = font.SemiBold
				return lbl.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 13, text)
				lbl.Color = t.Text2
				lbl.MaxLines = 1
				return lbl.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	// latar quote-bg membulat di belakang isi, lalu bar accent di tepi kiri.
	paint.FillShape(gtx.Ops, bxQuoteBg(t), clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	paint.FillShape(gtx.Ops, bxQuoteBar(t), clip.Rect{Max: image.Pt(bar, dims.Size.Y)}.Op())
	call.Add(gtx.Ops)
	return dims
}

// ---- (2) REAKSI chip di bawah bubble keluar (.reaction pil, rata kanan) ----
func bxReactionBubble(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	reacts := []bxReaction{{emoji: "👍", count: 2}, {emoji: "❤️", count: 1}}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return bxBubble(gtx, t, true, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, 14.5, "Mantap! Sampai nanti 🙌")
						lbl.Color = t.Text
						return lbl.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return bxMeta(gtx, th, t, "19.06", false)
					}),
				)
			})
		}),
		// .reactions { gap: 3px } rata kanan (.msg.out .reactions { right: 8px }).
		layout.Rigid(layout.Spacer{Height: unit.Dp(3)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					children := make([]layout.FlexChild, 0, len(reacts)*2)
					for i, rc := range reacts {
						if i > 0 {
							children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(3)}.Layout)) // gap: 3px
						}
						r := rc
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return bxReactionChip(gtx, th, t, r)
						}))
					}
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
				})
			})
		}),
	)
}

// bxReactionChip — .reaction: latar out-bg (.msg.out .reaction → var(--out-bg)),
// border 1px var(--divider), radius 12, padding 1/6, font 12, gap 2: emoji + jumlah.
func bxReactionChip(gtx layout.Context, th *material.Theme, t Theme, rc bxReaction) layout.Dimensions {
	r := gtx.Dp(12) // border-radius: 12px
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 12, rc.emoji)
				lbl.Color = t.Text2 // .reaction { color: var(--text2) }
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout), // gap: 2px
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 12, itoa(rc.count))
				lbl.Color = t.Text2
				return lbl.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	// border = frame var(--divider) 1px lalu isi var(--out-bg) di-inset (chip pada
	// bubble keluar: .msg.out .reaction { background: var(--out-bg) }).
	bw := gtx.Dp(1)
	full := image.Rectangle{Max: dims.Size}
	paint.FillShape(gtx.Ops, t.Divider, clip.RRect{Rect: full, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	inner := image.Rectangle{Min: image.Pt(bw, bw), Max: image.Pt(dims.Size.X-bw, dims.Size.Y-bw)}
	paint.FillShape(gtx.Ops, t.OutBg, clip.RRect{Rect: inner, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// ---- (3) CENTANG BACA ganda di samping jam (.ticks.read → warna tick) ----
func bxReadBubble(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	return bxBubble(gtx, t, true, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 14.5, "Oke, aku sudah baca pesanmu ya")
				lbl.Color = t.Text
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
			// .meta: jam + centang baca ganda (tick).
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return bxMeta(gtx, th, t, "19.07", true)
			}),
		)
	})
}

// bxTicks — .ticks 16x12 read: dua centang (✓✓) warna --tick (t.Tick).
// Tiap centang = dua batang diagonal (stroke ~1.7) via deret kotak (API ui.go).
func bxTicks(gtx layout.Context, t Theme) layout.Dimensions {
	w := gtx.Dp(16)
	h := gtx.Dp(12)
	col := t.Tick // .ticks.read { stroke: var(--tick) }
	bxCheck(gtx, col, 0, h)
	bxCheck(gtx, col, gtx.Dp(5), h) // centang kedua geser kanan (tumpang ala ✓✓)
	return layout.Dimensions{Size: image.Pt(w, h)}
}

// bxCheck — satu centang ✓: batang pendek turun (kiri) + batang panjang naik
// (kanan), bertemu di titik bawah. Digambar via deret kotak sw×sw sepanjang
// diagonal — sama gaya spArrow di sidepanesview.go.
func bxCheck(gtx layout.Context, col color.NRGBA, ox, h int) {
	sw := gtx.Dp(2) // stroke ~1.7 → 2px
	short := gtx.Dp(3)
	long := gtx.Dp(6)
	// titik bawah (lembah) centang.
	vx := ox + gtx.Dp(3)
	vy := h - gtx.Dp(2)
	// kaki kiri: dari kiri-atas turun ke lembah.
	for i := 0; i <= short; i++ {
		x := vx - short + i
		y := vy - short + i
		paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(x, y), Max: image.Pt(x+sw, y+sw)}.Op())
	}
	// kaki kanan: dari lembah naik ke kanan-atas (lebih panjang).
	for i := 0; i <= long; i++ {
		x := vx + i
		y := vy - i
		paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(x, y), Max: image.Pt(x+sw, y+sw)}.Op())
	}
}

// ---- (4) MENTION @Budi accent inline (.mention { color: accent; weight 600 }) ----
func bxMentionBubble(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	return bxBubble(gtx, t, false, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// satu baris teks dgn potongan mention berwarna accent (inline label).
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, 14.5, "Tolong bantu cek file-nya ya ")
						lbl.Color = t.Text
						lbl.MaxLines = 1
						return lbl.Layout(gtx)
					}),
					// .mention: accent + weight 600.
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, 14.5, "@Budi")
						lbl.Color = t.Accent
						lbl.Font.Weight = font.SemiBold
						lbl.MaxLines = 1
						return lbl.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return bxMeta(gtx, th, t, "19.08", false)
			}),
		)
	})
}
