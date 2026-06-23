// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// contactspaneview.go — sidebar pane KONTAK (paritas frontend/src/lib/sidebar/
// ContactsPane.svelte + app.css): .pane-head 56px ("Kontak" 19/SemiBold), .ct-top
// (pil pencarian searchBg + tombol accent "Kontak baru"), lalu daftar .ct-list
// dgn pemisah huruf .ct-letter (accent 12/Bold, pad 5/16, bg --bg) + baris .ct-row
// (avatar 40 + nama 15/Normal + about/status 12.5 text2 + ikon info "i" text2 kanan).
// Fungsi murni, data demo inline (standalone render).
package gioui

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// ctRowTag — tag pointer stabil per-baris kontak (deteksi klik-kanan → menu konteks).
type ctRowTag int

// cpContact = satu kontak (.ct-row). idx = indeks clickable datar (buka chat).
type cpContact struct {
	name   string
	about  string
	jid    string // utk muat foto profil asli (bukan fallback inisial)
	online bool   // .ct-dot (titik hijau online)
	idx    int    // indeks ke dalam clicks (untuk buka chat); -1 = tak bisa diklik
}

// cpAvatarFn — penggambar avatar (foto asli + fallback inisial). nil = inisial saja.
type cpAvatarFn func(gtx layout.Context, name, jid string, dp int) layout.Dimensions

// cpGroup = satu kelompok huruf (.ct-letter + items).
type cpGroup struct {
	letter string
	items  []cpContact
}

// ContactsPaneView menggambar sidebar 380px (t.SidebarBg) berisi pane KONTAK.
// Fungsi murni, mandiri (standalone render).
// cpFlat — satu baris daftar kontak (header huruf ATAU kontak) utk material.List
// yang bisa di-scroll.
type cpFlat struct {
	letter   string
	isLetter bool
	c        cpContact
}

func ContactsPaneView(gtx layout.Context, th *material.Theme, t Theme, groups []cpGroup, clicks []widget.Clickable, newGroup *widget.Clickable, list *widget.List, avFn cpAvatarFn, search *widget.Editor, infoClicks []widget.Clickable, newContact *widget.Clickable, onCtx func(idx int)) layout.Dimensions {
	w := gtx.Dp(468)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	if groups == nil { // data demo (render standalone / gio-shot)
		groups = []cpGroup{
			{letter: "A", items: []cpContact{{name: "Alice", about: "Tersedia", online: true, idx: -1}}},
			{letter: "B", items: []cpContact{{name: "Bob", about: "Di tempat kerja", idx: -1}}},
			{letter: "C", items: []cpContact{{name: "Carol", about: "Sibuk · jangan ganggu", idx: -1}}},
		}
	}

	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// .pane-head 56px "Kontak" (19/SemiBold).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return cpPaneHead(gtx, th, t, w, "Kontak")
		}),
		// .ct-top { gap: 8px; padding: 6px 12px 10px } : pil cari + tombol baru.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return cpTop(gtx, th, t, newGroup, search, newContact)
		}),
		// .ct-list : pemisah huruf + baris kontak — SCROLLABLE (material.List).
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			flat := make([]cpFlat, 0, len(groups)*2)
			for _, g := range groups {
				flat = append(flat, cpFlat{letter: g.letter, isLetter: true})
				for _, c := range g.items {
					flat = append(flat, cpFlat{c: c})
				}
			}
			if list == nil {
				list = &widget.List{}
				list.Axis = layout.Vertical
			}
			return material.List(th, list).Layout(gtx, len(flat), func(gtx layout.Context, i int) layout.Dimensions {
				it := flat[i]
				if it.isLetter {
					return cpLetter(gtx, th, t, it.letter)
				}
				var infoC, rowC *widget.Clickable // "i" → info-drawer; baris → buka chat
				if it.c.idx >= 0 && it.c.idx < len(infoClicks) {
					infoC = &infoClicks[it.c.idx]
				}
				if it.c.idx >= 0 && it.c.idx < len(clicks) {
					rowC = &clicks[it.c.idx]
				}
				// rowC & infoC = clickable BERSEBELAHAN (bukan nested) → klik "i" tak
				// memicu buka-chat. idx<0 (demo) → keduanya nil (tak bisa diklik).
				// PENTING (routing pointer Gio): event jatuh ke area TERDALAM lalu
				// merambat ke induk. Tag klik-kanan harus jadi INDUK clickable: push
				// clip baris + event.Op(tag), lalu REPLAY cpRow di dalamnya. Maka klik
				// di baris → clickable (anak) dapat primary, tag (induk) dapat secondary.
				// (Tag sebagai sibling di atas/bawah hanya bikin salah satu jalan.)
				macro := op.Record(gtx.Ops)
				dims := cpRow(gtx, th, t, it.c, avFn, rowC, infoC)
				call := macro.Stop()
				if onCtx != nil && it.c.idx >= 0 {
					tag := ctRowTag(it.c.idx)
					for {
						ev, ok := gtx.Event(pointer.Filter{Target: tag, Kinds: pointer.Press})
						if !ok {
							break
						}
						if pe, ok := ev.(pointer.Event); ok && pe.Buttons.Contain(pointer.ButtonSecondary) {
							onCtx(it.c.idx)
						}
					}
					area := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
					event.Op(gtx.Ops, tag) // tag = induk area baris
					call.Add(gtx.Ops)      // clickable = anak (nested di dalam clip tag)
					area.Pop()
				} else {
					call.Add(gtx.Ops)
				}
				return dims
			})
		}),
	)
	return layout.Dimensions{Size: sz}
}

// paneHead — header pane sidebar SERAGAM (sumber tunggal; paritas header "Chat"):
// tinggi 60, SidebarBg, divider bawah 1px, judul 23/Bold, inset kiri 18, terpusat
// vertikal. Semua pane utama (Kontak/Status/Saluran/Komunitas/Panggilan/Chat)
// memakai ini agar judul konsisten — jangan bikin header per-pane sendiri lagi.
func paneHead(gtx layout.Context, th *material.Theme, t Theme, w int, title string) layout.Dimensions {
	h := gtx.Dp(60)
	sz := image.Pt(w, h)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())
	paint.FillShape(gtx.Ops, t.Divider, clip.Rect{Min: image.Pt(0, h-gtx.Dp(1)), Max: sz}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	// top-aligned (Top16) — sama persis header "Chat" lama yg sudah benar.
	layout.Inset{Left: unit.Dp(18), Top: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, 23, title)
		lbl.Color = t.Text
		lbl.Font.Weight = font.Bold
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}

// cpPaneHead — header pane Kontak (delegasi ke paneHead seragam).
func cpPaneHead(gtx layout.Context, th *material.Theme, t Theme, w int, title string) layout.Dimensions {
	return paneHead(gtx, th, t, w, title)
}

// cpTop — .ct-top { display:flex; gap:8px; padding:6px 12px 10px } : pil pencarian
// (searchBg, r-pill, magnifier + placeholder) + tombol accent "Kontak baru".
func cpTop(gtx layout.Context, th *material.Theme, t Theme, newGroup *widget.Clickable, search *widget.Editor, newContact *widget.Clickable) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return cpSearchPill(gtx, th, t, search)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout), // gap: 8px
			// tombol ikon "Kontak baru" (addmember = orang+).
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return cpIconBtn(gtx, th, t, newContact, "addmember")
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			// tombol ikon "Grup baru" (peoplegroup = dua orang).
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return cpIconBtn(gtx, th, t, newGroup, "peoplegroup")
			}),
		)
	})
}

// cpIconBtn — tombol ikon-saja accent (r-pill) 38x38 utk aksi top kontak.
func cpIconBtn(gtx layout.Context, th *material.Theme, t Theme, c *widget.Clickable, ic string) layout.Dimensions {
	_ = th
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	body := func(gtx layout.Context) layout.Dimensions {
		d := gtx.Dp(38)
		r := d / 2
		paint.FillShape(gtx.Ops, t.Accent, clip.RRect{Rect: image.Rectangle{Max: image.Pt(d, d)}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		gtx.Constraints.Min, gtx.Constraints.Max = image.Pt(d, d), image.Pt(d, d)
		layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return icon(gtx, ic, 19, white) })
		return layout.Dimensions{Size: image.Pt(d, d)}
	}
	if c != nil {
		return c.Layout(gtx, body)
	}
	return body(gtx)
}

// cpSearchPill — .ct-search { background: var(--bg2); border-radius: 10px;
// padding: 9px 14px } : magnifier 18 + editor "Cari" (ed nil = label statis).
func cpSearchPill(gtx layout.Context, th *material.Theme, t Theme, ed *widget.Editor) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon(gtx, "search", 18, t.Text2)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if ed != nil {
					e := material.Editor(th, ed, "Cari nama atau nomor")
					e.Color, e.HintColor, e.TextSize = t.Text, t.Text2, unit.Sp(14.5)
					return e.Layout(gtx)
				}
				lbl := material.Label(th, unit.Sp(14.5), "Cari")
				lbl.Color = t.Text2
				lbl.MaxLines = 1
				return lbl.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	r := gtx.Dp(10) // border-radius: 10px
	paint.FillShape(gtx.Ops, t.Bg2, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// cpLetter — .ct-letter { background: var(--bg); color: var(--accent);
// font-size: 12px; font-weight: 700; padding: 5px 16px } pemisah huruf alfabet.
func cpLetter(gtx layout.Context, th *material.Theme, t Theme, letter string) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		lbl := material.Label(th, 12, letter)
		lbl.Color = t.Accent
		lbl.MaxLines = 1
		lbl.Font.Weight = font.Bold
		return lbl.Layout(gtx)
	})
	call := macro.Stop()
	paint.FillShape(gtx.Ops, t.Bg, clip.Rect{Max: dims.Size}.Op()) // background: var(--bg)
	call.Add(gtx.Ops)
	return dims
}

// cpRow — .ct-row { gap: 12px; padding: 8px 14px } : avatar 40 (.avatar.sm) +
// kolom .ct-meta (.ct-name 15/Normal + .ct-sub 12.5 text2) + ikon "i" kanan.
// rowC (buka chat) membungkus avatar+meta; infoC ("i") = sibling TERPISAH agar
// klik "i" tak ikut memicu buka-chat. Keduanya nil → baris tak bisa diklik (demo).
func cpRow(gtx layout.Context, th *material.Theme, t Theme, c cpContact, avFn cpAvatarFn, rowC, infoC *widget.Clickable) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			// area buka-chat (avatar + meta), flexed mengisi sisa lebar.
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				openArea := func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return cpAvatarDot(gtx, th, t, c, avFn)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout), // gap: 12px
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									lbl := material.Label(th, 15, c.name) // .ct-name 15px
									lbl.Color = t.Text
									lbl.MaxLines = 1
									lbl.Font.Weight = font.Normal
									return lbl.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									lbl := material.Label(th, unit.Sp(12.5), c.about) // .ct-sub 12.5px
									lbl.Color = t.Text2
									lbl.MaxLines = 1
									return lbl.Layout(gtx)
								}),
							)
						}),
					)
				}
				if rowC != nil {
					return rowC.Layout(gtx, openArea)
				}
				return openArea(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			// ikon "i" — clickable terpisah (info-drawer).
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				infoIcon := func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return icon(gtx, "info", 20, t.Text2)
					})
				}
				if infoC != nil { // ketuk "i" → info-drawer kontak
					return infoC.Layout(gtx, infoIcon)
				}
				return infoIcon(gtx)
			}),
		)
	})
}

// cpAvatarDot — .ct-av { position: relative } : avatar 40 (.avatar.sm) + titik
// online .ct-dot { right:-1; bottom:-1; 12x12; bg:#28c840; border:2px var(--bg) }.
func cpAvatarDot(gtx layout.Context, th *material.Theme, t Theme, c cpContact, avFn cpAvatarFn) layout.Dimensions {
	var av layout.Dimensions
	if avFn != nil {
		av = avFn(gtx, c.name, c.jid, 40) // foto profil asli (fallback inisial di dalam)
	} else {
		av = cpAvatar(gtx, th, t, c.name, 40)
	}
	if c.online {
		dot := gtx.Dp(12)
		bw := gtx.Dp(2)
		off := gtx.Dp(1) // right:-1px; bottom:-1px
		x := av.Size.X - dot + off
		y := av.Size.Y - dot + off
		green := color.NRGBA{R: 0x28, G: 0xc8, B: 0x40, A: 0xff} // #28c840
		// border 2px var(--bg).
		paint.FillShape(gtx.Ops, t.Bg, clip.Ellipse{Min: image.Pt(x, y), Max: image.Pt(x+dot, y+dot)}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, green, clip.Ellipse{Min: image.Pt(x+bw, y+bw), Max: image.Pt(x+dot-bw, y+dot-bw)}.Op(gtx.Ops))
	}
	return av
}

// cpAvatar — .ct-av lingkaran avatarColor(name) + inisial putih (paritas
// u.avatar: font 0.4*d, Bold, putih, di tengah).
func cpAvatar(gtx layout.Context, th *material.Theme, t Theme, name string, dp int) layout.Dimensions {
	_ = t
	d := gtx.Dp(unit.Dp(dp))
	sz := image.Pt(d, d)
	paint.FillShape(gtx.Ops, avatarColor(name), clip.Ellipse{Max: sz}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, unit.Sp(float32(dp)*0.4), initial(name))
		lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		lbl.Font.Weight = font.Bold
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}
