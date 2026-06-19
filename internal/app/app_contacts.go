// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_contacts.go — info kontak, blokir, dan profil akun sendiri.

import (
	"sort"
	"strings"
)

// ProfileDTO = profil akun sendiri untuk UI.
type ProfileDTO struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	About string `json:"about"`
	Jid   string `json:"jid"`
}

// WACheckDTO = hasil cek "ada di WhatsApp?".
type WACheckDTO struct {
	Query      string `json:"query"`
	JID        string `json:"jid"`
	Registered bool   `json:"registered"`
}

// IsOnWhatsApp memeriksa nomor (mis. sebelum mulai chat / simpan kontak).
func (a *App) IsOnWhatsApp(phones []string) []WACheckDTO {
	out := []WACheckDTO{}
	if a.eng == nil || len(phones) == 0 {
		return out
	}
	res, err := a.eng.IsOnWhatsApp(a.ctx, phones)
	if err != nil {
		a.emit("wa:error", err.Error())
		return out
	}
	for _, r := range res {
		out = append(out, WACheckDTO{Query: r.Query, JID: r.JID, Registered: r.Registered})
	}
	return out
}

// GetProfile mengembalikan profil akun yang sedang login.
func (a *App) GetProfile() ProfileDTO {
	if a.eng == nil {
		return ProfileDTO{Name: "Saya"}
	}
	self := a.eng.SelfJID()
	name := a.eng.ChatName(self)
	if name == "" {
		name = "Saya"
	}
	phone := a.eng.ReadableID(self)
	if phone == "" {
		phone = shortJID(self)
	}
	return ProfileDTO{Name: name, Phone: phone, About: a.eng.ContactAbout(a.ctx, self), Jid: a.eng.CanonicalJID(self)}
}

// SetMyPhoto mengganti foto profil sendiri (full + preview data-URI JPEG dari FE).
func (a *App) SetMyPhoto(fullURI, previewURI string) {
	if a.eng == nil {
		return
	}
	_, full, err := decodeDataURI(fullURI)
	if err != nil {
		a.emit("wa:error", err.Error())
		return
	}
	var preview []byte
	if previewURI != "" {
		_, preview, _ = decodeDataURI(previewURI)
	}
	if err := a.eng.SetOwnPhoto(a.ctx, full, preview); err != nil {
		a.emit("wa:error", err.Error())
		return
	}
	a.emit("wa:sync", "")
}

// SubscribePresence berlangganan presence (online/last seen) satu kontak —
// dipakai daftar Kontak utk indikator titik hijau.
func (a *App) SubscribePresence(jid string) {
	if a.eng == nil {
		return
	}
	a.eng.SendAvailable()
	a.eng.SubscribePresence(jid)
}

// BizProfileDTO = profil bisnis kontak (kosong bila bukan bisnis).
type BizProfileDTO struct {
	Address  string `json:"address"`
	Email    string `json:"email"`
	Category string `json:"category"`
	IsBiz    bool   `json:"isBiz"`
}

// GetBusinessProfile mengambil info bisnis kontak (alamat/email/kategori).
func (a *App) GetBusinessProfile(jid string) BizProfileDTO {
	if a.eng == nil {
		return BizProfileDTO{}
	}
	bp := a.eng.BusinessProfile(a.ctx, jid)
	if bp == nil {
		return BizProfileDTO{}
	}
	return BizProfileDTO{Address: bp.Address, Email: bp.Email, Category: bp.Category, IsBiz: true}
}

// AddViaQR menukar kode QR-kontak (hasil scan/tempel) → JID, lalu siap di-chat.
func (a *App) AddViaQR(code string) string {
	if a.eng == nil || code == "" {
		return ""
	}
	jid, _, err := a.eng.ResolveQR(a.ctx, code)
	if err != nil {
		a.emit("wa:error", err.Error())
		return ""
	}
	return jid
}

// GetContactAbout mengambil teks info/status seorang kontak.
func (a *App) GetContactAbout(jid string) string {
	if a.eng == nil {
		return ""
	}
	return a.eng.ContactAbout(a.ctx, jid)
}

// Block memblokir / membuka blokir kontak.
func (a *App) Block(jid string, block bool) {
	if a.eng == nil {
		return
	}
	if err := a.eng.Block(a.ctx, jid, block); err != nil {
		a.emit("wa:error", err.Error())
	}
}

// ContactRowDTO = kontak ringkas (daftar blokir / daftar Kontak sidebar).
type ContactRowDTO struct {
	JID   string `json:"jid"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Saved bool   `json:"saved"`
}

// GetContacts mengembalikan daftar kontak (buku-alamat WA + label lokal) urut
// abjad — utk panel "Kontak" di sidebar. Lewati diri sendiri.
func (a *App) GetContacts() []ContactRowDTO {
	out := []ContactRowDTO{}
	if a.eng == nil {
		return out
	}
	self := userPart(a.eng.SelfJID())
	seen := map[string]bool{}
	add := func(jid string) {
		if jid == "" || seen[jid] || userPart(jid) == self {
			return
		}
		seen[jid] = true
		name, saved := a.nameOf(jid)
		phone := a.phoneOf(jid)
		if name == "" {
			name = phone
			if name == "" {
				name = shortJID(jid)
			}
		}
		out = append(out, ContactRowDTO{JID: jid, Name: name, Phone: phone, Saved: saved})
	}
	a.labelsMu.RLock()
	labels := make([]string, 0, len(a.labels))
	for jid := range a.labels {
		labels = append(labels, jid)
	}
	a.labelsMu.RUnlock()
	for _, jid := range labels {
		add(jid)
	}
	for _, jid := range a.eng.ContactJIDs() {
		add(jid)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// GetBlockedContacts mengembalikan kontak yang diblokir (jid + nama).
func (a *App) GetBlockedContacts() (out []ContactRowDTO) {
	out = []ContactRowDTO{}
	if a.eng == nil {
		return
	}
	jids, err := a.eng.Blocklist(a.ctx)
	if err != nil {
		a.emit("wa:error", err.Error())
		return
	}
	for _, j := range jids {
		name := a.eng.ChatName(j)
		if name == "" {
			name = a.eng.ReadableID(j)
		}
		if name == "" {
			name = shortJID(j)
		}
		out = append(out, ContactRowDTO{JID: j, Name: name})
	}
	return
}

// GetPrivacy mengembalikan setelan privasi (name→value).
func (a *App) GetPrivacy() map[string]string {
	if a.eng == nil {
		return map[string]string{}
	}
	return a.eng.PrivacyMap(a.ctx)
}

// SetPrivacy mengubah satu setelan privasi (mis. "lastseen"→"contacts").
func (a *App) SetPrivacy(name, value string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.SetPrivacy(a.ctx, name, value); err != nil {
		a.emit("wa:error", err.Error())
	}
}

// SetMyName memperbarui nama tampil akun sendiri.
func (a *App) SetMyName(name string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.SetMyName(a.ctx, name); err != nil {
		a.emit("wa:error", err.Error())
	}
}

// SetMyAbout memperbarui teks info/status akun sendiri.
func (a *App) SetMyAbout(text string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.SetAbout(a.ctx, text); err != nil {
		a.emit("wa:error", err.Error())
	}
}
