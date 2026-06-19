// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_gifs.go — koleksi GIF tersimpan (CRUD GIF dari teman). Pola identik dgn
// app_stickers.go: salin byte GIF ke dir PERMANEN ({dataDir}/gifs/<hash>.<ext>,
// di luar LRU media) + catat metadata di saved_gifs (de-dup by hash). GIF
// WhatsApp = video mp4 (GifPlayback), jadi ext bervariasi → simpan & glob.

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/storage"
)

// SavedGifDTO = satu GIF koleksi untuk FE. Byte dimuat via /savedgif/<hash>.
type SavedGifDTO struct {
	Hash   string `json:"hash"`
	Mime   string `json:"mime"`
	Source string `json:"source"` // jid pengirim asal ("" bila tak diketahui)
}

// SaveGif menyimpan GIF sebuah pesan ke koleksi pribadi. Mengembalikan hash
// bila sukses, "" bila gagal.
func (a *App) SaveGif(chatJID, msgID string) string {
	if a.eng == nil || a.store == nil {
		return ""
	}
	chatJID = a.canon(chatJID)
	pb, err := a.store.GetMedia(a.ctx, chatJID, msgID)
	if err != nil || pb == "" {
		a.emit("wa:error", "GIF tak bisa disimpan (media tak tersedia)")
		return ""
	}
	data, mime, err := a.eng.DownloadMediaRaw(pb)
	if err != nil || len(data) == 0 {
		a.emit("wa:error", "GIF tak bisa diunduh")
		return ""
	}
	if mime == "" {
		mime = "video/mp4"
	}
	sum := sha1.Sum(data)
	hash := hex.EncodeToString(sum[:])

	if a.gifDir != "" {
		writeFileAtomic(filepath.Join(a.gifDir, hash+extForMime(mime)), data)
	}

	source := ""
	if m, e := a.store.GetMessage(a.ctx, chatJID, msgID); e == nil {
		source = m.Sender
	}

	if err := a.store.SaveGif(a.ctx, storage.SavedGif{
		Hash: hash, Mime: mime, Source: source, Added: time.Now().Unix(),
	}); err != nil {
		a.emit("wa:error", err.Error())
		return ""
	}
	a.emit("wa:gifs", "")
	return hash
}

// ListSavedGifs mengembalikan koleksi (terbaru disimpan dulu) untuk FE.
func (a *App) ListSavedGifs() []SavedGifDTO {
	out := []SavedGifDTO{}
	if a.store == nil {
		return out
	}
	items, err := a.store.ListSavedGifs(a.ctx)
	if err != nil {
		return out
	}
	for _, g := range items {
		out = append(out, SavedGifDTO{Hash: g.Hash, Mime: g.Mime, Source: g.Source})
	}
	return out
}

// DeleteSavedGif menghapus GIF dari koleksi (baris DB + file).
func (a *App) DeleteSavedGif(hash string) bool {
	if a.store == nil || hash == "" {
		return false
	}
	if err := a.store.DeleteSavedGif(a.ctx, hash); err != nil {
		return false
	}
	if a.gifDir != "" {
		// ext bervariasi (.mp4/.gif) → buang semua <hash>.* milik koleksi.
		if hits, _ := filepath.Glob(filepath.Join(a.gifDir, hash+".*")); len(hits) > 0 {
			for _, p := range hits {
				_ = os.Remove(p)
			}
		}
	}
	a.emit("wa:gifs", "")
	return true
}

// SendSavedGif mengirim GIF dari koleksi ke sebuah chat (baca file → kirim ulang).
func (a *App) SendSavedGif(chatJID, hash string) string {
	if a.eng == nil || a.store == nil || a.gifDir == "" || hash == "" {
		return ""
	}
	chatJID = a.canon(chatJID)
	g, ok := a.store.GetSavedGif(a.ctx, hash)
	if !ok {
		a.emit("wa:error", "GIF tersimpan tak ditemukan")
		return ""
	}
	data, err := os.ReadFile(filepath.Join(a.gifDir, hash+extForMime(g.Mime)))
	if err != nil || len(data) == 0 {
		a.emit("wa:error", "GIF tersimpan tak ditemukan")
		return ""
	}
	id, err := a.eng.SendGif(a.ctx, chatJID, g.Mime, data)
	if err != nil {
		a.emit("wa:error", err.Error())
		return ""
	}
	a.cacheSentMedia(chatJID, id, data, g.Mime)
	_ = a.store.SaveMessage(a.ctx, storage.Message{
		ID: id, ChatJID: chatJID, Kind: "gif", Timestamp: time.Now(), FromMe: true,
	})
	a.emit("wa:message", chatJID)
	return id
}
