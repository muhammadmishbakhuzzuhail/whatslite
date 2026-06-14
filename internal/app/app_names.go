// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_names.go — resolusi nama tampilan terpusat (label lokal > buku-alamat >
// pushname) + profil kontak + simpan/hapus label lokal.
//
// Prioritas nama (lihat nameOf):
//  1. Label lokal (app.db, disimpan pengguna)  → saved=true
//  2. Buku alamat WA (FullName/FirstName)       → saved=true
//  3. Pushname (kontak WA / pesan terakhir)     → saved=false
//  4. "" → pemanggil fallback ke nomor (phoneOf) / shortJID
//
// "Tersimpan" (saved) menentukan format UI: grup + tak-tersimpan → "Nama + nomor".

import (
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// loadLabels mengisi cache label dari DB (dipanggil saat startup).
func (a *App) loadLabels() {
	a.labelsMu.Lock()
	defer a.labelsMu.Unlock()
	a.labels = map[string]string{}
	if a.store == nil {
		return
	}
	if m, err := a.store.AllContactLabels(a.ctx); err == nil {
		a.labels = m
	}
}

// labelOf mengembalikan label lokal sebuah jid ("" bila tak ada).
func (a *App) labelOf(jid string) string {
	a.labelsMu.RLock()
	defer a.labelsMu.RUnlock()
	return a.labels[jid]
}

// nameOf me-resolve nama tampil + apakah "tersimpan" (label lokal / buku alamat).
func (a *App) nameOf(jid string) (name string, saved bool) {
	if jid == "" {
		return "", false
	}
	if lbl := a.labelOf(jid); lbl != "" {
		return lbl, true
	}
	if a.eng != nil {
		if n, sv := a.eng.ResolveName(jid); n != "" {
			return n, sv
		}
	}
	// DM yg kontaknya kosong di store WA → pakai pushname terakhir dari DB.
	if a.store != nil {
		if pn := a.store.LastPushName(a.ctx, jid); pn != "" {
			return pn, false
		}
	}
	return "", false
}

// phoneOf mengembalikan nomor terbaca ("+62…") atau "" bila tak terpetakan.
func (a *App) phoneOf(jid string) string {
	if a.eng == nil {
		return ""
	}
	return a.eng.ReadableID(jid)
}

// displayName: nama tampil dgn fallback berjenjang (nama > nomor > shortJID).
func (a *App) displayName(jid string) string {
	if n, _ := a.nameOf(jid); n != "" {
		return n
	}
	if p := a.phoneOf(jid); p != "" {
		return p
	}
	return shortJID(jid)
}

// ContactProfileDTO = profil kontak utk panel (klik mention / pengirim grup).
type ContactProfileDTO struct {
	JID   string `json:"jid"`
	Name  string `json:"name"`  // nama tampil (label/buku-alamat/pushname/nomor)
	Phone string `json:"phone"` // "+62…" bila terpetakan
	About string `json:"about"` // teks info/status (butuh koneksi)
	Saved bool   `json:"saved"` // ada label lokal / di buku alamat
}

// GetContactProfile mengumpulkan profil satu kontak utk panel profil.
func (a *App) GetContactProfile(jid string) ContactProfileDTO {
	name, saved := a.nameOf(jid)
	phone := a.phoneOf(jid)
	if name == "" {
		name = phone
		if name == "" {
			name = shortJID(jid)
		}
	}
	about := ""
	if a.eng != nil {
		about = a.eng.ContactAbout(a.ctx, jid)
	}
	return ContactProfileDTO{JID: jid, Name: name, Phone: phone, About: about, Saved: saved}
}

// SaveContactLabel menyimpan nama lokal utk jid (label app, BUKAN sync ke HP/WA).
func (a *App) SaveContactLabel(jid, name string) {
	name = strings.TrimSpace(name)
	if jid == "" || name == "" || a.store == nil {
		return
	}
	if err := a.store.SetContactLabel(a.ctx, jid, name); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	a.labelsMu.Lock()
	if a.labels == nil {
		a.labels = map[string]string{}
	}
	a.labels[jid] = name
	a.labelsMu.Unlock()
	runtime.EventsEmit(a.ctx, "wa:sync", "") // UI refresh nama
}

// RemoveContactLabel menghapus label lokal sebuah jid.
func (a *App) RemoveContactLabel(jid string) {
	if jid == "" || a.store == nil {
		return
	}
	if err := a.store.DeleteContactLabel(a.ctx, jid); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	a.labelsMu.Lock()
	delete(a.labels, jid)
	a.labelsMu.Unlock()
	runtime.EventsEmit(a.ctx, "wa:sync", "")
}
