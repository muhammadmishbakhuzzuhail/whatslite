// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// gio-shot — render UI Gio (internal/gioui) ke PNG secara HEADLESS (EGL
// surfaceless, tanpa display) untuk audit paritas vs Svelte. Data demo statis.
//
//	go run ./cmd/gio-shot [out.png] [w] [h]
//
// Jalankan dgn: LIBGL_ALWAYS_SOFTWARE=1 EGL_PLATFORM=surfaceless go run ./cmd/gio-shot
package main

import (
	"image"
	"image/png"
	"os"
	"strconv"

	"gioui.org/font/gofont"
	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/gioui"
)

func main() {
	out := "/tmp/gio_shot.png"
	w, h := 1000, 680
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	if len(os.Args) > 3 {
		w, _ = strconv.Atoi(os.Args[2])
		h, _ = strconv.Atoi(os.Args[3])
	}

	hw, err := headless.NewWindow(w, h)
	must(err)
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	ui := gioui.NewUI(th, nil) // core nil → data demo

	ops := new(op.Ops)
	// dua frame: frame-1 memicu refresh() (load data demo), frame-2 menggambarnya.
	for i := 0; i < 2; i++ {
		ops.Reset()
		gtx := layout.Context{Ops: ops, Constraints: layout.Exact(image.Pt(w, h)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
		ui.Layout(gtx)
		must(hw.Frame(ops))
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	must(hw.Screenshot(img))
	f, err := os.Create(out)
	must(err)
	defer f.Close()
	must(png.Encode(f, img))
	println("gio → " + out)
}

func must(err error) {
	if err != nil {
		println("gio-shot ERR:", err.Error())
		os.Exit(1)
	}
}
