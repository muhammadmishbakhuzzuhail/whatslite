// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// images.go — gambar IN-PROCESS: byte foto (avatar/media) dari engine whatsmeow
// langsung → image.Image → paint.ImageOp, digambar Gio TANPA HTTP/IPC. Inilah
// keunggulan all-Go: foto via memori, bukan server media.
package gioui

import (
	"bytes"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/paint"
)

// decodeImage: byte (jpeg/png) → image.Image (nil bila gagal). Dipakai utk byte
// avatar (engine.ProfilePictureRaw) & media dari store, in-memory.
func decodeImage(b []byte) image.Image {
	if len(b) == 0 {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil
	}
	return img
}

// drawImageFill: gambar img memenuhi kotak d×d (cover, di-scale), di dalam clip
// yang sudah aktif (mis. lingkaran avatar). Pakai op.Affine scale.
func drawImageFill(ops *op.Ops, imgOp paint.ImageOp, d int) {
	src := imgOp.Size()
	if src.X == 0 || src.Y == 0 {
		return
	}
	// cover: skala = max(d/src.X, d/src.Y) agar penuh tanpa celah.
	s := float32(d) / float32(src.X)
	if sy := float32(d) / float32(src.Y); sy > s {
		s = sy
	}
	defer op.Affine(f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(s, s))).Push(ops).Pop()
	imgOp.Add(ops)
	paint.PaintOp{}.Add(ops)
}

// synthPhoto: foto sintetis (gradient sunset) utk uji render avatar-foto headless
// tanpa engine. Membuktikan jalur gambar bulat Gio bekerja.
func synthPhoto() paint.ImageOp {
	const n = 96
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		fy := float32(y) / n
		r := uint8(255 - fy*150)
		g := uint8(150 - fy*70)
		b := uint8(90 + fy*120)
		for x := 0; x < n; x++ {
			img.SetRGBA(x, y, color.RGBA{r, g, b, 255})
		}
	}
	return paint.NewImageOp(img)
}
