// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_lock.go — kunci aplikasi (PIN). Hash PIN (sha256 ber-salt) disimpan di meta
// "app_pin"; ini gate UI saja (data tetap di home; bukan enkripsi-PIN).

import (
	"crypto/sha256"
	"encoding/hex"
)

// hashPIN — sha256 ber-salt dari PIN (jangan simpan plaintext). Pure → bisa diuji.
func hashPIN(pin string) string {
	sum := sha256.Sum256([]byte("walite-pin:" + pin))
	return hex.EncodeToString(sum[:])
}

// HasAppPIN: true bila kunci aplikasi aktif (PIN sudah diset).
func (a *App) HasAppPIN() bool {
	return a.store != nil && a.store.GetMeta(a.ctx, "app_pin", "") != ""
}

// SetAppPIN menyetel PIN baru (>=4 digit/char). PIN kosong diabaikan (pakai Clear).
func (a *App) SetAppPIN(pin string) {
	if a.store == nil || len(pin) < 4 {
		return
	}
	_ = a.store.SetMeta(a.ctx, "app_pin", hashPIN(pin))
}

// ClearAppPIN menonaktifkan kunci aplikasi.
func (a *App) ClearAppPIN() {
	if a.store != nil {
		_ = a.store.SetMeta(a.ctx, "app_pin", "")
	}
}

// CheckAppPIN: true bila pin cocok (atau tak ada PIN aktif).
func (a *App) CheckAppPIN(pin string) bool {
	if a.store == nil {
		return true
	}
	h := a.store.GetMeta(a.ctx, "app_pin", "")
	return h == "" || h == hashPIN(pin)
}
