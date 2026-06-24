// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_calls.go — API panggilan (signaling-only: log + tolak). whatsmeow tak
// punya media call, jadi tak ada menjawab/menelepon.

import (
	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/storage"
)

// GetCalls mengembalikan log panggilan terbaru. Nama di-RESOLVE ULANG saat ambil
// (bukan pakai yg dibekukan saat event) — sebab saat panggilan masuk tepat setelah
// reconnect, kontak/lid_map sering belum sinkron → tersimpan sbg nomor. Setelah
// kontak sinkron, panggilan lama tetap nomor bila tak di-resolve lagi.
func (a *App) GetCalls() []storage.Call {
	if a.store == nil {
		return []storage.Call{}
	}
	out, err := a.store.ListCalls(a.ctx)
	if err != nil || out == nil {
		return []storage.Call{}
	}
	for i := range out {
		if out[i].JID == "" {
			continue
		}
		if n := a.displayName(out[i].JID); n != "" && !isPhoneLike(n) {
			out[i].Name = n // nama nyata tersedia sekarang → ganti nomor beku
		}
	}
	return out
}

// isPhoneLike — heuristik: string hanya digit/+/spasi/strip (mis. "+62 812-…") →
// bukan nama nyata. Dipakai agar tak menimpa nomor dgn nomor.
func isPhoneLike(s string) bool {
	hasDigit := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case r == '+' || r == ' ' || r == '-' || r == '(' || r == ')':
		default:
			return false // ada huruf → nama
		}
	}
	return hasDigit
}

// DeleteCall menghapus satu entri dari log panggilan.
func (a *App) DeleteCall(id string) {
	if a.store == nil || id == "" {
		return
	}
	_ = a.store.DeleteCall(a.ctx, id)
	a.emit("wa:callupdate", "")
}

// ClearCallLog mengosongkan seluruh log panggilan.
func (a *App) ClearCallLog() {
	if a.store == nil {
		return
	}
	_ = a.store.ClearCalls(a.ctx)
	a.emit("wa:callupdate", "")
}

// RejectCall menolak panggilan masuk (callID) dari jid, lalu tandai log "rejected".
func (a *App) RejectCall(jid, callID string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.RejectCall(a.ctx, a.canon(jid), callID); err != nil {
		a.emit("wa:error", err.Error())
		return
	}
	if a.store != nil {
		_ = a.store.SetCallStatus(a.ctx, callID, "rejected")
	}
	a.emit("wa:callupdate", "")
}
