// Package storage menyimpan chat & pesan ke SQLite (app.db).
//
// Sesuai prinsip ringan: DB hanya menyimpan teks/metadata; media (nanti)
// disimpan sebagai file terpisah, DB cukup menyimpan path-nya.
//
// File-file paket:
//   - storage.go  : koneksi, skema/migrasi, model (Chat/Message), util
//   - chats.go    : tulis/baca ringkasan chat (sidebar) + perbaikan urutan
//   - messages.go : tulis/baca pesan + media on-demand
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Chat adalah ringkasan satu percakapan untuk daftar sidebar.
type Chat struct {
	JID      string
	Name     string
	LastText string
	LastTS     time.Time
	Unread     int
	Pinned     bool
	Archived   bool
	Muted      bool
	LastSender string // nama pengirim pesan terakhir (utk prefix preview grup)
	LastFromMe bool
	LastStatus string // status pesan keluar terakhir (sent/delivered/read) → centang sidebar
}

// Message adalah satu pesan dalam percakapan.
type Message struct {
	ID        string
	ChatJID   string
	Sender    string
	PushName  string
	Text      string
	Kind      string // text | image | video | sticker
	Thumb     string // data-URI thumbnail (bila ada)
	Media     string // base64 proto (utk download media penuh)
	Timestamp time.Time
	FromMe    bool

	QuotedID     string // balasan: id pesan dikutip ("" = bukan balasan)
	QuotedSender string
	QuotedText   string

	Status   string // sent | delivered | read (pesan sendiri)
	Pinned   bool   // disematkan di chat
	Edited   bool   // pernah disunting
}

// Store membungkus koneksi SQLite ke app.db.
type Store struct {
	db *sql.DB
}

// New membuka app.db di path dan menjalankan migrasi skema.
func New(ctx context.Context, path string) (*Store, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open app.db: %w", err)
	}
	// WAL: satu penulis + banyak pembaca BISA bersamaan. Dgn MaxOpenConns(1)
	// pembaca (buka chat) antre di belakang penulis → saat history-sync membanjiri
	// DB, chat tampak KOSONG karena GetMessages menunggu koneksi. Beberapa koneksi
	// → pembaca jalan paralel dgn penulis (snapshot WAL); tulis tetap di-serialkan
	// oleh antrian writer app + busy_timeout menangani tabrakan jarang.
	db.SetMaxOpenConns(4)

	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS chats (
	jid       TEXT PRIMARY KEY,
	name      TEXT NOT NULL DEFAULT '',
	last_text TEXT NOT NULL DEFAULT '',
	last_ts   INTEGER NOT NULL DEFAULT 0,
	unread    INTEGER NOT NULL DEFAULT 0,
	pinned    INTEGER NOT NULL DEFAULT 0,
	archived  INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS messages (
	id        TEXT NOT NULL,
	chat_jid  TEXT NOT NULL,
	sender    TEXT NOT NULL DEFAULT '',
	push_name TEXT NOT NULL DEFAULT '',
	text      TEXT NOT NULL DEFAULT '',
	kind      TEXT NOT NULL DEFAULT 'text',
	thumb     TEXT NOT NULL DEFAULT '',
	media     TEXT NOT NULL DEFAULT '',
	ts        INTEGER NOT NULL DEFAULT 0,
	from_me   INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (chat_jid, id)
);
CREATE INDEX IF NOT EXISTS idx_messages_chat_ts ON messages(chat_jid, ts);
CREATE INDEX IF NOT EXISTS idx_messages_sender_ts ON messages(sender, ts);
`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	// Untuk DB lama: tambah kolom bila belum ada (abaikan error "duplicate column").
	s.db.ExecContext(ctx, `ALTER TABLE messages ADD COLUMN kind TEXT NOT NULL DEFAULT 'text'`)
	s.db.ExecContext(ctx, `ALTER TABLE messages ADD COLUMN thumb TEXT NOT NULL DEFAULT ''`)
	s.db.ExecContext(ctx, `ALTER TABLE messages ADD COLUMN media TEXT NOT NULL DEFAULT ''`)
	s.db.ExecContext(ctx, `ALTER TABLE chats ADD COLUMN unread INTEGER NOT NULL DEFAULT 0`)
	s.db.ExecContext(ctx, `ALTER TABLE chats ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0`)
	s.db.ExecContext(ctx, `ALTER TABLE chats ADD COLUMN archived INTEGER NOT NULL DEFAULT 0`)
	s.db.ExecContext(ctx, `ALTER TABLE chats ADD COLUMN muted INTEGER NOT NULL DEFAULT 0`)
	s.db.ExecContext(ctx, `ALTER TABLE chats ADD COLUMN last_sender TEXT NOT NULL DEFAULT ''`)
	s.db.ExecContext(ctx, `ALTER TABLE chats ADD COLUMN last_from_me INTEGER NOT NULL DEFAULT 0`)
	s.db.ExecContext(ctx, `ALTER TABLE messages ADD COLUMN quoted_id TEXT NOT NULL DEFAULT ''`)
	s.db.ExecContext(ctx, `ALTER TABLE messages ADD COLUMN quoted_sender TEXT NOT NULL DEFAULT ''`)
	s.db.ExecContext(ctx, `ALTER TABLE messages ADD COLUMN quoted_text TEXT NOT NULL DEFAULT ''`)
	s.db.ExecContext(ctx, `ALTER TABLE messages ADD COLUMN starred INTEGER NOT NULL DEFAULT 0`)
	s.db.ExecContext(ctx, `ALTER TABLE messages ADD COLUMN status TEXT NOT NULL DEFAULT 'sent'`)
	s.db.ExecContext(ctx, `ALTER TABLE messages ADD COLUMN pinned_in_chat INTEGER NOT NULL DEFAULT 0`)
	s.db.ExecContext(ctx, `ALTER TABLE messages ADD COLUMN edited INTEGER NOT NULL DEFAULT 0`)
	// Reaksi per (pesan, pengirim) — satu reaksi terakhir per orang.
	s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS reactions (
		chat_jid TEXT NOT NULL,
		msg_id   TEXT NOT NULL,
		sender   TEXT NOT NULL,
		emoji    TEXT NOT NULL,
		ts       INTEGER NOT NULL,
		PRIMARY KEY (chat_jid, msg_id, sender)
	)`)
	// Suara polling per-pemilih (satu baris terakhir per voter) → rekap hasil.
	s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS poll_votes (
		poll_id TEXT NOT NULL,
		voter   TEXT NOT NULL,
		options TEXT NOT NULL,
		ts      INTEGER NOT NULL,
		PRIMARY KEY (poll_id, voter)
	)`)
	// Label kontak lokal — nama yg disimpan pengguna di app ini (BUKAN sync ke
	// HP/WA). Otoritatif atas nama tampil: kalau ada di sini, pakai ini.
	s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS contact_labels (
		jid     TEXT PRIMARY KEY,
		name    TEXT NOT NULL,
		created INTEGER NOT NULL DEFAULT 0
	)`)
	// Tanda terima per-penerima (grup: per anggota) → daftar baca di "Info pesan".
	s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS receipts (
		chat_jid  TEXT NOT NULL,
		msg_id    TEXT NOT NULL,
		recipient TEXT NOT NULL,
		status    TEXT NOT NULL,
		ts        INTEGER NOT NULL,
		PRIMARY KEY (chat_jid, msg_id, recipient)
	)`)

	// FTS5 EXTERNAL-CONTENT: index hanya kolom `text`, baris asli tetap di
	// `messages` (tanpa duplikasi teks). Sinkron OTOMATIS via TRIGGER → tak ada
	// sinkron manual yg bisa drift/duplikat. Migrasi sekali dari FTS standalone
	// lama (skema tanpa content=) → drop & bangun ulang dari messages.
	var ftsDef string
	s.db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name='messages_fts'`).Scan(&ftsDef)
	if !strings.Contains(ftsDef, "content=") {
		s.db.ExecContext(ctx, `DROP TABLE IF EXISTS messages_fts`)
		if _, err := s.db.ExecContext(ctx,
			`CREATE VIRTUAL TABLE messages_fts USING fts5(text, content='messages', content_rowid='rowid')`); err != nil {
			return fmt.Errorf("fts create: %w", err)
		}
		s.db.ExecContext(ctx, `INSERT INTO messages_fts(rowid, text) SELECT rowid, text FROM messages WHERE text <> ''`)
	}
	// Trigger sinkron (idempoten: drop+create tiap boot).
	for _, q := range []string{
		`DROP TRIGGER IF EXISTS messages_fts_ai`,
		`CREATE TRIGGER messages_fts_ai AFTER INSERT ON messages BEGIN
			INSERT INTO messages_fts(rowid, text) VALUES (new.rowid, new.text); END`,
		`DROP TRIGGER IF EXISTS messages_fts_ad`,
		`CREATE TRIGGER messages_fts_ad AFTER DELETE ON messages BEGIN
			INSERT INTO messages_fts(messages_fts, rowid, text) VALUES('delete', old.rowid, old.text); END`,
		`DROP TRIGGER IF EXISTS messages_fts_au`,
		`CREATE TRIGGER messages_fts_au AFTER UPDATE ON messages BEGIN
			INSERT INTO messages_fts(messages_fts, rowid, text) VALUES('delete', old.rowid, old.text);
			INSERT INTO messages_fts(rowid, text) VALUES (new.rowid, new.text); END`,
	} {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("fts trigger: %w", err)
		}
	}
	return nil
}

// Close menutup koneksi DB.
func (s *Store) Close() error { return s.db.Close() }

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func kindOr(k string) string {
	if k == "" {
		return "text"
	}
	return k
}

// previewLabel = teks ringkas untuk daftar chat saat pesan media tanpa caption.
func previewLabel(kind string) string {
	switch kind {
	case "image":
		return "🖼️ Foto"
	case "video":
		return "🎬 Video"
	case "sticker":
		return "🏷️ Stiker"
	}
	return ""
}
