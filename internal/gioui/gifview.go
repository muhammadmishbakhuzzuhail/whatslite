// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// gifview.go — GIF animasi pure-Go (stdlib image/gif). Decode → frame-frame
// paint.ImageOp; di app nyata frame dipilih per waktu (loop). Bukti render
// headless pakai GIF sintetis.
package gioui

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// GifView — kotak GIF (320x240) memutar GIF sintetis (frame ke-2 ditampilkan,
// menunjukkan titik bergerak) + badge "GIF". Membuktikan jalur decode+gambar.
func GifView(gtx layout.Context, th *material.Theme, t Theme) layout.Dimensions {
	paint.FillShape(gtx.Ops, t.Wallpaper, clip.Rect{Max: gtx.Constraints.Max}.Op())
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		boxW, boxH := gtx.Dp(320), gtx.Dp(240)
		sz := image.Pt(boxW, boxH)
		// kartu rounded 14, clip
		rr := gtx.Dp(14)
		cl := clip.RRect{Rect: image.Rectangle{Max: sz}, NW: rr, NE: rr, SE: rr, SW: rr}.Push(gtx.Ops)
		paint.FillShape(gtx.Ops, t.Bg2, clip.Rect{Max: sz}.Op())
		frames, delays, _, _ := gifFrames(synthGif())
		if len(frames) > 0 {
			idx := frameAt(delays, 600) // ~frame ke-2/3 (titik di tengah)
			if idx >= len(frames) {
				idx = len(frames) - 1
			}
			fcl := clip.Rect{Max: sz}.Push(gtx.Ops)
			drawImageFill(gtx.Ops, frames[idx], boxW) // skala gif → lebar kotak
			fcl.Pop()
		}
		cl.Pop()
		// badge "GIF" kiri-bawah
		gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
		layout.SW.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				bw, bh := gtx.Dp(40), gtx.Dp(18)
				bsz := image.Pt(bw, bh)
				br := gtx.Dp(4)
				paint.FillShape(gtx.Ops, color.NRGBA{A: 0x77}, clip.RRect{Rect: image.Rectangle{Max: bsz}, NW: br, NE: br, SE: br, SW: br}.Op(gtx.Ops))
				gtx.Constraints.Min = bsz
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, 11, "GIF")
					lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
					lbl.Font.Weight = font.Bold
					return lbl.Layout(gtx)
				})
			})
		})
		return layout.Dimensions{Size: sz}
	})
}
