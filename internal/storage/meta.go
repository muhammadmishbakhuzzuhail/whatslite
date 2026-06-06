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

// Vacuum mengembalikan ruang ke OS (pasca-prune) + memangkas WAL.
func (s *Store) Vacuum(ctx context.Context) error {
	_, _ = s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	_, err := s.db.ExecContext(ctx, `VACUUM`)
	return err
}
