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

// RpCtl = state interaktif pemilih emoji. Clicks: 1 per emoji KATEGORI AKTIF.
type RpCtl struct {
	Clicks    []widget.Clickable
	List      *widget.List       // grid emoji bisa di-scroll
	ActiveCat int                // kategori aktif (tab)
	TabClicks []widget.Clickable // paralel rpCats (ganti kategori)
}

// rpCat — satu kategori emoji (ikon tab + label tooltip + glyph).
type rpCat struct {
	icon  string
	label string
	emoji []string
}

// rpCats — emoji per kategori (ala keyboard HP), dikurasi agar tak tofu di
// NotoColorEmoji. Kategori 0 (Smiley) default + dipakai utk reaksi.
var rpCats = []rpCat{
	{"emojiface", "Smiley & emosi", rpSmileys},
	{"contacts", "Orang & gestur", rpPeople},
	{"emoji", "Hewan & alam", rpNature},
	{"locpin", "Makanan", rpFood},
	{"play", "Aktivitas", rpActivity},
	{"sticker", "Objek", rpObjects},
	{"globe", "Simbol", rpSymbols},
}

// RpEmoji — emoji bar reaksi cepat (kategori smiley); pemetaan indeks→glyph.
func RpEmoji() []string { return rpSmileys }

// rpCatCount — jumlah kategori emoji.
func rpCatCount() int { return len(rpCats) }

// rpMaxCatLen — panjang kategori terbesar (utk alokasi clickable).
func rpMaxCatLen() int {
	m := 0
	for _, c := range rpCats {
		if len(c.emoji) > m {
			m = len(c.emoji)
		}
	}
	return m
}

// rpCatEmoji — glyph kategori ke-i (aman bila indeks di luar batas).
func rpCatEmoji(i int) []string {
	if i < 0 || i >= len(rpCats) {
		i = 0
	}
	return rpCats[i].emoji
}

var rpSmileys = []string{
	"😀", "😃", "😄", "😁", "😆", "😅", "🤣", "😂", "🙂", "🙃", "😉", "😊",
	"😇", "🥰", "😍", "🤩", "😘", "😗", "😚", "😙", "😋", "😛", "😜", "🤪",
	"😝", "🤑", "🤗", "🤭", "🤫", "🤔", "🤐", "🤨", "😐", "😑", "😶", "😏",
	"😒", "🙄", "😬", "😌", "😔", "😪", "🤤", "😴", "😷", "🤒", "🤕", "🤢",
	"🤮", "🤧", "🥵", "🥶", "🥴", "😵", "🤯", "🤠", "🥳", "😎", "🤓", "🧐",
	"😕", "😟", "🙁", "😮", "😯", "😲", "😳", "🥺", "😦", "😧", "😨", "😰",
	"😥", "😢", "😭", "😱", "😖", "😣", "😞", "😓", "😩", "😫", "😤", "😠",
	"😡", "🤬", "😈", "👿", "💀", "💩", "🤡", "👹", "👺", "👻", "👽", "🤖",
	"😺", "😸", "😹", "😻", "😼", "😽", "🙀", "😿", "😾",
}

var rpPeople = []string{
	"👍", "👎", "👌", "🤌", "🤏", "✌️", "🤞", "🫰", "🤟", "🤘", "🤙", "👈",
	"👉", "👆", "👇", "☝️", "✋", "🤚", "🖐️", "🖖", "👋", "🤝", "👏", "🙌",
	"👐", "🤲", "🙏", "✊", "👊", "🤛", "🤜", "💪", "🦾", "✍️", "🫶", "🤳",
	"💅", "👀", "👁️", "👅", "👄", "🧠", "🫀", "🦷", "👶", "🧒", "👦", "👧",
	"🧑", "👨", "👩", "🧓", "👴", "👵", "🧔", "👮", "🕵️", "💂", "👷", "🤴",
	"👸", "👰", "🤵", "🧑‍⚕️", "🧑‍🏫", "🧑‍💻", "🧑‍🍳", "🧑‍🌾", "👼", "🎅", "🤶", "🦸",
	"🦹", "🧙", "🧚", "🧛", "🧜", "🧝", "🙅", "🙆", "💁", "🙋", "🙇", "🤦",
	"🤷", "💆", "💇", "🚶", "🏃", "💃", "🕺", "👯", "🧘", "🛀",
}

var rpNature = []string{
	"🐶", "🐱", "🐭", "🐹", "🐰", "🦊", "🐻", "🐼", "🐻‍❄️", "🐨", "🐯", "🦁",
	"🐮", "🐷", "🐽", "🐸", "🐵", "🙈", "🙉", "🙊", "🐒", "🐔", "🐧", "🐦",
	"🐤", "🐣", "🦆", "🦅", "🦉", "🦇", "🐺", "🐗", "🐴", "🦄", "🐝", "🐛",
	"🦋", "🐌", "🐞", "🐜", "🦗", "🕷️", "🦂", "🐢", "🐍", "🦎", "🦖", "🐙",
	"🦑", "🦐", "🦀", "🐡", "🐠", "🐟", "🐬", "🐳", "🐋", "🦈", "🐊", "🐅",
	"🐆", "🦓", "🦍", "🐘", "🦏", "🐪", "🐫", "🦒", "🐃", "🐂", "🐄", "🐎",
	"🐖", "🐏", "🐑", "🐐", "🦌", "🐕", "🐩", "🐈", "🐓", "🦃", "🕊️", "🐇",
	"🐁", "🐀", "🌸", "🌹", "🌺", "🌻", "🌷", "🌼", "🌵", "🌲", "🌳", "🌴",
	"🍀", "🍁", "🍂", "🍃", "🌿", "☘️", "🌍", "🌙", "⭐", "🌟", "☀️", "🌈",
}

var rpFood = []string{
	"🍏", "🍎", "🍐", "🍊", "🍋", "🍌", "🍉", "🍇", "🍓", "🫐", "🍈", "🍒",
	"🍑", "🥭", "🍍", "🥥", "🥝", "🍅", "🍆", "🥑", "🥦", "🥬", "🥒", "🌶️",
	"🌽", "🥕", "🧄", "🧅", "🥔", "🍠", "🥐", "🍞", "🥖", "🥨", "🧀", "🥚",
	"🍳", "🧇", "🥞", "🥓", "🍔", "🍟", "🍕", "🌭", "🥪", "🌮", "🌯", "🥙",
	"🥗", "🍝", "🍜", "🍲", "🍛", "🍣", "🍱", "🍤", "🍙", "🍚", "🍘", "🍢",
	"🍡", "🍧", "🍨", "🍦", "🥧", "🍰", "🎂", "🍮", "🍭", "🍬", "🍫", "🍿",
	"🍩", "🍪", "🌰", "🥜", "🍯", "☕", "🍵", "🧋", "🥤", "🍺", "🍻", "🥂",
	"🍷", "🥃", "🍸", "🍹", "🍾", "🥄", "🍴", "🍽️",
}

var rpActivity = []string{
	"⚽", "🏀", "🏈", "⚾", "🥎", "🎾", "🏐", "🏉", "🥏", "🎱", "🏓", "🏸",
	"🏒", "🏑", "🥍", "🏏", "⛳", "🏹", "🎣", "🥊", "🥋", "🎽", "⛸️", "🥌",
	"🛷", "🎿", "⛷️", "🏂", "🏋️", "🤼", "🤸", "🤺", "🤾", "🏌️", "🏇", "🧘",
	"🏄", "🏊", "🤽", "🚣", "🧗", "🚵", "🚴", "🏆", "🥇", "🥈", "🥉", "🏅",
	"🎖️", "🏵️", "🎗️", "🎫", "🎟️", "🎪", "🤹", "🎭", "🎨", "🎬", "🎤", "🎧",
	"🎼", "🎹", "🥁", "🎷", "🎺", "🎸", "🎻", "🎲", "♟️", "🎯", "🎳", "🎮",
	"🎰", "🧩",
}

var rpObjects = []string{
	"⌚", "📱", "💻", "⌨️", "🖥️", "🖨️", "🖱️", "💽", "💾", "💿", "📀", "📷",
	"📸", "📹", "🎥", "📽️", "🎞️", "📞", "☎️", "📟", "📠", "📺", "📻", "🎙️",
	"⏱️", "⏰", "🕰️", "⏳", "🔋", "🔌", "💡", "🔦", "🕯️", "🧯", "🛢️", "💸",
	"💵", "💴", "💶", "💷", "💰", "💳", "💎", "⚖️", "🔧", "🔨", "⚒️", "🛠️",
	"⛏️", "🔩", "⚙️", "🧱", "⛓️", "🧲", "🔫", "💣", "🧨", "🔪", "🗡️", "⚔️",
	"🛡️", "🚪", "🪑", "🚽", "🚿", "🛁", "🧴", "🧷", "🧹", "🧺", "🧻", "🧼",
	"🔑", "🗝️", "📦", "📫", "📮", "📜", "📃", "📄", "📑", "📊", "📈", "📉",
	"📅", "📆", "📌", "📍", "📎", "✂️", "🖊️", "✏️", "📚", "📖", "🔖", "🔗",
}

var rpSymbols = []string{
	"❤️", "🧡", "💛", "💚", "💙", "💜", "🤎", "🖤", "🤍", "💔", "❣️", "💕",
	"💞", "💓", "💗", "💖", "💘", "💝", "💟", "❤️‍🔥", "💯", "💢", "💥", "💫",
	"💦", "💨", "🕳️", "💬", "💭", "🔥", "✨", "🌟", "⭐", "🌠", "⚡", "☄️",
	"✅", "❌", "❎", "✔️", "❓", "❗", "❕", "❔", "‼️", "⁉️", "💲", "💱",
	"⚠️", "🚫", "🔞", "📵", "🚭", "❇️", "✳️", "❄️", "🎉", "🎊", "🎈", "🎁",
	"🎀", "🏆", "🥇", "👑", "💎", "🔔", "🔕", "🎵", "🎶", "➕", "➖", "➗",
	"♻️", "🔱", "⚜️", "🔰", "✅", "🆗", "🆕", "🆙", "🆒", "🆓", "🔟", "💠",
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

	// tab kategori (atas) + grid emoji kategori aktif (scroll).
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return rpTabs(gtx, th, t, ctl) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return rpGrid(gtx, th, t, w, h, ctl) }),
	)
	return layout.Dimensions{Size: sz}
}

// rpTabs — baris ikon kategori; tab aktif diberi garis-bawah accent.
func rpTabs(gtx layout.Context, th *material.Theme, t Theme, ctl *RpCtl) layout.Dimensions {
	active := 0
	if ctl != nil {
		active = ctl.ActiveCat
	}
	return layout.Inset{Top: unit.Dp(6), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, 0, len(rpCats))
		for i := range rpCats {
			i := i
			children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				hovered := ctl != nil && i < len(ctl.TabClicks) && ctl.TabClicks[i].Hovered()
				cell := func(gtx layout.Context) layout.Dimensions {
					col := t.Text2
					if i == active {
						col = t.Accent
					}
					if hovered { // tooltip nama kategori DI BAWAH ikon (di atas grid).
						m := op.Record(gtx.Ops)
						rpTipBelow(gtx, th, t, rpCats[i].label)
						op.Defer(gtx.Ops, m.Stop())
					}
					return layout.Stack{Alignment: layout.S}.Layout(gtx,
						layout.Stacked(func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								gtx.Constraints.Min.X = gtx.Constraints.Max.X
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return icon(gtx, rpCats[i].icon, 20, col)
								})
							})
						}),
						layout.Expanded(func(gtx layout.Context) layout.Dimensions {
							if i != active {
								return layout.Dimensions{}
							}
							hh := gtx.Dp(2)
							return layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								paint.FillShape(gtx.Ops, t.Accent, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, hh)}.Op())
								return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, hh)}
							})
						}),
					)
				}
				if ctl != nil && i < len(ctl.TabClicks) {
					return ctl.TabClicks[i].Layout(gtx, cell)
				}
				return cell(gtx)
			}))
		}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
	})
}

// rpTipBelow — tooltip kecil di bawah tab kategori (kotak inverse + teks).
func rpTipBelow(gtx layout.Context, th *material.Theme, t Theme, txt string) {
	cg := gtx
	cg.Constraints = layout.Constraints{Max: image.Pt(gtx.Dp(160), gtx.Dp(28))}
	m := op.Record(gtx.Ops)
	lbl := material.Label(th, 11, txt)
	lbl.Color, lbl.MaxLines = t.SidebarBg, 1
	dims := layout.UniformInset(unit.Dp(6)).Layout(cg, lbl.Layout)
	call := m.Stop()
	r := gtx.Dp(6)
	cx := gtx.Constraints.Max.X / 2
	y := gtx.Dp(40)
	off := op.Offset(image.Pt(cx-dims.Size.X/2, y)).Push(gtx.Ops)
	paint.FillShape(gtx.Ops, t.Text, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	off.Pop()
}

// rpGrid menata emoji KATEGORI AKTIF dlm grid 8 kolom yg BISA DI-SCROLL.
func rpGrid(gtx layout.Context, th *material.Theme, t Theme, w, h int, ctl *RpCtl) layout.Dimensions {
	const cols = 8
	active := 0
	if ctl != nil {
		active = ctl.ActiveCat
	}
	emojis := rpCatEmoji(active)
	rows := (len(emojis) + cols - 1) / cols
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
					if i >= len(emojis) {
						return layout.Dimensions{Size: image.Pt(0, gtx.Dp(42))}
					}
					var clk *widget.Clickable
					if ctl != nil && i < len(ctl.Clicks) {
						clk = &ctl.Clicks[i]
					}
					return rpCell(gtx, th, t, emojis[i], clk)
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
