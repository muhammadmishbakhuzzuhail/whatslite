// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_connect.go — sesi WhatsApp: menyambung + alur QR pairing, logout,
// status koneksi, dan kirim pesan teks.

import (
	"encoding/base64"
	"os"
	"time"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/storage"
)

// Connect memulai koneksi WhatsApp. Emit event: wa:qr (data-URI PNG), wa:ready, wa:error.
// Aman untuk pengujian: bila WALITE_NO_CONNECT=1, tidak menyambung.
func (a *App) Connect() {
	if a.eng == nil {
		return
	}
	if os.Getenv("WALITE_NO_CONNECT") == "1" {
		a.emit("wa:state", "offline")
		return
	}
	qr, err := a.eng.Start(a.ctx)
	if err != nil {
		a.emit("wa:error", err.Error())
		return
	}
	if qr == nil {
		a.emit("wa:ready", a.eng.SelfJID())
		return
	}
	go func() {
		for evt := range qr {
			switch evt.Event {
			case "code":
				a.setQR(evt.Code) // simpan kode mentah utk UI in-process (Gio)
				if png, e := qrcode.Encode(evt.Code, qrcode.Medium, 320); e == nil {
					a.emit("wa:qr", "data:image/png;base64,"+base64.StdEncoding.EncodeToString(png))
				}
			case "success":
				a.setQR("") // sudah tertaut → kosongkan QR
				a.emit("wa:ready", a.eng.SelfJID())
			case "timeout":
				a.emit("wa:qr_timeout", "")
			case "error":
				msg := ""
				if evt.Err != nil {
					msg = evt.Err.Error()
				}
				a.emit("wa:error", msg)
			}
		}
	}()
}

// LinkWithPhone meminta kode tautan via nomor telepon (alternatif QR).
// phone = nomor internasional (digit saja, dgn kode negara, tanpa +).
// Kembalikan kode 8-karakter utk diketik di HP, atau "" bila gagal.
func (a *App) LinkWithPhone(phone string) string {
	if a.eng == nil {
		return ""
	}
	digits := make([]rune, 0, len(phone))
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	code, err := a.eng.PairPhone(a.ctx, string(digits))
	if err != nil {
		a.emit("wa:error", err.Error())
		return ""
	}
	return code
}

// Logout mengeluarkan akun saat ini (unpair). UI akan kembali ke layar QR.
func (a *App) Logout() {
	if a.eng == nil {
		return
	}
	_ = a.eng.Logout(a.ctx)
	a.emit("wa:loggedout", "")
}

// setQR menyimpan kode QR pairing mentah terbaru (thread-safe).
func (a *App) setQR(code string) {
	a.qrMu.Lock()
	a.qrCode = code
	a.qrMu.Unlock()
}

// QRCode mengembalikan kode QR pairing mentah terbaru, atau "" bila belum ada /
// sudah tertaut. UI in-process (Gio) memanggil ini tiap refresh utk meng-encode
// QR asli sendiri (tanpa lewat jalur event Wails/IPC).
func (a *App) QRCode() string {
	a.qrMu.RLock()
	defer a.qrMu.RUnlock()
	return a.qrCode
}

// typingStateT — status "mengetik" terakhir per chat (utk subtitle header in-process).
type typingStateT struct {
	on     bool
	who    string    // nama (grup) atau "" (DM)
	whoJID string    // jid pengirim (grup) → avatar di bubble mengetik
	rec    bool      // merekam audio
	at     time.Time // kapan status "mengetik" diterima (utk kedaluwarsa)
}

// typingTTL — "mengetik…" dianggap basi setelah ini bila tak ada event "paused"
// (peer tutup app / putus jaringan). WhatsApp meng-auto-expire komposisi.
const typingTTL = 12 * time.Second

func (a *App) setPresence(jid, txt string) {
	a.presMu.Lock()
	if a.presence == nil {
		a.presence = map[string]string{}
	}
	a.presence[jid] = txt
	a.presMu.Unlock()
}

func (a *App) setTyping(jid string, st typingStateT) {
	a.presMu.Lock()
	if a.typing == nil {
		a.typing = map[string]typingStateT{}
	}
	if st.on {
		st.at = time.Now()
	}
	a.typing[jid] = st
	a.presMu.Unlock()
}

// ChatSubtitle mengembalikan subtitle header chat: "mengetik…"/"merekam audio…"
// bila sedang mengetik, jika tidak "online"/"terakhir dilihat .."/"" (presence).
// Dipoll UI in-process (Gio) tiap refresh — tanpa lewat jalur event Wails/IPC.
func (a *App) ChatSubtitle(jid string) string {
	a.presMu.RLock()
	defer a.presMu.RUnlock()
	if st, ok := a.typing[jid]; ok && st.on && time.Since(st.at) < typingTTL {
		if st.rec {
			if st.who != "" {
				return st.who + " sedang merekam audio…"
			}
			return "merekam audio…"
		}
		if st.who != "" {
			return st.who + " sedang mengetik…"
		}
		return "mengetik…"
	}
	return a.presence[jid]
}

// TypingLabel mengembalikan label mengetik/merekam bila chat sedang mengetik,
// atau "" (TANPA presence) — utk override preview di daftar chat (section 2).
func (a *App) TypingLabel(jid string) string {
	a.presMu.RLock()
	defer a.presMu.RUnlock()
	st, ok := a.typing[jid]
	if !ok || !st.on || time.Since(st.at) >= typingTTL {
		return ""
	}
	if st.rec {
		if st.who != "" {
			return st.who + " merekam…"
		}
		return "merekam audio…"
	}
	if st.who != "" {
		return st.who + " mengetik…"
	}
	return "mengetik…"
}

// TypingWho mengembalikan (nama, jid) pengirim yg sedang mengetik di grup, atau
// ("","") bila tak ada / DM. Dipakai UI utk menampilkan avatar di bubble mengetik.
func (a *App) TypingWho(jid string) (string, string) {
	a.presMu.RLock()
	defer a.presMu.RUnlock()
	st, ok := a.typing[jid]
	if !ok || !st.on || time.Since(st.at) >= typingTTL {
		return "", ""
	}
	return st.who, st.whoJID
}

// GetState: "offline" | "qr" | "ready".
func (a *App) GetState() string {
	if a.eng == nil {
		return "offline"
	}
	if a.eng.NeedsLogin() {
		return "qr"
	}
	return "ready"
}

// SendText mengirim & menyimpan pesan keluar; kembalikan ID (atau "" bila gagal).
// MetaAIJID mengembalikan JID bot Meta AI (gratis, server-side). "" bila engine
// belum siap. UI memakai ini untuk membuka chat Meta AI dari rail.
func (a *App) MetaAIJID() string {
	if a.eng == nil {
		return ""
	}
	return a.eng.MetaAIJID()
}

func (a *App) SendText(jid, text string) string {
	if a.eng == nil {
		return ""
	}
	jid = a.canon(jid)
	id, err := a.eng.SendText(a.ctx, jid, text)
	if err != nil {
		a.emit("wa:error", err.Error())
		return ""
	}
	_ = a.store.SaveMessage(a.ctx, storage.Message{
		ID: id, ChatJID: jid, Text: text, Timestamp: time.Now(), FromMe: true,
	})
	_ = a.store.SetUnread(a.ctx, jid, 0) // kirim = aktif di chat → tandai terbaca
	return id
}
