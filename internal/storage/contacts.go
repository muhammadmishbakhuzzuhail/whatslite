package storage

// contacts.go — label kontak lokal (nama yg disimpan pengguna di app ini) +
// lookup pushname terakhir. Label lokal TIDAK sinkron ke buku-alamat HP/WA;
// hanya tersimpan di app.db dan dipakai sebagai nama tampil otoritatif.

import (
	"context"
	"time"
)

// SetContactLabel menyimpan/menimpa nama lokal utk sebuah jid.
func (s *Store) SetContactLabel(ctx context.Context, jid, name string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO contact_labels (jid, name, created) VALUES (?, ?, ?)
		 ON CONFLICT(jid) DO UPDATE SET name=excluded.name`,
		jid, name, time.Now().Unix())
	return err
}

// DeleteContactLabel menghapus label lokal sebuah jid.
func (s *Store) DeleteContactLabel(ctx context.Context, jid string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM contact_labels WHERE jid = ?`, jid)
	return err
}

// AllContactLabels mengembalikan seluruh label lokal (jid → nama) utk di-cache.
func (s *Store) AllContactLabels(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT jid, name FROM contact_labels`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var jid, name string
		if err := rows.Scan(&jid, &name); err != nil {
			return nil, err
		}
		out[jid] = name
	}
	return out, rows.Err()
}

// LastPushName mencari pushname terakhir yg tercatat utk seorang pengirim
// (lintas semua chat). Berguna utk menamai DM yg kontaknya kosong di store WA.
func (s *Store) LastPushName(ctx context.Context, sender string) string {
	if sender == "" {
		return ""
	}
	var name string
	_ = s.db.QueryRowContext(ctx,
		`SELECT push_name FROM messages
		 WHERE sender = ? AND push_name != '' AND from_me = 0
		 ORDER BY ts DESC LIMIT 1`, sender).Scan(&name)
	return name
}
