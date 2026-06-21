// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package engine

// names.go — resolusi nama tampilan chat/kontak + subjek grup + sinkronisasi
// buku alamat (app-state). Inti masalah @lid: chat JID kini sering berupa JID
// privasi (@lid) sedangkan kontak/pushname ter-index ke nomor (@s.whatsapp.net),
// jadi perlu dijembatani lewat lid_map.

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// ChatName me-resolve nama tampilan sebuah chat:
//   - 1:1  : nama kontak dari address book (FullName/PushName/BusinessName) — offline OK.
//   - grup : subjek grup (di-cache; ambil dari server hanya bila tersambung).
//
// Mengembalikan "" bila tak diketahui (pemanggil fallback ke nomor/JID).
func (e *Engine) ChatName(jid string) string {
	n, _ := e.ResolveName(jid)
	return n
}

// ResolveName seperti ChatName tetapi juga mengembalikan apakah nama itu berasal
// dari buku-alamat (saved=true → FullName/FirstName tersimpan) atau hanya
// pushname yg di-set sendiri pengirim (saved=false). Pemanggil pakai flag ini
// utk memutuskan tampil "Nama + nomor" (grup, tak tersimpan) vs "Nama" saja.
func (e *Engine) ResolveName(jid string) (string, bool) {
	if e == nil || e.Client == nil || e.Client.Store == nil {
		return "", false
	}
	j, err := types.ParseJID(jid)
	if err != nil {
		return "", false
	}
	// Bot Meta AI (server "bot") → label tetap "Meta AI", bukan nomor mentah.
	if j.Server == "bot" {
		return "Meta AI", true
	}
	if j.Server == types.GroupServer {
		e.mu.Lock()
		n := e.groupNames[jid]
		e.mu.Unlock()
		if n != "" {
			return n, true
		}
		if e.Client.IsConnected() {
			if info, err := e.Client.GetGroupInfo(context.Background(), j); err == nil && info.Name != "" {
				e.cacheGroupName(jid, info.Name)
				return info.Name, true
			}
		}
		return "", false
	}
	// 1:1 — resolve nama. Chat JID sekarang sering @lid (JID privasi);
	// kontak/pushname ter-index ke nomor. Jembatani @lid → nomor via lid_map,
	// lalu cari kontak di KEDUA bentuk. FullName/FirstName = tersimpan di buku
	// alamat; PushName/BusinessName = nama yg di-set pemilik akun sendiri.
	ctx := context.Background()
	cand := []types.JID{j}
	if j.Server == types.HiddenUserServer && e.Client.Store.LIDs != nil {
		if pn, err := e.Client.Store.LIDs.GetPNForLID(ctx, j); err == nil && !pn.IsEmpty() {
			cand = append(cand, pn)
		}
	}
	var push string // pushname/business sebagai fallback (tak-tersimpan)
	if e.Client.Store.Contacts != nil {
		for _, cj := range cand {
			c, err := e.Client.Store.Contacts.GetContact(ctx, cj)
			if err != nil || !c.Found {
				continue
			}
			switch {
			case c.FullName != "":
				return c.FullName, true
			case c.FirstName != "":
				return c.FirstName, true
			}
			if push == "" {
				if c.PushName != "" {
					push = c.PushName
				} else if c.BusinessName != "" {
					push = c.BusinessName
				}
			}
		}
	}
	return push, false
}

// CanonicalJID menyatukan identitas chat 1:1 ke SATU bentuk kanonik agar tak ada
// chat ganda. whatsmeow kini sering memakai JID privasi (@lid) untuk percakapan
// yang sama yang sebelumnya tersimpan sebagai nomor (@s.whatsapp.net) → 2 baris
// chat utk 1 orang. Kanonik = nomor (@s.whatsapp.net) bila pemetaan lid→nomor
// diketahui; selainnya JID dikembalikan apa adanya (grup/bot/nomor tak berubah).
func (e *Engine) CanonicalJID(jid string) string {
	if e == nil || e.Client == nil || e.Client.Store == nil {
		return jid
	}
	j, err := types.ParseJID(jid)
	if err != nil {
		return jid
	}
	if j.Server == types.HiddenUserServer && e.Client.Store.LIDs != nil {
		if pn, err := e.Client.Store.LIDs.GetPNForLID(context.Background(), j); err == nil && !pn.IsEmpty() {
			return pn.ToNonAD().String()
		}
	}
	return jid
}

// ReadableID memberi label terbaca utk chat tanpa nama: nomor "+62…" alih-alih
// @lid 15-digit mentah. Jembatani @lid → nomor via lid_map bila bisa.
func (e *Engine) ReadableID(jid string) string {
	if e == nil || e.Client == nil || e.Client.Store == nil {
		return ""
	}
	j, err := types.ParseJID(jid)
	if err != nil {
		return ""
	}
	if j.Server == types.HiddenUserServer && e.Client.Store.LIDs != nil {
		if pn, err := e.Client.Store.LIDs.GetPNForLID(context.Background(), j); err == nil && pn.User != "" {
			return formatPhone(pn.User)
		}
	}
	if j.Server == types.DefaultUserServer && j.User != "" {
		return formatPhone(j.User)
	}
	return ""
}

// ccSet = kode panggil negara (ITU) yang dikenal, utk pisah CC dari nomor
// nasional. Dipakai longest-match (coba 3 digit, lalu 2, lalu 1).
var ccSet = map[string]bool{
	"1": true, "7": true,
	"20": true, "27": true, "30": true, "31": true, "32": true, "33": true, "34": true,
	"36": true, "39": true, "40": true, "41": true, "43": true, "44": true, "45": true,
	"46": true, "47": true, "48": true, "49": true, "51": true, "52": true, "53": true,
	"54": true, "55": true, "56": true, "57": true, "58": true, "60": true, "61": true,
	"62": true, "63": true, "64": true, "65": true, "66": true, "81": true, "82": true,
	"84": true, "86": true, "90": true, "91": true, "92": true, "93": true, "94": true,
	"95": true, "98": true,
	"212": true, "213": true, "216": true, "218": true, "220": true, "221": true, "234": true,
	"254": true, "255": true, "256": true, "351": true, "352": true, "353": true, "354": true,
	"355": true, "358": true, "359": true, "370": true, "371": true, "372": true, "380": true,
	"381": true, "385": true, "386": true, "420": true, "421": true, "852": true, "853": true,
	"855": true, "856": true, "880": true, "886": true, "960": true, "961": true, "962": true,
	"963": true, "964": true, "965": true, "966": true, "967": true, "968": true, "971": true,
	"972": true, "973": true, "974": true, "975": true, "976": true, "977": true, "992": true,
	"993": true, "994": true, "995": true, "996": true, "998": true,
}

// formatPhone memformat nomor mentah (hanya digit, tanpa +) ala WhatsApp:
// "+<CC> NNN-NNNN-NNNN". CC dipisah via longest-match ccSet; bila tak dikenal,
// kembalikan "+<digit>" apa adanya (aman, tak salah-pisah). Indonesia (62) →
// "+62 815-1934-6661".
func formatPhone(user string) string {
	if user == "" {
		return ""
	}
	for _, r := range user {
		if r < '0' || r > '9' {
			return "+" + user // non-digit (tak terduga) → jangan format
		}
	}
	cc := ""
	for _, n := range []int{3, 2, 1} {
		if len(user) > n && ccSet[user[:n]] {
			cc = user[:n]
			break
		}
	}
	if cc == "" {
		return "+" + user
	}
	return "+" + cc + " " + groupDigits(user[len(cc):])
}

// groupDigits mengelompokkan nomor nasional: 3 digit pertama lalu blok 4
// (815-1934-6661). Sisa terakhir <=4 digit jadi satu blok.
func groupDigits(s string) string {
	if len(s) <= 4 {
		return s
	}
	parts := []string{s[:3]}
	s = s[3:]
	for len(s) > 4 {
		parts = append(parts, s[:4])
		s = s[4:]
	}
	return strings.Join(append(parts, s), "-")
}

// SyncContacts menarik app-state buku alamat (nama tersimpan) + aksi chat
// (pin/arsip/mute). Tanpa ini ContactStore kosong → nama tampil sebagai nomor.
//
// onlyIfNotSynced SELALU false: tiap reconnect kita REKONSILIASI (ambil patch yg
// berubah saat offline) — bukan skip bila "sudah sinkron" (itu penyebab nama jadi
// nomor setelah lama offline; lihat riset arsitektur Telegram: catch-up eksplisit
// di tiap reconnect). force=true → fullSync snapshot kontak (rebuild bersih).
func (e *Engine) SyncContacts(force bool) {
	if e == nil || e.Client == nil || !e.Client.IsConnected() {
		return
	}
	ctx := context.Background()
	for _, name := range []appstate.WAPatchName{
		appstate.WAPatchCriticalUnblockLow, // kontak (nama tersimpan)
		appstate.WAPatchRegularHigh,        // pin + bintang
		appstate.WAPatchRegularLow,         // arsip/mute
		appstate.WAPatchCriticalBlock,      // pushname/locale sendiri
	} {
		n := name
		full := force && n == appstate.WAPatchCriticalUnblockLow // snapshot kontak penuh
		_ = retry(ctx, 3, func() error { return e.Client.FetchAppState(ctx, n, full, false) })
	}
}

// OnContactsSynced memanggil fn saat kontak/app-state berubah (mis. setelah
// SyncContacts) sehingga UI bisa refresh nama.
func (e *Engine) OnContactsSynced(fn func()) {
	e.Client.AddEventHandler(func(evt interface{}) {
		switch evt.(type) {
		case *events.Contact, *events.PushName, *events.AppStateSyncComplete:
			fn()
		}
	})
}
