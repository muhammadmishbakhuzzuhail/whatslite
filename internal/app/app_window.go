// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_window.go — mode latar belakang (tutup window → tetap jalan agar
// notifikasi/pesan-terjadwal/pengingat aktif) + badge unread di judul window.
// Wails v2 tak punya systray native; ini pendekatan terbaik tanpa systray.
// Buka lagi: jalankan app → IPC single-instance menampilkan window yang ada.

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var bgClose atomic.Bool

// SetBackgroundClose: true → tombol tutup menyembunyikan window (app jalan di
// latar), bukan keluar. Disimpan agar persist antar-sesi.
func (a *App) SetBackgroundClose(on bool) {
	bgClose.Store(on)
	if a.store != nil {
		v := "0"
		if on {
			v = "1"
		}
		_ = a.store.SetMeta(a.ctx, "bg_close", v)
	}
}

func (a *App) GetBackgroundClose() bool { return bgClose.Load() }

// BeforeClose dipanggil Wails saat user menutup window. Bila mode latar aktif →
// sembunyikan window & cegah keluar (true = batal tutup).
func (a *App) BeforeClose(ctx context.Context) bool {
	if bgClose.Load() {
		runtime.WindowHide(a.ctx)
		return true
	}
	return false
}

// Quit keluar dari app sepenuhnya (dipakai tombol "Keluar" di setelan saat mode
// latar aktif).
func (a *App) Quit() { runtime.Quit(a.ctx) }

// SetUnreadBadge memperbarui judul window dgn jumlah belum dibaca ("(3) …").
func (a *App) SetUnreadBadge(n int) {
	title := "WhatsLite"
	if n > 0 {
		title = fmt.Sprintf("(%d) %s", n, title)
	}
	runtime.WindowSetTitle(a.ctx, title)
}
