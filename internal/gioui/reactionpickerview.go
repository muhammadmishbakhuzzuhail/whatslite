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
	List   *widget.List // grid emoji bisa di-scroll (banyak emoji)
}

// RpEmoji mengekspos daftar emoji reaksi (utk pemetaan indeks→glyph di handler UI).
func RpEmoji() []string { return rpEmoji }

// rpEmoji — daftar glyph emoji warna utk grid (render via material.Label).
var rpEmoji = []string{
	// Smileys & emosi
	"😀", "😃", "😄", "😁", "😆", "😅", "🤣", "😂", "🙂", "🙃", "😉", "😊",
	"😇", "🥰", "😍", "🤩", "😘", "😗", "😚", "😙", "😋", "😛", "😜", "🤪",
	"😝", "🤑", "🤗", "🤭", "🤫", "🤔", "🤐", "🤨", "😐", "😑", "😶", "😏",
	"😒", "🙄", "😬", "😌", "😔", "😪", "🤤", "😴", "😷", "🤒", "🤕", "🤢",
	"🤮", "🤧", "🥵", "🥶", "🥴", "😵", "🤯", "🤠", "🥳", "😎", "🤓", "🧐",
	"😕", "😟", "🙁", "😮", "😯", "😲", "😳", "🥺", "😦", "😧", "😨", "😰",
	"😥", "😢", "😭", "😱", "😖", "😣", "😞", "😓", "😩", "😫", "😤", "😡",
	"😠", "🤬", "😈", "👿", "💀", "💩", "🤡", "👻", "👽", "🤖",
	// Gestur & tubuh
	"👍", "👎", "👌", "✌️", "🤞", "🤟", "🤘", "👏", "🙌", "🙏", "🤝", "💪",
	"👋", "🤙", "👈", "👉", "👆", "👇", "✊", "👊", "🫶", "❤️‍🔥", "👀", "🧠",
	// Hati & simbol
	"❤️", "🧡", "💛", "💚", "💙", "💜", "🤎", "🖤", "🤍", "💔", "❣️", "💕",
	"💞", "💓", "💗", "💖", "💘", "💝", "💯", "💢", "💥", "💫", "💦", "💨",
	"🔥", "✨", "🌟", "⭐", "🎉", "🎊", "🎈", "🎁", "🏆", "🥇", "👑", "💎",
	// Hewan & alam
	"🐶", "🐱", "🐭", "🐹", "🐰", "🦊", "🐻", "🐼", "🐨", "🐯", "🦁", "🐮",
	"🐷", "🐸", "🐵", "🐔", "🦄", "🐝", "🦋", "🌸", "🌹", "🌻", "🌈", "🍀",
	// Makanan & aktivitas
	"🍕", "🍔", "🍟", "🌮", "🍩", "🍪", "🎂", "🍰", "🍫", "🍿", "☕", "🍺",
	"⚽", "🏀", "🎮", "🎵", "🎸", "🚀", "⚡", "💡", "📌", "✅", "❌", "❓",
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

// rpGrid menata emoji dlm grid 8 kolom yg BISA DI-SCROLL (banyak emoji). Tiap baris
// = Flex 8 sel; daftar baris via material.List vertikal. Hover sel → bg bulat.
func rpGrid(gtx layout.Context, th *material.Theme, t Theme, w, h int, ctl *RpCtl) layout.Dimensions {
	const cols = 8
	rows := (len(rpEmoji) + cols - 1) / cols
	var lst *widget.List
	if ctl != nil && ctl.List != nil {
		lst = ctl.List
	} else {
		lst = &widget.List{}
	}
	lst.Axis = layout.Vertical
	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.List(th, lst).Layout(gtx, rows, func(gtx layout.Context, row int) layout.Dimensions {
			children := make([]layout.FlexChild, 0, cols)
			for c := 0; c < cols; c++ {
				i := row*cols + c
				children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					if i >= len(rpEmoji) {
						return layout.Dimensions{Size: image.Pt(0, gtx.Dp(42))}
					}
					var clk *widget.Clickable
					if ctl != nil && i < len(ctl.Clicks) {
						clk = &ctl.Clicks[i]
					}
					return rpCell(gtx, th, t, rpEmoji[i], clk)
				}))
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
		})
	})
}

// rpCell — satu sel emoji (klik = pilih). Hover → bg bulat (jelas mana yg ditunjuk).
func rpCell(gtx layout.Context, th *material.Theme, t Theme, glyph string, clk *widget.Clickable) layout.Dimensions {
	d := gtx.Dp(42)
	sz := image.Pt(gtx.Constraints.Max.X, d)
	body := func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
		if clk != nil && clk.Hovered() { // bg bulat hover
			cd := gtx.Dp(38)
			off := op.Offset(image.Pt((sz.X-cd)/2, (d-cd)/2)).Push(gtx.Ops)
			rad := cd / 2
			paint.FillShape(gtx.Ops, t.Hover, clip.RRect{Rect: image.Rectangle{Max: image.Pt(cd, cd)}, NW: rad, NE: rad, SE: rad, SW: rad}.Op(gtx.Ops))
			off.Pop()
		}
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(23), glyph)
			lbl.Color, lbl.MaxLines = t.Text, 1
			return lbl.Layout(gtx)
		})
	}
	if clk != nil {
		return clk.Layout(gtx, body)
	}
	return body(gtx)
}
