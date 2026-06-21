// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// reactionpickerview.go — popup pemilih reaksi emoji (paritas ReactionPicker.svelte
// + app.css .rp-pop): kartu t.Bg radius 14, ~352x400, border lebih terang utk kesan
// bayangan, diisi grid emoji warna (8 per baris, sel ~40px, terpusat). Fungsi murni,
// data demo inline (standalone render).
package gioui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// RpCtl = state interaktif pemilih reaksi. nil → grid statis (gio-shot). Clicks:
// 1 clickable per emoji (urut rpEmoji).
type RpCtl struct {
	Clicks []widget.Clickable
}

// RpEmoji mengekspos daftar emoji reaksi (utk pemetaan indeks→glyph di handler UI).
func RpEmoji() []string { return rpEmoji }

// rpEmoji — daftar glyph emoji warna utk grid (render via material.Label).
var rpEmoji = []string{
	"😀", "😂", "😍", "👍", "🙏", "🎉", "❤️", "🔥",
	"😢", "😮", "😘", "😎", "🤔", "😭", "🥰", "😅",
	"😊", "😡", "🤣", "👏", "💯", "🙌", "🤩", "😴",
	"😱", "🤗", "😇", "🥳", "😏", "😉", "😋", "🤯",
	"👌", "✌️", "🤝", "💪", "🙈", "👀", "💀", "🤡",
	"⭐", "🌟", "💥", "✨", "🎈", "🍕", "☕", "🌈",
	"🐶", "🐱", "🦄", "🌸", "🍀", "🚀", "⚡", "💖",
}

// ReactionPickerView menggambar backdrop transparan penuh lalu kartu popup terpusat
// berisi grid emoji bare.
func ReactionPickerView(gtx layout.Context, th *material.Theme, t Theme, ctl *RpCtl) layout.Dimensions {
	// latar app (var(--bg)) penuh sbg konteks; backdrop .rp-backdrop transparan.
	paint.FillShape(gtx.Ops, t.Bg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return rpCard(gtx, th, t, ctl)
	})
}

// rpCard — .rp-pop: t.Bg, radius 14, 352x400, border Line (kesan box-shadow-lg).
func rpCard(gtx layout.Context, th *material.Theme, t Theme, ctl *RpCtl) layout.Dimensions {
	w := gtx.Dp(352)
	h := gtx.Dp(400)
	sz := image.Pt(w, h)
	r := gtx.Dp(14)
	bw := gtx.Dp(1)

	// border lebih terang (Line) penuh, lalu isi t.Bg inset 1px → kesan bingkai bayangan.
	paint.FillShape(gtx.Ops, t.Line, clip.RRect{Rect: image.Rectangle{Max: sz}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	inner := image.Rectangle{Min: image.Pt(bw, bw), Max: image.Pt(w-bw, h-bw)}
	paint.FillShape(gtx.Ops, t.Bg, clip.RRect{Rect: inner, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))

	// grid emoji mengisi kartu: 8 per baris, sel 40px, terpusat.
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	rpGrid(gtx, th, t, w, h, ctl)
	return layout.Dimensions{Size: sz}
}

// rpGrid menata emoji dlm grid 8 kolom, tiap sel ~40px, glyph terpusat. Padding 12.
func rpGrid(gtx layout.Context, th *material.Theme, t Theme, w, h int, ctl *RpCtl) layout.Dimensions {
	cols := 8
	pad := gtx.Dp(12)
	cell := gtx.Dp(40)
	gap := (w - 2*pad - cols*cell) / (cols - 1)
	if gap < 0 {
		gap = 0
	}

	x0 := pad
	y := pad
	for i := 0; i < len(rpEmoji); i++ {
		col := i % cols
		if col == 0 && i != 0 {
			y += cell + gap
		}
		if y+cell > h-pad {
			break
		}
		x := x0 + col*(cell+gap)
		var clk *widget.Clickable
		if ctl != nil && i < len(ctl.Clicks) {
			clk = &ctl.Clicks[i]
		}
		rpCell(gtx, th, t, rpEmoji[i], x, y, cell, clk)
	}
	return layout.Dimensions{Size: image.Pt(w, h)}
}

// rpCell menggambar satu glyph emoji terpusat dlm sel cellxcell pd offset (x,y).
// clk != nil → sel bisa diklik (registrasi area pointer ikut ter-offset).
func rpCell(gtx layout.Context, th *material.Theme, t Theme, glyph string, x, y, cell int, clk *widget.Clickable) {
	sz := image.Pt(cell, cell)
	macro := op.Record(gtx.Ops)
	cgtx := gtx
	cgtx.Constraints.Min, cgtx.Constraints.Max = sz, sz
	body := func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(22), glyph)
			lbl.Color = t.Text
			lbl.MaxLines = 1
			return lbl.Layout(gtx)
		})
	}
	if clk != nil {
		clk.Layout(cgtx, body)
	} else {
		body(cgtx)
	}
	call := macro.Stop()
	off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
}
