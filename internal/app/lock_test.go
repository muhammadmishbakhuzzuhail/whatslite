// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/storage"
)

func TestAppLock(t *testing.T) {
	ctx := context.Background()
	s, err := storage.New(ctx, filepath.Join(t.TempDir(), "l.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	a := &App{store: s, ctx: ctx}

	if a.HasAppPIN() {
		t.Errorf("awal: tak boleh ada PIN")
	}
	if !a.CheckAppPIN("anything") {
		t.Errorf("tanpa PIN, Check harus selalu true")
	}
	a.SetAppPIN("123") // <4 → diabaikan
	if a.HasAppPIN() {
		t.Errorf("PIN <4 karakter harus ditolak")
	}
	a.SetAppPIN("2468")
	if !a.HasAppPIN() {
		t.Errorf("PIN harus aktif setelah set")
	}
	if !a.CheckAppPIN("2468") {
		t.Errorf("PIN benar harus cocok")
	}
	if a.CheckAppPIN("0000") {
		t.Errorf("PIN salah tak boleh cocok")
	}
	// hash disimpan, bukan plaintext.
	if got := s.GetMeta(ctx, "app_pin", ""); got == "2468" || got == "" {
		t.Errorf("harus simpan HASH, bukan plaintext/kosong (got %q)", got)
	}
	a.ClearAppPIN()
	if a.HasAppPIN() || !a.CheckAppPIN("2468") {
		t.Errorf("setelah Clear: tak ada PIN, Check selalu true")
	}
}
