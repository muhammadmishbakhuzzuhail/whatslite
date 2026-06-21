// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// settingsview.go — pane Setelan (paritas SettingsPane.svelte + app.css).
// Fungsi murni mandiri (data demo dibakar di dalam) agar bisa dirender
// standalone: .pane-head, .settings-profile, lalu daftar .settings-item
// (sebagian dgn .switch, satu .danger). Semua px/warna mengikuti app.css.
package gioui

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// SettingsCtl = state interaktif pane setelan. nil → render statis (gio-shot).
// Clicks: 1 clickable per baris (urut spt setList). Dark: status tema saat ini
// (refleksi toggle "Tema").
type SettingsCtl struct {
	Dark        bool
	KeepDeleted bool // status anti-delete (toggle baris "Simpan pesan dihapus")
	Clicks      []widget.Clickable
}

// SettingsView merender pane setelan penuh ke seluruh area gtx.
func SettingsView(gtx layout.Context, th *material.Theme, t Theme, ctl *SettingsCtl) layout.Dimensions {
	// latar pane (sidebarBg seperti SettingsPane yg menempati sidebar)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return setHead(gtx, th, t)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return setProfile(gtx, th, t, "Saya", "Tentang — Hidup itu indah ✨", "#00a884")
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return setList(gtx, th, t, ctl)
		}),
	)
}

// .pane-head { height:56; padding:0 16; head-bg }  h2 { 19/600 }
func setHead(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	h := gtx.Dp(56)
	sz := image.Pt(gtx.Constraints.Max.X, h)
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Rect{Max: sz}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(19), "Setelan")
			lbl.Color = t.Text
			lbl.Font.Weight = font.SemiBold
			return lbl.Layout(gtx)
		})
	})
	return layout.Dimensions{Size: sz}
}

// .settings-profile { gap:16; padding:18 16; border-bottom 1px divider }
func setProfile(gtx layout.Context, th *material.Theme, t Theme, name, about, accent string) layout.Dimensions {
	w := gtx.Constraints.Max.X
	dims := layout.Inset{Top: unit.Dp(18), Bottom: unit.Dp(18), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return setAvatar(gtx, th, name, accent, 49)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(18), name) // sp-name 18/500
						lbl.Color = t.Text
						lbl.Font.Weight = font.Medium
						lbl.MaxLines = 1
						return lbl.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(14), about) // sp-about 14/text2
						lbl.Color = t.Text2
						lbl.MaxLines = 1
						return lbl.Layout(gtx)
					}),
				)
			}),
		)
	})
	// border-bottom 1px divider
	paint.FillShape(gtx.Ops, t.Divider, clip.Rect{Min: image.Pt(0, dims.Size.Y - gtx.Dp(1)), Max: image.Pt(w, dims.Size.Y)}.Op())
	return layout.Dimensions{Size: image.Pt(w, dims.Size.Y)}
}

// setItem = struct baris .settings-item (demo)
type setItem struct {
	name   string
	desc   string
	icon   string // nama ikon garis WhatsApp (lihat icons.go)
	hasSw  bool
	swOn   bool
	danger bool
}

func setList(gtx layout.Context, th *material.Theme, t Theme, ctl *SettingsCtl) layout.Dimensions {
	themeDesc := "Terang, gelap, atau ikuti sistem"
	themeOn := false
	if ctl != nil { // Tema jadi toggle nyata yg merefleksikan mode gelap aktif
		themeOn = ctl.Dark
		if ctl.Dark {
			themeDesc = "Mode gelap"
		} else {
			themeDesc = "Mode terang"
		}
	}
	items := []setItem{
		{name: "Tema", desc: themeDesc, icon: "theme", hasSw: ctl != nil, swOn: themeOn},
		{name: "Bahasa", desc: "Bahasa Indonesia", icon: "globe"},
		{name: "Notifikasi", desc: "Aktif", icon: "bell", hasSw: true, swOn: true},
		{name: "Simpan pesan dihapus", desc: "Lihat pesan yang ditarik pengirim", icon: "eyeoff", hasSw: true, swOn: ctl == nil || ctl.KeepDeleted},
		{name: "Retensi", desc: "Hapus pesan setelah 90 hari", icon: "disk"},
		{name: "Privasi", desc: "Terakhir dilihat, blokir, kunci aplikasi", icon: "lock"},
		{name: "Penyimpanan", desc: "Kelola ruang & data", icon: "disk"},
		{name: "Keluar", icon: "power", danger: true},
	}
	flex := layout.Flex{Axis: layout.Vertical}
	children := make([]layout.FlexChild, len(items))
	for i := range items {
		it, idx := items[i], i
		children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			row := func(gtx layout.Context) layout.Dimensions { return setRow(gtx, th, t, it) }
			if ctl != nil && idx < len(ctl.Clicks) { // baris jadi clickable
				return ctl.Clicks[idx].Layout(gtx, row)
			}
			return row(gtx)
		})
	}
	return flex.Layout(gtx, children...)
}

// .settings-item { gap:20; padding:14 20; border-bottom 1px divider }
func setRow(gtx layout.Context, th *material.Theme, t Theme, it setItem) layout.Dimensions {
	danger := color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff} // .danger #e35d6a
	w := gtx.Constraints.Max.X
	nameCol := t.Text
	icoCol := t.Text2
	if it.danger {
		nameCol = danger
		icoCol = danger
	}
	dims := layout.Inset{Top: unit.Dp(14), Bottom: unit.Dp(14), Left: unit.Dp(20), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			// ikon garis WhatsApp 24px (svg width/height 24, color text2/danger)
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if it.icon == "" {
					return layout.Dimensions{Size: image.Pt(gtx.Dp(24), gtx.Dp(24))}
				}
				return icon(gtx, it.icon, 24, icoCol)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(20)}.Layout), // gap 20
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(16), it.name) // si-name 16
						lbl.Color = nameCol
						lbl.MaxLines = 1
						return lbl.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if it.desc == "" {
							return layout.Dimensions{}
						}
						return layout.Inset{Top: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, unit.Sp(13), it.desc) // si-desc 13/text2 mt2
							lbl.Color = t.Text2
							lbl.MaxLines = 1
							return lbl.Layout(gtx)
						})
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !it.hasSw {
					return layout.Dimensions{}
				}
				return layout.Inset{Left: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return setSwitch(gtx, t, it.swOn)
				})
			}),
		)
	})
	// border-bottom 1px divider
	paint.FillShape(gtx.Ops, t.Divider, clip.Rect{Min: image.Pt(0, dims.Size.Y - gtx.Dp(1)), Max: image.Pt(w, dims.Size.Y)}.Op())
	return layout.Dimensions{Size: image.Pt(w, dims.Size.Y)}
}

// .switch { 38x22 radius12 accent } knob 18x18 white inset 2; off => text2 kiri
func setSwitch(gtx layout.Context, t Theme, on bool) layout.Dimensions {
	w := gtx.Dp(38)
	h := gtx.Dp(22)
	sz := image.Pt(w, h)
	r := gtx.Dp(12)
	track := t.Accent
	if !on {
		track = t.Text2
	}
	paint.FillShape(gtx.Ops, track, clip.RRect{Rect: image.Rectangle{Max: sz}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	// knob 18x18, top 2; on => kanan (right:2), off => kiri (left:2)
	k := gtx.Dp(18)
	pad := gtx.Dp(2)
	kx := pad
	if on {
		kx = w - pad - k
	}
	ky := pad
	knob := image.Rectangle{Min: image.Pt(kx, ky), Max: image.Pt(kx+k, ky+k)}
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, clip.Ellipse{Min: knob.Min, Max: knob.Max}.Op(gtx.Ops))
	return layout.Dimensions{Size: sz}
}

// avatar profil: lingkaran warna aksen + inisial (paritas .avatar)
func setAvatar(gtx layout.Context, th *material.Theme, name, accent string, dp int) layout.Dimensions {
	d := gtx.Dp(unit.Dp(dp))
	sz := image.Pt(d, d)
	col := hex(accent)
	paint.FillShape(gtx.Ops, col, clip.Ellipse{Max: sz}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, unit.Sp(float32(dp)*0.4), initial(name))
		lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
		lbl.Font.Weight = font.Bold
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}
