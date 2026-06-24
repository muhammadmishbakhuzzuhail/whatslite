// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// communitiespaneview.go — sidebar pane KOMUNITAS: .pane-head + daftar kartu
// komunitas (ikon communities + nama + "N grup" + ringkas sub-grup). Fungsi murni,
// data demo inline (standalone render). nil → demo.
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

// comSub = satu sub-grup di dalam komunitas.
type comSub struct {
	jid       string
	name      string
	isDefault bool // grup "Pengumuman" (default community group)
}

// comItem = satu komunitas di daftar (nama + sub-baris jumlah/nama grup + sub-grup).
type comItem struct {
	jid    string
	name   string
	sub    string // "N grup · Grup A, Grup B, …"
	groups []comSub
}

// ComCtl = state interaktif pane komunitas. nil → demo. Open != nil → tampil
// detail (daftar sub-grup); else daftar komunitas (RowClicks paralel Items).
type ComCtl struct {
	Items     []comItem
	RowClicks []widget.Clickable
	NewBtn    *widget.Clickable
	Open      *comItem
	Back      *widget.Clickable
	SubClicks []widget.Clickable                         // paralel Open.groups (tap → buka chat grup)
	Pill      func(gtx layout.Context) layout.Dimensions // kotak cari ala chat (filter komunitas); nil = tak ditampilkan
}

// CommunitiesPaneView — sidebar 408px (t.SidebarBg) berisi pane KOMUNITAS:
// head + tombol "Komunitas baru" (ala WhatsApp) + daftar kartu komunitas
// (tiap kartu menampilkan grup Pengumuman dgn ikon megafon). newBtn nil → tombol
// tetap digambar tapi tak bisa diklik (render standalone).
func CommunitiesPaneView(gtx layout.Context, th *material.Theme, t Theme, ctl *ComCtl) layout.Dimensions {
	w := gtx.Dp(468)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	if ctl == nil { // data demo (render standalone / gio-shot)
		ctl = &ComCtl{Items: []comItem{
			{name: "Tim Kantor", sub: "3 grup · Umum, Proyek X, Acara"},
			{name: "Keluarga Besar", sub: "2 grup · Umum, Liburan"},
		}}
	}

	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	if ctl.Open != nil { // detail komunitas (daftar sub-grup)
		return comDetailView(gtx, th, t, w, ctl)
	}
	items := ctl.Items
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return comPaneHead(gtx, th, t, w, "Komunitas")
		}),
		// kotak cari ala chat (filter komunitas lokal).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if ctl == nil || ctl.Pill == nil {
				return layout.Dimensions{}
			}
			return ctl.Pill(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			row := func(gtx layout.Context) layout.Dimensions { return comNewRow(gtx, th, t) }
			if ctl.NewBtn != nil {
				return ctl.NewBtn.Layout(gtx, row)
			}
			return row(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(items) == 0 {
				return layout.Inset{Top: unit.Dp(40)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(th, 14, "Belum ada komunitas")
						l.Color = t.Text2
						return l.Layout(gtx)
					})
				})
			}
			children := make([]layout.FlexChild, 0, len(items))
			for i := range items {
				it, idx := items[i], i
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if ctl.RowClicks != nil && idx < len(ctl.RowClicks) {
						c := &ctl.RowClicks[idx]
						return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							macro := op.Record(gtx.Ops)
							dims := comRow(gtx, th, t, it)
							call := macro.Stop()
							if c.Hovered() {
								paint.FillShape(gtx.Ops, t.Hover, clip.Rect{Max: dims.Size}.Op())
							}
							call.Add(gtx.Ops)
							return dims
						})
					}
					return comRow(gtx, th, t, it)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)
}

// comDetailView — detail satu komunitas: header ← + nama, lalu daftar sub-grup
// (ikon + nama + "Pengumuman" utk grup default). Tap sub-grup → buka chat.
func comDetailView(gtx layout.Context, th *material.Theme, t Theme, w int, ctl *ComCtl) layout.Dimensions {
	c := ctl.Open
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return comDetailHead(gtx, th, t, w, c.name, ctl.Back)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(4), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				l := material.Label(th, 12.5, itoa(len(c.groups))+" GRUP")
				l.Color = t.Text2
				return l.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, len(c.groups))
			for i := range c.groups {
				g, idx := c.groups[i], i
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					row := func(gtx layout.Context) layout.Dimensions { return comSubRow(gtx, th, t, g) }
					if ctl.SubClicks != nil && idx < len(ctl.SubClicks) {
						return ctl.SubClicks[idx].Layout(gtx, row)
					}
					return row(gtx)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)
}

// comDetailHead — header detail komunitas: tombol ← + nama komunitas.
func comDetailHead(gtx layout.Context, th *material.Theme, t Theme, w int, name string, back *widget.Clickable) layout.Dimensions {
	h := gtx.Dp(60)
	sz := image.Pt(w, h)
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Rect{Max: sz}.Op())
	paint.FillShape(gtx.Ops, t.Divider, clip.Rect{Min: image.Pt(0, h-1), Max: sz}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				b := func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return icon(gtx, "back", 22, t.Text)
					})
				}
				if back != nil {
					return back.Layout(gtx, b)
				}
				return b(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				l := material.Label(th, 16.5, name)
				l.Color, l.Font.Weight, l.MaxLines = t.Text, font.Medium, 1
				return l.Layout(gtx)
			}),
		)
	})
}

// comSubRow — baris sub-grup di detail komunitas: ikon + nama + (default → "Pengumuman").
func comSubRow(gtx layout.Context, th *material.Theme, t Theme, g comSub) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		glyph := "communities"
		if g.isDefault {
			glyph = "channels" // megafon utk grup Pengumuman
		}
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				d := gtx.Dp(44)
				bsz := image.Pt(d, d)
				r := gtx.Dp(11)
				paint.FillShape(gtx.Ops, t.Bg2, clip.RRect{Rect: image.Rectangle{Max: bsz}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
				gtx.Constraints.Min, gtx.Constraints.Max = bsz, bsz
				layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return icon(gtx, glyph, 22, t.Accent)
				})
				return layout.Dimensions{Size: bsz}
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(13)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(th, 15.5, g.name)
						l.Color, l.MaxLines, l.Font.Weight = t.Text, 1, font.Medium
						return l.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !g.isDefault {
							return layout.Dimensions{}
						}
						l := material.Label(th, 12.5, "Pengumuman")
						l.Color, l.MaxLines = t.Text2, 1
						return l.Layout(gtx)
					}),
				)
			}),
		)
	})
}

// comNewRow — tombol "Komunitas baru": lingkaran aksen + plus, lalu label.
func comNewRow(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				d := gtx.Dp(46)
				bsz := image.Pt(d, d)
				paint.FillShape(gtx.Ops, t.Accent, clip.Ellipse{Max: bsz}.Op(gtx.Ops))
				gtx.Constraints.Min, gtx.Constraints.Max = bsz, bsz
				layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return icon(gtx, "plus", 24, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
				})
				return layout.Dimensions{Size: bsz}
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(13)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				l := material.Label(th, 16, "Komunitas baru")
				l.Color, l.Font.Weight = t.Text, font.Medium
				return l.Layout(gtx)
			}),
		)
	})
}

func comPaneHead(gtx layout.Context, th *material.Theme, t Theme, w int, title string) layout.Dimensions {
	return paneHead(gtx, th, t, w, title)
}

// comRow — kartu komunitas: ikon communities (kotak membulat) + nama + sub.
func comRow(gtx layout.Context, th *material.Theme, t Theme, it comItem) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				d := gtx.Dp(46)
				bsz := image.Pt(d, d)
				r := gtx.Dp(12)
				paint.FillShape(gtx.Ops, t.Bg2, clip.RRect{Rect: image.Rectangle{Max: bsz}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
				gtx.Constraints.Min, gtx.Constraints.Max = bsz, bsz
				layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return icon(gtx, "communities", 24, t.Accent)
				})
				return layout.Dimensions{Size: bsz}
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(13)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(th, 16, it.name)
						l.Color, l.MaxLines, l.Font.Weight = t.Text, 1, font.Medium
						return l.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(th, 13, it.sub)
						l.Color, l.MaxLines = t.Text2, 1
						return l.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
					// grup Pengumuman (read-only, megafon) — baris pertama tiap komunitas di WhatsApp.
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return icon(gtx, "channels", 16, t.Text2)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								l := material.Label(th, 13, "Pengumuman")
								l.Color, l.MaxLines = t.Text2, 1
								return l.Layout(gtx)
							}),
						)
					}),
				)
			}),
		)
	})
}
