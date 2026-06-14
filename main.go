// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

// Command whatslite (Wails): app Linux native — UI web (Svelte) di dalam
// WebView sistem (WebKitGTK), engine whatsmeow (disambung belakangan).
package main

import (
	"embed"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"syscall"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/app"
	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/engine"
)

// singleInstance memastikan hanya satu proses app jalan (flock). false = sudah
// ada instance lain. Gagal cek → izinkan jalan (jangan blok user).
func singleInstance() bool {
	dir, err := engine.DefaultDataDir()
	if err != nil {
		return true
	}
	f, err := os.OpenFile(filepath.Join(dir, ".singleton.lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return true
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return false
	}
	return true // fd sengaja dibiarkan terbuka → lock dipegang seumur proses
}

//go:embed all:frontend/dist
var assets embed.FS

// version di-stamp saat build: `wails build -ldflags "-X main.version=v0.1.0"`.
// Default "dev" untuk build lokal tanpa stamp.
var version = "dev"

// ensureRuntimeEnv memperbaiki crash Go+WebKitGTK di Linux dengan re-exec sekali
// memakai env yang aman:
//   - GODEBUG=asyncpreemptoff=1 : matikan sinyal preemption Go yang dapat merusak
//     heap saat berada di kode C WebKit/JavaScriptCore (penyebab "free():
//     corrupted unsorted chunks").
//   - WEBKIT_DISABLE_DMABUF_RENDERER=1 : hindari crash renderer DMABUF di
//     WebKitGTK baru pada sebagian driver GPU.
func ensureRuntimeEnv() {
	if os.Getenv("WALITE_REEXEC") == "1" {
		return // sudah re-exec; jangan loop
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	env := append(os.Environ(),
		"GODEBUG=asyncpreemptoff=1",
		"WALITE_REEXEC=1",
	)
	if os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER") == "" {
		env = append(env, "WEBKIT_DISABLE_DMABUF_RENDERER=1")
	}
	_ = syscall.Exec(exe, os.Args, env) // ganti proses ini; tak kembali bila sukses
}

func main() {
	// Mode ekspor data nyata (offline, tanpa GUI) → JSON untuk autopilot/compare.
	if len(os.Args) >= 3 && os.Args[1] == "--export-json" {
		if err := app.RunExport(os.Args[2]); err != nil {
			println("export error:", err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}

	ensureRuntimeEnv()

	// Cegah dua instance (setelah re-exec, di proses final). Bila sudah ada,
	// beri sinyal instance lama untuk tampil ke depan, lalu keluar.
	if !singleInstance() {
		if dir, err := engine.DefaultDataDir(); err == nil {
			if c, e := net.Dial("unix", filepath.Join(dir, ".ipc.sock")); e == nil {
				c.Close()
			}
		}
		println("WhatsLite sudah berjalan.")
		os.Exit(0)
	}

	// Resolver DNS murni-Go (hindari jalur CGo getaddrinfo yang rawan crash).
	net.DefaultResolver.PreferGo = true

	println("WhatsLite", version)

	application := app.NewApp()
	application.SetVersion(version)

	err := wails.Run(&options.App{
		Title:     "WhatsLite",
		Width:     1100,
		Height:    720,
		MinWidth:  760,
		MinHeight: 480,
		AssetServer: &assetserver.Options{
			Assets: assets,
			// GET non-aset (/media/* & /avatar/*) → handler cache-file.
			Handler: http.HandlerFunc(application.ServeHTTP),
		},
		BackgroundColour: &options.RGBA{R: 240, G: 242, B: 245, A: 1},
		OnStartup:        application.Startup,
		OnDomReady:       application.DomReady,
		OnShutdown:       application.Shutdown,
		OnBeforeClose:    application.BeforeClose,
		Bind:             []interface{}{application},
		Linux: &linux.Options{
			ProgramName: "WhatsLite",
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
