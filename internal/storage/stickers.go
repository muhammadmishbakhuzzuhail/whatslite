// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package storage

// stickers.go — koleksi stiker tersimpan (saved_stickers). CRUD untuk menyimpan
// stiker yang dikirim teman ke koleksi pribadi. De-dup by hash isi (sha1 byte
// webp): stiker yang sama dari banyak orang hanya satu baris. Byte stiker
// disimpan sebagai FILE terpisah (lihat app_stickers.go); DB hanya metadata.

import "context"

// SavedSticker = satu stiker dalam koleksi pribadi.
type SavedSticker struct {
	Hash     string // sha1 byte webp (de-dup + nama file)
	Mime     string // image/webp
	Animated bool   // stiker animasi (webp ANMF)
	Source   string // jid pengirim asal (opsional, "" bila tak diketahui)
	Added    int64  // unix saat disimpan → urut terbaru dulu
}

// SaveSticker menambah stiker ke koleksi. De-dup: bila hash sudah ada, no-op
// (INSERT OR IGNORE) → menyimpan stiker yang sama dua kali aman & idempoten.
func (s *Store) SaveSticker(ctx context.Context, st SavedSticker) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO saved_stickers (hash, mime, animated, source, added)
		 VALUES (?, ?, ?, ?, ?)`,
		st.Hash, st.Mime, b2i(st.Animated), st.Source, st.Added)
	return err
}

// ListSavedStickers mengembalikan seluruh koleksi, terbaru disimpan dulu.
func (s *Store) ListSavedStickers(ctx context.Context) ([]SavedSticker, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT hash, mime, animated, source, added FROM saved_stickers ORDER BY added DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SavedSticker{}
	for rows.Next() {
		var st SavedSticker
		var anim int
		if err := rows.Scan(&st.Hash, &st.Mime, &anim, &st.Source, &st.Added); err != nil {
			return nil, err
		}
		st.Animated = anim != 0
		out = append(out, st)
	}
	return out, rows.Err()
}

// DeleteSavedSticker menghapus satu stiker dari koleksi (file dibuang terpisah
// oleh pemanggil di app_stickers.go).
func (s *Store) DeleteSavedSticker(ctx context.Context, hash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM saved_stickers WHERE hash=?`, hash)
	return err
}

// IsStickerSaved = true bila hash sudah ada di koleksi (UI: tombol simpan/terhapus).
func (s *Store) IsStickerSaved(ctx context.Context, hash string) bool {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM saved_stickers WHERE hash=?`, hash).Scan(&one)
	return err == nil
}
