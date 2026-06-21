// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// whatslite-gio — UI WhatsLite native pure-Go (Gio) yang memanggil engine
// whatsmeow IN-PROCESS (tanpa jembatan IPC/HTTP). Backend = internal/app yang
// SAMA dgn versi Wails/Qt; di sini cukup panggil method-nya langsung.
package main

import (
	"context"
	"log"
	"os"

	gioapp "gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/app"
)

func main() {
	// Engine in-process: init store+engine+WA persis seperti host headless,
	// lalu kita panggil GetChats/GetMessages/Connect LANGSUNG (bukan via socket).
	// WLGIO_DEMO=1 → render UI dgn data statis (tanpa engine/jaringan) utk uji.
	var core *app.App
	if os.Getenv("WLGIO_DEMO") == "" {
		core = app.NewApp()
		if err := core.StartupHeadless(context.Background()); err != nil {
			log.Fatal("[gio] startup engine: ", err)
		}
		core.Connect() // sambungkan sesi WA (QR bila belum login)
	}

	go func() {
		w := new(gioapp.Window)
		w.Option(gioapp.Title("WhatsLite"), gioapp.Size(unit.Dp(1000), unit.Dp(680)),
			gioapp.MinSize(unit.Dp(720), unit.Dp(480)))
		if err := run(w, core); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	gioapp.Main()
}

func run(w *gioapp.Window, core *app.App) error {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	ui := NewUI(th, core)

	// Repaint berkala agar data engine yang masuk (chat/pesan baru, koneksi)
	// ter-refresh. Sederhana untuk POC; nanti diganti event-driven.
	go func() {
		for range tick() {
			w.Invalidate()
		}
	}()

	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case gioapp.DestroyEvent:
			return e.Err
		case gioapp.FrameEvent:
			gtx := gioapp.NewContext(&ops, e)
			ui.Layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}
