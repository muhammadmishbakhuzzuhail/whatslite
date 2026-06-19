// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package storage

// gifs.go — koleksi GIF tersimpan (saved_gifs). CRUD untuk menyimpan GIF yang
// dikirim teman ke koleksi pribadi. Pola identik dgn stickers.go: de-dup by
// hash isi, byte disimpan sebagai FILE terpisah (lihat app_gifs.go); DB hanya
// metadata. GIF WhatsApp = video mp4 (GifPlayback) → mime disimpan utk kirim ulang.

import "context"

// SavedGif = satu GIF dalam koleksi pribadi.
type SavedGif struct {
	Hash   string // sha1 byte (de-dup + nama file)
	Mime   string // video/mp4 (umumnya) | image/gif
	Source string // jid pengirim asal (opsional)
	Added  int64  // unix saat disimpan → urut terbaru dulu
}

// SaveGif menambah GIF ke koleksi. De-dup: hash sudah ada → no-op (idempoten).
func (s *Store) SaveGif(ctx context.Context, g SavedGif) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO saved_gifs (hash, mime, source, added) VALUES (?, ?, ?, ?)`,
		g.Hash, g.Mime, g.Source, g.Added)
	return err
}

// ListSavedGifs mengembalikan seluruh koleksi, terbaru disimpan dulu.
func (s *Store) ListSavedGifs(ctx context.Context) ([]SavedGif, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT hash, mime, source, added FROM saved_gifs ORDER BY added DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SavedGif{}
	for rows.Next() {
		var g SavedGif
		if err := rows.Scan(&g.Hash, &g.Mime, &g.Source, &g.Added); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// GetSavedGif mengembalikan satu baris (utk tahu mime saat kirim ulang).
func (s *Store) GetSavedGif(ctx context.Context, hash string) (SavedGif, bool) {
	var g SavedGif
	err := s.db.QueryRowContext(ctx,
		`SELECT hash, mime, source, added FROM saved_gifs WHERE hash=?`, hash).
		Scan(&g.Hash, &g.Mime, &g.Source, &g.Added)
	return g, err == nil
}

// DeleteSavedGif menghapus satu GIF dari koleksi (file dibuang oleh pemanggil).
func (s *Store) DeleteSavedGif(ctx context.Context, hash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM saved_gifs WHERE hash=?`, hash)
	return err
}

// IsGifSaved = true bila hash sudah ada di koleksi.
func (s *Store) IsGifSaved(ctx context.Context, hash string) bool {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM saved_gifs WHERE hash=?`, hash).Scan(&one)
	return err == nil
}
