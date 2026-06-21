// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// whatslite-gio — UI WhatsLite native pure-Go (Gio) yang memanggil engine
// whatsmeow IN-PROCESS (tanpa jembatan IPC/HTTP). UI di internal/gioui agar
// bisa dipakai ulang oleh render-tool (cmd/gio-shot) untuk audit headless.
package main

import (
	"context"
	"log"
	"os"
	"time"

	gioapp "gioui.org/app"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/app"
	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/gioui"
)

func main() {
	var core *app.App
	if os.Getenv("WLGIO_DEMO") == "" {
		core = app.NewApp()
		if err := core.StartupHeadless(context.Background()); err != nil {
			log.Fatal("[gio] startup engine: ", err)
		}
		core.Connect()
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
	th.Shaper = gioui.NewShaper()
	ui := gioui.NewUI(th, core)

	go func() {
		for range time.NewTicker(700 * time.Millisecond).C {
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
