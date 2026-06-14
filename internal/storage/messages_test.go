// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()
	s, err := New(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { s.db.Close() })
	return s
}

func saveOut(t *testing.T, s *Store, id string) {
	t.Helper()
	if err := s.SaveMessage(context.Background(), Message{
		ID: id, ChatJID: "x@s.whatsapp.net", Text: "hi", Timestamp: time.Now(), FromMe: true,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
}

func statusOf(t *testing.T, s *Store, id string) string {
	t.Helper()
	ms, err := s.ListMessages(context.Background(), "x@s.whatsapp.net", 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, m := range ms {
		if m.ID == id {
			return m.Status
		}
	}
	t.Fatalf("msg %s not found", id)
	return ""
}

func TestSetMessageStatusUpgradeOnly(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	saveOut(t, s, "m1")
	if got := statusOf(t, s, "m1"); got != "sent" {
		t.Fatalf("initial status = %q, want sent", got)
	}
	// sent → read
	if err := s.SetMessageStatus(ctx, "x@s.whatsapp.net", []string{"m1"}, "read"); err != nil {
		t.Fatal(err)
	}
	if got := statusOf(t, s, "m1"); got != "read" {
		t.Fatalf("after read = %q, want read", got)
	}
	// read → delivered must NOT downgrade
	if err := s.SetMessageStatus(ctx, "x@s.whatsapp.net", []string{"m1"}, "delivered"); err != nil {
		t.Fatal(err)
	}
	if got := statusOf(t, s, "m1"); got != "read" {
		t.Fatalf("no-downgrade failed = %q, want read", got)
	}
}

func TestReceiptUpgradeOnly(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	chat, msg, rcpt := "g@g.us", "m1", "62811@s.whatsapp.net"
	now := time.Now()
	if err := s.SetReceipt(ctx, chat, msg, rcpt, "delivered", now); err != nil {
		t.Fatal(err)
	}
	if err := s.SetReceipt(ctx, chat, msg, rcpt, "read", now); err != nil {
		t.Fatal(err)
	}
	// read must persist even if a stale delivered arrives later
	if err := s.SetReceipt(ctx, chat, msg, rcpt, "delivered", now); err != nil {
		t.Fatal(err)
	}
	rs, err := s.ListReceipts(ctx, chat, msg)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 1 {
		t.Fatalf("receipts = %d, want 1", len(rs))
	}
	if rs[0].Status != "read" {
		t.Fatalf("status = %q, want read", rs[0].Status)
	}
}

func TestPinnedRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	saveOut(t, s, "m1")
	if err := s.SetPinnedInChat(ctx, "x@s.whatsapp.net", "m1", true); err != nil {
		t.Fatal(err)
	}
	ps, err := s.ListPinned(ctx, "x@s.whatsapp.net")
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 1 || ps[0].ID != "m1" {
		t.Fatalf("pinned = %+v, want [m1]", ps)
	}
	if err := s.SetPinnedInChat(ctx, "x@s.whatsapp.net", "m1", false); err != nil {
		t.Fatal(err)
	}
	ps, _ = s.ListPinned(ctx, "x@s.whatsapp.net")
	if len(ps) != 0 {
		t.Fatalf("after unpin = %d, want 0", len(ps))
	}
}
