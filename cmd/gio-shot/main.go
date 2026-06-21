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

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/gioui"
)

func main() {
	out := "/tmp/gio_shot.png"
	screen := "main"
	w, h := 1000, 680
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	if len(os.Args) > 2 {
		screen = os.Args[2]
	}
	if len(os.Args) > 4 {
		w, _ = strconv.Atoi(os.Args[3])
		h, _ = strconv.Atoi(os.Args[4])
	}

	hw, err := headless.NewWindow(w, h)
	must(err)
	th := material.NewTheme()
	th.Shaper = gioui.NewShaper()
	t := gioui.DarkTheme()
	ui := gioui.NewUI(th, nil) // core nil → data demo

	draw := func(gtx layout.Context) {
		switch screen {
		case "login":
			gioui.LoginView(gtx, th, t)
		case "settings":
			gioui.SettingsView(gtx, th, t)
		case "bubbles":
			gioui.BubbleTypesView(gtx, th, t)
		case "states":
			gioui.StatesView(gtx, th, t)
		case "convheader":
			gioui.ConvHeaderView(gtx, th, t)
		case "sidepanes":
			gioui.SidePanesView(gtx, th, t)
		case "modals":
			gioui.ModalsView(gtx, th, t)
		case "infodrawer":
			gioui.InfoDrawerView(gtx, th, t)
		case "app-settings":
			ui.SetView("settings")
			ui.Layout(gtx)
		case "app-calls":
			ui.SetView("calls")
			ui.Layout(gtx)
		case "app-splash":
			ui.Deselect()
			ui.Layout(gtx)
		default:
			ui.Layout(gtx)
		}
	}

	ops := new(op.Ops)
	// dua frame: frame-1 memicu refresh() (load data demo), frame-2 menggambarnya.
	for i := 0; i < 2; i++ {
		ops.Reset()
		gtx := layout.Context{Ops: ops, Constraints: layout.Exact(image.Pt(w, h)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
		draw(gtx)
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
