// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// mockengine — harness uji Fase-3: pakai paket internal/ipc ASLI (server bridge
// yang sama dgn produksi) untuk membuktikan klien Qt bisa call method + terima
// event lewat protokol NDJSON/UDS. Bukan bagian app; hanya untuk PoC.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/ipc"
)

// State tiruan: pesan terkirim per-chat (membuktikan jalur TULIS end-to-end).
var (
	mu   sync.Mutex
	sent = map[string][]map[string]any{}
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: mockengine <sock>")
		os.Exit(2)
	}
	srv, err := ipc.Listen(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer srv.Close()

	// Dispatcher tiruan: GetChats balas 2 chat; lainnya error.
	srv.SetHandler(func(method string, args []json.RawMessage) (any, error) {
		switch method {
		case "GetChats":
			// Skenario IDENTIK dgn frontend/src/lib/data/mock.js → render Svelte &
			// QML bisa di-DSSIM apple-to-apple (tools/diff.sh). Jangan diubah lepas
			// dari mock.js; itu yang bikin perbandingan objektif.
			return []map[string]any{
				{"id": "1", "name": "Andi Pratama", "preview": "Mantap! Sampai nanti malam 🙌", "time": "19.08", "sent": true, "status": "sent", "pinned": true},
				{"id": "6", "name": "Mama", "preview": "Sudah sampai rumah?", "time": "12.30", "sent": true, "status": "sent", "pinned": true},
				{"id": "2", "name": "Keluarga 👨‍👩‍👧", "preview": "Ibu: Jangan lupa makan ya nak", "time": "18.41", "badge": 2, "unread": true, "group": true},
				{"id": "3", "name": "Sarah", "preview": "Oke besok aku kabarin lagi", "time": "17.55", "sent": true, "status": "sent"},
				{"id": "4", "name": "Tim Proyek X", "preview": "Budi: file-nya udah aku upload", "time": "16.20", "badge": 12, "unread": true, "group": true},
				{"id": "5", "name": "Rian", "preview": "Haha iya bener banget 😂", "time": "14.03", "muted": true},
				{"id": "7", "name": "Info Kampus", "preview": "Pengumuman: jadwal UAS telah...", "time": "Kemarin", "muted": true, "group": true},
				{"id": "8", "name": "Dimas", "preview": "Nanti aku telpon ya", "time": "Kemarin", "sent": true, "status": "sent"},
				{"id": "9", "name": "Grup Futsal", "preview": "Anto: yang ikut absen dong", "time": "Kemarin", "badge": 5, "unread": true, "group": true},
				{"id": "10", "name": "Nadia", "preview": "Makasih banyak ya 🙏", "time": "Senin", "sent": true, "status": "sent"},
				{"id": "11", "name": "Kerja Kelompok", "preview": "Kamu: oke aku kerjain bagian 2", "time": "Senin", "sent": true, "status": "sent", "group": true},
				{"id": "12", "name": "Bayu", "preview": "📷 Foto", "time": "Minggu", "muted": true},
			}, nil
		case "GetMessages":
			var chatID string
			if len(args) > 0 {
				_ = json.Unmarshal(args[0], &chatID)
			}
			// Pesan chat "Andi Pratama" (mock.js messagesByChat[1]) — chat pertama
			// yang auto-dipilih → render percakapan setara Svelte utk DSSIM.
			base := []map[string]any{
				{"id": "m1", "dir": "in", "type": "text", "text": "Halo! Jadi nanti malam ngumpul jam berapa?", "time": "19.02",
					"reactions": []map[string]any{{"emoji": "👍", "count": 2}, {"emoji": "❤️", "count": 1}, {"emoji": "😂", "count": 5}}},
				{"id": "m2", "dir": "out", "type": "text", "text": "Jam 8 ya, di tempat biasa 👌", "time": "19.03", "status": "read",
					"reactions": []map[string]any{{"emoji": "🔥", "count": 1}}},
				{"id": "m3", "dir": "in", "type": "text", "text": "Oke sip. Oh iya aku bawa kamera, sekalian foto-foto. Kamu bawa speaker yang kemarin gak?", "time": "19.04"},
				{"id": "m4", "dir": "out", "type": "text", "text": "Bawa dong, udah aku charge full 🔋", "time": "19.05", "status": "read",
					"quoteId": "m3", "quoteName": "Andi Pratama", "quoteText": "Kamu bawa speaker yang kemarin gak?"},
				{"id": "m5", "dir": "in", "type": "image", "text": "Spot kemarin, bagus banget buat sunset 🌅", "time": "19.06",
					"thumb": "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='320' height='240'%3E%3Cdefs%3E%3ClinearGradient id='s' x1='0' y1='0' x2='0' y2='1'%3E%3Cstop offset='0' stop-color='rgb(255,176,99)'/%3E%3Cstop offset='0.45' stop-color='rgb(255,118,92)'/%3E%3Cstop offset='1' stop-color='rgb(108,58,128)'/%3E%3C/linearGradient%3E%3C/defs%3E%3Crect width='320' height='240' fill='url(%23s)'/%3E%3Ccircle cx='160' cy='128' r='38' fill='rgb(255,228,158)'/%3E%3C/svg%3E"},
				{"id": "m6", "dir": "in", "type": "voice", "text": "0:12", "time": "19.07"},
				{"id": "m7", "dir": "out", "type": "text", "text": "Mantap! Sampai nanti malam 🙌", "time": "19.08", "status": "sending"},
			}
			mu.Lock()
			base = append(base, sent[chatID]...)
			mu.Unlock()
			return base, nil
		case "SendText":
			// args: [chatID, text] → simpan sebagai pesan keluar.
			var chatID, text string
			if len(args) > 0 {
				_ = json.Unmarshal(args[0], &chatID)
			}
			if len(args) > 1 {
				_ = json.Unmarshal(args[1], &text)
			}
			mu.Lock()
			sent[chatID] = append(sent[chatID], map[string]any{
				"id": fmt.Sprintf("s%d", len(sent[chatID])+1), "dir": "out",
				"type": "text", "text": text, "time": "now",
			})
			mu.Unlock()
			return "sent-id", nil
		case "ListSavedStickers":
			return []map[string]any{
				{"hash": "aaa", "animated": false, "source": "alice@s.whatsapp.net"},
				{"hash": "bbb", "animated": true, "source": "bob@s.whatsapp.net"},
				{"hash": "ccc", "animated": false, "source": ""},
			}, nil
		case "SendSavedSticker":
			var chatID string
			if len(args) > 0 {
				_ = json.Unmarshal(args[0], &chatID)
			}
			mu.Lock()
			sent[chatID] = append(sent[chatID], map[string]any{
				"id": fmt.Sprintf("s%d", len(sent[chatID])+1), "dir": "out",
				"type": "sticker", "text": "", "time": "now",
			})
			mu.Unlock()
			return "sticker-id", nil
		case "ListSavedGifs":
			return []map[string]any{
				{"hash": "g1", "mime": "video/mp4", "source": "alice@s.whatsapp.net"},
				{"hash": "g2", "mime": "video/mp4", "source": ""},
			}, nil
		case "SendSavedGif":
			var chatID string
			if len(args) > 0 {
				_ = json.Unmarshal(args[0], &chatID)
			}
			mu.Lock()
			sent[chatID] = append(sent[chatID], map[string]any{
				"id": fmt.Sprintf("s%d", len(sent[chatID])+1), "dir": "out", "type": "gif", "text": "", "time": "now",
			})
			mu.Unlock()
			return "gif-id", nil
		case "React", "StarMessage", "DeleteMessage", "SaveSticker", "SaveGif", "SetKeepDeleted", "Connect", "Forward", "LeaveGroup",
			"MarkRead", "MarkUnread", "Pin", "Mute", "Archive", "DeleteChat", "PinMessage", "EditMessage", "SendTyping", "Logout":
			return nil, nil // terima (efek nyata di engine asli)
		case "MyQR":
			return "", nil
		case "Reply":
			var chatID, text string
			if len(args) > 0 {
				_ = json.Unmarshal(args[0], &chatID)
			}
			if len(args) > 1 {
				_ = json.Unmarshal(args[1], &text)
			}
			mu.Lock()
			sent[chatID] = append(sent[chatID], map[string]any{
				"id": fmt.Sprintf("s%d", len(sent[chatID])+1), "dir": "out", "type": "text", "text": text, "time": "now"})
			mu.Unlock()
			return "reply-id", nil
		case "GetMessagesBefore":
			return []map[string]any{
				{"id": "old1", "dir": "in", "type": "text", "text": "(pesan lebih lama 1)", "time": "08:50", "ts": 1},
				{"id": "old2", "dir": "out", "type": "text", "text": "(pesan lebih lama 2)", "time": "08:51", "ts": 2},
			}, nil
		case "GetGroupInfo":
			return map[string]any{
				"name": "Grup Kerja", "desc": "Koordinasi tim proyek WhatsLite", "count": 4,
				"members": []map[string]any{
					{"name": "Kamu", "admin": true},
					{"name": "Budi", "admin": true},
					{"name": "Citra", "admin": false},
					{"name": "Dewi", "admin": false},
				},
			}, nil
		case "GetContactProfile":
			return map[string]any{
				"name": "Alice", "about": "Hai! Pakai WhatsLite", "phone": "+62 812-3456-7890",
			}, nil
		case "GetPrivacy":
			return map[string]any{
				"lastseen": "everyone", "profile": "contacts", "status": "everyone",
				"readreceipts": "all", "groupadd": "contacts", "online": "everyone",
			}, nil
		case "SetPrivacy":
			return nil, nil
		case "GetMessageInfo":
			return map[string]any{
				"readBy":      []map[string]any{{"name": "Budi", "time": "09:03"}, {"name": "Citra", "time": "09:04"}},
				"deliveredTo": []map[string]any{{"name": "Dewi", "time": "09:02"}},
			}, nil
		case "SendMedia":
			var chatID, kind, name string
			if len(args) > 0 {
				_ = json.Unmarshal(args[0], &chatID)
			}
			if len(args) > 1 {
				_ = json.Unmarshal(args[1], &kind)
			}
			if len(args) > 3 {
				_ = json.Unmarshal(args[3], &name)
			}
			mu.Lock()
			sent[chatID] = append(sent[chatID], map[string]any{
				"id": fmt.Sprintf("s%d", len(sent[chatID])+1), "dir": "out", "type": kind,
				"text": name, "time": "now", "docMime": "application/pdf", "docSize": 524288,
			})
			mu.Unlock()
			return "media-id", nil
		case "GetStatuses":
			return []map[string]any{
				{"id": "st1", "name": "Alice", "count": 2, "time": "30 mnt lalu", "seen": false},
				{"id": "st2", "name": "Bob", "count": 1, "time": "2 jam lalu", "seen": true},
			}, nil
		case "GetContacts":
			return []map[string]any{
				{"jid": "alice@s.whatsapp.net", "name": "Alice", "about": "Hai!"},
				{"jid": "bob@s.whatsapp.net", "name": "Bob", "about": "Sibuk"},
				{"jid": "carol@s.whatsapp.net", "name": "Carol", "about": ""},
			}, nil
		case "SearchMessages":
			var q string
			if len(args) > 0 {
				_ = json.Unmarshal(args[0], &q)
			}
			return []map[string]any{
				{"chatName": "Bob", "text": "hasil cocok untuk \"" + q + "\"", "time": "11/06"},
				{"chatName": "Grup Kerja", "text": "lagi satu hit: " + q, "time": "10/06"},
			}, nil
		case "GetKeepDeleted":
			return true, nil
		case "GetState":
			return "ready", nil
		case "GetChannels":
			return []map[string]any{
				{"name": "WhatsLite News", "preview": "v2.0 rilis 🎉"},
				{"name": "Tech Daily", "preview": "5 berita baru"},
			}, nil
		case "GetCommunities":
			return []map[string]any{
				{"name": "Komunitas TI", "subtitle": "4 grup · 312 anggota"},
				{"name": "RT 07", "subtitle": "2 grup · 48 anggota"},
			}, nil
		case "GetArchivedChats":
			return []map[string]any{
				{"name": "Promo Toko", "preview": "Diskon 50%..."},
				{"name": "Bot Spam", "preview": "diarsipkan"},
			}, nil
		case "GetScheduled":
			return []map[string]any{
				{"chatName": "Bob", "text": "Selamat ulang tahun! 🎂", "time": "besok 08:00"},
				{"chatName": "Grup Kerja", "text": "Reminder standup", "time": "Sen 09:00"},
			}, nil
		case "GetCalls":
			return []map[string]any{
				{"id": "c1", "name": "Alice", "video": true, "group": false, "status": "missed", "time": "Kemarin 19:20"},
				{"id": "c2", "name": "Grup Kerja", "video": false, "group": true, "status": "rejected", "time": "Senin 08:05"},
			}, nil
		case "GetStarred":
			return []map[string]any{
				{"chatName": "Bob", "text": "Jangan lupa meeting jam 3", "time": "11/06"},
				{"chatName": "Grup Kerja", "text": "Link dokumen: ...", "time": "10/06"},
			}, nil
		case "GetProfile":
			return map[string]any{"name": "Saya", "about": "Pakai WhatsLite", "phone": "+62 811-0000-0000"}, nil
		case "GetStorageUsage":
			return map[string]any{"messages": 12450, "dbBytes": 18874368, "mediaBytes": 134217728}, nil
		case "GetProxy":
			return "", nil
		case "GetRetention":
			return 90, nil
		}
		// Default: terima method aksi apa pun (efek nyata ada di engine asli).
		return nil, nil
	})

	// Siar event berkala → klien pasti menangkap satu (terlepas timing connect).
	t := time.NewTicker(150 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		srv.Broadcast("wa:message", "a@s.whatsapp.net")
		srv.Broadcast("wa:typing", map[string]any{"chat": "a@s.whatsapp.net", "on": true})
	}
}
