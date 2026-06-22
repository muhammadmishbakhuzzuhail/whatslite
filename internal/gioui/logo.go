// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// logo.go — logo aplikasi WhatsLite (build/linux/whatslite.svg: kotak membulat
// hijau + bubble putih + 3 titik). Diraster penuh-warna via oksvg (fill + warna),
// cache per ukuran. Dipakai di titlebar (waLogo) — bukan glyph stroke satu-warna.
package gioui

import (
	"bytes"
	_ "embed"
	"image"
	"sync"

	"gioui.org/op/paint"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

//go:embed logo.svg
var logoSVG []byte

var (
	logoMu    sync.Mutex
	logoCache = map[int]paint.ImageOp{}
)

// logoOp — raster logo WhatsLite ukuran size×size (penuh-warna, ber-fill). Cache.
func logoOp(size int) (paint.ImageOp, bool) {
	if size <= 0 {
		return paint.ImageOp{}, false
	}
	logoMu.Lock()
	defer logoMu.Unlock()
	if op, ok := logoCache[size]; ok {
		return op, true
	}
	ic, err := oksvg.ReadIconStream(bytes.NewReader(logoSVG))
	if err != nil {
		return paint.ImageOp{}, false
	}
	ic.SetTarget(0, 0, float64(size), float64(size))
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, rgba, rgba.Bounds())
	ic.Draw(rasterx.NewDasher(size, size, scanner), 1.0)
	op := paint.NewImageOp(rgba)
	logoCache[size] = op
	return op, true
}
