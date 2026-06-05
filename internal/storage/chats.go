package storage

// chats.go — ringkasan chat untuk sidebar: penamaan, metadata history sync,
// dan perbaikan urutan/preview (RecomputeSummaries) + pembacaan daftar.

import (
	"context"
	"time"
)

// SetChatName memperbarui nama chat (subjek grup / nama kontak) bila chat sudah
// ada. UPDATE-only agar tak membuat baris chat kosong untuk grup tanpa riwayat.
func (s *Store) SetChatName(ctx context.Context, jid, name string) error {
	if name == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `UPDATE chats SET name = ? WHERE jid = ?`, name, jid)
	return err
}

// UpsertChat menulis metadata chat dari history sync (Conversation): timestamp
// aktivitas terakhir (otoritatif → urutan benar), unread, pinned, archived.
func (s *Store) UpsertChat(ctx context.Context, jid, name string, ts int64, unread int, pinned, archived bool) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO chats (jid, name, last_text, last_ts, unread, pinned, archived)
VALUES (?, ?, '', ?, ?, ?, ?)
ON CONFLICT(jid) DO UPDATE SET
	last_ts  = MAX(chats.last_ts, excluded.last_ts),
	unread   = excluded.unread,
	pinned   = excluded.pinned,
	archived = excluded.archived,
	name     = CASE WHEN excluded.name != '' THEN excluded.name ELSE chats.name END`,
		jid, name, ts, unread, b2i(pinned), b2i(archived))
	return err
}

// RecomputeSummaries menurunkan last_ts + last_text tiap chat dari pesan NYATA
// terbaru di tabel messages (sumber kebenaran). Memperbaiki drift di mana
// last_ts (cache dari ConversationTimestamp) lebih lama dari pesan asli →
// chat baru terkubur di bawah chat lama. Idempoten; dipanggil saat startup.
func (s *Store) RecomputeSummaries(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE chats SET
  last_ts = COALESCE((SELECT MAX(ts) FROM messages m WHERE m.chat_jid = chats.jid), last_ts),
  last_text = COALESCE((
    SELECT CASE
      WHEN text != '' THEN text
      WHEN kind = 'image'   THEN '🖼️ Foto'
      WHEN kind = 'video'   THEN '🎬 Video'
      WHEN kind = 'sticker' THEN '🏷️ Stiker'
      WHEN kind = 'voice'   THEN '🎤 Pesan suara'
      ELSE last_text END
    FROM messages m WHERE m.chat_jid = chats.jid ORDER BY ts DESC LIMIT 1
  ), last_text),
  last_sender = COALESCE((SELECT push_name FROM messages m WHERE m.chat_jid = chats.jid ORDER BY ts DESC LIMIT 1), last_sender),
  last_from_me = COALESCE((SELECT from_me FROM messages m WHERE m.chat_jid = chats.jid ORDER BY ts DESC LIMIT 1), last_from_me)
WHERE EXISTS (SELECT 1 FROM messages m WHERE m.chat_jid = chats.jid)`)
	return err
}

// ListChats mengembalikan semua chat (kecuali diarsipkan), terbaru di atas.
func (s *Store) ListChats(ctx context.Context) ([]Chat, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT jid, name, last_text, last_ts, unread, pinned, archived, muted, last_sender, last_from_me
		 FROM chats WHERE archived = 0 ORDER BY pinned DESC, last_ts DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Chat
	for rows.Next() {
		var c Chat
		var ts int64
		var pinned, archived, muted, fromMe int
		if err := rows.Scan(&c.JID, &c.Name, &c.LastText, &ts, &c.Unread, &pinned, &archived, &muted, &c.LastSender, &fromMe); err != nil {
			return nil, err
		}
		c.LastTS = time.Unix(ts, 0)
		c.Pinned = pinned != 0
		c.Archived = archived != 0
		c.Muted = muted != 0
		c.LastFromMe = fromMe != 0
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListArchivedChats mengembalikan chat yang diarsipkan (panel terpisah).
func (s *Store) ListArchivedChats(ctx context.Context) ([]Chat, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT jid, name, last_text, last_ts, unread, pinned, archived, muted, last_sender, last_from_me
		 FROM chats WHERE archived = 1 ORDER BY last_ts DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Chat
	for rows.Next() {
		var c Chat
		var ts int64
		var pinned, archived, muted, fromMe int
		if err := rows.Scan(&c.JID, &c.Name, &c.LastText, &ts, &c.Unread, &pinned, &archived, &muted, &c.LastSender, &fromMe); err != nil {
			return nil, err
		}
		c.LastTS = time.Unix(ts, 0)
		c.Pinned = pinned != 0
		c.Archived = archived != 0
		c.Muted = muted != 0
		c.LastFromMe = fromMe != 0
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetPinned / SetArchived / SetMuted / SetUnread memperbarui flag chat lokal
// agar UI langsung mencerminkan aksi (server di-sinkron terpisah via app-state).
func (s *Store) SetPinned(ctx context.Context, jid string, v bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE chats SET pinned = ? WHERE jid = ?`, b2i(v), jid)
	return err
}
func (s *Store) SetArchived(ctx context.Context, jid string, v bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE chats SET archived = ? WHERE jid = ?`, b2i(v), jid)
	return err
}
func (s *Store) SetMuted(ctx context.Context, jid string, v bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE chats SET muted = ? WHERE jid = ?`, b2i(v), jid)
	return err
}
func (s *Store) SetUnread(ctx context.Context, jid string, n int) error {
	_, err := s.db.ExecContext(ctx, `UPDATE chats SET unread = ? WHERE jid = ?`, n, jid)
	return err
}

// DeleteChat menghapus chat beserta pesannya dari penyimpanan lokal.
func (s *Store) DeleteChat(ctx context.Context, jid string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE chat_jid = ?`, jid); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM chats WHERE jid = ?`, jid)
	return err
}
