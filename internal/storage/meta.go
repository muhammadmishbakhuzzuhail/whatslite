// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package storage

// meta.go — KV setelan ringan (app_meta) + retensi pesan & pemeliharaan DB.
// Berat app berasal dari riwayat pesan tak terbatas; retensi + VACUUM menjaganya
// ramping tanpa mengorbankan pesan penting (berbintang/disematkan dikecualikan).

import "context"

// GetMeta membaca nilai setelan; def bila belum ada.
func (s *Store) GetMeta(ctx context.Context, key, def string) string {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM app_meta WHERE key=?`, key).Scan(&v)
	if err != nil {
		return def
	}
	return v
}

// SetMeta menyimpan nilai setelan.
func (s *Store) SetMeta(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO app_meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, value)
	return err
}

// DelMeta menghapus satu kunci setelan (no-op bila tak ada).
func (s *Store) DelMeta(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM app_meta WHERE key=?`, key)
	return err
}

// ListMeta mengembalikan semua kunci berawalan prefix → nilai (kunci utuh).
func (s *Store) ListMeta(ctx context.Context, prefix string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM app_meta WHERE key LIKE ?`, prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// PruneMessages menghapus pesan lebih lama dari cutoff (unix), KECUALI yang
// berbintang/disematkan. Membersihkan reaksi/receipt yatim. cutoff<=0 → no-op.
// FTS dibersihkan otomatis oleh trigger messages_fts_ad.
func (s *Store) PruneMessages(ctx context.Context, cutoff int64) (int64, error) {
	if cutoff <= 0 {
		return 0, nil
	}
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM messages WHERE ts < ? AND starred=0 AND pinned_in_chat=0`, cutoff)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	// Buang turunan yang pesannya tak ada lagi.
	_, _ = s.db.ExecContext(ctx,
		`DELETE FROM reactions WHERE NOT EXISTS (SELECT 1 FROM messages m WHERE m.chat_jid=reactions.chat_jid AND m.id=reactions.msg_id)`)
	_, _ = s.db.ExecContext(ctx,
		`DELETE FROM receipts WHERE NOT EXISTS (SELECT 1 FROM messages m WHERE m.chat_jid=receipts.chat_jid AND m.id=receipts.msg_id)`)
	return n, nil
}

// SweepExpired menghapus pesan disappearing yang sudah kedaluwarsa (expire_at>0
// dan < now). Mengembalikan jumlah terhapus.
func (s *Store) SweepExpired(ctx context.Context, now int64) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE expire_at > 0 AND expire_at < ?`, now)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// KindStat = ringkasan penyimpanan per jenis pesan.
type KindStat struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
	Bytes int64  `json:"bytes"`
}

// StorageStats mengembalikan ukuran app.db + rincian per-jenis pesan.
func (s *Store) StorageStats(ctx context.Context) (msgCount int, dbBytes int64, kinds []KindStat, err error) {
	kinds = []KindStat{}
	_ = s.db.QueryRowContext(ctx, `SELECT count(*) FROM messages`).Scan(&msgCount)
	var pc, ps int64
	_ = s.db.QueryRowContext(ctx, `PRAGMA page_count`).Scan(&pc)
	_ = s.db.QueryRowContext(ctx, `PRAGMA page_size`).Scan(&ps)
	dbBytes = pc * ps
	rows, e := s.db.QueryContext(ctx,
		`SELECT kind, count(*), COALESCE(SUM(length(media)+length(thumb)+length(text)),0)
		 FROM messages GROUP BY kind ORDER BY 3 DESC`)
	if e != nil {
		return msgCount, dbBytes, kinds, e
	}
	defer rows.Close()
	for rows.Next() {
		var k KindStat
		if rows.Scan(&k.Kind, &k.Count, &k.Bytes) == nil {
			kinds = append(kinds, k)
		}
	}
	return msgCount, dbBytes, kinds, rows.Err()
}

// Vacuum mengembalikan ruang ke OS (pasca-prune) + memangkas WAL.
func (s *Store) Vacuum(ctx context.Context) error {
	_, _ = s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	_, err := s.db.ExecContext(ctx, `VACUUM`)
	return err
}
