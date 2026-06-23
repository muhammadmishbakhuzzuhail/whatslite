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
	Name    string
	Sub     string // "N anggota" (grup) / presence (DM)
	Desc    string // topik grup / about kontak
	Group   bool
	Muted   bool // status bisu chat (label baris Bisukan)
	Blocked bool // status blokir kontak (label baris Blokir / Buka blokir)
	// aksi (nil = baris statis/demo): Block (DM), Leave (grup), Invite (link grup),
	// Edit (info grup: nama+deskripsi), Mute (toggle bisu), Media (galeri), Enc (info enkripsi).
	Block        *widget.Clickable
	Leave        *widget.Clickable
	Invite       *widget.Clickable
	Edit         *widget.Clickable
	Mute         *widget.Clickable
	Media        *widget.Clickable
	Enc          *widget.Clickable
	Timer        *widget.Clickable  // pesan sementara (buka picker)
	Rename       *widget.Clickable  // edit nama kontak (DM)
	Add          *widget.Clickable  // tambah anggota (grup)
	TimerLabel   string             // label aktif: "Mati" / "24 jam" / "7 hari" / "90 hari"
	Members      []InfoMember       // grup: daftar anggota
	MemberClicks []widget.Clickable // paralel Members (ketuk → menu, opsional)
}

// InfoMember = satu anggota grup di laci info (nama + admin + jid utk buka DM).
type InfoMember struct {
	Name  string
	Admin bool
	JID   string
}

func InfoDrawerView(gtx layout.Context, th *material.Theme, t Theme, d *InfoDrawerData) layout.Dimensions {
	// .info-panel { width: 400px; background: var(--sidebar-bg); }
	w := gtx.Dp(400)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	if d == nil { // demo (render standalone / gio-shot)
		d = &InfoDrawerData{Name: "Grup Kerja", Sub: "4 anggota", Desc: "Koordinasi tim proyek", Group: true,
			Members: []InfoMember{
				{Name: "Andi Pratama", Admin: true}, {Name: "Sarah", Admin: false},
				{Name: "Tim Proyek X", Admin: false}, {Name: "Rian", Admin: false},
			}}
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
		// blok umum (DM + grup) ala WhatsApp: media, bisukan, pesan sementara, enkripsi.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return infoDrawerRow(gtx, th, t, infoDrawerMediaIcon, "Media, tautan, dokumen", t.Text2, t.Text, d.Media)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := "Bisukan notifikasi"
			if d.Muted {
				lbl = "Aktifkan notifikasi"
			}
			return infoDrawerRow(gtx, th, t, infoDrawerMuteIcon, lbl, t.Text2, t.Text, d.Mute)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := "Pesan sementara"
			if d.TimerLabel != "" {
				lbl += " — " + d.TimerLabel
			}
			return infoDrawerRow(gtx, th, t, infoDrawerTimerIcon, lbl, t.Text2, t.Text, d.Timer)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return infoDrawerRow(gtx, th, t, infoDrawerLockIcon, "Enkripsi", t.Text2, t.Text, d.Enc)
		}),
		// pemisah 6px var(--wallpaper).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return infoDrawerSep(gtx, t, w)
		}),
		// baris aksi (.info-row): grup → tambah/link/keluar; DM → blokir.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !d.Group {
				return infoDrawerRow(gtx, th, t, infoDrawerEditIcon, "Edit nama kontak", t.Text2, t.Text, d.Rename)
			}
			return infoDrawerRow(gtx, th, t, infoDrawerEditIcon, "Edit info grup", t.Text2, t.Text, d.Edit)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !d.Group {
				return layout.Dimensions{}
			}
			return infoDrawerRow(gtx, th, t, infoDrawerAddIcon, "Tambah anggota", t.Text2, t.Text, d.Add)
		}),
		// daftar anggota grup (avatar + nama + lencana admin).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !d.Group || len(d.Members) == 0 {
				return layout.Dimensions{}
			}
			return infoDrawerMembers(gtx, th, t, d.Members, d.MemberClicks)
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
		// Blokir / Buka blokir (DM, merah/accent sesuai status).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if d.Group {
				return layout.Dimensions{}
			}
			lbl, col := "Blokir kontak", dangerCol
			if d.Blocked {
				lbl, col = "Buka blokir", t.Accent
			}
			return infoDrawerRow(gtx, th, t, infoDrawerBlockIcon, lbl, col, col, d.Block)
		}),
		// Laporkan (paling bawah, merah) — paritas WhatsApp.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := "Laporkan kontak"
			if d.Group {
				lbl = "Laporkan grup"
			}
			return infoDrawerRow(gtx, th, t, infoDrawerReportIcon, lbl, dangerCol, dangerCol, nil)
		}),
	)
}

// infoDrawerMembers — header "N anggota" + baris anggota (avatar 40 + nama + admin).
// clicks (paralel members) → baris bisa diketik (buka DM anggota).
func infoDrawerMembers(gtx layout.Context, th *material.Theme, t Theme, members []InfoMember, clicks []widget.Clickable) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(members)+1)
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(4), Left: unit.Dp(24), Right: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			l := material.Label(th, 13, itoa(len(members))+" anggota")
			l.Color = t.Accent
			return l.Layout(gtx)
		})
	}))
	for i := range members {
		m, idx := members[i], i
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			row := func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(24), Right: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return infoDrawerAvatar(gtx, th, m.Name, 40)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(th, 15, m.Name)
							l.Color, l.MaxLines = t.Text, 1
							return l.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if !m.Admin {
								return layout.Dimensions{}
							}
							l := material.Label(th, 12, "Admin grup")
							l.Color = t.Accent
							return l.Layout(gtx)
						}),
					)
				})
			}
			if idx < len(clicks) {
				return clicks[idx].Layout(gtx, row)
			}
			return row(gtx)
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
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
		// font-size ≈ 0.4× diameter (80px untuk .avatar.big 200).
		lbl := material.Label(th, unit.Sp(float32(dp)*0.4), initial(name))
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
	body := func(gtx layout.Context) layout.Dimensions {
		return infoDrawerRowBody(gtx, th, t, icon, label, iconCol, textCol)
	}
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

// infoDrawerEditIcon: edit info grup → ikon "editpen".
func infoDrawerEditIcon(gtx layout.Context, col color.NRGBA) {
	icon(gtx, "editpen", 22, col)
}

// infoDrawerLeaveIcon: keluar grup → ikon "leavegroup".
func infoDrawerLeaveIcon(gtx layout.Context, col color.NRGBA) {
	icon(gtx, "leavegroup", 22, col)
}

// infoDrawerMediaIcon: media, tautan, dokumen → ikon "media".
func infoDrawerMediaIcon(gtx layout.Context, col color.NRGBA) {
	icon(gtx, "media", 22, col)
}

// infoDrawerMuteIcon: bisukan notifikasi → ikon "mute".
func infoDrawerMuteIcon(gtx layout.Context, col color.NRGBA) {
	icon(gtx, "mute", 22, col)
}

// infoDrawerTimerIcon: pesan sementara → ikon "clock".
func infoDrawerTimerIcon(gtx layout.Context, col color.NRGBA) {
	icon(gtx, "clock", 22, col)
}

// infoDrawerLockIcon: enkripsi → ikon "lock".
func infoDrawerLockIcon(gtx layout.Context, col color.NRGBA) {
	icon(gtx, "lock", 22, col)
}

// infoDrawerBlockIcon: blokir/buka blokir → ikon "block".
func infoDrawerBlockIcon(gtx layout.Context, col color.NRGBA) {
	icon(gtx, "block", 22, col)
}

// infoDrawerReportIcon: laporkan → ikon "report" (bendera).
func infoDrawerReportIcon(gtx layout.Context, col color.NRGBA) {
	icon(gtx, "report", 22, col)
}
