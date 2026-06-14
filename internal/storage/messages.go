// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package storage

// messages.go — tulis/baca pesan. SaveMessage juga memperbarui ringkasan chat
// (last_text/last_ts) agar sidebar mengikuti pesan terbaru.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// SavePlaceholder menyisipkan baris pesan "pending" (gagal didekripsi, sedang
// diminta ulang dari HP). INSERT-OR-IGNORE: bila pesan asli sudah/akan tiba,
// SaveMessage meng-UPSERT menimpa kind/text. Tak menyentuh ringkasan chat
// (sidebar tak terganggu). Hilang sendiri saat isi asli tiba.
func (s *Store) SavePlaceholder(ctx context.Context, id, chatJID, sender string, ts time.Time, fromMe bool) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO messages (id, chat_jid, sender, push_name, text, kind, thumb, media, ts, from_me, quoted_id, quoted_sender, quoted_text, status, expire_at)
VALUES (?, ?, ?, '', '', 'pending', '', '', ?, ?, '', '', '', 'sent', 0)
ON CONFLICT(chat_jid, id) DO NOTHING`,
		id, chatJID, sender, ts.Unix(), b2i(fromMe))
	if err != nil {
		return fmt.Errorf("save placeholder: %w", err)
	}
	return nil
}

// SaveMessage menyimpan pesan dan memperbarui ringkasan chat.
func (s *Store) SaveMessage(ctx context.Context, m Message) error {
	st := m.Status
	if st == "" {
		st = "sent"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO messages (id, chat_jid, sender, push_name, text, kind, thumb, media, ts, from_me, quoted_id, quoted_sender, quoted_text, status, expire_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(chat_jid, id) DO UPDATE SET
	text = excluded.text, kind = excluded.kind, thumb = excluded.thumb, media = excluded.media,
	sender = excluded.sender, push_name = excluded.push_name,
	quoted_id = excluded.quoted_id, quoted_sender = excluded.quoted_sender, quoted_text = excluded.quoted_text,
	expire_at = CASE WHEN excluded.expire_at > 0 THEN excluded.expire_at ELSE messages.expire_at END,
	status = CASE
		WHEN (CASE excluded.status WHEN 'read' THEN 3 WHEN 'delivered' THEN 2 ELSE 1 END)
		   > (CASE messages.status WHEN 'read' THEN 3 WHEN 'delivered' THEN 2 ELSE 1 END)
		THEN excluded.status ELSE messages.status END`,
		m.ID, m.ChatJID, m.Sender, m.PushName, m.Text, kindOr(m.Kind), m.Thumb, m.Media, m.Timestamp.Unix(), b2i(m.FromMe),
		m.QuotedID, m.QuotedSender, m.QuotedText, st, m.ExpireAt)
	if err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	// Ringkasan chat. Preview = teks; bila kosong (mis. foto tanpa caption) → label.
	preview := m.Text
	if preview == "" {
		preview = previewLabel(m.Kind)
	}
	// Nama HANYA untuk 1:1 dari push name lawan bicara; grup di-resolve di atas.
	name := ""
	if !m.FromMe && !strings.HasSuffix(m.ChatJID, "@g.us") {
		name = m.PushName
	}
	// Pengirim utk prefix preview grup ("Budi: …"); kosong utk 1:1 / from_me.
	lastSender := ""
	if !m.FromMe && strings.HasSuffix(m.ChatJID, "@g.us") {
		lastSender = m.PushName
	}
	// last_from_me / last_sender HARUS ikut ter-update (dulu cuma RecomputeSummaries
	// startup → stale → centang sidebar tak muncul utk pesan baru terkirim).
	_, err = s.db.ExecContext(ctx, `
INSERT INTO chats (jid, name, last_text, last_ts, last_from_me, last_sender)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(jid) DO UPDATE SET
	last_text    = CASE WHEN excluded.last_ts >= chats.last_ts THEN excluded.last_text    ELSE chats.last_text    END,
	last_from_me = CASE WHEN excluded.last_ts >= chats.last_ts THEN excluded.last_from_me ELSE chats.last_from_me END,
	last_sender  = CASE WHEN excluded.last_ts >= chats.last_ts THEN excluded.last_sender  ELSE chats.last_sender  END,
	last_ts      = MAX(chats.last_ts, excluded.last_ts),
	name         = CASE WHEN chats.name = '' THEN excluded.name ELSE chats.name END`,
		m.ChatJID, name, preview, m.Timestamp.Unix(), b2i(m.FromMe), lastSender)
	if err != nil {
		return fmt.Errorf("upsert chat: %w", err)
	}
	// FTS disinkronkan otomatis oleh trigger messages_fts_* (insert/update).
	return nil
}

// ListMessages mengembalikan hingga `limit` pesan terbaru, diurutkan lama->baru.
func (s *Store) ListMessages(ctx context.Context, chatJID string, limit int) ([]Message, error) {
	// Pilih kolom eksplisit (TANPA media/proto besar) → list ringan.
	rows, err := s.db.QueryContext(ctx, `
SELECT id, chat_jid, sender, push_name, text, kind, thumb, ts, from_me, quoted_id, quoted_sender, quoted_text, status, pinned_in_chat, edited, revoked FROM (
	SELECT id, chat_jid, sender, push_name, text, kind, thumb, ts, from_me, quoted_id, quoted_sender, quoted_text, status, pinned_in_chat, edited, revoked
	FROM messages WHERE chat_jid = ? ORDER BY ts DESC LIMIT ?
) ORDER BY ts ASC`, chatJID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var m Message
		var ts int64
		var fromMe, pinned, edited, revoked int
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.Sender, &m.PushName, &m.Text, &m.Kind, &m.Thumb, &ts, &fromMe,
			&m.QuotedID, &m.QuotedSender, &m.QuotedText, &m.Status, &pinned, &edited, &revoked); err != nil {
			return nil, err
		}
		m.Timestamp = time.Unix(ts, 0)
		m.FromMe = fromMe != 0
		m.Pinned = pinned != 0
		m.Edited = edited != 0
		m.Revoked = revoked != 0
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListStatuses mengembalikan pesan status (chat=status@broadcast) sejak `since`,
// terbaru dulu, dgn thumb (utk ring + viewer). Tanpa media proto (ringan).
func (s *Store) ListStatuses(ctx context.Context, since time.Time) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, chat_jid, sender, push_name, text, kind, thumb, ts, from_me
FROM messages WHERE chat_jid = 'status@broadcast' AND ts >= ?
ORDER BY ts DESC`, since.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		var ts int64
		var fromMe int
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.Sender, &m.PushName, &m.Text, &m.Kind, &m.Thumb, &ts, &fromMe); err != nil {
			return nil, err
		}
		m.Timestamp = time.Unix(ts, 0)
		m.FromMe = fromMe != 0
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListMedia mengembalikan pesan media satu chat (foto/video/stiker/gif/doc),
// terbaru dulu, untuk galeri "Media, tautan & dok" di panel info. Ringan (thumb).
func (s *Store) ListMedia(ctx context.Context, jid string) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, chat_jid, sender, push_name, text, kind, thumb, ts, from_me
FROM messages
WHERE chat_jid = ? AND kind IN ('image','video','sticker','gif','document')
ORDER BY ts DESC LIMIT 300`, jid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		var ts int64
		var fromMe int
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.Sender, &m.PushName, &m.Text, &m.Kind, &m.Thumb, &ts, &fromMe); err != nil {
			return nil, err
		}
		m.Timestamp = time.Unix(ts, 0)
		m.FromMe = fromMe != 0
		out = append(out, m)
	}
	return out, rows.Err()
}

// SetMessageStatus menaikkan status pesan sendiri (sent→delivered→read). Tak
// pernah menurunkan (receipt bisa tiba tak berurutan).
func (s *Store) SetMessageStatus(ctx context.Context, chatJID string, ids []string, status string) error {
	rank := map[string]int{"sent": 1, "delivered": 2, "read": 3}
	r := rank[status]
	if r == 0 || len(ids) == 0 {
		return nil
	}
	ph := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	q := `UPDATE messages SET status = ? WHERE chat_jid = ? AND from_me = 1 AND id IN (` + ph + `)
AND (CASE status WHEN 'read' THEN 3 WHEN 'delivered' THEN 2 ELSE 1 END) < ?`
	args := []any{status, chatJID}
	for _, id := range ids {
		args = append(args, id)
	}
	args = append(args, r)
	_, err := s.db.ExecContext(ctx, q, args...)
	return err
}

// Reaction = satu reaksi (pengirim + emoji).
type Reaction struct {
	Sender string
	Emoji  string
}

// SetReaction menyimpan/menghapus reaksi seseorang pada pesan (emoji ""=hapus).
func (s *Store) SetReaction(ctx context.Context, chatJID, msgID, sender, emoji string, ts time.Time) error {
	if strings.TrimSpace(emoji) == "" {
		_, err := s.db.ExecContext(ctx,
			`DELETE FROM reactions WHERE chat_jid=? AND msg_id=? AND sender=?`, chatJID, msgID, sender)
		return err
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO reactions (chat_jid, msg_id, sender, emoji, ts) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(chat_jid, msg_id, sender) DO UPDATE SET emoji=excluded.emoji, ts=excluded.ts`,
		chatJID, msgID, sender, emoji, ts.Unix())
	return err
}

// ReactionsForChat mengembalikan semua reaksi chat, dipetakan per msg_id.
func (s *Store) ReactionsForChat(ctx context.Context, chatJID string) (map[string][]Reaction, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT msg_id, sender, emoji FROM reactions WHERE chat_jid=?`, chatJID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]Reaction{}
	for rows.Next() {
		var id string
		var r Reaction
		if err := rows.Scan(&id, &r.Sender, &r.Emoji); err != nil {
			return nil, err
		}
		out[id] = append(out[id], r)
	}
	return out, rows.Err()
}

// SetPollVote menyimpan suara terbaru seorang pemilih (mengganti yang lama).
func (s *Store) SetPollVote(ctx context.Context, pollID, voter string, options []string, ts time.Time) error {
	j, _ := json.Marshal(options)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO poll_votes (poll_id, voter, options, ts) VALUES (?, ?, ?, ?)
ON CONFLICT(poll_id, voter) DO UPDATE SET options = excluded.options, ts = excluded.ts`,
		pollID, voter, string(j), ts.Unix())
	return err
}

// PollVoteCounts mengembalikan jumlah suara per opsi + total pemilih.
func (s *Store) PollVoteCounts(ctx context.Context, pollID string) (map[string]int, int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT options FROM poll_votes WHERE poll_id = ?`, pollID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	counts := map[string]int{}
	total := 0
	for rows.Next() {
		var oj string
		if err := rows.Scan(&oj); err != nil {
			return nil, 0, err
		}
		var opts []string
		json.Unmarshal([]byte(oj), &opts)
		if len(opts) > 0 {
			total++
		}
		for _, o := range opts {
			counts[o]++
		}
	}
	return counts, total, rows.Err()
}

// Receipt = tanda terima satu penerima atas satu pesan.
type Receipt struct {
	Recipient string
	Status    string // delivered | read
	Timestamp time.Time
}

// SetReceipt mencatat tanda terima per-penerima (naik saja: delivered→read).
func (s *Store) SetReceipt(ctx context.Context, chatJID, msgID, recipient, status string, ts time.Time) error {
	if recipient == "" || (status != "delivered" && status != "read") {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO receipts (chat_jid, msg_id, recipient, status, ts)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(chat_jid, msg_id, recipient) DO UPDATE SET
	status = CASE WHEN excluded.status = 'read' THEN 'read' ELSE receipts.status END,
	ts     = excluded.ts`,
		chatJID, msgID, recipient, status, ts.Unix())
	return err
}

// SetReceipts mencatat tanda terima satu penerima atas BANYAK pesan sekaligus
// dalam SATU transaksi. Receipt grup bisa berisi puluhan id; menulisnya satu
// per satu = puluhan fsync sinkron yang memblok loop baca socket whatsmeow
// (gejala: "Node handling took 9s" lalu websocket EOF). Batch → 1 fsync.
func (s *Store) SetReceipts(ctx context.Context, chatJID string, ids []string, recipient, status string, ts time.Time) error {
	if recipient == "" || len(ids) == 0 || (status != "delivered" && status != "read") {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO receipts (chat_jid, msg_id, recipient, status, ts)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(chat_jid, msg_id, recipient) DO UPDATE SET
	status = CASE WHEN excluded.status = 'read' THEN 'read' ELSE receipts.status END,
	ts     = excluded.ts`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, id := range ids {
		if _, err := stmt.ExecContext(ctx, chatJID, id, recipient, status, ts.Unix()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListReceipts mengembalikan tanda terima sebuah pesan (terbaru dulu).
func (s *Store) ListReceipts(ctx context.Context, chatJID, msgID string) ([]Receipt, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT recipient, status, ts FROM receipts WHERE chat_jid = ? AND msg_id = ? ORDER BY ts DESC`, chatJID, msgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Receipt
	for rows.Next() {
		var r Receipt
		var ts int64
		if err := rows.Scan(&r.Recipient, &r.Status, &ts); err != nil {
			return nil, err
		}
		r.Timestamp = time.Unix(ts, 0)
		out = append(out, r)
	}
	return out, rows.Err()
}

// SetPinnedInChat menyemat / melepas sebuah pesan di dalam chat.
func (s *Store) SetPinnedInChat(ctx context.Context, chatJID, id string, pinned bool) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE messages SET pinned_in_chat = ? WHERE chat_jid = ? AND id = ?`, b2i(pinned), chatJID, id)
	return err
}

// ListPinned mengembalikan pesan yang disematkan di chat (terbaru dulu).
func (s *Store) ListPinned(ctx context.Context, chatJID string) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, chat_jid, sender, push_name, text, kind, thumb, ts, from_me
FROM messages WHERE chat_jid = ? AND pinned_in_chat = 1 ORDER BY ts DESC`, chatJID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		var ts int64
		var fromMe int
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.Sender, &m.PushName, &m.Text, &m.Kind, &m.Thumb, &ts, &fromMe); err != nil {
			return nil, err
		}
		m.Timestamp = time.Unix(ts, 0)
		m.FromMe = fromMe != 0
		m.Pinned = true
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetMedia mengembalikan proto media (base64) satu pesan utk download on-demand.
func (s *Store) GetMedia(ctx context.Context, chatJID, id string) (string, error) {
	var media string
	err := s.db.QueryRowContext(ctx,
		`SELECT media FROM messages WHERE chat_jid = ? AND id = ?`, chatJID, id).Scan(&media)
	if err != nil {
		return "", err
	}
	return media, nil
}

// ListMessagesBefore mengembalikan hingga `limit` pesan yang lebih LAMA dari
// beforeTS (untuk pagination "scroll ke atas" muat riwayat lama), urut lama→baru.
func (s *Store) ListMessagesBefore(ctx context.Context, chatJID string, beforeTS int64, limit int) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, chat_jid, sender, push_name, text, kind, thumb, ts, from_me, quoted_id, quoted_sender, quoted_text, status, pinned_in_chat, edited, revoked FROM (
	SELECT id, chat_jid, sender, push_name, text, kind, thumb, ts, from_me, quoted_id, quoted_sender, quoted_text, status, pinned_in_chat, edited, revoked
	FROM messages WHERE chat_jid = ? AND ts < ? ORDER BY ts DESC LIMIT ?
) ORDER BY ts ASC`, chatJID, beforeTS, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		var ts int64
		var fromMe, pinned, edited, revoked int
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.Sender, &m.PushName, &m.Text, &m.Kind, &m.Thumb, &ts, &fromMe,
			&m.QuotedID, &m.QuotedSender, &m.QuotedText, &m.Status, &pinned, &edited, &revoked); err != nil {
			return nil, err
		}
		m.Timestamp = time.Unix(ts, 0)
		m.FromMe = fromMe != 0
		m.Pinned = pinned != 0
		m.Edited = edited != 0
		m.Revoked = revoked != 0
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetMessage mengambil satu pesan (utk teruskan/forward).
func (s *Store) GetMessage(ctx context.Context, chatJID, id string) (Message, error) {
	var m Message
	var ts int64
	var fromMe int
	err := s.db.QueryRowContext(ctx, `
SELECT id, chat_jid, sender, push_name, text, kind, thumb, media, ts, from_me, status
FROM messages WHERE chat_jid = ? AND id = ?`, chatJID, id).
		Scan(&m.ID, &m.ChatJID, &m.Sender, &m.PushName, &m.Text, &m.Kind, &m.Thumb, &m.Media, &ts, &fromMe, &m.Status)
	if err != nil {
		return Message{}, err
	}
	m.Timestamp = time.Unix(ts, 0)
	m.FromMe = fromMe != 0
	return m, nil
}

// LastMessage mengembalikan metadata pesan terakhir chat (utk patch app-state
// archive/mark-read yang butuh kunci pesan terakhir). found=false bila kosong.
func (s *Store) LastMessage(ctx context.Context, chatJID string) (id string, ts time.Time, fromMe, found bool, err error) {
	var unix int64
	var fm int
	row := s.db.QueryRowContext(ctx,
		`SELECT id, ts, from_me FROM messages WHERE chat_jid = ? ORDER BY ts DESC LIMIT 1`, chatJID)
	switch e := row.Scan(&id, &unix, &fm); e {
	case nil:
		return id, time.Unix(unix, 0), fm != 0, true, nil
	case sql.ErrNoRows:
		return "", time.Time{}, false, false, nil
	default:
		return "", time.Time{}, false, false, e
	}
}

// SetStarred menandai/melepas bintang satu pesan (lokal).
func (s *Store) SetStarred(ctx context.Context, chatJID, id string, v bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE messages SET starred = ? WHERE chat_jid = ? AND id = ?`, b2i(v), chatJID, id)
	return err
}

// ListStarred mengembalikan pesan berbintang lintas chat (terbaru dulu).
func (s *Store) ListStarred(ctx context.Context, limit int) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, chat_jid, sender, push_name, text, kind, thumb, ts, from_me, quoted_id, quoted_sender, quoted_text
FROM messages WHERE starred = 1 ORDER BY ts DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		var ts int64
		var fromMe int
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.Sender, &m.PushName, &m.Text, &m.Kind, &m.Thumb, &ts, &fromMe,
			&m.QuotedID, &m.QuotedSender, &m.QuotedText); err != nil {
			return nil, err
		}
		m.Timestamp = time.Unix(ts, 0)
		m.FromMe = fromMe != 0
		out = append(out, m)
	}
	return out, rows.Err()
}

// EditText mengubah teks pesan (sunting) tanpa mengubah timestamp/urutan + sync FTS.
func (s *Store) EditText(ctx context.Context, chatJID, id, newText string) error {
	// Trigger messages_fts_au menyinkronkan FTS dari UPDATE ini.
	_, err := s.db.ExecContext(ctx, `UPDATE messages SET text = ?, edited = 1 WHERE chat_jid = ? AND id = ?`, newText, chatJID, id)
	return err
}

// DeleteLocalMessage menghapus satu pesan dari penyimpanan lokal (delete-for-me).
func (s *Store) DeleteLocalMessage(ctx context.Context, chatJID, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE chat_jid = ? AND id = ?`, chatJID, id)
	// FTS dibersihkan oleh trigger messages_fts_ad; reaksi dibuang manual.
	_, _ = s.db.ExecContext(ctx, `DELETE FROM reactions WHERE chat_jid = ? AND msg_id = ?`, chatJID, id)
	return err
}

// MarkDeleted menandai pesan "dihapus" (revoke/hapus-utk-semua) — baris tetap ada
// agar muncul placeholder "pesan dihapus" seperti WhatsApp. Kosongkan isi/media.
// MarkDeleted menandai pesan ditarik pengirim (hapus-utk-semua). ANTI-DELETE:
// konten asli TETAP disimpan (text/media/thumb/kind utuh), hanya set revoked=1.
// UI memutuskan tampilkan isi+tag "dihapus" atau placeholder (toggle pengguna).
func (s *Store) MarkDeleted(ctx context.Context, chatJID, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE messages SET revoked = 1 WHERE chat_jid = ? AND id = ?`, chatJID, id)
	return err
}

// MarkDeletedHard = perilaku WhatsApp asli (anti-delete OFF): tandai ditarik DAN
// kosongkan isi (teks/thumb/media) → tampil "pesan dihapus" tanpa konten.
func (s *Store) MarkDeletedHard(ctx context.Context, chatJID, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE messages SET revoked = 1, text = '', thumb = '', media = '' WHERE chat_jid = ? AND id = ?`, chatJID, id)
	return err
}

// OldestMessage mengembalikan pesan TERTUA tersimpan utk chat (utk minta history
// lebih lama on-demand: server kirim pesan SEBELUM ini).
func (s *Store) OldestMessage(ctx context.Context, chatJID string) (id string, fromMe bool, ts int64, ok bool) {
	var fm int
	if err := s.db.QueryRowContext(ctx, `SELECT id, from_me, ts FROM messages WHERE chat_jid = ? ORDER BY ts ASC LIMIT 1`, chatJID).Scan(&id, &fm, &ts); err != nil {
		return "", false, 0, false
	}
	return id, fm != 0, ts, true
}

// SearchMessages mencari isi pesan via FTS5 (cepat), lintas chat. Query
// disanitasi → prefix-match per token (mis. "es cend" → es* cend*).
func (s *Store) SearchMessages(ctx context.Context, query string, limit int) ([]Message, error) {
	match := ftsQuery(query)
	if match == "" {
		return nil, nil
	}
	// External-content FTS: join balik ke messages via rowid utk ambil metadata.
	rows, err := s.db.QueryContext(ctx, `
SELECT m.id, m.chat_jid, m.text, m.ts
FROM messages_fts f JOIN messages m ON m.rowid = f.rowid
WHERE messages_fts MATCH ? ORDER BY m.ts DESC LIMIT ?`, match, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		var ts int64
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.Text, &ts); err != nil {
			return nil, err
		}
		m.Timestamp = time.Unix(ts, 0)
		out = append(out, m)
	}
	return out, rows.Err()
}

// SearchAdvanced: pencarian lintas-chat berfilter jenis. typ ""/"text" → FTS;
// "link" → pesan mengandung URL; lainnya → by kind. query opsional (boleh kosong
// untuk jelajah semua media/dok).
func (s *Store) SearchAdvanced(ctx context.Context, query, typ string, limit int) ([]Message, error) {
	if typ == "" || typ == "text" {
		return s.SearchMessages(ctx, query, limit)
	}
	where := "kind = ?"
	args := []any{typ}
	if typ == "link" {
		where = "text LIKE '%http%'"
		args = []any{}
	}
	if q := strings.TrimSpace(query); q != "" {
		where += " AND lower(text) LIKE ?"
		args = append(args, "%"+strings.ToLower(q)+"%")
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, chat_jid, text, kind, ts FROM messages WHERE `+where+` ORDER BY ts DESC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		var ts int64
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.Text, &m.Kind, &ts); err != nil {
			return nil, err
		}
		m.Timestamp = time.Unix(ts, 0)
		out = append(out, m)
	}
	return out, rows.Err()
}

// ftsQuery membangun ekspresi MATCH aman: ambil token alnum, tiap token jadi
// prefix ("tok*"), digabung AND. Cegah error sintaks FTS dari tanda baca.
func ftsQuery(q string) string {
	var toks []string
	// Pisah pada non-(huruf/angka) UNICODE → token CJK/diakritik/non-latin ikut
	// tercari (bukan cuma ASCII). Kutip tiap token utk aman dari sintaks FTS5.
	for _, f := range strings.FieldsFunc(q, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r))
	}) {
		toks = append(toks, `"`+f+`"*`)
	}
	return strings.Join(toks, " ")
}
