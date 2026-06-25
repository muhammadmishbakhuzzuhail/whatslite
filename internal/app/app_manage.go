// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_manage.go — kelola chat: pin, mute, arsip, tandai belum dibaca, hapus
// chat, dan pencarian isi pesan. Aksi memperbarui DB lokal (UI langsung) lalu
// menyinkron ke server via app-state.

import (
	"time"
)

// Pin menyematkan / melepas sematan chat.
func (a *App) Pin(jid string, pin bool) {
	if a.eng == nil || a.store == nil {
		return
	}
	_ = a.store.SetPinned(a.ctx, jid, pin)
	if err := a.eng.Pin(a.ctx, jid, pin); err != nil {
		a.emit("wa:error", err.Error())
	}
	a.emit("wa:sync", "")
}

// Mute membisukan / mengaktifkan notifikasi chat. dur = lama bisu (0 + mute=true
// → tak terbatas/selamanya). WhatsApp menyimpan waktu-kedaluwarsa server-side &
// otomatis melepas bisu saat sinkron berikutnya.
func (a *App) Mute(jid string, mute bool, dur time.Duration) {
	if a.eng == nil || a.store == nil {
		return
	}
	_ = a.store.SetMuted(a.ctx, jid, mute)
	if mute && dur == 0 {
		dur = 365 * 24 * time.Hour // "selamanya" praktis
	}
	if !mute {
		dur = 0
	}
	if err := a.eng.Mute(a.ctx, jid, mute, dur); err != nil {
		a.emit("wa:error", err.Error())
	}
	a.emit("wa:sync", "")
}

// Archive mengarsip / mengeluarkan chat dari arsip.
func (a *App) Archive(jid string, archive bool) {
	if a.eng == nil || a.store == nil {
		return
	}
	_ = a.store.SetArchived(a.ctx, jid, archive)
	id, ts, fromMe, _, _ := a.store.LastMessage(a.ctx, jid)
	if err := a.eng.Archive(a.ctx, jid, archive, ts, id, fromMe); err != nil {
		a.emit("wa:error", err.Error())
	}
	a.emit("wa:sync", "")
}

// MarkUnread menandai chat belum dibaca (true) / dibaca (false).
func (a *App) MarkUnread(jid string, unread bool) {
	if a.eng == nil || a.store == nil {
		return
	}
	n := 0
	if unread {
		n = 1
	}
	_ = a.store.SetUnread(a.ctx, jid, n)
	id, ts, fromMe, _, _ := a.store.LastMessage(a.ctx, jid)
	if err := a.eng.MarkChatRead(a.ctx, jid, !unread, ts, id, fromMe); err != nil {
		a.emit("wa:error", err.Error())
	}
	a.emit("wa:sync", "")
}

// DeleteChat menghapus chat dari penyimpanan lokal.
func (a *App) DeleteChat(jid string) {
	if a.store == nil {
		return
	}
	_ = a.store.DeleteChat(a.ctx, jid)
	a.emit("wa:sync", "")
}

// ClearChat mengosongkan isi chat (hapus semua pesan) tapi chat tetap ada.
func (a *App) ClearChat(jid string) {
	if a.store == nil {
		return
	}
	_ = a.store.ClearMessages(a.ctx, a.canon(jid))
	a.emit("wa:sync", "")
}

// SearchHitDTO = satu hasil pencarian isi pesan.
type SearchHitDTO struct {
	ChatJID  string `json:"chatJid"`
	ChatName string `json:"chatName"`
	MsgID    string `json:"msgId"`
	Text     string `json:"text"`
	Time     string `json:"time"`
	Group    bool   `json:"group"`
}

// SearchMessages mencari isi pesan lintas chat (maks 50 hasil terbaru). typ:
// ""/"text" (FTS) | "image"|"video"|"document"|"sticker"|"gif"|"voice" | "link".
// Untuk filter jenis, query boleh kosong (jelajah semua).
func (a *App) SearchMessages(query, typ string) (out []SearchHitDTO) {
	out = []SearchHitDTO{}
	if a.store == nil {
		return
	}
	if (typ == "" || typ == "text") && query == "" {
		return
	}
	ms, err := a.store.SearchAdvanced(a.ctx, query, typ, 50)
	if err != nil {
		return
	}
	for _, m := range ms {
		name := ""
		if a.eng != nil {
			name = a.eng.ChatName(m.ChatJID)
		}
		if name == "" {
			name = shortJID(m.ChatJID)
		}
		txt := m.Text
		if txt == "" && typ != "" && typ != "text" {
			txt = "[" + typ + "]"
		}
		out = append(out, SearchHitDTO{
			ChatJID: m.ChatJID, ChatName: name, MsgID: m.ID, Text: txt,
			Time: hm(m.Timestamp), Group: isGroupJID(m.ChatJID),
		})
	}
	return out
}
