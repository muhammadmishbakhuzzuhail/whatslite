// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package storage

import (
	"context"
	"testing"
	"time"
)

// Latih fungsi baru sesi ini: meta KV, calls log, prune retensi, clear chat,
// dan galeri media. Memastikan SQL valid terhadap skema nyata (migrasi v3/v4).
func TestNewFeatureStorage(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now()

	// --- meta KV (retensi/proxy) ---
	if v := s.GetMeta(ctx, "retention_days", "90"); v != "90" {
		t.Fatalf("GetMeta default = %q, want 90", v)
	}
	if err := s.SetMeta(ctx, "retention_days", "30"); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	if v := s.GetMeta(ctx, "retention_days", "90"); v != "30" {
		t.Fatalf("GetMeta after set = %q, want 30", v)
	}

	// --- calls log ---
	if err := s.SaveCall(ctx, Call{ID: "c1", JID: "u1@s.whatsapp.net", Name: "Budi", Video: true, Status: "missed", Time: now}); err != nil {
		t.Fatalf("SaveCall: %v", err)
	}
	if err := s.SetCallStatus(ctx, "c1", "rejected"); err != nil {
		t.Fatalf("SetCallStatus: %v", err)
	}
	calls, err := s.ListCalls(ctx)
	if err != nil || len(calls) != 1 || calls[0].Status != "rejected" || !calls[0].Video {
		t.Fatalf("ListCalls = %+v, err %v", calls, err)
	}

	// --- pesan: prune retensi + clear + galeri media ---
	old := now.Add(-100 * 24 * time.Hour)
	mustSave := func(id, kind string, ts time.Time, starred bool) {
		m := Message{ID: id, ChatJID: "u1@s.whatsapp.net", Text: "x", Kind: kind, Timestamp: ts}
		if err := s.SaveMessage(ctx, m); err != nil {
			t.Fatalf("SaveMessage %s: %v", id, err)
		}
		if starred {
			if err := s.SetStarred(ctx, "u1@s.whatsapp.net", id, true); err != nil {
				t.Fatalf("Star: %v", err)
			}
		}
	}
	mustSave("old1", "text", old, false)  // harus ter-prune
	mustSave("oldS", "text", old, true)   // berbintang → tetap
	mustSave("img1", "image", now, false) // media → galeri
	mustSave("new1", "text", now, false)  // baru → tetap

	cutoff := now.Add(-90 * 24 * time.Hour).Unix()
	n, err := s.PruneMessages(ctx, cutoff)
	if err != nil || n != 1 {
		t.Fatalf("PruneMessages deleted %d (want 1), err %v", n, err)
	}

	media, err := s.ListMedia(ctx, "u1@s.whatsapp.net")
	if err != nil || len(media) != 1 || media[0].ID != "img1" {
		t.Fatalf("ListMedia = %+v, err %v", media, err)
	}

	if err := s.ClearMessages(ctx, "u1@s.whatsapp.net"); err != nil {
		t.Fatalf("ClearMessages: %v", err)
	}
	left, _ := s.ListMessages(ctx, "u1@s.whatsapp.net", 50)
	if len(left) != 0 {
		t.Fatalf("after ClearMessages still %d msgs", len(left))
	}

	if err := s.Vacuum(ctx); err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
}
