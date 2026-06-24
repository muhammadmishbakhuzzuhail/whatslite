// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// images.go — gambar IN-PROCESS: byte foto (avatar/media) dari engine whatsmeow
// langsung → image.Image → paint.ImageOp, digambar Gio TANPA HTTP/IPC. Inilah
// keunggulan all-Go: foto via memori, bukan server media.
package gioui

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	_ "image/png"

	xdraw "golang.org/x/image/draw"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/paint"
	_ "golang.org/x/image/webp" // decode stiker .webp (statis) via image.Decode
)

// gifAnim — frame animasi GIF ter-decode (auto-loop di thumbnail bubble).
type gifAnim struct {
	frames []paint.ImageOp
	delays []int
}

// gifFrames: byte GIF → frame-frame ter-komposit (paint.ImageOp) + delay(ms) +
// ukuran. Pure-Go (stdlib image/gif). Disposal disederhanakan (akumulasi Over —
// benar utk "keep" yg paling umum). Animasi: pilih frame per waktu.
func gifFrames(b []byte) (frames []paint.ImageOp, delaysMs []int, w, h int) {
	g, err := gif.DecodeAll(bytes.NewReader(b))
	if err != nil || len(g.Image) == 0 {
		return nil, nil, 0, 0
	}
	w, h = g.Config.Width, g.Config.Height
	if w == 0 || h == 0 {
		b0 := g.Image[0].Bounds()
		w, h = b0.Dx(), b0.Dy()
	}
	canvas := image.NewRGBA(image.Rect(0, 0, w, h))
	for i, fr := range g.Image {
		draw.Draw(canvas, fr.Bounds(), fr, fr.Bounds().Min, draw.Over)
		snap := image.NewRGBA(canvas.Bounds())
		copy(snap.Pix, canvas.Pix)
		frames = append(frames, paint.NewImageOp(snap))
		d := 100 // default 100ms
		if i < len(g.Delay) && g.Delay[i] > 0 {
			d = g.Delay[i] * 10 // centidetik → ms
		}
		delaysMs = append(delaysMs, d)
	}
	return frames, delaysMs, w, h
}

// frameAt: index frame utk total durasi `t` ms (loop).
func frameAt(delaysMs []int, tMs int) int {
	total := 0
	for _, d := range delaysMs {
		total += d
	}
	if total <= 0 {
		return 0
	}
	tMs %= total
	for i, d := range delaysMs {
		if tMs < d {
			return i
		}
		tMs -= d
	}
	return len(delaysMs) - 1
}

// synthGif: GIF animasi sintetis (titik accent bergerak) utk uji render headless.
func synthGif() []byte {
	const n = 120
	pal := color.Palette{
		color.RGBA{0x0a, 0x0f, 0x14, 0xff}, // 0 bg
		color.RGBA{0x06, 0xc9, 0x8c, 0xff}, // 1 accent
	}
	g := &gif.GIF{}
	for f := 0; f < 5; f++ {
		img := image.NewPaletted(image.Rect(0, 0, n, n), pal)
		cx, cy := 20+f*20, 60
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				if (x-cx)*(x-cx)+(y-cy)*(y-cy) < 220 {
					img.SetColorIndex(x, y, 1)
				}
			}
		}
		g.Image = append(g.Image, img)
		g.Delay = append(g.Delay, 15)
	}
	var buf bytes.Buffer
	_ = gif.EncodeAll(&buf, g)
	return buf.Bytes()
}

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

// decodeImageScaled — seperti decodeImage tapi DI-DOWNSCALE ke sisi-terpanjang
// maxDim sebelum dipakai (kunci hemat RAM ala Telegram: thumbnail bubble ~640px,
// avatar ~256px — bukan foto 12MP full-res yg jadi RGBA puluhan MB). Sudah kecil →
// dipakai apa adanya. maxDim<=0 → tanpa skala.
func decodeImageScaled(b []byte, maxDim int) image.Image {
	if len(b) == 0 {
		return nil
	}
	if maxDim <= 0 {
		return decodeImage(b)
	}
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(b)); err == nil && cfg.Width <= maxDim && cfg.Height <= maxDim {
		return decodeImage(b) // sumber sudah ≤ target → tak perlu skala
	}
	src := decodeImage(b)
	if src == nil {
		return nil
	}
	sb := src.Bounds()
	w, h := sb.Dx(), sb.Dy()
	if w <= maxDim && h <= maxDim {
		return src
	}
	if w >= h {
		h = h * maxDim / w
		w = maxDim
	} else {
		w = w * maxDim / h
		h = maxDim
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, sb, xdraw.Src, nil) // src full-res dibuang stlh ini
	return dst
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

// drawImageRect: gambar img di-scale uniform mengisi sz (dipakai lightbox aspek-
// fit — sz sudah dihitung sesuai aspek gambar, jadi scale-by-width = scale-by-height).
func drawImageRect(ops *op.Ops, imgOp paint.ImageOp, sz image.Point) {
	src := imgOp.Size()
	if src.X == 0 || src.Y == 0 {
		return
	}
	s := float32(sz.X) / float32(src.X)
	defer op.Affine(f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(s, s))).Push(ops).Pop()
	imgOp.Add(ops)
	paint.PaintOp{}.Add(ops)
}

// rotateDataURI90 — putar gambar data-URI 90° searah jarum jam, encode ulang JPEG.
// Kembalikan (uri baru, ImageOp baru, true). Dipakai editor pratinjau media.
func rotateDataURI90(uri string) (string, paint.ImageOp, bool) {
	img := decodeImage(decodeDataURI(uri))
	if img == nil {
		return uri, paint.ImageOp{}, false
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, h, w)) // dimensi tertukar
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(h-1-y, x, img.At(b.Min.X+x, b.Min.Y+y)) // 90° CW
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 88}); err != nil {
		return uri, paint.ImageOp{}, false
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), paint.NewImageOp(dst), true
}

// flipDataURIH — cermin horizontal gambar data-URI (kiri↔kanan), encode ulang JPEG.
func flipDataURIH(uri string) (string, paint.ImageOp, bool) {
	img := decodeImage(decodeDataURI(uri))
	if img == nil {
		return uri, paint.ImageOp{}, false
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(w-1-x, y, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 88}); err != nil {
		return uri, paint.ImageOp{}, false
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), paint.NewImageOp(dst), true
}

// compressDataURI — encode ulang gambar JPEG kualitas lebih rendah (q55) → ukuran
// kirim lebih kecil. Piksel tak berubah (ImageOp sama), hanya byte URI menyusut.
func compressDataURI(uri string) (string, paint.ImageOp, bool) {
	img := decodeImage(decodeDataURI(uri))
	if img == nil {
		return uri, paint.ImageOp{}, false
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 55}); err != nil {
		return uri, paint.ImageOp{}, false
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), paint.NewImageOp(img), true
}

// cropDataURI — potong gambar data-URI ke rect r (koordinat piksel gambar),
// encode ulang JPEG. Kembalikan (uri baru, ImageOp baru, true). r<4px → gagal.
func cropDataURI(uri string, r image.Rectangle) (string, paint.ImageOp, bool) {
	img := decodeImage(decodeDataURI(uri))
	if img == nil {
		return uri, paint.ImageOp{}, false
	}
	b := img.Bounds()
	rr := r.Add(b.Min).Intersect(b)
	if rr.Dx() < 4 || rr.Dy() < 4 {
		return uri, paint.ImageOp{}, false
	}
	dst := image.NewRGBA(image.Rect(0, 0, rr.Dx(), rr.Dy()))
	draw.Draw(dst, dst.Bounds(), img, rr.Min, draw.Src)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 88}); err != nil {
		return uri, paint.ImageOp{}, false
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), paint.NewImageOp(dst), true
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
