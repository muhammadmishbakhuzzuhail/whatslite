// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// channelspaneview.go — sidebar pane CHANNELS (paritas frontend/src/lib/sidebar/
// ChannelsPane.svelte + app.css): .pane-head 56px ("Channels" 19/SemiBold), lalu
// baris tab .ch-tabs (Diikuti aktif accent/#fff, Jelajahi pasif bg2, radius), lalu
// daftar .ch-row: avatar 48 + nama 15/SemiBold + "0 subscriber" 13.5 text2 +
// kanan ikon-btn lonceng (text2) + "✕" batal-ikuti. Fungsi murni, data demo.
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

// chnChannel = satu baris saluran (.ch-row). follow=true → baris jelajah (tombol
// "Ikuti"); jid utk aksi engine.
type chnChannel struct {
	name     string
	subs     string
	jid      string
	pic      string // URL CDN foto saluran (dari ChannelDTO.Picture); "" = inisial
	role     string // peran kita: owner/admin (boleh posting) | subscriber
	follow   bool
	verified bool   // lencana centang (channel terverifikasi)
	desc     string // deskripsi saluran (utk panel info)
	invite   string // kode undangan (tautan bagikan …/channel/<kode>)
	muted    bool   // notifikasi saluran dibisukan
}

// ChnCtl = state interaktif pane channels (nil → statis/demo). Tab Diikuti/Jelajahi
// + aksi per-baris (ikuti/batal-ikuti).
type ChnCtl struct {
	Tabs      []widget.Clickable // 0=Diikuti, 1=Jelajahi
	Active    int
	Rows      []widget.Clickable                         // aksi per-channel (ikuti/unfollow)
	Opens     []widget.Clickable                         // buka channel (tap baris diikuti → reader)
	Pill      func(gtx layout.Context) layout.Dimensions // kotak cari ala chat (Diikuti=filter lokal, Jelajahi=direktori)
	Av        cpAvatarFn                                 // penggambar avatar (foto channel asli); nil = inisial
	List      *widget.List                               // gulir daftar saluran (nil → Flex biasa)
	ActiveJID string                                     // jid saluran terbuka (sorot baris aktif ala chat)
	Menu      []widget.Clickable                         // chevron menu per-baris diikuti (muncul saat hover)
}

// chnTab = satu tombol tab (.ch-tabs button).
type chnTab struct {
	label  string
	active bool
}

// ChannelsPaneView menggambar sidebar 380px (t.SidebarBg) berisi pane CHANNELS:
// .pane-head + .ch-tabs + daftar .ch-row. Fungsi murni, mandiri (standalone).
func ChannelsPaneView(gtx layout.Context, th *material.Theme, t Theme, channels []chnChannel, ctl *ChnCtl) layout.Dimensions {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}

	w := gtx.Dp(468)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	active := 0
	if ctl != nil {
		active = ctl.Active
	}
	tabs := []chnTab{
		{label: "Diikuti", active: active == 0},
		{label: "Jelajahi", active: active == 1},
	}
	if channels == nil { // data demo (render standalone / gio-shot)
		channels = []chnChannel{
			{name: "WhatsLite News", subs: "1,2 jt pengikut"},
			{name: "Tech Daily", subs: "850 rb pengikut", follow: true}, // baris jelajah → "Ikuti"
		}
	}

	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// .pane-head { height: 56px; padding: 0 16px; background: head-bg } h2 19/SemiBold.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return chnPaneHead(gtx, th, t, w, "Saluran")
		}),
		// .ch-tabs { gap: 6px; padding: 2px 12px 10px } : Diikuti / Jelajahi.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return chnTabs(gtx, th, t, white, tabs, ctl)
		}),
		// kotak cari ala chat (kedua tab): Diikuti = filter lokal, Jelajahi = direktori.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if ctl == nil || ctl.Pill == nil {
				return layout.Dimensions{}
			}
			return ctl.Pill(gtx)
		}),
		// daftar .ch-row (Flexed → isi sisa tinggi; material.List → gulir).
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(channels) == 0 { // kosong → ajak jelajah
				return layout.Inset{Top: unit.Dp(40)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						msg := "Belum mengikuti channel"
						if active == 1 {
							msg = "Tak ada channel ditemukan"
						}
						l := material.Label(th, 14, msg)
						l.Color = t.Text2
						return l.Layout(gtx)
					})
				})
			}
			var avFn cpAvatarFn
			var list *widget.List
			if ctl != nil {
				avFn = ctl.Av
				list = ctl.List
			}
			renderRow := func(gtx layout.Context, idx int) layout.Dimensions {
				cc := channels[idx]
				var rc, oc, mc *widget.Clickable
				if ctl != nil && idx < len(ctl.Rows) {
					rc = &ctl.Rows[idx]
				}
				if ctl != nil && idx < len(ctl.Opens) {
					oc = &ctl.Opens[idx]
				}
				if ctl != nil && idx < len(ctl.Menu) {
					mc = &ctl.Menu[idx]
				}
				// Hover SELURUH baris (area buka oc / tombol aksi rc / chevron mc).
				hov := (oc != nil && oc.Hovered()) || (rc != nil && rc.Hovered()) || (mc != nil && mc.Hovered())
				macro := op.Record(gtx.Ops)
				dims := chnChannelRow(gtx, th, t, cc, rc, oc, mc, hov, avFn)
				call := macro.Stop()
				// kartu MEMBULAT ala baris chat: aktif (saluran terbuka) > hover.
				bg := color.NRGBA{}
				if ctl != nil && ctl.ActiveJID != "" && cc.jid == ctl.ActiveJID {
					bg = t.Selected
				} else if hov {
					bg = t.Hover
				}
				if bg.A > 0 {
					m, vy, rr := gtx.Dp(7), gtx.Dp(3), gtx.Dp(12)
					rect := image.Rectangle{Min: image.Pt(m, vy), Max: image.Pt(dims.Size.X-m, dims.Size.Y-vy)}
					paint.FillShape(gtx.Ops, bg, clip.RRect{Rect: rect, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
				}
				call.Add(gtx.Ops)
				return dims
			}
			if list != nil {
				list.Axis = layout.Vertical
				return material.List(th, list).Layout(gtx, len(channels), renderRow)
			}
			children := make([]layout.FlexChild, 0, len(channels))
			for i := range channels {
				idx := i
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return renderRow(gtx, idx) }))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)
	return layout.Dimensions{Size: sz}
}

// chnPaneHead — .pane-head { height: 56px; padding: 0 16px; background: head-bg }
// h2 19/SemiBold.
func chnPaneHead(gtx layout.Context, th *material.Theme, t Theme, w int, title string) layout.Dimensions {
	return paneHead(gtx, th, t, w, title)
}

// chnTabs — .ch-tabs { gap: 6px; padding: 2px 12px 10px } : dua tombol flex-1.
func chnTabs(gtx layout.Context, th *material.Theme, t Theme, white color.NRGBA, tabs []chnTab, ctl *ChnCtl) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		children := make([]layout.FlexChild, 0, len(tabs)*2)
		for i, tab := range tabs {
			if i > 0 {
				children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout)) // gap: 6px
			}
			tt, idx := tab, i
			children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				btn := func(gtx layout.Context) layout.Dimensions { return chnTabBtn(gtx, th, t, white, tt) }
				if ctl != nil && idx < len(ctl.Tabs) {
					return ctl.Tabs[idx].Layout(gtx, btn)
				}
				return btn(gtx)
			}))
		}
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
	})
}

// chnTabBtn — .ch-tabs button { flex:1; padding:8px; radius:9px; font:13/600 }.
// Aktif: accent/#fff; pasif: bg2/text2.
func chnTabBtn(gtx layout.Context, th *material.Theme, t Theme, white color.NRGBA, tab chnTab) layout.Dimensions {
	bg := t.Bg2
	fg := t.Text2
	if tab.active {
		bg = t.Accent
		fg = white
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 13, tab.label)
			lbl.Color = fg
			lbl.MaxLines = 1
			lbl.Font.Weight = font.SemiBold
			return lbl.Layout(gtx)
		})
	})
	call := macro.Stop()
	r := gtx.Dp(9) // border-radius: 9px
	paint.FillShape(gtx.Ops, bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// chnChannelRow — .ch-row { padding 14; gap 13; align center } : avatar 48 +
// kolom (nama 15/SemiBold + sub 13.5 text2) + ikon lonceng + "✕" batal-ikuti.
func chnChannelRow(gtx layout.Context, th *material.Theme, t Theme, c chnChannel, rc, oc, menuC *widget.Clickable, hov bool, avFn cpAvatarFn) layout.Dimensions {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(20), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			// area buka channel (avatar + meta), flexed; tap → reader (oc). Tombol aksi terpisah.
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				open := func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if avFn != nil { // foto channel asli (fallback inisial di dalam)
								return avFn(gtx, c.name, c.jid, 54)
							}
							return chnAvatar(gtx, th, c.name, 54)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout), // gap: 14px (selaras chat)
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { // nama menyusut+ellipsis (jangan tumpang-tindih tombol Ikuti)
											lbl := material.Label(th, 16.5, c.name)
											lbl.Color, lbl.MaxLines, lbl.Font.Weight = t.Text, 1, font.Medium
											return lbl.Layout(gtx)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											if !c.verified {
												return layout.Dimensions{}
											}
											return layout.Inset{Left: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return icon(gtx, "verif", 15, t.Accent)
											})
										}),
									)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(3)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									lbl := material.Label(th, unit.Sp(13.5), c.subs)
									lbl.Color, lbl.MaxLines = t.Text2, 1
									return lbl.Layout(gtx)
								}),
							)
						}),
					)
				}
				if oc != nil && !c.follow { // hanya channel diikuti yg bisa dibuka
					return oc.Layout(gtx, open)
				}
				return open(gtx)
			}),
			// kanan: baris jelajah → tombol "Ikuti" (accent); baris diikuti → lonceng + ✕.
			layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if c.follow { // jelajah → tombol Ikuti
					btn := func(gtx layout.Context) layout.Dimensions {
						macro := op.Record(gtx.Ops)
						dims := layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(th, 13, "Ikuti")
							l.Color, l.Font.Weight = white, font.SemiBold
							return l.Layout(gtx)
						})
						call := macro.Stop()
						r := dims.Size.Y / 2
						paint.FillShape(gtx.Ops, t.Accent, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
						call.Add(gtx.Ops)
						return dims
					}
					if rc != nil {
						return rc.Layout(gtx, btn)
					}
					return btn(gtx)
				}
				// diikuti → indikator bisu (bila dibisukan) + chevron menu saat hover
				// (ala baris chat: aksi disembunyikan sampai hover, bukan tombol permanen).
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !c.muted {
							return layout.Dimensions{}
						}
						return layout.Inset{Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return icon(gtx, "mute", 16, t.Text2)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !hov || menuC == nil {
							return layout.Dimensions{}
						}
						return menuC.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return icon(gtx, "chevrondown", 18, t.Text2)
							})
						})
					}),
				)
			}),
		)
	})
}

// chnAvatar — .ch-av: lingkaran 48 avatarColor(name) + inisial 0.4*d Bold putih.
func chnAvatar(gtx layout.Context, th *material.Theme, name string, dp int) layout.Dimensions {
	d := gtx.Dp(unit.Dp(dp))
	sz := image.Pt(d, d)
	paint.FillShape(gtx.Ops, avatarColor(name), clip.Ellipse{Max: sz}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// .ch-av { font-size:18px; font-weight:600 } (fixed, tdk skala 0.4*d).
		lbl := material.Label(th, 18, initial(name))
		lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		lbl.Font.Weight = font.SemiBold
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}
