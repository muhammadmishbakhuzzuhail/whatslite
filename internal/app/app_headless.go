// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_headless.go — startup TANPA Wails (dipakai UI Gio in-process). Inisialisasi
// engine + storage + wireEvents — sama dgn mode Wails KECUALI: tak ada
// WebKitGTK/window. Gio memakai media IN-PROCESS (MediaBytes/AvatarBytes), jadi
// tak ada media-server HTTP maupun bridge IPC (keduanya peninggalan FE Qt dihapus).

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/engine"
	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/storage"
)

// StartupHeadless menyalakan engine untuk UI Gio in-process. Mengembalikan error
// agar pemanggil bisa keluar bersih bila init gagal.
func (a *App) StartupHeadless(ctx context.Context) error {
	a.ctx = ctx

	dataDir, err := engine.DefaultDataDir()
	if err != nil {
		return fmt.Errorf("data dir: %w", err)
	}
	eng, err := engine.New(ctx, filepath.Join(dataDir, "whatslite.db"), os.Getenv("WALITE_DEBUG") != "")
	if err != nil {
		return fmt.Errorf("engine: %w", err)
	}
	store, err := storage.New(ctx, filepath.Join(dataDir, "app.db"))
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}
	a.eng = eng
	a.store = store
	a.loadLabels()

	// Writer DB tunggal (sama pola dgn Wails) — recover pakai log (bukan runtime).
	a.wq = make(chan func(), 8192)
	go func() {
		for fn := range a.wq {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Println("[engine] bg write panic:", r)
					}
				}()
				fn()
			}()
		}
	}()

	a.mediaDir = filepath.Join(dataDir, "media")
	_ = os.MkdirAll(a.mediaDir, 0o755)
	a.startMediaEviction(512 << 20)
	a.stickerDir = filepath.Join(dataDir, "stickers")
	_ = os.MkdirAll(a.stickerDir, 0o755)
	a.gifDir = filepath.Join(dataDir, "gifs")
	_ = os.MkdirAll(a.gifDir, 0o755)

	if px := store.GetMeta(ctx, "proxy", ""); px != "" {
		_ = eng.SetProxy(px)
	}
	a.retentionDays = atoiDef(store.GetMeta(ctx, "retention_days", "90"), 90)
	a.keepDeleted.Store(store.GetMeta(ctx, "keep_deleted", "1") == "1")

	// Sapu disappearing kedaluwarsa berkala (sama spt Wails).
	go func() {
		t := time.NewTicker(60 * time.Second)
		defer t.Stop()
		for range t.C {
			if a.store == nil {
				return
			}
			if n, _ := a.store.SweepExpired(a.ctx, time.Now().Unix()); n > 0 {
				a.emit("wa:sync", "")
			}
		}
	}()

	_ = store.RecomputeSummaries(ctx)
	a.startScheduler()
	a.wireEvents(eng, store)

	log.Printf("[engine] headless siap (in-process, Gio) — data=%s", dataDir)
	return nil
}
