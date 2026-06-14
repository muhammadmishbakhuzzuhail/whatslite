package storage

// chats.go — ringkasan chat untuk sidebar: penamaan, metadata history sync,
// dan perbaikan urutan/preview (RecomputeSummaries) + pembacaan daftar.

import (
	"context"
	"database/sql"
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
	name     = CASE WHEN excluded.name != '' THEN excluded.name ELSE chats.name END`,
		jid, name, ts, unread, b2i(pinned), b2i(archived))
	return err
}

// HistoryChat = metadata satu percakapan dari history-sync (utk SaveHistory).
type HistoryChat struct {
	JID      string
	Name     string
	TS       int64
	Unread   int
	Pinned   bool
	Archived bool
}

// SaveHistory menulis SELURUH history-sync (metadata chat + pesan + nama) dalam
// SATU transaksi. Penting: history-sync awal bisa berisi ribuan pesan; menulis
// satu-satu lewat SaveMessage = ribuan fsync WAL sinkron di 1 koneksi → handler
// whatsmeow ke-blok berdetik → socket putus → loop sinkron tak selesai (sidebar
// kosong, spinner nyangkut). Satu transaksi = 1 fsync saat commit (~100x).
func (s *Store) SaveHistory(ctx context.Context, chats []HistoryChat, msgs []Message, names map[string]string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	chatStmt, err := tx.PrepareContext(ctx, `
INSERT INTO chats (jid, name, last_text, last_ts, unread, pinned, archived)
VALUES (?, ?, '', ?, ?, ?, ?)
ON CONFLICT(jid) DO UPDATE SET
	last_ts  = MAX(chats.last_ts, excluded.last_ts),
	unread   = excluded.unread,
	name     = CASE WHEN excluded.name != '' THEN excluded.name ELSE chats.name END`)
	if err != nil {
		return err
	}
	defer chatStmt.Close()
	for _, c := range chats {
		if _, err := chatStmt.ExecContext(ctx, c.JID, c.Name, c.TS, c.Unread, b2i(c.Pinned), b2i(c.Archived)); err != nil {
			return err
		}
	}

	msgStmt, err := tx.PrepareContext(ctx, `
INSERT INTO messages (id, chat_jid, sender, push_name, text, kind, thumb, media, ts, from_me, status)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(chat_jid, id) DO UPDATE SET
	text = excluded.text, kind = excluded.kind, thumb = excluded.thumb, media = excluded.media,
	sender = excluded.sender, push_name = excluded.push_name,
	status = CASE
		WHEN (CASE excluded.status WHEN 'read' THEN 3 WHEN 'delivered' THEN 2 ELSE 1 END)
		   > (CASE messages.status WHEN 'read' THEN 3 WHEN 'delivered' THEN 2 ELSE 1 END)
		THEN excluded.status ELSE messages.status END`)
	if err != nil {
		return err
	}
	defer msgStmt.Close()
	// FTS disinkronkan otomatis oleh trigger messages_fts_* saat msgStmt menulis.
	for _, m := range msgs {
		st := m.Status
		if st == "" {
			st = "sent"
		}
		if _, err := msgStmt.ExecContext(ctx, m.ID, m.ChatJID, m.Sender, m.PushName, m.Text, kindOr(m.Kind), m.Thumb, m.Media, m.Timestamp.Unix(), b2i(m.FromMe), st); err != nil {
			return err
		}
	}

	if len(names) > 0 {
		nameStmt, err := tx.PrepareContext(ctx, `UPDATE chats SET name = ? WHERE jid = ?`)
		if err != nil {
			return err
		}
		defer nameStmt.Close()
		for jid, name := range names {
			if name == "" {
				continue
			}
			if _, err := nameStmt.ExecContext(ctx, name, jid); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
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
		`SELECT jid, name, last_text, last_ts, unread, pinned, archived, muted, last_sender, last_from_me,
		   (SELECT status FROM messages WHERE chat_jid = chats.jid AND from_me = 1 ORDER BY ts DESC LIMIT 1) AS last_status
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
		var status sql.NullString
		if err := rows.Scan(&c.JID, &c.Name, &c.LastText, &ts, &c.Unread, &pinned, &archived, &muted, &c.LastSender, &fromMe, &status); err != nil {
			return nil, err
		}
		c.LastTS = time.Unix(ts, 0)
		c.Pinned = pinned != 0
		c.Archived = archived != 0
		c.Muted = muted != 0
		c.LastFromMe = fromMe != 0
		c.LastStatus = status.String
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListArchivedChats mengembalikan chat yang diarsipkan (panel terpisah).
func (s *Store) ListArchivedChats(ctx context.Context) ([]Chat, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT jid, name, last_text, last_ts, unread, pinned, archived, muted, last_sender, last_from_me,
		   (SELECT status FROM messages WHERE chat_jid = chats.jid AND from_me = 1 ORDER BY ts DESC LIMIT 1) AS last_status
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
		var status sql.NullString
		if err := rows.Scan(&c.JID, &c.Name, &c.LastText, &ts, &c.Unread, &pinned, &archived, &muted, &c.LastSender, &fromMe, &status); err != nil {
			return nil, err
		}
		c.LastTS = time.Unix(ts, 0)
		c.Pinned = pinned != 0
		c.Archived = archived != 0
		c.Muted = muted != 0
		c.LastFromMe = fromMe != 0
		c.LastStatus = status.String
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetPinned / SetArchived / SetMuted / SetUnread memperbarui flag chat lokal
// agar UI langsung mencerminkan aksi (server di-sinkron terpisah via app-state).
// SetPinned/SetMuted/SetArchived = UPSERT: bila baris chat belum dibuat saat
// event app-state (events.Pin/Mute/Archive) tiba — sebelum history-sync —
// UPDATE biasa kena 0 baris → status hilang. Upsert membuat baris bila perlu.
func (s *Store) SetPinned(ctx context.Context, jid string, v bool) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO chats (jid, pinned) VALUES (?, ?) ON CONFLICT(jid) DO UPDATE SET pinned = excluded.pinned`, jid, b2i(v))
	return err
}
func (s *Store) SetArchived(ctx context.Context, jid string, v bool) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO chats (jid, archived) VALUES (?, ?) ON CONFLICT(jid) DO UPDATE SET archived = excluded.archived`, jid, b2i(v))
	return err
}
func (s *Store) SetMuted(ctx context.Context, jid string, v bool) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO chats (jid, muted) VALUES (?, ?) ON CONFLICT(jid) DO UPDATE SET muted = excluded.muted`, jid, b2i(v))
	return err
}
func (s *Store) SetUnread(ctx context.Context, jid string, n int) error {
	_, err := s.db.ExecContext(ctx, `UPDATE chats SET unread = ? WHERE jid = ?`, n, jid)
	return err
}

// MergeChat memindahkan seluruh pesan + metadata dari fromJID ke toJID lalu
// menghapus baris fromJID. Dipakai utk menyatukan chat ganda @lid↔nomor (lihat
// engine.CanonicalJID). last_ts/unread mengambil nilai tertinggi; nama non-kosong
// dipertahankan. ON CONFLICT pada (chat_jid,id) → pesan duplikat di-skip.
func (s *Store) MergeChat(ctx context.Context, fromJID, toJID string) error {
	if fromJID == toJID || fromJID == "" || toJID == "" {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Pastikan baris tujuan ada (salin metadata sumber bila belum).
	if _, err := tx.ExecContext(ctx, `
INSERT INTO chats (jid, name, last_text, last_ts, unread, pinned, archived, muted, last_sender, last_from_me)
SELECT ?, name, last_text, last_ts, unread, pinned, archived, muted, last_sender, last_from_me
FROM chats WHERE jid = ?
ON CONFLICT(jid) DO UPDATE SET
	last_ts = MAX(chats.last_ts, excluded.last_ts),
	unread  = chats.unread + excluded.unread,
	pinned  = MAX(chats.pinned, excluded.pinned),
	name    = CASE WHEN chats.name != '' THEN chats.name ELSE excluded.name END`, toJID, fromJID); err != nil {
		return err
	}
	// Pindahkan pesan (skip yg bentrok id), lalu sisa pesan duplikat dibuang.
	// FTS ikut otomatis via trigger messages_fts_au (update) & _ad (delete) —
	// ini juga membereskan duplikat FTS yg dulu muncul dari UPDATE OR IGNORE.
	if _, err := tx.ExecContext(ctx, `UPDATE OR IGNORE messages SET chat_jid = ? WHERE chat_jid = ?`, toJID, fromJID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM messages WHERE chat_jid = ?`, fromJID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM chats WHERE jid = ?`, fromJID); err != nil {
		return err
	}
	return tx.Commit()
}

// ListChatJIDs mengembalikan semua jid chat (utk pass kanonikalisasi startup).
func (s *Store) ListChatJIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT jid FROM chats`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var j string
		if err := rows.Scan(&j); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// ListRecentChatJIDs mengembalikan jid chat TERBARU (urut last_ts turun, batas
// limit). Dipakai utk subscribe presence hanya chat yg relevan — bukan ratusan
// (hemat IQ/baterai; riset whatsmeow: jangan bulk-subscribe).
func (s *Store) ListRecentChatJIDs(ctx context.Context, limit int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT jid FROM chats ORDER BY last_ts DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var j string
		if err := rows.Scan(&j); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// ClearMessages menghapus SEMUA pesan satu chat tapi MEMPERTAHANKAN baris chat
// (chat tetap di sidebar, hanya isinya kosong). Ringkasan direset.
func (s *Store) ClearMessages(ctx context.Context, jid string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE chat_jid = ?`, jid); err != nil {
		return err
	}
	for _, q := range []string{
		`DELETE FROM reactions WHERE chat_jid = ?`,
		`DELETE FROM receipts WHERE chat_jid = ?`,
	} {
		_, _ = s.db.ExecContext(ctx, q, jid)
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE chats SET last_text='', last_sender='', last_from_me=0, unread=0 WHERE jid = ?`, jid)
	return err
}

// DeleteChat menghapus chat beserta pesannya dari penyimpanan lokal.
func (s *Store) DeleteChat(ctx context.Context, jid string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE chat_jid = ?`, jid); err != nil {
		return err
	}
	// FTS dibersihkan otomatis oleh trigger messages_fts_ad (per pesan dihapus).
	// Tabel turunan lain dibuang manual.
	for _, q := range []string{
		`DELETE FROM reactions WHERE chat_jid = ?`,
		`DELETE FROM receipts WHERE chat_jid = ?`,
	} {
		_, _ = s.db.ExecContext(ctx, q, jid)
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM chats WHERE jid = ?`, jid)
	return err
}
