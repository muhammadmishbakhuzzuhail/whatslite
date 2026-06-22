// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// searchview.go — state hasil pencarian sidebar (paritas
// frontend/src/lib/sidebar/ChatList.svelte mode `searching` + app.css). Sidebar
// 380 (sidebarBg): pil pencarian (.search r-pill, magnifier + query), chip row
// jenis (.sc-types/.chip — aktif "Semua" accent/#fff), label .list-label "PESAN",
// lalu 3 baris hit (.hit-row: avatar 40 + nama 15/Medium + teks 13.5 + jam 12).
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

// svHit = satu baris hasil pencarian isi pesan (.hit-row).
type svHit struct {
	name string
	text string
	time string
	jid  string // chat tujuan (klik → buka)
}

// SvCtl = state interaktif pencarian pesan global. nil → demo. Query editor +
// hasil nyata (SearchMessages) + clickable per-hit + tombol kembali.
type SvCtl struct {
	Query     *widget.Editor
	Hits      []svHit
	HitClicks []widget.Clickable
	Back      *widget.Clickable
}

// svChip = satu chip jenis (.chip) di .sc-types.
type svChip struct {
	label  string
	active bool
}

// SearchView menggambar sidebar 380 dalam state hasil pencarian.
func SearchView(gtx layout.Context, th *material.Theme, t Theme, ctl *SvCtl) layout.Dimensions {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}

	w := gtx.Dp(380)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	chips := []svChip{
		{"Semua", true},
		{"Foto", false},
		{"Video", false},
		{"Dokumen", false},
		{"Tautan", false},
		{"Suara", false},
	}
	hits := []svHit{
		{name: "Andi Pratama", text: "Oke nanti aku kirim file rapat-nya ya", time: "19.08"},
		{name: "Keluarga", text: "Ibu: jadwal rapat keluarga minggu depan", time: "18.41"},
		{name: "Tim Proyek X", text: "Budi: notulen rapat sudah aku rapat-kan", time: "16.20"},
	}
	if ctl != nil {
		hits = ctl.Hits
	}

	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// .search-wrap { padding: 8px 12px; } berisi pil .search (+ tombol kembali).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return svSearchWrap(gtx, th, t, ctl)
		}),
		// .sc-types { gap: 6px; padding: 6px 12px 8px; } (override inline ChatList).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return svChips(gtx, th, t, white, chips)
		}),
		// .list-label "PESAN" (messages_label).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return svListLabel(gtx, th, t, "PESAN")
		}),
		// daftar .hit-row (klik → buka chat).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(hits) == 0 {
				return layout.Inset{Top: unit.Dp(30)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(th, 14, "Ketik untuk mencari pesan")
						l.Color = t.Text2
						return l.Layout(gtx)
					})
				})
			}
			children := make([]layout.FlexChild, 0, len(hits))
			for i := range hits {
				hh, idx := hits[i], i
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					row := func(gtx layout.Context) layout.Dimensions { return svHitRow(gtx, th, t, hh) }
					if ctl != nil && idx < len(ctl.HitClicks) {
						return ctl.HitClicks[idx].Layout(gtx, row)
					}
					return row(gtx)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)
	return layout.Dimensions{Size: sz}
}

// svSearchWrap — .search-wrap padding 8/12 + pil .search (searchBg, r-pill,
// padding 9/14, gap 10): magnifier 18 text2 + query "rapat".
func svSearchWrap(gtx layout.Context, th *material.Theme, t Theme, ctl *SvCtl) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(6), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			// tombol kembali (← ) ke daftar chat.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				b := func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return icon(gtx, "back", 22, t.Text2)
					})
				}
				if ctl != nil && ctl.Back != nil {
					return ctl.Back.Layout(gtx, b)
				}
				return b(gtx)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				macro := op.Record(gtx.Ops)
				dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return svMagnifier(gtx, t) }),
						layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							if ctl != nil && ctl.Query != nil {
								e := material.Editor(th, ctl.Query, "Cari pesan")
								e.Color, e.HintColor, e.TextSize = t.Text, t.Text2, unit.Sp(14.5)
								return e.Layout(gtx)
							}
							lbl := material.Label(th, unit.Sp(14.5), "rapat")
							lbl.Color, lbl.MaxLines = t.Text, 1
							return lbl.Layout(gtx)
						}),
					)
				})
				call := macro.Stop()
				r := dims.Size.Y / 2
				paint.FillShape(gtx.Ops, t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
				call.Add(gtx.Ops)
				return dims
			}),
		)
	})
}

// svMagnifier — ikon kaca pembesar 18 (.search-ico text2): cincin lingkaran +
// gagang. Digambar via ellipse + rect (API yg dipakai ui.go).
func svMagnifier(gtx layout.Context, t Theme) layout.Dimensions {
	d := gtx.Dp(18)
	sz := image.Pt(d, d)
	ring := gtx.Dp(12) // diameter cincin
	bw := gtx.Dp(2)    // tebal garis
	// cincin: ellipse text2 lalu lubang searchBg di dalam.
	paint.FillShape(gtx.Ops, t.Text2, clip.Ellipse{Max: image.Pt(ring, ring)}.Op(gtx.Ops))
	paint.FillShape(gtx.Ops, t.SearchBg, clip.Ellipse{Min: image.Pt(bw, bw), Max: image.Pt(ring-bw, ring-bw)}.Op(gtx.Ops))
	// gagang: garis diagonal pendek dari sudut kanan-bawah cincin ke pojok.
	g := gtx.Dp(6)
	hx := ring - gtx.Dp(3)
	hy := ring - gtx.Dp(3)
	paint.FillShape(gtx.Ops, t.Text2, clip.Rect{Min: image.Pt(hx, hy), Max: image.Pt(hx+g, hy+bw)}.Op())
	paint.FillShape(gtx.Ops, t.Text2, clip.Rect{Min: image.Pt(hx, hy), Max: image.Pt(hx+bw, hy+g)}.Op())
	return layout.Dimensions{Size: sz}
}

// svChips — .sc-types { gap 6; padding 6px 12px 8px } berisi chip jenis.
func svChips(gtx layout.Context, th *material.Theme, t Theme, white color.NRGBA, chips []svChip) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, 0, len(chips)*2)
		for i, c := range chips {
			if i > 0 {
				children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout)) // gap: 6px
			}
			cc := c
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return svChipBtn(gtx, th, t, white, cc)
			}))
		}
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
	})
}

// svChipBtn — .chip: radius 16, padding 5/13, font 13. Aktif: accent/#fff;
// pasif: searchBg/text2.
func svChipBtn(gtx layout.Context, th *material.Theme, t Theme, white color.NRGBA, c svChip) layout.Dimensions {
	bg := t.SearchBg
	fg := t.Text2
	if c.active {
		bg = t.Accent
		fg = white
	}
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(13), Right: unit.Dp(13)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, 13, c.label)
		lbl.Color = fg
		lbl.MaxLines = 1
		return lbl.Layout(gtx)
	})
	call := macro.Stop()
	r := gtx.Dp(16)
	paint.FillShape(gtx.Ops, bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// svListLabel — .list-label: padding 8/16/4, 12.5px text2 uppercase.
func svListLabel(gtx layout.Context, th *material.Theme, t Theme, txt string) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(4), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, unit.Sp(12.5), txt)
		lbl.Color = t.Text2
		lbl.MaxLines = 1
		return lbl.Layout(gtx)
	})
}

// svHitRow — .hit-row: padding 9/12, gap 12, avatar 40 + .hit-main (top: nama
// 15/Medium + jam 12 text2; bawah: teks 13.5 text2).
func svHitRow(gtx layout.Context, th *material.Theme, t Theme, h svHit) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return svHitAvatar(gtx, th, h.name)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					// .hit-top: nama (flex) + jam kanan.
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(th, 15, h.name)
								lbl.Color = t.Text
								lbl.MaxLines = 1
								lbl.Font.Weight = font.Medium
								return lbl.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(th, 12, h.time)
								lbl.Color = t.Text2
								lbl.MaxLines = 1
								return lbl.Layout(gtx)
							}),
						)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
					// .hit-text: 13.5px text2.
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(13.5), h.text)
						lbl.Color = t.Text2
						lbl.MaxLines = 1
						return lbl.Layout(gtx)
					}),
				)
			}),
		)
	})
}

// svHitAvatar — .hit-av: lingkaran 40 avatarColor(name) + inisial 16/Bold putih.
func svHitAvatar(gtx layout.Context, th *material.Theme, name string) layout.Dimensions {
	d := gtx.Dp(40)
	sz := image.Pt(d, d)
	paint.FillShape(gtx.Ops, avatarColor(name), clip.Ellipse{Max: sz}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, 16, initial(name))
		lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		lbl.Font.Weight = font.Bold
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}
