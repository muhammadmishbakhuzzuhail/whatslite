// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package storage

import (
	"context"
	"testing"
)

// UnreadChats: hitung chat unread>0, kecuali arsip + status@broadcast.
func TestUnreadChats(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	mustUpsert := func(jid string, unread int, archived bool) {
		t.Helper()
		if err := s.UpsertChat(ctx, jid, jid, 1, unread, false, archived); err != nil {
			t.Fatalf("upsert %s: %v", jid, err)
		}
	}
	mustUpsert("a@s.whatsapp.net", 3, false) // unread → hitung
	mustUpsert("b@s.whatsapp.net", 1, false) // unread → hitung
	mustUpsert("c@s.whatsapp.net", 0, false) // dibaca → tidak
	mustUpsert("d@s.whatsapp.net", 5, true)  // arsip → tidak
	mustUpsert("status@broadcast", 9, false) // status → tidak

	n, err := s.UnreadChats(ctx)
	if err != nil {
		t.Fatalf("UnreadChats: %v", err)
	}
	if n != 2 {
		t.Errorf("UnreadChats = %d, want 2 (a + b)", n)
	}
}
