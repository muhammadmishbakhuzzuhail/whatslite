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
	"gioui.org/widget/material"

	qrcode "github.com/skip2/go-qrcode"
)

// LoginView menggambar layar login penuh (bar accent + kartu QR + hint).
func LoginView(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}

	// latar: var(--head-bg)
	paint.FillShape(gtx.Ops, t.HeadBg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
		// .login-bar — align-self: stretch; height 56; accent bg; white 17/600
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return loginBar(gtx, th, t, white)
		}),
		// .login-card + .login-hint dipusatkan pada sisa ruang
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return loginCard(gtx, th, t, white)
					}),
					// .login-hint { margin-top: 30px; color: text2; font-size: 14px; }
					layout.Rigid(layout.Spacer{Height: unit.Dp(30)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, 14, "Buka WhatsLite di ponsel untuk memindai.")
						lbl.Color = t.Text2
						return lbl.Layout(gtx)
					}),
				)
			})
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
func loginCard(gtx layout.Context, th *material.Theme, t Theme, white color.NRGBA) layout.Dimensions {
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
				return loginRight(gtx, th, t, white)
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
func loginRight(gtx layout.Context, th *material.Theme, t Theme, white color.NRGBA) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return loginQR(gtx, white)
		}),
		// gap: 12px
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		// .login-waiting { font-size: 13px; color: text2; }
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 13, "Menunggu kode QR…")
			lbl.Color = t.Text2
			return lbl.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		// .pl-switch — link accent 13px ("Tautkan dengan nomor telepon").
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, 13, "Tautkan dengan nomor telepon")
			lbl.Color = t.Accent
			return lbl.Layout(gtx)
		}),
	)
}

// .qr / .qr-img — kotak 220x220 putih, radius 8, padding 8 → QR 200px asli.
func loginQR(gtx layout.Context, white color.NRGBA) layout.Dimensions {
	box := gtx.Dp(220)
	pad := gtx.Dp(8)
	r := gtx.Dp(8)
	sz := image.Pt(box, box)
	// kotak putih membulat
	paint.FillShape(gtx.Ops, white, clip.RRect{Rect: image.Rectangle{Max: sz}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))

	// QR asli (200px), digambar di dalam clip kotak 200 setelah padding 8.
	if img := loginQRImage(); img != nil {
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

// loginQRImage menghasilkan QR PNG demo lalu mendekodenya ke image.Image.
func loginQRImage() image.Image {
	pngBytes, err := qrcode.Encode("https://wa.me/link/demo", qrcode.Medium, 200)
	if err != nil {
		return nil
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil
	}
	return img
}
