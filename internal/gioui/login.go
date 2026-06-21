// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// login.go — layar Login/QR (paritas frontend/src/lib/Login.svelte + app.css).
// Bar atas accent "WhatsLite", kartu tengah dgn langkah bernomor + QR asli
// (github.com/skip2/go-qrcode), lalu hint. Fungsi murni, data demo baked-in.
package gioui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	qrcode "github.com/skip2/go-qrcode"
)

// LoginCtl = state interaktif layar login (alur nomor telepon). nil → render QR
// statis saja (dipakai render-tool gio-shot). Disuplai UI.loginCtl() saat hidup.
type LoginCtl struct {
	PhoneMode bool             // true = panel nomor telepon; false = QR
	Phone     *widget.Editor   // input nomor (mode telepon)
	Switch    *widget.Clickable // link toggle QR↔nomor
	Submit    *widget.Clickable // tombol "Minta kode"
	Code      string           // kode pairing 8-karakter (hasil LinkWithPhone)
}

// LoginView menggambar layar login penuh (bar accent + kartu QR + hint).
// qr = kode QR pairing mentah dari engine; "" → tampilkan placeholder + "menunggu".
func LoginView(gtx layout.Context, th *material.Theme, t Theme, qr string, ctl *LoginCtl) layout.Dimensions {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}

	// latar: var(--head-bg)
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	// .login { flex-direction: column; align-items: center } — kartu mengalir
	// dari atas (margin-top), bukan dipusatkan vertikal.
	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
		// .login-bar — align-self: stretch; height 56; accent bg; white 17/600
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return loginBar(gtx, th, t, white)
		}),
		// .login-card { margin-top: 56px; }
		layout.Rigid(layout.Spacer{Height: unit.Dp(56)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return loginCard(gtx, th, t, white, qr, ctl)
		}),
		// .login-hint { margin-top: 30px; color: text2; font-size: 14px; }
		layout.Rigid(layout.Spacer{Height: unit.Dp(30)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 14, "Buka WhatsLite di ponsel untuk memindai.")
			lbl.Color = t.Text2
			return lbl.Layout(gtx)
		}),
	)
}

// .login-bar — accent, tinggi 56, padding 0 24, teks 17/600 putih.
func loginBar(gtx layout.Context, th *material.Theme, t Theme, white color.NRGBA) layout.Dimensions {
	h := gtx.Dp(56)
	sz := image.Pt(gtx.Constraints.Max.X, h)
	paint.FillShape(gtx.Ops, t.Accent, clip.Rect{Max: sz}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Inset{Left: unit.Dp(24), Right: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 17, "WhatsLite")
			lbl.Color = white
			lbl.Font.Weight = font.SemiBold
			return lbl.Layout(gtx)
		})
	})
	return layout.Dimensions{Size: sz}
}

// .login-card — sidebarBg, radius 14, padding 44, gap 56, row align center.
func loginCard(gtx layout.Context, th *material.Theme, t Theme, white color.NRGBA, qr string, ctl *LoginCtl) layout.Dimensions {
	pad := unit.Dp(44)
	// rekam isi utk ukur, gambar latar RRect di belakang.
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: pad, Bottom: pad, Left: pad, Right: pad}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return loginLeft(gtx, th, t)
			}),
			// gap: 56px
			layout.Rigid(layout.Spacer{Width: unit.Dp(56)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return loginRight(gtx, th, t, white, qr, ctl)
			}),
		)
	})
	call := macro.Stop()
	r := gtx.Dp(14)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// .login-left — judul h2 (26/500) + langkah bernomor (15, Text, gap 16, maxw 320).
func loginLeft(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	gtx.Constraints.Max.X = gtx.Dp(320) // max-width: 320px
	steps := []string{
		"Buka WhatsApp di ponsel Anda",
		"Ketuk Menu, lalu pilih Perangkat Tertaut",
		"Ketuk Tautkan perangkat dan arahkan kamera ke layar ini",
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// h2 { font-size: 26px; font-weight: 500; margin-bottom: 22px; }
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 26, "Tautkan perangkat")
			lbl.Color = t.Text
			lbl.Font.Weight = font.Medium
			return lbl.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(22)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, len(steps)*2)
			for i, s := range steps {
				if i > 0 {
					children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout)) // gap: 16px
				}
				num, txt := i+1, s
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return loginStep(gtx, th, t, num, txt)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)
}

// satu <li> bernomor dlm <ol class=login-steps> — angka + teks 15px line-height 1.45.
func loginStep(gtx layout.Context, th *material.Theme, t Theme, n int, s string) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 15, itoa(n)+".")
			lbl.Color = t.Text
			return lbl.Layout(gtx)
		}),
		// padding-left: 22px setara jarak penanda <ol>
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 15, s)
			lbl.Color = t.Text
			lbl.LineHeight = unit.Sp(15 * 1.45) // line-height: 1.45
			return lbl.Layout(gtx)
		}),
	)
}

// .login-right — kotak QR 220 putih (padding 8) + waiting + link nomor telepon.
func loginRight(gtx layout.Context, th *material.Theme, t Theme, white color.NRGBA, qr string, ctl *LoginCtl) layout.Dimensions {
	phoneMode := ctl != nil && ctl.PhoneMode
	switchTxt := "Tautkan dengan nomor telepon"
	if phoneMode {
		switchTxt = "Gunakan kode QR"
	}
	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
		// area utama: panel nomor telepon ATAU QR (kotak 220 sama).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if phoneMode {
				return loginPhonePanel(gtx, th, t, ctl)
			}
			return loginQRBlock(gtx, th, t, white, qr)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		// .pl-switch — link accent 13px (toggle QR↔nomor). Clickable bila hidup.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := func(gtx layout.Context) layout.Dimensions {
				l := material.Label(th, 13, switchTxt)
				l.Color = t.Accent
				return l.Layout(gtx)
			}
			if ctl != nil && ctl.Switch != nil {
				return ctl.Switch.Layout(gtx, lbl)
			}
			return lbl(gtx)
		}),
	)
}

// loginQRBlock — kotak QR 220 + label status di bawahnya.
func loginQRBlock(gtx layout.Context, th *material.Theme, t Theme, white color.NRGBA, qr string) layout.Dimensions {
	waiting := "Pindai dengan ponsel Anda"
	if qr == "" {
		waiting = "Menunggu kode QR…"
	}
	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return loginQR(gtx, white, qr) }),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 13, waiting)
			lbl.Color = t.Text2
			return lbl.Layout(gtx)
		}),
	)
}

// loginPhonePanel — input nomor + tombol "Minta kode"; bila Code terisi tampilkan
// kode pairing 8-karakter besar. Lebar disamakan dgn kotak QR (220).
func loginPhonePanel(gtx layout.Context, th *material.Theme, t Theme, ctl *LoginCtl) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Dp(220)
	gtx.Constraints.Max.X = gtx.Dp(220)
	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 14, "Nomor telepon (kode negara, tanpa +)")
			lbl.Color = t.Text2
			return lbl.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		// kotak input membulat (var --search-bg)
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return loginField(gtx, th, t, ctl.Phone, "628123456789")
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		// tombol accent "Minta kode"
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th, ctl.Submit, "Minta kode")
			btn.Background = t.Accent
			btn.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			btn.CornerRadius = unit.Dp(8)
			btn.TextSize = unit.Sp(14)
			return btn.Layout(gtx)
		}),
		// kode pairing (bila ada)
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if ctl.Code == "" {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, 24, ctl.Code)
				lbl.Color = t.Text
				lbl.Font.Weight = font.SemiBold
				return lbl.Layout(gtx)
			})
		}),
	)
}

// loginField — input teks 1-baris dlm kotak membulat (var --search-bg).
func loginField(gtx layout.Context, th *material.Theme, t Theme, ed *widget.Editor, hint string) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		e := material.Editor(th, ed, hint)
		e.Color = t.Text
		e.HintColor = t.Text2
		e.TextSize = unit.Sp(14)
		return e.Layout(gtx)
	})
	call := macro.Stop()
	r := gtx.Dp(8)
	paint.FillShape(gtx.Ops, t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// .qr / .qr-img — kotak 220x220 putih, radius 8, padding 8 → QR 200px asli.
func loginQR(gtx layout.Context, white color.NRGBA, qr string) layout.Dimensions {
	box := gtx.Dp(220)
	pad := gtx.Dp(8)
	r := gtx.Dp(8)
	sz := image.Pt(box, box)
	// kotak putih membulat
	paint.FillShape(gtx.Ops, white, clip.RRect{Rect: image.Rectangle{Max: sz}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))

	// QR asli (200px), digambar di dalam clip kotak 200 setelah padding 8.
	if img := loginQRImage(qr); img != nil {
		inner := box - 2*pad
		off := op.Offset(image.Pt(pad, pad)).Push(gtx.Ops)
		area := clip.Rect{Max: image.Pt(inner, inner)}.Push(gtx.Ops)
		paint.NewImageOp(img).Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		area.Pop()
		off.Pop()
	}
	return layout.Dimensions{Size: sz}
}

// loginQRImage meng-encode kode QR `qr` (dari engine) → image.Image. Bila kosong
// (belum ada kode), pakai placeholder demo agar kotak tak kosong saat menunggu.
func loginQRImage(qr string) image.Image {
	if qr == "" {
		qr = "https://wa.me/link/demo"
	}
	pngBytes, err := qrcode.Encode(qr, qrcode.Medium, 200)
	if err != nil {
		return nil
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil
	}
	return img
}
