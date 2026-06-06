package app

// app_media.go — unduh media penuh on-demand + asset-server /media & /avatar
// (foto profil di-serve via file-cache, lihat serveAvatar/ProfilePictureRaw).

import (
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DownloadMedia mengunduh media penuh satu pesan → data-URI (atau "").
func (a *App) DownloadMedia(chatJID, msgID string) string {
	if a.eng == nil || a.store == nil {
		return ""
	}
	pb, err := a.store.GetMedia(a.ctx, chatJID, msgID)
	if err != nil || pb == "" {
		return ""
	}
	uri, err := a.eng.DownloadMedia(pb)
	if err != nil {
		return ""
	}
	return uri
}

// TranscribeVoice mentranskrip voice note → teks (STT lokal, best-effort).
// "" bila whisper/model tak terpasang (lihat transcribeAudio).
func (a *App) TranscribeVoice(chatJID, msgID string) string {
	if a.eng == nil || a.store == nil {
		return ""
	}
	pb, err := a.store.GetMedia(a.ctx, a.canon(chatJID), msgID)
	if err != nil || pb == "" {
		return ""
	}
	data, _, err := a.eng.DownloadMediaRaw(pb)
	if err != nil || len(data) == 0 {
		return ""
	}
	txt, ok := transcribeAudio(a.ctx, data)
	if !ok {
		return ""
	}
	return txt
}

// serveMedia menyajikan media via asset-server: GET /media/<chatJID>/<msgID>.
// Cache-first: kalau file sudah ada → kirim; kalau belum → unduh dari proto
// (tersimpan di DB), tulis ke FILE (bukan DB/memori), lalu kirim. Ringan +
// browser ikut meng-cache. Pesan lama tetap bisa dimuat ulang (file persist).
func (a *App) serveMedia(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/media/") {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/media/")
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		http.NotFound(w, r)
		return
	}
	chat, _ := url.PathUnescape(rest[:slash])
	id, _ := url.PathUnescape(rest[slash+1:])
	if chat == "" || id == "" || a.store == nil || a.eng == nil {
		http.NotFound(w, r)
		return
	}

	sum := sha1.Sum([]byte(chat + "|" + id))
	base := filepath.Join(a.mediaDir, hex.EncodeToString(sum[:]))

	// cache hit: cari <sha>.<ext> apa pun
	if hits, _ := filepath.Glob(base + ".*"); len(hits) > 0 {
		w.Header().Set("Cache-Control", "max-age=31536000")
		http.ServeFile(w, r, hits[0])
		return
	}

	// cache miss → unduh dari proto tersimpan → tulis file
	protoB64, err := a.store.GetMedia(a.ctx, chat, id)
	if err != nil || protoB64 == "" {
		// Fallback: pesan terkirim menyimpan data-URI penuh di kolom thumb
		// (tanpa proto). Sajikan itu agar gambar/video keluar tetap tampil.
		if m, e := a.store.GetMessage(a.ctx, chat, id); e == nil && strings.HasPrefix(m.Thumb, "data:") {
			if mime, raw, e2 := decodeDataURI(m.Thumb); e2 == nil && len(raw) > 0 {
				path := base + extForMime(mime)
				writeFileAtomic(path, raw)
				w.Header().Set("Cache-Control", "max-age=31536000")
				http.ServeFile(w, r, path)
				return
			}
		}
		http.NotFound(w, r)
		return
	}
	data, mime, err := a.eng.DownloadMediaRaw(protoB64)
	if err != nil || len(data) == 0 {
		http.Error(w, "media unavailable", http.StatusBadGateway)
		return
	}
	path := base + extForMime(mime)
	writeFileAtomic(path, data)
	w.Header().Set("Cache-Control", "max-age=31536000")
	http.ServeFile(w, r, path)
}

// writeFileAtomic menulis ke file sementara unik lalu rename → request paralel
// utk media/avatar yang sama tak pernah menyajikan file separuh-tulis.
func writeFileAtomic(path string, data []byte) {
	f, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		_ = os.WriteFile(path, data, 0o644)
		return
	}
	tmp := f.Name()
	if _, err := f.Write(data); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return
	}
	f.Close()
	_ = os.Chmod(tmp, 0o644)
	if os.Rename(tmp, path) != nil {
		_ = os.Remove(tmp)
	}
}

// cacheSentMedia menulis byte media KELUAR ke file-cache persisten
// (<sha>.sent.<ext>) → /media cache-hit menyajikannya, jadi tak perlu simpan
// data-URI raksasa di DB. Diberi sufiks ".sent." agar TAK ikut di-evict (tak
// ada proto utk unduh ulang). Aman dipanggil di goroutine (atomic write).
func (a *App) cacheSentMedia(chat, id string, data []byte, mime string) {
	if a.mediaDir == "" || len(data) == 0 {
		return
	}
	sum := sha1.Sum([]byte(chat + "|" + id))
	base := filepath.Join(a.mediaDir, hex.EncodeToString(sum[:]))
	writeFileAtomic(base+".sent"+extForMime(mime), data)
}

func extForMime(mime string) string {
	switch {
	case strings.HasPrefix(mime, "image/webp"):
		return ".webp"
	case strings.HasPrefix(mime, "image/png"):
		return ".png"
	case strings.HasPrefix(mime, "image/gif"):
		return ".gif"
	case strings.HasPrefix(mime, "image/"):
		return ".jpg"
	case strings.HasPrefix(mime, "video/"):
		return ".mp4"
	case strings.HasPrefix(mime, "audio/ogg"):
		return ".ogg"
	case strings.HasPrefix(mime, "audio/mpeg"):
		return ".mp3"
	case strings.HasPrefix(mime, "audio/"):
		return ".ogg"
	}
	return ".bin"
}

// serveHTTP = router asset non-embed: /media/* dan /avatar/*.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/media/"):
		a.serveMedia(w, r)
	case strings.HasPrefix(r.URL.Path, "/avatar/"):
		a.serveAvatar(w, r)
	default:
		http.NotFound(w, r)
	}
}

// serveAvatar: GET /avatar/<jid> → foto profil cache-FILE (lazy, hanya avatar
// yang benar-benar dirender). Negatif-cache (.none) agar tak fetch berulang.
func (a *App) serveAvatar(w http.ResponseWriter, r *http.Request) {
	jid, _ := url.PathUnescape(strings.TrimPrefix(r.URL.Path, "/avatar/"))
	if jid == "" || a.eng == nil {
		http.NotFound(w, r)
		return
	}
	sum := sha1.Sum([]byte(jid))
	base := filepath.Join(a.mediaDir, "av_"+hex.EncodeToString(sum[:]))
	const ttl = 24 * time.Hour
	if st, err := os.Stat(base + ".jpg"); err == nil && time.Since(st.ModTime()) < ttl {
		w.Header().Set("Cache-Control", "max-age=3600")
		http.ServeFile(w, r, base+".jpg")
		return
	}
	if st, err := os.Stat(base + ".none"); err == nil && time.Since(st.ModTime()) < ttl {
		http.NotFound(w, r) // baru-baru ini diketahui tak ada foto
		return
	}
	// stale / belum ada → fetch ulang (refresh bila foto berubah).
	b, err := a.eng.ProfilePictureRaw(jid)
	if err != nil {
		http.Error(w, "", http.StatusBadGateway)
		return
	}
	if len(b) == 0 {
		_ = os.WriteFile(base+".none", []byte{}, 0o644)
		http.NotFound(w, r)
		return
	}
	writeFileAtomic(base+".jpg", b)
	w.Header().Set("Cache-Control", "max-age=86400")
	http.ServeFile(w, r, base+".jpg")
}

// startMediaEviction menjaga cache media tak tumbuh tanpa batas: bila total file
// media penuh (bukan avatar/.none) > cap, hapus yang paling lama diakses.
func (a *App) startMediaEviction(capBytes int64) {
	sweep := func() {
		entries, err := os.ReadDir(a.mediaDir)
		if err != nil {
			return
		}
		type f struct {
			path string
			size int64
			mod  int64
		}
		var files []f
		var total int64
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, "av_") || strings.HasSuffix(name, ".none") || strings.Contains(name, ".sent.") {
				continue // avatar kecil, marker, media KELUAR (tanpa proto unduh-ulang) → biarkan
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			files = append(files, f{filepath.Join(a.mediaDir, name), info.Size(), info.ModTime().Unix()})
			total += info.Size()
		}
		if total <= capBytes {
			return
		}
		sort.Slice(files, func(i, j int) bool { return files[i].mod < files[j].mod }) // terlama dulu
		for _, x := range files {
			if total <= capBytes {
				break
			}
			if os.Remove(x.path) == nil {
				total -= x.size
			}
		}
	}
	go func() {
		sweep()
		t := time.NewTicker(30 * time.Minute)
		defer t.Stop()
		for range t.C {
			sweep()
		}
	}()
}
