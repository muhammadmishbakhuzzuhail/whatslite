// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/storage"
)

// HasMoreHistory: default true; false hanya setelah flag histend di-set (HP habis).
func TestHasMoreHistory(t *testing.T) {
	ctx := context.Background()
	s, err := storage.New(ctx, filepath.Join(t.TempDir(), "h.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	a := &App{store: s, ctx: ctx}
	const jid = "x@s.whatsapp.net"

	if !a.HasMoreHistory(jid) {
		t.Errorf("default harus true (belum EndOfHistory)")
	}
	if err := s.SetMeta(ctx, "histend:"+jid, "1"); err != nil {
		t.Fatalf("setmeta: %v", err)
	}
	if a.HasMoreHistory(jid) {
		t.Errorf("setelah histend=1 harus false")
	}
	// chat lain tak terpengaruh.
	if !a.HasMoreHistory("y@s.whatsapp.net") {
		t.Errorf("chat lain harus tetap true")
	}
	// store nil → true (tak bisa tahu → izinkan minta).
	if !(&App{}).HasMoreHistory(jid) {
		t.Errorf("store nil harus true")
	}
}
