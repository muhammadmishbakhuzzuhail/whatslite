// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// infodrawerview.go — laci info grup kanan (paritas frontend/src/lib/chat/
// InfoPanel.svelte + app.css): header 56, hero avatar 200, bar pemisah
// wallpaper 6px, blok Deskripsi, lalu baris aksi (tambah/undangan/keluar).
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

// InfoDrawerView menggambar laci info grup 400px di sisi kanan (sidebarBg).
// InfoDrawerData = data nyata drawer info (nil → demo grup statis).
type InfoDrawerData struct {
	Name  string
	Sub   string // "N anggota" (grup) / presence (DM)
	Desc  string // topik grup / about kontak
	Group bool
	// aksi (nil = baris statis/demo): Block (DM), Leave (grup), Invite (link grup).
	Block  *widget.Clickable
	Leave  *widget.Clickable
	Invite *widget.Clickable
}

func InfoDrawerView(gtx layout.Context, th *material.Theme, t Theme, d *InfoDrawerData) layout.Dimensions {
	// .info-panel { width: 400px; background: var(--sidebar-bg); }
	w := gtx.Dp(400)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	if d == nil { // demo (render standalone / gio-shot)
		d = &InfoDrawerData{Name: "Grup Kerja", Sub: "4 anggota", Desc: "Koordinasi tim proyek", Group: true}
	}
	dangerCol := color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff} // #e35d6a
	headTitle := "Info kontak"
	if d.Group {
		headTitle = "Info grup"
	}
	desc := d.Desc
	if desc == "" {
		desc = "—"
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// .info-head — height 56, head-bg, title 16/500.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return infoDrawerHead(gtx, th, t, w, headTitle)
		}),
		// .info-hero — pad 28/24, avatar 200, nama 24/500, sub 15.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return infoDrawerHero(gtx, th, t, d.Name, d.Sub)
		}),
		// pemisah 6px var(--wallpaper) (border-bottom .info-hero).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return infoDrawerSep(gtx, t, w)
		}),
		// .info-block — Deskripsi/Tentang.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := "Tentang"
			if d.Group {
				lbl = "Deskripsi"
			}
			return infoDrawerBlock(gtx, th, t, lbl, desc)
		}),
		// pemisah 6px var(--wallpaper) (border-bottom .info-block).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return infoDrawerSep(gtx, t, w)
		}),
		// baris aksi (.info-row): grup → tambah/link/keluar; DM → blokir.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !d.Group {
				return infoDrawerRow(gtx, th, t, infoDrawerLeaveIcon, "Blokir kontak", dangerCol, dangerCol, d.Block)
			}
			return infoDrawerRow(gtx, th, t, infoDrawerAddIcon, "Tambah anggota", t.Text2, t.Text, nil)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !d.Group {
				return layout.Dimensions{}
			}
			return infoDrawerRow(gtx, th, t, infoDrawerLinkIcon, "Link undangan", t.Text2, t.Text, d.Invite)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !d.Group {
				return layout.Dimensions{}
			}
			return infoDrawerRow(gtx, th, t, infoDrawerLeaveIcon, "Keluar grup", dangerCol, dangerCol, d.Leave)
		}),
	)
}

// infoDrawerHead: .info-head — tinggi 56, latar head-bg, pad 0 16, title 16/500.
func infoDrawerHead(gtx layout.Context, th *material.Theme, t Theme, w int, title string) layout.Dimensions {
	h := gtx.Dp(56)
	sz := image.Pt(w, h)
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Rect{Max: sz}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 16, title)
			lbl.Color = t.Text
			lbl.Font.Weight = font.Medium
			return lbl.Layout(gtx)
		})
	})
	return layout.Dimensions{Size: sz}
}

// infoDrawerHero: .info-hero — pad 28/24, avatar 200 di tengah + nama + sub.
func infoDrawerHero(gtx layout.Context, th *material.Theme, t Theme, name, sub string) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(28), Bottom: unit.Dp(28), Left: unit.Dp(24), Right: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			// .avatar.big — 200x200, lingkaran avatarColor + inisial putih 80.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return infoDrawerAvatar(gtx, th, name, 200)
			}),
			// margin: 0 auto 16px.
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			// .info-hero .iname — 24/500.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 24, name)
				lbl.Color = t.Text
				lbl.Font.Weight = font.Medium
				return lbl.Layout(gtx)
			}),
			// .info-hero .iphone — margin-top 4, 15, text2.
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 15, sub)
				lbl.Color = t.Text2
				return lbl.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: dims.Size}.Op())
	call.Add(gtx.Ops)
	return dims
}

// infoDrawerAvatar: lingkaran avatarColor(name) + inisial putih, ukuran dp.
func infoDrawerAvatar(gtx layout.Context, th *material.Theme, name string, dp int) layout.Dimensions {
	d := gtx.Dp(unit.Dp(dp))
	sz := image.Pt(d, d)
	paint.FillShape(gtx.Ops, avatarColor(name), clip.Ellipse{Max: sz}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// font-size: 80px untuk .avatar.big.
		lbl := material.Label(th, 80, initial(name))
		lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		lbl.Font.Weight = font.SemiBold
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}

// infoDrawerSep: border-bottom 6px var(--wallpaper) sebagai bar penuh-lebar.
func infoDrawerSep(gtx layout.Context, t Theme, w int) layout.Dimensions {
	h := gtx.Dp(6)
	sz := image.Pt(w, h)
	paint.FillShape(gtx.Ops, t.Wallpaper, clip.Rect{Max: sz}.Op())
	return layout.Dimensions{Size: sz}
}

// infoDrawerBlock: .info-block — pad 14/24, lbl accent 13 (mb 5) + val 15 text.
func infoDrawerBlock(gtx layout.Context, th *material.Theme, t Theme, label, value string) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(14), Bottom: unit.Dp(14), Left: unit.Dp(24), Right: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 13, label)
				lbl.Color = t.Accent
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 15, value)
				lbl.Color = t.Text
				return lbl.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: dims.Size}.Op())
	call.Add(gtx.Ops)
	return dims
}

// infoDrawerRow: .info-row — pad 14/24, gap 18, ikon 22 + label 15.
func infoDrawerRow(gtx layout.Context, th *material.Theme, t Theme, icon func(layout.Context, color.NRGBA), label string, iconCol, textCol color.NRGBA, c *widget.Clickable) layout.Dimensions {
	body := func(gtx layout.Context) layout.Dimensions { return infoDrawerRowBody(gtx, th, t, icon, label, iconCol, textCol) }
	if c != nil {
		return c.Layout(gtx, body)
	}
	return body(gtx)
}

func infoDrawerRowBody(gtx layout.Context, th *material.Theme, t Theme, icon func(layout.Context, color.NRGBA), label string, iconCol, textCol color.NRGBA) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(14), Bottom: unit.Dp(14), Left: unit.Dp(24), Right: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return infoDrawerIconBox(gtx, icon, iconCol)
			}),
			// gap: 18px.
			layout.Rigid(layout.Spacer{Width: unit.Dp(18)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 15, label)
				lbl.Color = textCol
				lbl.MaxLines = 1
				return lbl.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: dims.Size}.Op())
	call.Add(gtx.Ops)
	return dims
}

// infoDrawerIconBox: kotak ikon 22x22 (.info-row svg) — gambar ikon WhatsApp via helper.
func infoDrawerIconBox(gtx layout.Context, draw func(layout.Context, color.NRGBA), col color.NRGBA) layout.Dimensions {
	d := gtx.Dp(22)
	draw(gtx, col)
	return layout.Dimensions{Size: image.Pt(d, d)}
}

// ---- ikon 22 (raster SVG WhatsApp via icon(), warna col) ----

// infoDrawerAddIcon: tambah anggota → ikon "addmember".
func infoDrawerAddIcon(gtx layout.Context, col color.NRGBA) {
	icon(gtx, "addmember", 22, col)
}

// infoDrawerLinkIcon: link undangan → ikon "invitelink".
func infoDrawerLinkIcon(gtx layout.Context, col color.NRGBA) {
	icon(gtx, "invitelink", 22, col)
}

// infoDrawerLeaveIcon: keluar grup → ikon "leavegroup".
func infoDrawerLeaveIcon(gtx layout.Context, col color.NRGBA) {
	icon(gtx, "leavegroup", 22, col)
}
