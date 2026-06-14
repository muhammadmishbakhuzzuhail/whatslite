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
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/storage"
)

// Connect memulai koneksi WhatsApp. Emit event: wa:qr (data-URI PNG), wa:ready, wa:error.
// Aman untuk pengujian: bila WALITE_NO_CONNECT=1, tidak menyambung.
func (a *App) Connect() {
	if a.eng == nil {
		return
	}
	if os.Getenv("WALITE_NO_CONNECT") == "1" {
		runtime.EventsEmit(a.ctx, "wa:state", "offline")
		return
	}
	qr, err := a.eng.Start(a.ctx)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	if qr == nil {
		runtime.EventsEmit(a.ctx, "wa:ready", a.eng.SelfJID())
		return
	}
	go func() {
		for evt := range qr {
			switch evt.Event {
			case "code":
				if png, e := qrcode.Encode(evt.Code, qrcode.Medium, 320); e == nil {
					runtime.EventsEmit(a.ctx, "wa:qr", "data:image/png;base64,"+base64.StdEncoding.EncodeToString(png))
				}
			case "success":
				runtime.EventsEmit(a.ctx, "wa:ready", a.eng.SelfJID())
			case "timeout":
				runtime.EventsEmit(a.ctx, "wa:qr_timeout", "")
			case "error":
				msg := ""
				if evt.Err != nil {
					msg = evt.Err.Error()
				}
				runtime.EventsEmit(a.ctx, "wa:error", msg)
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
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
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
	runtime.EventsEmit(a.ctx, "wa:loggedout", "")
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
func (a *App) SendText(jid, text string) string {
	if a.eng == nil {
		return ""
	}
	jid = a.canon(jid)
	id, err := a.eng.SendText(a.ctx, jid, text)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	_ = a.store.SaveMessage(a.ctx, storage.Message{
		ID: id, ChatJID: jid, Text: text, Timestamp: time.Now(), FromMe: true,
	})
	return id
}
