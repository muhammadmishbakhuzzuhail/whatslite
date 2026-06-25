// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// sidepanesview.go — sidebar pane CALLS (paritas frontend/src/lib/sidebar/
// CallsPane.svelte + app.css): .sidebar-head 60px ("Panggilan" 23/Bold), lalu
// daftar baris panggilan — avatar 49 + nama + sub-baris (panah arah + label,
// merah utk tak terjawab) + ikon panggil accent kanan. Fungsi murni, data demo.
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
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// spCall = satu baris panggilan.
type spCall struct {
	id     string // id log (utk hapus); "" = demo
	jid    string // kontak (utk tap → buka chat)
	name   string
	time   string
	video  bool
	missed bool
}

// spCallTag — tag pointer per baris panggilan (klik-kanan → menu konteks).
type spCallTag struct{ i int }

// SidePanesView menggambar sidebar 380px (t.SidebarBg) berisi pane CALLS:
// header .pane-head + 4 baris panggilan demo. Fungsi murni, mandiri (standalone).
func SidePanesView(gtx layout.Context, th *material.Theme, t Theme, calls []spCall, clicks []widget.Clickable, onCtx func(idx int), top func(gtx layout.Context) layout.Dimensions, list *widget.List) layout.Dimensions {
	w := gtx.Dp(468)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())

	demo := calls == nil
	if demo { // data demo (render standalone / gio-shot)
		calls = []spCall{
			{name: "Andi Pratama", time: "19.08", video: true, missed: true},
			{name: "Keluarga", time: "18.41", video: false, missed: false},
			{name: "Sarah", time: "17.55", video: false, missed: true},
			{name: "Tim Proyek X", time: "16.20", video: true, missed: false},
		}
	}

	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return spPaneHead(gtx, th, t, w, "Panggilan")
		}),
		// area cari + filter + bersihkan (dirender ui.go; demo/headless → nil).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if top == nil {
				return layout.Dimensions{}
			}
			return top(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(calls) == 0 { // empty-state modern: ikon + teks muted terpusat
				return spEmptyState(gtx, th, t, "calls", "Belum ada panggilan", "Riwayat panggilan masuk muncul di sini.")
			}
			// daftar baris: padding lega, hover membulat lembut (gaya kartu modern).
			// material.List → gulir bila banyak baris.
			if list != nil {
				list.Axis = layout.Vertical
			}
			row := func(gtx layout.Context, idx int) layout.Dimensions {
				c := calls[idx]
				var rc *widget.Clickable
				if idx < len(clicks) {
					rc = &clicks[idx]
				}
				body := func(gtx layout.Context) layout.Dimensions {
					macro := op.Record(gtx.Ops)
					dims := spCallRow(gtx, th, t, c)
					call := macro.Stop()
					if rc != nil && rc.Hovered() { // hover membulat (modern)
						rr := gtx.Dp(10)
						paint.FillShape(gtx.Ops, t.Hover, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
					}
					// klik-kanan baris → menu konteks (Hapus / Bersihkan). Demo → non-interaktif.
					if onCtx != nil && c.id != "" {
						tag := spCallTag{idx}
						for {
							ev, ok := gtx.Event(pointer.Filter{Target: tag, Kinds: pointer.Press})
							if !ok {
								break
							}
							if pe, ok := ev.(pointer.Event); ok && pe.Buttons.Contain(pointer.ButtonSecondary) {
								onCtx(idx)
							}
						}
						area := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
						event.Op(gtx.Ops, tag)
						call.Add(gtx.Ops)
						area.Pop()
					} else {
						call.Add(gtx.Ops)
					}
					return dims
				}
				if rc != nil {
					return rc.Layout(gtx, body)
				}
				return body(gtx)
			}
			ins := layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(8), Right: unit.Dp(8)}
			if list == nil { // demo/headless → tanpa list state, render Flex biasa
				return ins.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					children := make([]layout.FlexChild, 0, len(calls))
					for i := range calls {
						idx := i
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return row(gtx, idx) }))
					}
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
				})
			}
			return ins.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.List(th, list).Layout(gtx, len(calls), row)
			})
		}),
	)
}

// spEmptyState — keadaan kosong modern (ikon garis besar + judul + subteks muted)
// terpusat di area pane. Dipakai pane Panggilan/Status saat tak ada data.
func spEmptyState(gtx layout.Context, th *material.Theme, t Theme, ico, title, sub string) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon(gtx, ico, 56, t.Text2)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(th, 16, title)
				l.Color, l.Font.Weight, l.Alignment = t.Text, font.Medium, text.Middle
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = gtx.Dp(300)
				l := material.Label(th, 13.5, sub)
				l.Color, l.Alignment = t.Text2, text.Middle
				return l.Layout(gtx)
			}),
		)
	})
}

// spPaneHead — .sidebar-head { height: 60px; padding: 0 18px; background: head-bg;
// border-bottom: 1px solid divider } ; h1 23/Bold (letter-spacing -.3px).
func spPaneHead(gtx layout.Context, th *material.Theme, t Theme, w int, title string) layout.Dimensions {
	return paneHead(gtx, th, t, w, title)
}

// spCallRow — .chat-row { padding: 10px 12px; gap: 13px } : avatar 49 + kolom
// (nama 16.5/Medium + sub-baris panah+label) + ikon panggil accent kanan.
func spCallRow(gtx layout.Context, th *material.Theme, t Theme, c spCall) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return spAvatar(gtx, th, t, c.name, 49)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(13)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					// .row-top : nama (flex 1) + waktu (text2 12)
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(th, 16.5, c.name) // .row-name 16.5px/500
								lbl.Color = t.Text
								lbl.MaxLines = 1
								lbl.Font.Weight = font.Medium
								return lbl.Layout(gtx)
							}),
							// .row-time { margin-left: 8px }
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(th, 12, c.time) // .row-time 12px
								lbl.Color = t.Text2
								return lbl.Layout(gtx)
							}),
						)
					}),
					// .row-bottom { margin-top: 2px }
					layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return spCallLine(gtx, th, t, c)
					}),
				)
			}),
			// ikon panggil accent kanan (telepon / kamera tergantung tipe)
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return spCallIcon(gtx, t, c.video)
			}),
		)
	})
}

// spCallLine — .call-line { gap: 6px } : panah arah kecil + label.
// Tak terjawab → merah #e35d6a; selain itu → text2.
func spCallLine(gtx layout.Context, th *material.Theme, t Theme, c spCall) layout.Dimensions {
	col := t.Text2
	if c.missed {
		col = color.NRGBA{R: 0xef, G: 0x53, B: 0x50, A: 0xff} // .call-line.missed #ef5350
	}
	var label string
	if c.video {
		label = "Panggilan video · "
	} else {
		label = "Panggilan suara · "
	}
	if c.missed {
		label += "Tak terjawab"
	} else {
		label += "Masuk"
	}
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return spArrow(gtx, col, c.missed)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 14, label)
			lbl.Color = col
			lbl.MaxLines = 1
			return lbl.Layout(gtx)
		}),
	)
}

// spArrow — .call-ico 15x15 : panah arah panggilan (callArrowOut, paritas SVG
// path M7 17L17 7M17 7H9M17 7v8). Warna mengikuti garis (currentColor / merah
// utk tak terjawab).
func spArrow(gtx layout.Context, col color.NRGBA, missed bool) layout.Dimensions {
	// whatsmeow = signaling-only → semua panggilan MASUK (panah masuk); merah saat
	// tak terjawab (warna sudah diset pemanggil), accent saat terjawab.
	_ = missed
	return icon(gtx, "callArrowIn", 15, col)
}

// spCallIcon — ikon panggil accent di kanan baris (.icon-btn ~40, glyph accent).
// Pakai ikon native "calls" (gagang telepon WhatsApp) ber-tint accent, 20dp glyph
// di tengah kotak 40dp.
func spCallIcon(gtx layout.Context, t Theme, video bool) layout.Dimensions {
	glyph := "calls" // gagang telepon (suara)
	if video {
		glyph = "video" // panggilan video → ikon kamera
	}
	box := gtx.Dp(40)
	sz := image.Pt(box, box)
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Text2 (bukan accent) → indikator JENIS panggilan, bukan tombol palsu
		// (kita tak bisa originasi panggilan; tap baris = buka chat).
		return icon(gtx, glyph, 20, t.Text2)
	})
	return layout.Dimensions{Size: sz}
}

// spAvatar — .avatar 49 : lingkaran avatarColor(name) + inisial putih (paritas
// u.avatar di ui.go: font 0.4*d, Bold, putih, di tengah).
func spAvatar(gtx layout.Context, th *material.Theme, t Theme, name string, dp int) layout.Dimensions {
	_ = t
	d := gtx.Dp(unit.Dp(dp))
	sz := image.Pt(d, d)
	col := avatarColor(name)
	// rekam latar lingkaran lalu inisial di tengah (tanpa op import bila tak perlu).
	paint.FillShape(gtx.Ops, col, clip.Ellipse{Max: sz}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th, unit.Sp(float32(dp)*0.4), initial(name))
		lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		lbl.Font.Weight = font.Bold
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}
