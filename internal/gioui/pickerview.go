// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// pickerview.go — popup pemilih stiker/GIF (paritas StickerPicker.svelte + app.css):
// kartu .stk-panel (Bg, radius 14, border Line, lebar maks 520) berisi baris tab
// (.stk-tabs), pil pencarian (.stk-search), baris chip kategori (.gif-cats), grid
// sel placeholder (.stk-grid/.stk-cell), lalu kredit "Powered by Sticker.ly".
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

// PkItem = satu stiker tersimpan: thumbnail (di-decode in-process) + ada/tidak.
type PkItem struct {
	Thumb paint.ImageOp
	Has   bool
}

// PkCtl = state interaktif picker stiker. nil → grid placeholder demo. Items +
// Clicks paralel (tap → kirim stiker).
type PkCtl struct {
	Items     []PkItem
	Clicks    []widget.Clickable
	Tab       int                // 0 = Stiker, 1 = GIF
	TabClicks []widget.Clickable // paralel [Stiker, GIF]
	Empty     string             // pesan saat hasil kosong
	SearchEd  *widget.Editor     // input cari online (nil = tanpa)
}

// PickerView menggambar kartu pemilih stiker sbg POPUP di atas composer (kiri-bawah),
// bukan layar penuh — latar app tetap terlihat (scrim redup). ala WhatsApp Desktop.
func PickerView(gtx layout.Context, th *material.Theme, t Theme, ctl *PkCtl) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 80}, clip.Rect{Max: gtx.Constraints.Max}.Op()) // scrim, bukan opaque
	// jangkar kiri-bawah area chat (di atas composer): kiri = lebar rail+sidebar.
	left := gtx.Dp(540)
	if left > gtx.Constraints.Max.X-gtx.Dp(280) {
		left = gtx.Dp(12) // jendela sempit → mepet kiri
	}
	return layout.Inset{Left: unit.Dp(0), Bottom: unit.Dp(72), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.SW.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: pxToDp(gtx, left)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return pkCard(gtx, th, t, ctl)
			})
		})
	})
}

// pxToDp — konversi px → unit.Dp (utk inset dinamis).
func pxToDp(gtx layout.Context, px int) unit.Dp { return unit.Dp(float32(px) / gtx.Metric.PxPerDp) }

// pkCard — .stk-panel: Bg, radius 14, border 1px Line, lebar 520, padding 10, kolom isi.
func pkCard(gtx layout.Context, th *material.Theme, t Theme, ctl *PkCtl) layout.Dimensions {
	w := gtx.Dp(520)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w

	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// tab Stiker | GIF.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return pkTabs(gtx, th, t, ctl)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			// kotak cari online (Enter → cari).
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if ctl == nil || ctl.SearchEd == nil {
					return layout.Dimensions{}
				}
				macro := op.Record(gtx.Ops)
				dims := layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, "search", 16, t.Text2) }),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							hint := "Cari GIF"
							if ctl.Tab == 0 {
								hint = "Cari stiker"
							}
							e := material.Editor(th, ctl.SearchEd, hint)
							e.Color, e.HintColor, e.TextSize = t.Text, t.Text2, unit.Sp(14)
							return e.Layout(gtx)
						}),
					)
				})
				call := macro.Stop()
				r := gtx.Dp(9)
				paint.FillShape(gtx.Ops, t.Bg2, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
				call.Add(gtx.Ops)
				return dims
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			// grid stiker/GIF (thumbnail), atau pesan kosong.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if ctl != nil && len(ctl.Items) == 0 {
					msg := ctl.Empty
					if msg == "" {
						msg = "Belum ada — simpan stiker/GIF dari chat dulu."
					}
					return layout.Inset{Top: unit.Dp(40), Bottom: unit.Dp(40)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, 14, msg)
							lbl.Color, lbl.MaxLines = t.Text2, 2
							return lbl.Layout(gtx)
						})
					})
				}
				return pkGrid(gtx, t, ctl)
			}),
		)
	})
	call := macro.Stop()
	// latar kartu + border: gambar Line penuh lalu Bg inset 1px (paritas border .stk-panel).
	r := gtx.Dp(14)
	bw := gtx.Dp(1)
	full := image.Rectangle{Max: dims.Size}
	paint.FillShape(gtx.Ops, t.Line, clip.RRect{Rect: full, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	inner := image.Rectangle{Min: image.Pt(bw, bw), Max: image.Pt(dims.Size.X-bw, dims.Size.Y-bw)}
	paint.FillShape(gtx.Ops, t.Bg, clip.RRect{Rect: inner, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// pkTab = satu tab di baris tab (label + aktif?).
type pkTab struct {
	label  string
	active bool
}

// pkTabs — .stk-tabs: 4 tombol flex sama lebar, gap 6, padding 8, radius 9.
// aktif = Accent + putih; lainnya = Bg2 + text2/600.
func pkTabs(gtx layout.Context, th *material.Theme, t Theme, ctl *PkCtl) layout.Dimensions {
	labels := []string{"Stiker", "GIF"}
	active := 0
	if ctl != nil {
		active = ctl.Tab
	}
	children := make([]layout.FlexChild, 0, len(labels)*2)
	for i, lb := range labels {
		if i > 0 {
			children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout))
		}
		i, lb := i, lb
		children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			btn := func(gtx layout.Context) layout.Dimensions {
				return pkTabBtn(gtx, th, t, pkTab{label: lb, active: i == active})
			}
			if ctl != nil && i < len(ctl.TabClicks) {
				return ctl.TabClicks[i].Layout(gtx, btn)
			}
			return btn(gtx)
		}))
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
}

// pkTabBtn — satu tab: latar (Accent jika aktif, else Bg2), radius 9, padding 8, label tengah.
func pkTabBtn(gtx layout.Context, th *material.Theme, t Theme, tb pkTab) layout.Dimensions {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	bg := t.Bg2
	fg := t.Text2
	if tb.active {
		bg = t.Accent
		fg = white
	}
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 13, tb.label)
			lbl.Color = fg
			lbl.MaxLines = 1
			lbl.Font.Weight = font.SemiBold
			return lbl.Layout(gtx)
		})
	})
	call := macro.Stop()
	r := gtx.Dp(9)
	paint.FillShape(gtx.Ops, bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// pkSearch — .stk-search: pil Bg2 (background:var(--bg2)), radius 9, padding 8/12, teks placeholder text2.
func pkSearch(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		lbl := material.Label(th, 14.5, "Cari pesan stiker")
		lbl.Color = t.Text2
		lbl.MaxLines = 1
		return lbl.Layout(gtx)
	})
	call := macro.Stop()
	r := gtx.Dp(9)
	paint.FillShape(gtx.Ops, t.Bg2, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// pkCat = satu chip kategori (glyph/teks + aktif?).
type pkCat struct {
	label  string
	active bool
}

// pkCats — .gif-cats: baris chip kategori (gap 5). Aktif = Accent, else Bg2.
func pkCats(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	cats := []pkCat{
		{label: "🔥 trending", active: true},
		{label: "❤️ love"},
		{label: "😂 lol"},
		{label: "😮 wow"},
	}
	children := make([]layout.FlexChild, 0, len(cats)*2)
	for i, c := range cats {
		if i > 0 {
			children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout))
		}
		c := c
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return pkCatChip(gtx, th, t, c)
		}))
	}
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
}

// pkCatChip — .gif-cat: pil radius 12, padding 4/10, latar Bg2 (Accent jika aktif), teks 12/400.
func pkCatChip(gtx layout.Context, th *material.Theme, t Theme, c pkCat) layout.Dimensions {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	bg := t.Bg2
	fg := t.Text2
	if c.active {
		bg = t.Accent
		fg = white
	}
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, 12, c.label)
		lbl.Color = fg
		lbl.MaxLines = 1
		lbl.Font.Weight = font.Normal
		return lbl.Layout(gtx)
	})
	call := macro.Stop()
	r := gtx.Dp(12)
	paint.FillShape(gtx.Ops, bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// pkGrid — .stk-grid: grid auto-fill minmax(84px,1fr), gap 6. Render stiker nyata
// (ctl.Items, tappable) atau 2 baris placeholder (ctl nil = demo).
func pkGrid(gtx layout.Context, t Theme, ctl *PkCtl) layout.Dimensions {
	gap := gtx.Dp(6)
	avail := gtx.Constraints.Max.X
	minCell := gtx.Dp(84)
	cols := (avail + gap) / (minCell + gap)
	if cols < 1 {
		cols = 1
	}
	cell := (avail - (cols-1)*gap) / cols

	n := 2 * cols // placeholder: 2 baris
	if ctl != nil {
		n = len(ctl.Items)
	}
	rows := (n + cols - 1) / cols
	if rows < 1 && ctl == nil {
		rows = 2
	}

	rowChildren := make([]layout.FlexChild, 0, rows*2)
	for ri := 0; ri < rows; ri++ {
		ri := ri
		if ri > 0 {
			rowChildren = append(rowChildren, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
		}
		rowChildren = append(rowChildren, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			cellChildren := make([]layout.FlexChild, 0, cols*2)
			for ci := 0; ci < cols; ci++ {
				idx := ri*cols + ci
				if ci > 0 {
					cellChildren = append(cellChildren, layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout))
				}
				cellChildren = append(cellChildren, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if ctl == nil {
						return pkCell(gtx, t, cell, PkItem{})
					}
					if idx >= len(ctl.Items) {
						return layout.Dimensions{Size: image.Pt(cell, cell)} // sel kosong
					}
					body := func(gtx layout.Context) layout.Dimensions { return pkCell(gtx, t, cell, ctl.Items[idx]) }
					if idx < len(ctl.Clicks) {
						return ctl.Clicks[idx].Layout(gtx, body)
					}
					return body(gtx)
				}))
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, cellChildren...)
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rowChildren...)
}

// pkCell — .stk-cell: kotak persegi Bg2 radius 10; gambar thumbnail bila ada.
func pkCell(gtx layout.Context, t Theme, side int, item PkItem) layout.Dimensions {
	sz := image.Pt(side, side)
	r := gtx.Dp(10)
	paint.FillShape(gtx.Ops, t.Bg2, clip.RRect{Rect: image.Rectangle{Max: sz}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	if item.Has {
		cl := clip.RRect{Rect: image.Rectangle{Max: sz}, NW: r, NE: r, SE: r, SW: r}.Push(gtx.Ops)
		drawImageFill(gtx.Ops, item.Thumb, side)
		cl.Pop()
	}
	return layout.Dimensions{Size: sz}
}
