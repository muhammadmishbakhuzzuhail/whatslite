// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_settings.go — setelan persisten ringan (app_meta). Saat ini: retensi pesan.

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/storage"
)

func atoiDef(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

// SetDefaultDisappearing menyetel timer hilang-otomatis default (detik; 0 = off).
func (a *App) SetDefaultDisappearing(seconds int) {
	if a.eng != nil && !a.emitErr(a.eng.SetDefaultDisappearing(a.ctx, seconds)) {
		a.emit("wa:sync", "")
	}
}

// SetChatDisappearing menyetel timer pesan sementara untuk SATU chat (detik; 0 = mati).
func (a *App) SetChatDisappearing(jid string, seconds int) {
	if a.eng != nil && !a.emitErr(a.eng.SetDisappearing(a.ctx, jid, seconds)) {
		a.emit("wa:sync", "")
	}
}

// MyQR mengembalikan QR kontak sendiri sebagai PNG data-URI (revoke=buat ulang).
func (a *App) MyQR(revoke bool) string {
	if a.eng == nil {
		return ""
	}
	link, err := a.eng.MyQRLink(a.ctx, revoke)
	if err != nil || link == "" {
		if err != nil {
			a.emit("wa:error", err.Error())
		}
		return ""
	}
	png, e := qrcode.Encode(link, qrcode.Medium, 320)
	if e != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}

// GetProxy mengembalikan alamat proxy tersimpan ("" = tanpa proxy).
func (a *App) GetProxy() string {
	if a.store == nil {
		return ""
	}
	return a.store.GetMeta(a.ctx, "proxy", "")
}

// SetProxy menyimpan proxy (berlaku setelah restart). "" = matikan.
func (a *App) SetProxy(addr string) {
	if a.store != nil {
		_ = a.store.SetMeta(a.ctx, "proxy", addr)
	}
	a.emit("wa:sync", "")
}

// StorageUsageDTO = rincian penyimpanan untuk layar setelan.
type StorageUsageDTO struct {
	DBBytes    int64              `json:"dbBytes"`
	MediaBytes int64              `json:"mediaBytes"`
	MsgCount   int                `json:"msgCount"`
	Kinds      []storage.KindStat `json:"kinds"`
}

// GetStorageUsage menghitung ukuran DB + cache media + rincian per-jenis.
func (a *App) GetStorageUsage() StorageUsageDTO {
	out := StorageUsageDTO{Kinds: []storage.KindStat{}}
	if a.store != nil {
		out.MsgCount, out.DBBytes, out.Kinds, _ = a.store.StorageStats(a.ctx)
	}
	if a.mediaDir != "" {
		_ = filepath.Walk(a.mediaDir, func(_ string, info os.FileInfo, err error) error {
			if err == nil && info != nil && !info.IsDir() {
				out.MediaBytes += info.Size()
			}
			return nil
		})
	}
	return out
}

// ThemeDark mengembalikan preferensi tema gelap tersimpan (default true bila
// belum pernah disetel) — dimuat saat start agar tema tak reset tiap buka.
func (a *App) ThemeDark() bool {
	if a.store == nil {
		return true
	}
	return a.store.GetMeta(a.ctx, "theme_dark", "1") != "0"
}

// SetThemeDark menyimpan preferensi tema gelap (persist lintas-restart).
func (a *App) SetThemeDark(dark bool) {
	if a.store == nil {
		return
	}
	v := "1"
	if !dark {
		v = "0"
	}
	_ = a.store.SetMeta(a.ctx, "theme_dark", v)
}

// NotificationsOn mengembalikan preferensi notifikasi (default aktif).
func (a *App) NotificationsOn() bool {
	if a.store == nil {
		return true
	}
	return a.store.GetMeta(a.ctx, "notifications", "1") != "0"
}

// SetNotificationsOn menyimpan preferensi notifikasi (persist).
func (a *App) SetNotificationsOn(on bool) {
	if a.store == nil {
		return
	}
	v := "1"
	if !on {
		v = "0"
	}
	_ = a.store.SetMeta(a.ctx, "notifications", v)
}

// Language mengembalikan kode bahasa UI tersimpan (default "id").
func (a *App) Language() string {
	if a.store == nil {
		return "id"
	}
	if v := a.store.GetMeta(a.ctx, "lang", "id"); v != "" {
		return v
	}
	return "id"
}

// SetLanguage menyimpan kode bahasa UI (persist).
func (a *App) SetLanguage(code string) {
	if a.store == nil || code == "" {
		return
	}
	_ = a.store.SetMeta(a.ctx, "lang", code)
}

// Username mengembalikan nama pengguna (handle) tersimpan, atau "".
func (a *App) Username() string {
	if a.store == nil {
		return ""
	}
	return a.store.GetMeta(a.ctx, "username", "")
}

// SetUsername memvalidasi (aturan WhatsApp) lalu menyimpan nama pengguna.
// Kembalikan "" bila sukses, atau pesan error bila tak valid.
func (a *App) SetUsername(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if msg := validateUsername(s); msg != "" {
		return msg
	}
	if a.store != nil {
		_ = a.store.SetMeta(a.ctx, "username", s)
	}
	return ""
}

// validateUsername — aturan WhatsApp: 3–35 char; huruf kecil a–z, angka, "." dan
// "_"; min. 1 huruf; tak diawali/diakhiri "."; tak ada ".."; tak diawali "www.".
func validateUsername(s string) string {
	if s == "" {
		return "" // kosong = hapus username (valid)
	}
	n := len(s)
	if n < 3 || n > 35 {
		return "Nama pengguna harus 3–35 karakter"
	}
	if strings.HasPrefix(s, ".") || strings.HasSuffix(s, ".") {
		return "Tak boleh diawali/diakhiri titik"
	}
	if strings.Contains(s, "..") {
		return "Tak boleh ada titik berurutan"
	}
	if strings.HasPrefix(s, "www.") {
		return "Tak boleh diawali \"www.\""
	}
	hasLetter := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			hasLetter = true
		case r >= '0' && r <= '9', r == '.', r == '_':
		default:
			return "Hanya huruf kecil, angka, titik, dan garis bawah"
		}
	}
	if !hasLetter {
		return "Harus memuat minimal satu huruf"
	}
	return ""
}

// GetRetention mengembalikan jumlah hari retensi pesan (0 = selamanya).
func (a *App) GetRetention() int { return a.retentionDays }

// SetRetention menyetel retensi (hari; 0 = selamanya), simpan, lalu prune+VACUUM.
func (a *App) SetRetention(days int) {
	if days < 0 {
		days = 0
	}
	a.retentionDays = days
	if a.store == nil {
		return
	}
	_ = a.store.SetMeta(a.ctx, "retention_days", strconv.Itoa(days))
	a.bg(func() {
		if cut := a.retentionCutoff(); cut > 0 {
			if n, _ := a.store.PruneMessages(a.ctx, cut); n > 0 {
				_ = a.store.Vacuum(a.ctx)
			}
		}
		a.emit("wa:sync", "")
	})
}

// --- Draft composer persisten (app_meta key "draft:<jid>") ---

// SetDraft menyimpan / menghapus draft teks composer per-chat (kosong = hapus).
func (a *App) SetDraft(jid, text string) {
	if a.store == nil {
		return
	}
	key := "draft:" + jid
	if strings.TrimSpace(text) == "" {
		_ = a.store.DelMeta(a.ctx, key)
	} else {
		_ = a.store.SetMeta(a.ctx, key, text)
	}
}

// GetDrafts mengembalikan semua draft tersimpan (jid → teks) untuk dimuat saat start.
func (a *App) GetDrafts() map[string]string {
	out := map[string]string{}
	if a.store == nil {
		return out
	}
	m, err := a.store.ListMeta(a.ctx, "draft:")
	if err != nil {
		return out
	}
	for k, v := range m {
		out[strings.TrimPrefix(k, "draft:")] = v
	}
	return out
}
