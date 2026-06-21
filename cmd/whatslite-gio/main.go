// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// whatslite-gio — UI WhatsLite native pure-Go (Gio) yang memanggil engine
// whatsmeow IN-PROCESS (tanpa jembatan IPC/HTTP). UI di internal/gioui agar
// bisa dipakai ulang oleh render-tool (cmd/gio-shot) untuk audit headless.
package main

import (
	"context"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	gioapp "gioui.org/app"
	"gioui.org/gpu/headless"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/app"
	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/gioui"
	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/video"
	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/voice"
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
	// Voice note (ogg-opus) in-process: byte engine → internal/voice (cgo libopus).
	if core != nil {
		ui.OnPlayVoice = func(chat, id string) {
			if b := core.MediaBytes(chat, id); len(b) > 0 {
				go func() { _, _ = voice.Play(b) }()
			}
		}
		ui.OnPlayVideo = func(chat, id, typ string) {
			if b := core.MediaBytes(chat, id); len(b) > 0 {
				ext := ".mp4"
				if typ == "gif" {
					ext = ".gif"
				}
				go func() { _, _ = video.PlayBytes(b, ext) }()
			}
		}
	}

	go func() {
		for range time.NewTicker(700 * time.Millisecond).C {
			w.Invalidate()
		}
	}()

	// Helper auto-capture: WLGIO_SHOTDIR set → simpan PNG frame LIVE (data WA asli)
	// tiap WLGIO_SHOT_EVERY detik (default 3) utk loop analisa→perbaikan UI/UX.
	// Render headless dgn ops yg SAMA spt window (satu goroutine → tanpa race).
	shot := newShooter()

	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case gioapp.DestroyEvent:
			return e.Err
		case gioapp.FrameEvent:
			gtx := gioapp.NewContext(&ops, e)
			ui.Layout(gtx)
			shot.maybeCapture(gtx.Ops, e.Size) // foto live sebelum frame window
			e.Frame(gtx.Ops)
		}
	}
}

// shooter mengambil PNG frame aplikasi yg sedang berjalan (data nyata) ke
// WLGIO_SHOTDIR. Render headless pakai ops yg sama → identik dgn yg di layar.
type shooter struct {
	dir   string
	every time.Duration
	last  time.Time
	hw    *headless.Window
	size  image.Point
	n     int
}

func newShooter() *shooter {
	dir := os.Getenv("WLGIO_SHOTDIR")
	if dir == "" {
		return &shooter{}
	}
	every := 3 * time.Second
	if v := os.Getenv("WLGIO_SHOT_EVERY"); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			every = time.Duration(sec) * time.Second
		}
	}
	_ = os.MkdirAll(dir, 0o755)
	log.Printf("[gio] auto-capture aktif → %s tiap %s", dir, every)
	return &shooter{dir: dir, every: every}
}

func (s *shooter) maybeCapture(ops *op.Ops, size image.Point) {
	if s.dir == "" || size.X <= 0 || size.Y <= 0 || time.Since(s.last) < s.every {
		return
	}
	s.last = time.Now()
	if s.hw == nil || s.size != size {
		if s.hw != nil {
			s.hw.Release()
		}
		hw, err := headless.NewWindow(size.X, size.Y)
		if err != nil {
			log.Printf("[gio] capture: headless gagal: %v", err)
			s.dir = "" // matikan agar tak spam error
			return
		}
		s.hw, s.size = hw, size
	}
	if err := s.hw.Frame(ops); err != nil {
		log.Printf("[gio] capture: frame: %v", err)
		return
	}
	img := image.NewRGBA(image.Rectangle{Max: size})
	if err := s.hw.Screenshot(img); err != nil {
		log.Printf("[gio] capture: screenshot: %v", err)
		return
	}
	s.n++
	name := filepath.Join(s.dir, "wlive-"+s.last.UTC().Format("20060102-150405")+".png")
	f, err := os.Create(name)
	if err != nil {
		log.Printf("[gio] capture: create: %v", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Printf("[gio] capture: encode: %v", err)
		return
	}
	log.Printf("[gio] capture #%d → %s", s.n, name)
}
