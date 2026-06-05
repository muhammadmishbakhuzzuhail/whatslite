package engine

// names.go — resolusi nama tampilan chat/kontak + subjek grup + sinkronisasi
// buku alamat (app-state). Inti masalah @lid: chat JID kini sering berupa JID
// privasi (@lid) sedangkan kontak/pushname ter-index ke nomor (@s.whatsapp.net),
// jadi perlu dijembatani lewat lid_map.

import (
	"context"

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
	if e == nil || e.Client == nil || e.Client.Store == nil {
		return ""
	}
	j, err := types.ParseJID(jid)
	if err != nil {
		return ""
	}
	if j.Server == types.GroupServer {
		e.mu.Lock()
		n := e.groupNames[jid]
		e.mu.Unlock()
		if n != "" {
			return n
		}
		if e.Client.IsConnected() {
			if info, err := e.Client.GetGroupInfo(context.Background(), j); err == nil && info.Name != "" {
				e.mu.Lock()
				e.groupNames[jid] = info.Name
				e.mu.Unlock()
				return info.Name
			}
		}
		return ""
	}
	// 1:1 — resolve nama. Chat JID sekarang sering @lid (JID privasi);
	// kontak/pushname ter-index ke nomor. Jembatani @lid → nomor via lid_map,
	// lalu cari kontak di KEDUA bentuk.
	ctx := context.Background()
	cand := []types.JID{j}
	if j.Server == types.HiddenUserServer && e.Client.Store.LIDs != nil {
		if pn, err := e.Client.Store.LIDs.GetPNForLID(ctx, j); err == nil && !pn.IsEmpty() {
			cand = append(cand, pn)
		}
	}
	if e.Client.Store.Contacts != nil {
		for _, cj := range cand {
			c, err := e.Client.Store.Contacts.GetContact(ctx, cj)
			if err != nil || !c.Found {
				continue
			}
			switch {
			case c.FullName != "":
				return c.FullName
			case c.PushName != "":
				return c.PushName
			case c.BusinessName != "":
				return c.BusinessName
			case c.FirstName != "":
				return c.FirstName
			}
		}
	}
	return ""
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
			return "+" + pn.User
		}
	}
	if j.Server == types.DefaultUserServer && j.User != "" {
		return "+" + j.User
	}
	return ""
}

// SyncContacts menarik app-state buku alamat (nama tersimpan) + aksi chat
// (pin/arsip). Tanpa ini ContactStore kosong → nama tampil sebagai nomor.
// Aman dipanggil tiap connect; onlyIfNotSynced=true → tak ulang bila sudah.
func (e *Engine) SyncContacts() {
	if e == nil || e.Client == nil || !e.Client.IsConnected() {
		return
	}
	ctx := context.Background()
	for _, name := range []appstate.WAPatchName{
		appstate.WAPatchCriticalUnblockLow, // kontak (nama tersimpan)
		appstate.WAPatchRegularHigh,        // pin
		appstate.WAPatchRegularLow,         // arsip/mute
	} {
		_ = e.Client.FetchAppState(ctx, name, false, true)
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

