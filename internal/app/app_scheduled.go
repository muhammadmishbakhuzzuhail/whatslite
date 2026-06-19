// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_scheduled.go — pesan terjadwal + pengingat. Eksekusi via ticker (lihat
// startScheduler). Client-side: hanya jalan saat app hidup; yang lewat saat app
// mati dikirim/ditampilkan saat app dibuka lagi (catch-up).

import (
	"fmt"
	"time"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/storage"
)

func (a *App) genID() string { return fmt.Sprintf("loc-%d", time.Now().UnixNano()) }

// ScheduleMessage menjadwalkan pesan teks ke chat pada waktu (unix detik).
func (a *App) ScheduleMessage(chatJID, text string, sendAt int64) {
	if a.store == nil || text == "" {
		return
	}
	_ = a.store.AddScheduled(a.ctx, storage.Scheduled{
		ID: a.genID(), ChatJID: a.canon(chatJID), ChatName: a.displayName(a.canon(chatJID)),
		Text: text, SendAt: sendAt, Created: time.Now().Unix(),
	})
	a.emit("wa:scheduled", "")
}

func (a *App) GetScheduled() []storage.Scheduled {
	if a.store == nil {
		return []storage.Scheduled{}
	}
	out, _ := a.store.ListScheduled(a.ctx)
	if out == nil {
		return []storage.Scheduled{}
	}
	return out
}

func (a *App) CancelScheduled(id string) {
	if a.store != nil {
		_ = a.store.DeleteScheduled(a.ctx, id)
		a.emit("wa:scheduled", "")
	}
}

// AddReminder membuat pengingat pada pesan/chat pada waktu (unix).
func (a *App) AddReminder(chatJID, msgID, note string, remindAt int64) {
	if a.store == nil {
		return
	}
	cj := a.canon(chatJID)
	_ = a.store.AddReminder(a.ctx, storage.Reminder{
		ID: a.genID(), ChatJID: cj, ChatName: a.displayName(cj), MsgID: msgID, Note: note, RemindAt: remindAt,
	})
	a.emit("wa:reminders", "")
}

func (a *App) GetReminders() []storage.Reminder {
	if a.store == nil {
		return []storage.Reminder{}
	}
	out, _ := a.store.ListReminders(a.ctx)
	if out == nil {
		return []storage.Reminder{}
	}
	return out
}

func (a *App) CancelReminder(id string) {
	if a.store != nil {
		_ = a.store.DeleteReminder(a.ctx, id)
		a.emit("wa:reminders", "")
	}
}

// startScheduler menjalankan pengirim terjadwal + pemicu pengingat tiap 20 dtk
// (+ sekali di boot, utk catch-up). Off-loop (goroutine sendiri).
func (a *App) startScheduler() {
	tick := func() {
		if a.store == nil {
			return
		}
		now := time.Now().Unix()
		// Pesan terjadwal jatuh tempo → kirim + hapus.
		if due, _ := a.store.DueScheduled(a.ctx, now); len(due) > 0 {
			for _, m := range due {
				if a.eng != nil {
					a.SendText(m.ChatJID, m.Text)
				}
				_ = a.store.DeleteScheduled(a.ctx, m.ID)
			}
			a.emit("wa:scheduled", "")
		}
		// Pengingat jatuh tempo → event in-app (toast/suara di FE) + hapus.
		// TANPA notif desktop (dihapus total).
		if due, _ := a.store.DueReminders(a.ctx, now); len(due) > 0 {
			for _, r := range due {
				a.emit("wa:reminder", map[string]interface{}{
					"chatJid": r.ChatJID, "chatName": r.ChatName, "msgId": r.MsgID, "note": r.Note,
				})
				_ = a.store.DeleteReminder(a.ctx, r.ID)
			}
			a.emit("wa:reminders", "")
		}
	}
	go func() {
		time.Sleep(3 * time.Second) // beri waktu koneksi siap utk catch-up kirim
		tick()
		t := time.NewTicker(20 * time.Second)
		defer t.Stop()
		for range t.C {
			tick()
		}
	}()
}
