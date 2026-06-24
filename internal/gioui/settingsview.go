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
	"gioui.org/op"
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
	// sub-pane navigasi
	Sub          string // ""|profile|storage
	Back         *widget.Clickable
	ProfileClick *widget.Clickable
	PhotoClick   *widget.Clickable                                                    // ketuk avatar profil → ganti foto
	Avatar       func(gtx layout.Context, name, jid string, dp int) layout.Dimensions // foto profil asli (nil → inisial)
	SelfJID      string                                                               // jid sendiri (utk foto asli)
	Retention    int                                                                  // hari retensi (0 = selamanya) — baris "Retensi"
	AppLock      bool                                                                 // PIN kunci aplikasi aktif? — baris "Kunci aplikasi"
	// data sub-pane
	ProfName, ProfAbout, ProfPhone string
	ProfNameEd, ProfAboutEd        *widget.Editor // edit profil (nil = read-only)
	ProfUsernameEd                 *widget.Editor // nama pengguna (@handle)
	ProfUsernameErr                string         // pesan validasi username
	ProfSave                       *widget.Clickable
	AboutClicks                    []widget.Clickable // chip saran "Tentang" (paralel aboutPresets)
	AboutOpen                      bool               // dropdown saran terbuka?
	AboutToggle                    *widget.Clickable  // tombol chevron buka/tutup saran
	StoreDB, StoreMedia            int64
	StoreMsgs                      int
	Privacy                        map[string]string // nama setelan → nilai
	PrivacyClicks                  []widget.Clickable
	Notifications                  bool               // toggle baris "Notifikasi" (persist)
	Language                       string             // kode bahasa UI aktif ("id"/"en"/…)
	LangClicks                     []widget.Clickable // baris pemilih bahasa (sub-pane)
}

// langOrder — pilihan bahasa UI (kode + label). Indeks = indeks LangClicks.
var langOrder = []struct{ code, label string }{
	{"id", "Bahasa Indonesia"}, {"en", "English"}, {"ms", "Bahasa Melayu"},
	{"es", "Español"}, {"ar", "العربية"},
}

// langLabel — label tampil dari kode bahasa (default Indonesia).
func langLabel(code string) string {
	for _, l := range langOrder {
		if l.code == code {
			return l.label
		}
	}
	return "Bahasa Indonesia"
}

// SettingsView merender pane setelan penuh ke seluruh area gtx.
func SettingsView(gtx layout.Context, th *material.Theme, t Theme, ctl *SettingsCtl) layout.Dimensions {
	// latar pane (sidebarBg seperti SettingsPane yg menempati sidebar)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	if ctl != nil && ctl.Sub != "" { // sub-pane (profil / penyimpanan)
		return setSubPane(gtx, th, t, ctl)
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return setHead(gtx, th, t)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			card := func(gtx layout.Context) layout.Dimensions {
				return setProfile(gtx, th, t, "Saya", "Tentang — Hidup itu indah ✨", "#00a884")
			}
			if ctl != nil && ctl.ProfileClick != nil { // kartu profil bisa diklik
				return ctl.ProfileClick.Layout(gtx, card)
			}
			return card(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return setList(gtx, th, t, ctl)
		}),
	)
}

// setSubPane — sub-pane setelan dgn header + tombol kembali.
func setSubPane(gtx layout.Context, th *material.Theme, t Theme, ctl *SettingsCtl) layout.Dimensions {
	title := "Profil"
	switch ctl.Sub {
	case "storage":
		title = "Penyimpanan"
	case "privacy":
		title = "Privasi"
	case "account":
		title = "Akun"
	case "help":
		title = "Bantuan"
	case "language":
		title = "Bahasa"
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return setSubHead(gtx, th, t, title, ctl.Back)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			switch ctl.Sub {
			case "storage":
				return setStoragePane(gtx, th, t, ctl)
			case "privacy":
				return setPrivacyPane(gtx, th, t, ctl)
			case "account":
				return setAccountPane(gtx, th, t, ctl)
			case "help":
				return setHelpPane(gtx, th, t, ctl)
			case "language":
				return setLanguagePane(gtx, th, t, ctl)
			}
			return setProfilePane(gtx, th, t, ctl)
		}),
	)
}

// privacyOrder — urutan + label baris privasi (indeks = indeks clickable).
var privacyOrder = []struct{ key, label string }{
	{"lastseen", "Terakhir dilihat"}, {"online", "Online"}, {"profile", "Foto profil"},
	{"about", "Tentang"}, {"status", "Status"}, {"readreceipts", "Laporan dibaca"},
	{"groupadd", "Grup"}, {"calladd", "Panggilan"},
}

// setPrivacyPane — daftar setelan privasi (label + nilai). Ketuk baris → siklus nilai.
func setPrivacyPane(gtx layout.Context, th *material.Theme, t Theme, ctl *SettingsCtl) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(privacyOrder))
	for i := range privacyOrder {
		o := privacyOrder[i]
		val, ok := ctl.Privacy[o.key]
		if !ok {
			continue
		}
		idx := i
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			row := func(gtx layout.Context) layout.Dimensions {
				return setProfileField(gtx, th, t, o.label, privValue(val))
			}
			if idx < len(ctl.PrivacyClicks) {
				return ctl.PrivacyClicks[idx].Layout(gtx, row)
			}
			return row(gtx)
		}))
	}
	if len(children) == 0 {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return setProfileField(gtx, th, t, "Privasi", "—")
		}))
	}
	return layout.Inset{Top: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

// setLanguagePane — daftar bahasa UI; baris aktif diberi centang. Ketuk → pilih.
// Catatan: teks aplikasi saat ini Indonesia; pilihan disimpan utk i18n mendatang.
func setLanguagePane(gtx layout.Context, th *material.Theme, t Theme, ctl *SettingsCtl) layout.Dimensions {
	cur := "id"
	if ctl != nil && ctl.Language != "" {
		cur = ctl.Language
	}
	children := make([]layout.FlexChild, 0, len(langOrder))
	for i := range langOrder {
		l, idx := langOrder[i], i
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			row := func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(14), Bottom: unit.Dp(14), Left: unit.Dp(20), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, 15, l.label)
							lbl.Color, lbl.MaxLines = t.Text, 1
							return lbl.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if l.code != cur {
								return layout.Dimensions{}
							}
							return icon(gtx, "check", 18, t.Accent)
						}),
					)
				})
			}
			if ctl != nil && idx < len(ctl.LangClicks) {
				return ctl.LangClicks[idx].Layout(gtx, row)
			}
			return row(gtx)
		}))
	}
	return layout.Inset{Top: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

// retentionDesc — keterangan baris Retensi dari nilai aktif (ctl nil = demo 90 hari).
func retentionDesc(ctl *SettingsCtl) string {
	d := 90
	if ctl != nil {
		d = ctl.Retention
	}
	if d <= 0 {
		return "Simpan pesan selamanya"
	}
	return "Hapus pesan setelah " + itoa(d) + " hari"
}

// lockDesc — keterangan baris Kunci aplikasi (aktif/nonaktif).
func lockDesc(ctl *SettingsCtl) string {
	if ctl != nil && ctl.AppLock {
		return "Aktif — ketuk untuk mengelola"
	}
	return "Nonaktif — ketuk untuk mengatur PIN"
}

// nextRetention — siklus retensi 90→180→365→0(selamanya)→30→90.
func nextRetention(d int) int {
	switch d {
	case 30:
		return 90
	case 90:
		return 180
	case 180:
		return 365
	case 365:
		return 0
	default: // 0 (selamanya) / lainnya
		return 30
	}
}

// nextPrivacy — siklus nilai privasi all→contacts→none→all.
func nextPrivacy(cur string) string {
	switch cur {
	case "all":
		return "contacts"
	case "contacts":
		return "none"
	default:
		return "all"
	}
}

// privValue — terjemahkan nilai privasi WA ke Indonesia.
func privValue(v string) string {
	switch v {
	case "all":
		return "Semua orang"
	case "contacts":
		return "Kontak saya"
	case "contact_blacklist":
		return "Kontak saya kecuali…"
	case "none":
		return "Tidak ada"
	case "match_last_seen":
		return "Sama spt terakhir dilihat"
	}
	return v
}

// setSubHead — header sub-pane: ikon back + judul.
func setSubHead(gtx layout.Context, th *material.Theme, t Theme, title string, back *widget.Clickable) layout.Dimensions {
	h := gtx.Dp(56)
	sz := image.Pt(gtx.Constraints.Max.X, h)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.Rect{Max: sz}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Inset{Left: unit.Dp(12), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				b := func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return icon(gtx, "back", 22, t.Text)
					})
				}
				if back != nil {
					return back.Layout(gtx, b)
				}
				return b(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(19), title)
				lbl.Color = t.Text
				lbl.Font.Weight = font.SemiBold
				return lbl.Layout(gtx)
			}),
		)
	})
	return layout.Dimensions{Size: sz}
}

// setProfilePane — avatar besar + nama + tentang + nomor (data GetProfile).
func setProfilePane(gtx layout.Context, th *material.Theme, t Theme, ctl *SettingsCtl) layout.Dimensions {
	name := ctl.ProfName
	if name == "" {
		name = "Saya"
	}
	editable := ctl.ProfNameEd != nil && ctl.ProfAboutEd != nil
	return layout.Inset{Top: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X // avatar TERPUSAT (baris lain lebar-penuh)
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					av := func(gtx layout.Context) layout.Dimensions {
						return setProfileAvatar(gtx, th, t, ctl, name, "#00a884", 120)
					}
					if ctl.PhotoClick != nil {
						return ctl.PhotoClick.Layout(gtx, av)
					}
					return av(gtx)
				})
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if editable {
					return setEditField(gtx, th, t, "Nama", ctl.ProfNameEd)
				}
				return setProfileField(gtx, th, t, "Nama", name)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if editable {
					return setAboutField(gtx, th, t, ctl) // editor + chevron toggle saran
				}
				return setProfileField(gtx, th, t, "Tentang", orDash(ctl.ProfAbout))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				// Saran = DROPDOWN: muncul hanya saat dibuka (klik chevron), bisa ditutup.
				if !editable || len(ctl.AboutClicks) == 0 || !ctl.AboutOpen {
					return layout.Dimensions{}
				}
				return setAboutPresets(gtx, th, t, ctl)
			}),
			// Nama pengguna (@handle) — fitur baru WhatsApp: kontak tanpa nomor.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if ctl.ProfUsernameEd == nil {
					return layout.Dimensions{}
				}
				return setEditField(gtx, th, t, "Nama pengguna", ctl.ProfUsernameEd)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if ctl.ProfUsernameEd == nil {
					return layout.Dimensions{}
				}
				msg, col := "Orang bisa hubungi kamu via nama pengguna tanpa nomor.", t.Text2
				if ctl.ProfUsernameErr != "" {
					msg, col = ctl.ProfUsernameErr, color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff}
				}
				return layout.Inset{Left: unit.Dp(20), Right: unit.Dp(20), Top: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(th, 12, msg)
					l.Color = col
					return l.Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return setProfileField(gtx, th, t, "Telepon", orDash(ctl.ProfPhone))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !editable || ctl.ProfSave == nil {
					return layout.Dimensions{}
				}
				return layout.Inset{Top: unit.Dp(14), Left: unit.Dp(20), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					btn := material.Button(th, ctl.ProfSave, "Simpan")
					btn.Background = t.Accent
					btn.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
					btn.CornerRadius = unit.Dp(8)
					btn.TextSize = unit.Sp(14)
					return btn.Layout(gtx)
				})
			}),
		)
	})
}

// setEditField — label accent + input teks membulat (var --search-bg).
func setEditField(gtx layout.Context, th *material.Theme, t Theme, label string, ed *widget.Editor) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(20), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(th, 13, label)
				l.Color = t.Accent
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				macro := op.Record(gtx.Ops)
				dims := layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					e := material.Editor(th, ed, "")
					e.Color = t.Text
					e.HintColor = t.Text2
					e.TextSize = unit.Sp(15)
					return e.Layout(gtx)
				})
				call := macro.Stop()
				r := gtx.Dp(8)
				paint.FillShape(gtx.Ops, t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
				call.Add(gtx.Ops)
				return dims
			}),
		)
	})
}

// setAboutField — field "Tentang": editor + tombol chevron kanan (buka/tutup
// dropdown saran). Chevron berubah arah sesuai AboutOpen.
func setAboutField(gtx layout.Context, th *material.Theme, t Theme, ctl *SettingsCtl) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(20), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(th, 13, "Tentang")
				l.Color = t.Accent
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				macro := op.Record(gtx.Ops)
				dims := layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							e := material.Editor(th, ctl.ProfAboutEd, "")
							e.Color, e.HintColor, e.TextSize = t.Text, t.Text2, unit.Sp(15)
							return e.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							ico := "chevrondown"
							if ctl.AboutOpen {
								ico = "chevronup"
							}
							btn := func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return icon(gtx, ico, 20, t.Text2)
								})
							}
							if ctl.AboutToggle != nil {
								return ctl.AboutToggle.Layout(gtx, btn)
							}
							return btn(gtx)
						}),
					)
				})
				call := macro.Stop()
				r := gtx.Dp(8)
				paint.FillShape(gtx.Ops, t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
				call.Add(gtx.Ops)
				return dims
			}),
		)
	})
}

func setProfileField(gtx layout.Context, th *material.Theme, t Theme, label, val string) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(20), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(th, 13, label)
				l.Color = t.Accent
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(3)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(th, 15.5, val)
				l.Color = t.Text
				return l.Layout(gtx)
			}),
		)
	})
}

// aboutPresets — frasa "Tentang" bawaan WhatsApp (Indonesia). Indeks = indeks
// AboutClicks. Ketuk → isi editor Tentang.
var aboutPresets = []string{
	"Tersedia", "Sibuk", "Di sekolah", "Di bioskop", "Sedang bekerja",
	"Baterai lemah", "Tidak bisa bicara, WhatsApp saja", "Dalam rapat",
	"Di gym", "Tidur", "Hanya panggilan darurat",
}

// setAboutPresets — daftar saran "Tentang" (tappable) di bawah editor, ala
// layar edit Tentang WhatsApp. Ketuk baris → set teks editor (lihat handleSettings).
func setAboutPresets(gtx layout.Context, th *material.Theme, t Theme, ctl *SettingsCtl) layout.Dimensions {
	n := len(aboutPresets)
	if len(ctl.AboutClicks) < n {
		n = len(ctl.AboutClicks)
	}
	children := make([]layout.FlexChild, 0, n+1)
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(20), Top: unit.Dp(6), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			l := material.Label(th, 13, "Pilih saran")
			l.Color = t.Text2
			return l.Layout(gtx)
		})
	}))
	for i := 0; i < n; i++ {
		txt := aboutPresets[i]
		idx := i
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			row := func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(20), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					l := material.Label(th, 15, txt)
					l.Color, l.MaxLines = t.Text, 1
					return l.Layout(gtx)
				})
			}
			return ctl.AboutClicks[idx].Layout(gtx, row)
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// setStoragePane — ringkasan penyimpanan (DB, media, jumlah pesan).
func setStoragePane(gtx layout.Context, th *material.Theme, t Theme, ctl *SettingsCtl) layout.Dimensions {
	rows := []struct{ label, val string }{
		{"Basis data", setBytes(ctl.StoreDB)},
		{"Media", setBytes(ctl.StoreMedia)},
		{"Total pesan", itoa(ctl.StoreMsgs)},
	}
	children := make([]layout.FlexChild, 0, len(rows))
	for i := range rows {
		r := rows[i]
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return setProfileField(gtx, th, t, r.label, r.val)
		}))
	}
	return layout.Inset{Top: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

// setAccountPane — bagian "Akun" ala WhatsApp: nomor + status keamanan akun.
// Baris informatif (read-only) — paritas Settings ▸ Account.
func setAccountPane(gtx layout.Context, th *material.Theme, t Theme, ctl *SettingsCtl) layout.Dimensions {
	rows := []struct{ label, val string }{
		{"Telepon", orDash(ctl.ProfPhone)},
		{"Verifikasi dua langkah", "Nonaktif"},
		{"Notifikasi keamanan", "Tampilkan di komputer ini"},
		{"Minta info akun", "Buat laporan data akun Anda"},
	}
	return setInfoRows(gtx, th, t, rows)
}

// setHelpPane — bagian "Bantuan" ala WhatsApp: pusat bantuan, ketentuan, lisensi,
// versi. Baris informatif (read-only) — paritas Settings ▸ Help.
func setHelpPane(gtx layout.Context, th *material.Theme, t Theme, ctl *SettingsCtl) layout.Dimensions {
	rows := []struct{ label, val string }{
		{"Pusat Bantuan", "faq.whatsapp.com"},
		{"Ketentuan & Kebijakan Privasi", "Baca ketentuan layanan"},
		{"Lisensi", "GPL-3.0-or-later"},
		{"Versi aplikasi", "WhatsLite (Gio)"},
	}
	return setInfoRows(gtx, th, t, rows)
}

// setInfoRows — daftar baris label+nilai (read-only) dgn inset atas, dipakai
// pane Akun/Bantuan/Penyimpanan.
func setInfoRows(gtx layout.Context, th *material.Theme, t Theme, rows []struct{ label, val string }) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(rows))
	for i := range rows {
		r := rows[i]
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return setProfileField(gtx, th, t, r.label, r.val)
		}))
	}
	return layout.Inset{Top: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func setBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return itoa(int(n>>20)) + " MB"
	case n >= 1<<10:
		return itoa(int(n>>10)) + " KB"
	default:
		return itoa(int(n)) + " B"
	}
}

// .pane-head { height:56; padding:0 16; head-bg }  h2 { 19/600 }
func setHead(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	// pakai header pane kanonik (sama spt "Chat"/"Komunitas") agar judul konsisten.
	return paneHead(gtx, th, t, gtx.Constraints.Max.X, "Setelan")
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
	paint.FillShape(gtx.Ops, t.Divider, clip.Rect{Min: image.Pt(0, dims.Size.Y-gtx.Dp(1)), Max: image.Pt(w, dims.Size.Y)}.Op())
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

// notifDesc — keterangan baris Notifikasi (Aktif/Nonaktif).
func notifDesc(ctl *SettingsCtl) string {
	if ctl == nil || ctl.Notifications {
		return "Aktif"
	}
	return "Nonaktif"
}

// langDesc — keterangan baris Bahasa (label bahasa aktif).
func langDesc(ctl *SettingsCtl) string {
	if ctl == nil {
		return "Bahasa Indonesia"
	}
	return langLabel(ctl.Language)
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
	// Urutan mengikuti WhatsApp Settings: Akun, Privasi, Notifikasi, Tema,
	// Bahasa, Penyimpanan, lalu fitur WhatsLite (Retensi, Simpan-dihapus,
	// Kunci), Bantuan, Keluar. Indeks = indeks clickable (lihat handleSettings).
	items := []setItem{
		{name: "Akun", desc: "Notifikasi keamanan, info akun", icon: "info"},                                                                        // 0
		{name: "Privasi", desc: "Terakhir dilihat, blokir, kunci aplikasi", icon: "lock"},                                                           // 1
		{name: "Notifikasi", desc: notifDesc(ctl), icon: "bell", hasSw: true, swOn: ctl == nil || ctl.Notifications},                                // 2
		{name: "Tema", desc: themeDesc, icon: "theme", hasSw: ctl != nil, swOn: themeOn},                                                            // 3
		{name: "Bahasa", desc: langDesc(ctl), icon: "globe"},                                                                                        // 4
		{name: "Penyimpanan", desc: "Kelola ruang & data", icon: "disk"},                                                                            // 5
		{name: "Retensi", desc: retentionDesc(ctl), icon: "clock"},                                                                                  // 6
		{name: "Simpan pesan dihapus", desc: "Lihat pesan yang ditarik pengirim", icon: "eyeoff", hasSw: true, swOn: ctl == nil || ctl.KeepDeleted}, // 7
		{name: "Kunci aplikasi", desc: lockDesc(ctl), icon: "lock"},                                                                                 // 8
		{name: "Bantuan", desc: "Pusat bantuan, ketentuan, lisensi", icon: "info"},                                                                  // 9
		{name: "Keluar", icon: "power", danger: true},                                                                                               // 10
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
	paint.FillShape(gtx.Ops, t.Divider, clip.Rect{Min: image.Pt(0, dims.Size.Y-gtx.Dp(1)), Max: image.Pt(w, dims.Size.Y)}.Op())
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

// setProfileAvatar — avatar besar + lencana kamera bulat di pojok kanan-bawah
// (paritas WhatsApp: ketuk foto utk ganti). Stack: avatar lalu badge di sudut.
func setProfileAvatar(gtx layout.Context, th *material.Theme, t Theme, ctl *SettingsCtl, name, accent string, dp int) layout.Dimensions {
	bd := gtx.Dp(34) // diameter lencana kamera
	return layout.Stack{Alignment: layout.SE}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if ctl != nil && ctl.Avatar != nil { // foto profil asli (bulat), else inisial
				return ctl.Avatar(gtx, name, ctl.SelfJID, dp)
			}
			return setAvatar(gtx, th, name, accent, dp)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			sz := image.Pt(bd, bd)
			// cincin warna pane di belakang agar lencana "menempel" rapi
			rd := gtx.Dp(3)
			ring := image.Pt(bd+rd*2, bd+rd*2)
			paint.FillShape(gtx.Ops, t.SidebarBg, clip.Ellipse{Max: ring}.Op(gtx.Ops))
			off := op.Offset(image.Pt(rd, rd)).Push(gtx.Ops)
			paint.FillShape(gtx.Ops, t.Accent, clip.Ellipse{Max: sz}.Op(gtx.Ops))
			gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
			layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return icon(gtx, "camera", 18, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
			})
			off.Pop()
			return layout.Dimensions{Size: ring}
		}),
	)
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
