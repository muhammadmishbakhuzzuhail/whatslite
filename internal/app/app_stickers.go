// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_stickers.go — koleksi stiker tersimpan (CRUD stiker dari teman).
//
// Stiker yang masuk biasa cuma jadi row `messages` (kind=sticker) + file media
// yang KENA evict LRU → hilang. Fitur ini menyalin byte stiker ke direktori
// PERMANEN ({dataDir}/stickers/<hash>.webp, di luar LRU media) dan mencatat
// metadata di tabel saved_stickers (de-dup by hash isi). Byte disimpan sekali
// walau stiker sama dikirim banyak orang (mengikuti pola Telegram: simpan
// referensi+metadata, byte di cache bersama keyed by hash).

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/storage"
)

// StickerDTO = satu stiker koleksi untuk FE. Byte dimuat via /sticker/<hash>.
type StickerDTO struct {
	Hash     string `json:"hash"`
	Animated bool   `json:"animated"`
	Source   string `json:"source"` // jid pengirim asal ("" bila tak diketahui)
}

// SaveSticker menyimpan stiker sebuah pesan ke koleksi pribadi. Mengembalikan
// hash (id koleksi) bila sukses, "" bila gagal (bukan stiker / media hilang).
func (a *App) SaveSticker(chatJID, msgID string) string {
	if a.eng == nil || a.store == nil {
		return ""
	}
	chatJID = a.canon(chatJID)
	// Ambil proto media tersimpan → unduh byte webp penuh.
	pb, err := a.store.GetMedia(a.ctx, chatJID, msgID)
	if err != nil || pb == "" {
		a.emit("wa:error", "stiker tak bisa disimpan (media tak tersedia)")
		return ""
	}
	data, mime, err := a.eng.DownloadMediaRaw(pb)
	if err != nil || len(data) == 0 {
		a.emit("wa:error", "stiker tak bisa diunduh")
		return ""
	}
	if mime == "" {
		mime = "image/webp"
	}
	sum := sha1.Sum(data)
	hash := hex.EncodeToString(sum[:])

	// Tulis byte ke dir stiker PERMANEN (bukan mediaDir → tak kena LRU evict).
	if a.stickerDir != "" {
		writeFileAtomic(filepath.Join(a.stickerDir, hash+".webp"), data)
	}

	// Sumber = pengirim asal pesan (provenance "disimpan dari teman").
	source := ""
	if m, e := a.store.GetMessage(a.ctx, chatJID, msgID); e == nil {
		source = m.Sender
	}

	if err := a.store.SaveSticker(a.ctx, storage.SavedSticker{
		Hash:     hash,
		Mime:     mime,
		Animated: isAnimatedWebp(data),
		Source:   source,
		Added:    time.Now().Unix(),
	}); err != nil {
		a.emit("wa:error", err.Error())
		return ""
	}
	a.emit("wa:stickers", "") // koleksi berubah → FE refresh
	return hash
}

// ListSavedStickers mengembalikan koleksi (terbaru disimpan dulu) untuk FE.
func (a *App) ListSavedStickers() []StickerDTO {
	out := []StickerDTO{}
	if a.store == nil {
		return out
	}
	items, err := a.store.ListSavedStickers(a.ctx)
	if err != nil {
		return out
	}
	for _, st := range items {
		out = append(out, StickerDTO{Hash: st.Hash, Animated: st.Animated, Source: st.Source})
	}
	return out
}

// DeleteSavedSticker menghapus stiker dari koleksi (baris DB + file). Karena
// koleksi tunggal & hash = PK, tak perlu refcount: hash hilang = file boleh hapus.
func (a *App) DeleteSavedSticker(hash string) bool {
	if a.store == nil || hash == "" {
		return false
	}
	if err := a.store.DeleteSavedSticker(a.ctx, hash); err != nil {
		return false
	}
	if a.stickerDir != "" {
		_ = os.Remove(filepath.Join(a.stickerDir, hash+".webp"))
	}
	a.emit("wa:stickers", "")
	return true
}

// SendSavedSticker mengirim stiker dari koleksi ke sebuah chat (baca file →
// kirim ulang). Mengembalikan id pesan, "" bila gagal.
func (a *App) SendSavedSticker(chatJID, hash string) string {
	if a.eng == nil || a.store == nil || a.stickerDir == "" || hash == "" {
		return ""
	}
	chatJID = a.canon(chatJID)
	data, err := os.ReadFile(filepath.Join(a.stickerDir, hash+".webp"))
	if err != nil || len(data) == 0 {
		a.emit("wa:error", "stiker tersimpan tak ditemukan")
		return ""
	}
	// Pastikan animasi berputar terus (loop-count=0) — sama seperti SendSticker.
	data = webpLoopForever(data)
	id, err := a.eng.SendSticker(a.ctx, chatJID, data)
	if err != nil {
		a.emit("wa:error", err.Error())
		return ""
	}
	a.cacheSentMedia(chatJID, id, data, "image/webp")
	_ = a.store.SaveMessage(a.ctx, storage.Message{
		ID: id, ChatJID: chatJID, Kind: "sticker", Timestamp: time.Now(), FromMe: true,
	})
	a.emit("wa:message", chatJID)
	return id
}

// isAnimatedWebp = true bila byte webp memuat sub-chunk "ANMF" (frame animasi).
// WebP animasi = kontainer VP8X dgn flag animasi + ≥1 chunk ANMF; statis tidak
// punya ANMF. Cukup pindai awal byte (header + chunk pertama).
func isAnimatedWebp(data []byte) bool {
	n := len(data)
	if n > 4096 {
		n = 4096
	}
	return indexBytes(data[:n], "ANMF") >= 0
}

// indexBytes = cari sub-string s dalam b (hindari import strings utk []byte).
func indexBytes(b []byte, s string) int {
	if len(s) == 0 {
		return 0
	}
	for i := 0; i+len(s) <= len(b); i++ {
		j := 0
		for ; j < len(s); j++ {
			if b[i+j] != s[j] {
				break
			}
		}
		if j == len(s) {
			return i
		}
	}
	return -1
}
