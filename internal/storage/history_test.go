package storage

import (
	"context"
	"testing"
	"time"
)

// SaveHistory harus menulis metadata chat + pesan + nama dalam satu transaksi,
// dan idempoten (commit ulang tak menggandakan baris).
func TestSaveHistoryBatch(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now()

	chats := []HistoryChat{
		{JID: "g1@g.us", Name: "Grup A", TS: now.Unix(), Unread: 2, Pinned: true},
		{JID: "u1@s.whatsapp.net", Name: "", TS: now.Unix()},
	}
	msgs := []Message{
		{ID: "m1", ChatJID: "g1@g.us", Sender: "u1@s.whatsapp.net", Text: "halo", Timestamp: now, Kind: "text"},
		{ID: "m2", ChatJID: "u1@s.whatsapp.net", PushName: "Budi", Text: "hai", Timestamp: now, Kind: "text"},
	}
	names := map[string]string{"u1@s.whatsapp.net": "Budi"}

	if err := s.SaveHistory(ctx, chats, msgs, names); err != nil {
		t.Fatalf("SaveHistory: %v", err)
	}
	// Commit ulang → idempoten.
	if err := s.SaveHistory(ctx, chats, msgs, names); err != nil {
		t.Fatalf("SaveHistory (2nd): %v", err)
	}
	_ = s.RecomputeSummaries(ctx)

	got, err := s.ListMessages(ctx, "g1@g.us", 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].ID != "m1" || got[0].Text != "halo" {
		t.Fatalf("messages g1 = %+v, want 1×m1", got)
	}

	cs, err := s.ListChats(ctx)
	if err != nil {
		t.Fatalf("list chats: %v", err)
	}
	byJID := map[string]Chat{}
	for _, c := range cs {
		byJID[c.JID] = c
	}
	if byJID["g1@g.us"].Name != "Grup A" {
		t.Fatalf("g1 name = %q, want Grup A", byJID["g1@g.us"].Name)
	}
	if byJID["g1@g.us"].Unread != 2 {
		t.Fatalf("g1 unread = %d, want 2", byJID["g1@g.us"].Unread)
	}
	if byJID["u1@s.whatsapp.net"].Name != "Budi" {
		t.Fatalf("u1 name = %q, want Budi (from names map)", byJID["u1@s.whatsapp.net"].Name)
	}
}
