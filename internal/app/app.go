// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app.go — siklus hidup aplikasi Wails + perkabelan event engine→UI.
//
// App di-bind ke JS sebagai window.go.main.App; method publik tersebar per
// domain agar mudah didokumentasikan & diperbaiki:
//   - app.go         : struct App, startup/domReady/shutdown, OpenChat, helper
//   - app_chats.go   : GetChats / GetMessages (+ DTO untuk frontend)
//   - app_connect.go : Connect / Logout / GetState / SendText (sesi & QR)
//   - app_media.go   : DownloadMedia + asset-server /media & /avatar

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/engine"
	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/storage"
)

// App = konteks aplikasi Wails. Method publik di sini otomatis ter-bind ke JS
// sebagai window.go.main.App.<Method>. Event ke UI lewat runtime.EventsEmit.
type App struct {
	ctx        context.Context
	eng        *engine.Engine
	store      *storage.Store
	mediaDir   string // cache file media (bukan di DB) → ringan
	stickerDir string // koleksi stiker tersimpan (permanen, di luar LRU media)
	gifDir     string // koleksi GIF tersimpan (permanen, di luar LRU media)

	labelsMu sync.RWMutex      // melindungi labels
	labels   map[string]string // label kontak lokal (jid → nama), cache dari DB

	retentionDays int // 0 = simpan selamanya; >0 = prune pesan lebih tua (kec. berbintang/disematkan)

	keepDeleted atomic.Bool // anti-delete: simpan isi pesan yang ditarik (default on)
	didFullSync atomic.Bool // sudah fullSync snapshot kontak sesi ini? (hindari ulang)

	qrMu   sync.RWMutex // melindungi qrCode
	qrCode string       // kode QR pairing mentah terbaru (utk UI in-process spt Gio)

	presMu   sync.RWMutex            // melindungi presence + typing
	presence map[string]string       // jid → "online"/"terakhir dilihat .."/""
	typing   map[string]typingStateT // jid → status mengetik (utk subtitle header)

	openMu   sync.RWMutex // melindungi openChat
	openChat string       // chat yg sedang dibuka (jangan naikkan unread-nya)

	version string // versi build (di-stamp via -ldflags -X main.version) → UI "Tentang"

	wq chan func() // antrian tulis-DB serial (off the whatsmeow socket loop)

	headless bool // true = jalan tanpa Wails (UI Gio in-process) → skip runtime.*
}

// emit menyiarkan event UI ke Wails (Svelte). Mode headless (Gio) tak pakai event
// — UI Gio polling state langsung (GetChats/GetMessages/QRCode/ChatSubtitle).
func (a *App) emit(event string, data ...any) {
	if !a.headless { // mode Wails: kirim ke JS/Svelte
		runtime.EventsEmit(a.ctx, event, data...)
	}
}

// logErr mencatat error secara aman di kedua mode: Wails (runtime.LogError) atau
// headless (log standar). Hindari runtime.* dgn ctx non-Wails yang bisa panik.
func (a *App) logErr(msg string) {
	if a.headless {
		log.Println("[engine]", msg)
		return
	}
	runtime.LogError(a.ctx, msg)
}

// SetVersion menyetel versi build (dipanggil dari main sebelum wails.Run).
func (a *App) SetVersion(v string) {
	if v != "" {
		a.version = v
	}
}

// Version mengembalikan versi build (di-bind ke FE untuk layar "Tentang").
func (a *App) Version() string {
	if a.version == "" {
		return "dev"
	}
	return a.version
}

// retentionCutoff = batas unix; pesan lebih tua di-prune. 0 bila retensi mati.
func (a *App) retentionCutoff() int64 {
	if a.retentionDays <= 0 {
		return 0
	}
	return time.Now().AddDate(0, 0, -a.retentionDays).Unix()
}

// bg mengantre kerja tulis-DB ke writer tunggal. Handler whatsmeow dipanggil
// SINKRON di goroutine pembaca socket; menulis langsung (apalagi ke 1 koneksi
// SQLite saat banjir history-sync) memblok loop → "Node handling took 9s" →
// websocket EOF → sinkron tak selesai. Enqueue = handler balik seketika; satu
// drainer = tanpa kontensi koneksi + urutan terjaga. Antrian penuh → jalankan
// inline (backpressure) agar tak ada yang hilang.
func (a *App) bg(fn func()) {
	if a.wq == nil {
		fn()
		return
	}
	select {
	case a.wq <- fn:
	default:
		fn()
	}
}

func NewApp() *App { return &App{} }

// startup: inisialisasi engine + storage (TANPA connect — connect lewat Connect()).
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	// WebKitGTK (terutama di Arch) memasang signal handler tanpa SA_ONSTACK →
	// crash "non-Go code set up signal handler without SA_ONSTACK". Pulihkan,
	// lalu ulangi berkala karena JSC bisa menimpanya lagi tiap siklus GC.
	runtime.ResetSignalHandlers()
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for range t.C {
			runtime.ResetSignalHandlers()
		}
	}()
	dataDir, err := engine.DefaultDataDir()
	if err != nil {
		runtime.LogError(ctx, "data dir: "+err.Error())
		return
	}
	eng, err := engine.New(ctx, filepath.Join(dataDir, "whatslite.db"), os.Getenv("WALITE_DEBUG") != "")
	if err != nil {
		runtime.LogError(ctx, "engine: "+err.Error())
		return
	}
	store, err := storage.New(ctx, filepath.Join(dataDir, "app.db"))
	if err != nil {
		runtime.LogError(ctx, "storage: "+err.Error())
		return
	}
	a.eng = eng
	a.store = store
	a.loadLabels()
	// Writer DB tunggal: serialkan semua tulis dari event handler off-socket-loop.
	a.wq = make(chan func(), 8192)
	go func() {
		for fn := range a.wq {
			// Recover per-tugas: satu panic (nil deref/proto buruk) tak boleh
			// membunuh drainer → kalau mati, SEMUA tulis DB berikutnya senyap hilang.
			func() {
				defer func() {
					if r := recover(); r != nil {
						runtime.LogError(a.ctx, fmt.Sprintf("bg write panic: %v", r))
					}
				}()
				fn()
			}()
		}
	}()
	a.mediaDir = filepath.Join(dataDir, "media")
	_ = os.MkdirAll(a.mediaDir, 0o755)
	a.startMediaEviction(512 << 20) // cap cache media ~512MB (LRU by modtime)

	// Koleksi stiker tersimpan: dir TERPISAH dari media → TAK kena LRU evict,
	// jadi stiker yang disimpan dari teman bertahan permanen.
	a.stickerDir = filepath.Join(dataDir, "stickers")
	_ = os.MkdirAll(a.stickerDir, 0o755)
	a.gifDir = filepath.Join(dataDir, "gifs")
	_ = os.MkdirAll(a.gifDir, 0o755)

	// Retensi pesan: default 90 hari. Prune + VACUUM sekali saat boot (off-loop)
	// → app.db tetap ramping walau riwayat besar. Berbintang/disematkan aman.
	// Proxy (opsional): terapkan SEBELUM Connect (dipicu FE setelah ini).
	if px := store.GetMeta(ctx, "proxy", ""); px != "" {
		_ = eng.SetProxy(px)
	}

	a.retentionDays = atoiDef(store.GetMeta(ctx, "retention_days", "90"), 90)
	a.keepDeleted.Store(store.GetMeta(ctx, "keep_deleted", "1") == "1") // anti-delete default ON
	a.bg(func() {
		if cut := a.retentionCutoff(); cut > 0 {
			if n, _ := a.store.PruneMessages(a.ctx, cut); n > 0 {
				_ = a.store.Vacuum(a.ctx)
				a.emit("wa:sync", "")
			}
		}
	})

	// Disappearing messages: sapu yang kedaluwarsa saat boot + tiap 60 dtk.
	go func() {
		sweep := func() {
			if a.store == nil {
				return
			}
			if n, _ := a.store.SweepExpired(a.ctx, time.Now().Unix()); n > 0 {
				a.emit("wa:sync", "")
			}
		}
		sweep()
		t := time.NewTicker(60 * time.Second)
		defer t.Stop()
		for range t.C {
			sweep()
		}
	}()

	bgClose.Store(store.GetMeta(ctx, "bg_close", "0") == "1") // mode latar
	a.startScheduler()                                        // pesan terjadwal + pengingat (ticker sendiri)

	// IPC single-instance: instance ke-2 yang gagal flock akan men-dial socket ini
	// → kita angkat window ke depan (bukan diam).
	sock := filepath.Join(dataDir, ".ipc.sock")
	os.Remove(sock)
	if l, err := net.Listen("unix", sock); err == nil {
		go func() {
			for {
				c, e := l.Accept()
				if e != nil {
					return
				}
				c.Close()
				runtime.WindowUnminimise(a.ctx)
				runtime.WindowShow(a.ctx)
			}
		}()
	}
	// Perbaiki drift urutan/preview dari data lama (build sebelumnya) sekali di awal.
	_ = store.RecomputeSummaries(ctx)

	a.wireEvents(eng, store)
}

// wireEvents menyambungkan callback engine → simpan ke storage → emit event UI.
func (a *App) wireEvents(eng *engine.Engine, store *storage.Store) {
	// Pesan masuk → simpan ke storage → beri tahu UI (chat JID-nya).
	eng.OnMessage(func(m engine.IncomingMessage) {
		chat := eng.CanonicalJID(m.Chat) // satukan @lid↔nomor → 1 chat
		var expireAt int64
		if m.ExpireSecs > 0 { // disappearing → set waktu kedaluwarsa
			expireAt = m.Timestamp.Add(time.Duration(m.ExpireSecs) * time.Second).Unix()
		}
		a.bg(func() {
			_ = store.SaveMessage(a.ctx, storage.Message{
				ID: m.ID, ChatJID: chat, Sender: m.Sender, PushName: m.PushName,
				Text: m.Text, Kind: m.Kind, Thumb: m.Thumb, Media: m.Media,
				Timestamp: m.Timestamp, FromMe: m.FromMe,
				QuotedID: m.QuotedID, QuotedSender: m.QuotedSender, QuotedText: m.QuotedText,
				ExpireAt: expireAt,
			})
			// Naikkan badge belum-dibaca utk pesan masuk live ke chat yg TIDAK
			// sedang dibuka (SaveMessage tak melakukannya; tanpa ini badge cuma
			// muncul setelah resync). Lewati status & pesan sendiri.
			if !m.FromMe && chat != "status@broadcast" && chat != a.currentOpen() {
				_ = store.IncrementUnread(a.ctx, chat)
			}
			a.emit("wa:message", chat)
		})
	})
	// History sync → simpan semua percakapan & riwayat dalam SATU transaksi
	// (ribuan baris → 1 fsync). Off-socket-loop lewat antrian writer.
	eng.OnHistorySync(func(convs []engine.HistoryConversation, pushnames map[string]string, onDemand bool) {
		a.bg(func() {
			// Retensi saat ingest: jangan simpan pesan lebih tua dari cutoff →
			// app.db tak pernah bengkak dari history-sync besar. KECUALI on-demand
			// (user sengaja scroll minta pesan lama) → jangan dipotong retensi.
			cutoff := a.retentionCutoff()
			if onDemand {
				cutoff = 0
			}
			// FASE 1 — METADATA chat dulu (nama/ts/unread), SATU tulis ringan →
			// sidebar + jumlah unread + urutan muncul SEGERA, tak menunggu ribuan
			// baris isi pesan. (Pinned/arsip TIDAK dari sini — itu app-state.)
			meta := make([]storage.HistoryChat, 0, len(convs))
			for _, c := range convs {
				cj := eng.CanonicalJID(c.JID) // satukan @lid↔nomor → 1 chat
				meta = append(meta, storage.HistoryChat{
					JID: cj, Name: c.Name, TS: c.Timestamp, Unread: c.Unread, Pinned: c.Pinned, Archived: c.Archived,
				})
				if c.EndOfHistory { // HP tak punya pesan lebih lama → tandai, stop minta
					_ = store.SetMeta(a.ctx, "histend:"+cj, "1")
				}
			}
			_ = store.SaveHistory(a.ctx, meta, nil, nil)
			a.syncChatSettings()                 // pin/arsip/bisukan dari HP (app-state)
			a.emit("wa:syncprogress", len(meta)) // indikator: jumlah chat
			a.emit("wa:sync", "")                // sidebar tampil cepat (unread!)

			// FASE 2 — ISI PESAN (lambat) streaming berkelompok (~2000/tx). TANPA
			// kirim ulang metadata chat → tak menimpa apa pun.
			const batch = 2000
			bmsgs := make([]storage.Message, 0, batch)
			flush := func() {
				if len(bmsgs) == 0 {
					return
				}
				_ = store.SaveHistory(a.ctx, nil, bmsgs, nil)
				bmsgs = bmsgs[:0]
			}
			for _, c := range convs {
				cj := eng.CanonicalJID(c.JID)
				for _, m := range c.Messages {
					if cutoff > 0 && m.Timestamp.Unix() < cutoff {
						continue // lebih tua dari retensi → lewati
					}
					bmsgs = append(bmsgs, storage.Message{
						ID: m.ID, ChatJID: cj, Sender: m.Sender, PushName: m.PushName,
						Text: m.Text, Kind: m.Kind, Thumb: m.Thumb, Media: m.Media,
						Timestamp: m.Timestamp, FromMe: m.FromMe, Status: m.Status,
					})
					if len(bmsgs) >= batch {
						flush()
					}
				}
			}
			flush()
			// Pushname → nama chat (UPDATE) HARUS setelah semua chat ter-insert.
			if len(pushnames) > 0 {
				_ = store.SaveHistory(a.ctx, nil, nil, pushnames)
			}
			a.dedupChats(eng, store)
			_ = store.RecomputeSummaries(a.ctx)
			a.emit("wa:sync", "")
		})
	})

	// Backlog offline selesai di-replay server → rekonsiliasi & refresh UI dgn data
	// terkini (akhir "get-difference" Telegram: tahu kapan data terbaru siap).
	eng.OnOfflineSyncCompleted(func(count int) {
		a.bg(func() {
			_ = store.RecomputeSummaries(a.ctx)
			a.emit("wa:syncprogress", count)
			a.emit("wa:sync", "")
		})
	})

	eng.OnConnected(func() {
		a.emit("wa:ready", eng.SelfJID())
		// Rekonsiliasi buku alamat tiap reconnect (incremental). fullSync snapshot
		// HANYA sekali per sesi (connect pertama) — reconnect berikutnya cukup
		// patch incremental agar tak banjir IQ/ratelimit saat jaringan flapping.
		full := !a.didFullSync.Swap(true)
		go func() {
			eng.SyncContacts(full)
			_ = store.RecomputeSummaries(a.ctx)
			a.emit("wa:sync", "")
		}()
		// Umumkan online → server mulai kirim presence balik. SubscribePresence
		// HANYA utk online/last-seen; "mengetik" (chatstate) di-push tanpa subscribe.
		// Riset whatsmeow: JANGAN bulk-subscribe ratusan chat (boros IQ/wakeup/
		// baterai, ekosistem warn ratelimit). Cukup ~30 chat 1:1 TERBARU; sisanya
		// di-subscribe saat dibuka (OpenChat) & panel Kontak self-subscribe.
		go func() {
			eng.SendAvailable()
			jids, err := store.ListRecentChatJIDs(a.ctx, 80)
			if err != nil {
				return
			}
			subscribed := 0
			for _, j := range jids {
				if subscribed >= 30 {
					break
				}
				if isGroupJID(j) || strings.HasSuffix(j, "@newsletter") {
					continue // grup/saluran: tak ada online/last-seen utk dilanggan
				}
				eng.SubscribePresence(j)
				subscribed++
				time.Sleep(40 * time.Millisecond) // hindari banjir IQ
			}
		}()
		// Satukan chat ganda @lid↔nomor dari data build lama (lid_map sudah persist).
		a.bg(func() {
			a.dedupChats(eng, store)
			a.syncChatSettings() // tarik pin/arsip/bisukan dari HP (app-state) → app.db
			a.emit("wa:sync", "")
		})
		// Ambil subjek grup yang diikuti → perbarui nama chat grup.
		go func() {
			gs, err := eng.JoinedGroups(a.ctx)
			if err != nil {
				return
			}
			for _, g := range gs {
				_ = store.SetChatName(a.ctx, g.JID, g.Name)
			}
			a.emit("wa:sync", "")
		}()
	})
	// Buku alamat / pushname berubah (app-state sync) → refresh nama di UI.
	eng.OnContactsSynced(func() {
		a.emit("wa:sync", "")
	})
	// Pesan ditarik/dihapus-untuk-semua (oleh siapa pun) → tandai placeholder.
	eng.OnRevoke(func(chat, msgID, sender string) {
		chat = eng.CanonicalJID(chat) // samakan dgn OnMessage (row tersimpan di JID kanonik)
		a.bg(func() {
			if a.keepDeleted.Load() {
				_ = store.MarkDeleted(a.ctx, chat, msgID) // anti-delete: simpan isi
			} else {
				_ = store.MarkDeletedHard(a.ctx, chat, msgID) // WA asli: kosongkan isi
			}
			a.emit("wa:message", chat)
		})
	})
	// Pin/mute/arsip dari perangkat lain (mis. di-pin dari HP) → sinkron ke DB lokal.
	eng.OnChatAction(func(jid, action string, on bool) {
		jid = eng.CanonicalJID(jid)
		a.bg(func() {
			switch action {
			case "pin":
				_ = store.SetPinned(a.ctx, jid, on)
			case "mute":
				_ = store.SetMuted(a.ctx, jid, on)
			case "archive":
				_ = store.SetArchived(a.ctx, jid, on)
			}
			a.emit("wa:sync", "")
		})
	})
	eng.OnLoggedOut(func() { a.emit("wa:loggedout", "") })
	// Sesi direbut proses kembar → beri tahu (jangan reconnect, whatsmeow stop).
	eng.OnStreamReplaced(func() { a.emit("wa:streamreplaced", "") })
	// Blokir sementara → tampilkan alasan + sisa menit (0 = tak diketahui).
	eng.OnTemporaryBan(func(reason string, expire time.Duration) {
		a.emit("wa:tempban", map[string]interface{}{"reason": reason, "mins": int(expire.Minutes())})
	})
	// Pesan gagal-dekripsi → placeholder "menunggu pesan…"; isi asli menyusul
	// (whatsmeow minta kirim ulang + rerequest ke HP). Off-loop lewat antrian.
	eng.OnUndecryptable(func(id, chat, sender string, ts time.Time, fromMe bool) {
		chat = eng.CanonicalJID(chat)
		a.bg(func() {
			if a.store.SavePlaceholder(a.ctx, id, chat, sender, ts, fromMe) == nil {
				a.emit("wa:message", chat)
			}
		})
	})
	// Keepalive gagal beruntun → reconnect lebih cepat dari batas 3 menit
	// whatsmeow (akun ramai bisa drop lebih awal). Single-flight + ambang.
	var kaReconnecting atomic.Bool
	eng.OnKeepAliveTimeout(func(errCount int) {
		if errCount < 3 {
			return // beri kesempatan pulih sendiri dulu
		}
		if !kaReconnecting.CompareAndSwap(false, true) {
			return
		}
		go func() { defer kaReconnecting.Store(false); eng.ForceReconnect() }()
	})

	// Panggilan masuk: whatsmeow hanya signaling (tanpa media) → catat + notif +
	// kirim event ke UI (banner + tombol Tolak). Tak bisa menjawab/bicara.
	eng.OnCall(func(fromJID string, video, group bool, callID string, ts time.Time) {
		jid := eng.CanonicalJID(fromJID)
		name := a.displayName(jid)
		if name == "" {
			name = eng.ReadableID(jid)
		}
		a.bg(func() {
			if a.store != nil {
				_ = a.store.SaveCall(a.ctx, storage.Call{
					ID: callID, JID: jid, Name: name, Video: video, Group: group,
					Status: "missed", Time: ts,
				})
			}
		})
		a.emit("wa:call", map[string]interface{}{
			"id": callID, "jid": jid, "name": name, "video": video, "group": group, "ts": ts.Unix(),
		})
	})

	eng.OnPresence(func(jid string, online bool, ls time.Time) {
		txt := "online"
		if !online {
			if ls.IsZero() {
				txt = ""
			} else {
				txt = "terakhir dilihat " + hm(ls)
			}
		}
		a.setPresence(eng.CanonicalJID(jid), txt) // cache utk UI in-process (Gio)
		a.emit("wa:presence", map[string]string{"jid": jid, "text": txt})
	})
	eng.OnChatPresence(func(chat, sender string, composing, recording bool) {
		chat = eng.CanonicalJID(chat) // samakan dgn id chat di UI (cegah @lid mismatch)
		who := ""
		if composing && isGroupJID(chat) && sender != "" {
			who = a.displayName(sender) // grup → "Budi sedang mengetik…"
		}
		a.setTyping(chat, typingStateT{on: composing, who: who, rec: recording})
		a.emit("wa:typing", map[string]interface{}{"chat": chat, "on": composing, "who": who, "rec": recording})
	})
	eng.OnReceipt(func(chat, sender string, ids []string, status string, ts time.Time) {
		chat = eng.CanonicalJID(chat)
		// Receipt grup = puluhan id × banyak penerima; tulis batch off-loop.
		if a.store != nil && len(ids) > 0 {
			a.bg(func() {
				_ = a.store.SetMessageStatus(a.ctx, chat, ids, status)
				_ = a.store.SetReceipts(a.ctx, chat, ids, sender, status, ts)
			})
		}
		a.emit("wa:receipt", map[string]interface{}{
			"chat": chat, "ids": ids, "status": status,
		})
	})
	// Edit masuk (lawan sunting pesannya) → perbarui teks lokal.
	eng.OnEdit(func(chat, msgID, newText string) {
		if a.store == nil {
			return
		}
		chat = eng.CanonicalJID(chat)
		a.bg(func() {
			_ = a.store.EditText(a.ctx, chat, msgID, newText)
			a.emit("wa:message", chat)
		})
	})
	// Reaksi masuk → simpan & beri tahu UI (reload chat aktif).
	eng.OnReaction(func(chat, targetID, sender, emoji string, fromMe bool) {
		if a.store == nil {
			return
		}
		chat = eng.CanonicalJID(chat)
		a.bg(func() {
			_ = a.store.SetReaction(a.ctx, chat, targetID, sender, emoji, time.Now())
			a.emit("wa:message", chat)
		})
	})
	// Suara polling masuk → cocokkan hash ke opsi, simpan, beri tahu UI.
	eng.OnPollVote(func(chat, pollID, voter string, selected [][]byte) {
		if a.store == nil {
			return
		}
		chat = eng.CanonicalJID(chat)
		a.bg(func() {
			m, err := a.store.GetMessage(a.ctx, chat, pollID)
			if err != nil {
				return
			}
			var opts []string
			_ = json.Unmarshal([]byte(m.Thumb), &opts) // poll: thumb = JSON opsi
			names := eng.MatchPollHashes(opts, selected)
			_ = a.store.SetPollVote(a.ctx, pollID, voter, names, time.Now())
			a.emit("wa:poll", pollID)
		})
	})
	// Pin/unpin dari perangkat atau anggota lain → perbarui banner tersemat.
	eng.OnPinInChat(func(chat, msgID string, pinned bool) {
		chat = eng.CanonicalJID(chat)
		a.bg(func() { // jangan tulis DB di socket-loop → cegah websocket drop
			if a.store != nil {
				_ = a.store.SetPinnedInChat(a.ctx, chat, msgID, pinned)
			}
			a.emit("wa:message", chat)
		})
	})
}

// dedupChats menyatukan baris chat ganda di mana @lid dan nomor menunjuk orang
// yang sama. Iterasi semua jid chat, hitung bentuk kanonik (engine.CanonicalJID);
// bila berbeda → merge baris @lid ke baris nomor. Idempoten; dijalankan setelah
// lid_map terisi (history-sync / connect). Murah: hanya jalan saat ada @lid.
// syncChatSettings menarik pin/arsip/bisukan TERKINI dari store app-state
// whatsmeow (engine.ChatSettings) ke app.db. Perlu karena events.Pin hanya fire
// saat BERUBAH — sematan/bisukan yang SUDAH ADA di HP tak pernah ter-emit, tapi
// tersimpan di tabel ChatSettings whatsmeow → query & terapkan. Emit wa:sync bila
// ada perubahan agar sidebar (urutan pinned) ikut terbarui.
func (a *App) syncChatSettings() {
	if a.eng == nil || a.store == nil {
		return
	}
	chats, err := a.store.ListChats(a.ctx)
	if err != nil {
		return
	}
	changed := false
	for _, c := range chats {
		pinned, archived, muted, ok := a.eng.ChatSettings(c.JID)
		if !ok {
			continue
		}
		if pinned != c.Pinned {
			_ = a.store.SetPinned(a.ctx, c.JID, pinned)
			changed = true
		}
		if muted != c.Muted {
			_ = a.store.SetMuted(a.ctx, c.JID, muted)
			changed = true
		}
		if archived && !c.Archived {
			_ = a.store.SetArchived(a.ctx, c.JID, true)
			changed = true
		}
	}
	if changed {
		a.emit("wa:sync", "")
	}
}

// GetKeepDeleted / SetKeepDeleted — toggle anti-delete (simpan isi pesan yang
// ditarik). ON = isi tetap terlihat; OFF = perilaku WhatsApp asli (kosong).
func (a *App) GetKeepDeleted() bool { return a.keepDeleted.Load() }
func (a *App) SetKeepDeleted(v bool) {
	a.keepDeleted.Store(v)
	if a.store != nil {
		val := "0"
		if v {
			val = "1"
		}
		_ = a.store.SetMeta(a.ctx, "keep_deleted", val)
	}
}

// LoadOlderHistory minta ~50 pesan lebih lama dari yang tertua tersimpan untuk
// chat ini (history on-demand). Respons tiba via OnHistorySync (ON_DEMAND) →
// tersimpan → wa:message → UI reload. Dipanggil FE saat scroll mentok ke atas.
func (a *App) LoadOlderHistory(chatJID string) {
	if a.eng == nil || a.store == nil || chatJID == "" || !a.HasMoreHistory(chatJID) {
		return
	}
	id, fromMe, ts, ok := a.store.OldestMessage(a.ctx, chatJID)
	if !ok {
		return
	}
	_ = a.eng.RequestOlderHistory(chatJID, id, fromMe, ts, 50)
}

// HasMoreHistory melaporkan apakah HP MASIH punya pesan lebih lama utk chat ini
// (false bila EndOfHistoryTransfer sudah diterima → UI berhenti minta). Sumber
// otoritatif: server WhatsApp tak simpan history, jadi flag ini dari HP.
func (a *App) HasMoreHistory(chatJID string) bool {
	if a.store == nil {
		return true
	}
	return a.store.GetMeta(a.ctx, "histend:"+a.canon(chatJID), "") != "1"
}

func (a *App) dedupChats(eng *engine.Engine, store *storage.Store) {
	jids, err := store.ListChatJIDs(a.ctx)
	if err != nil {
		return
	}
	for _, j := range jids {
		canon := eng.CanonicalJID(j)
		if canon != j {
			_ = store.MergeChat(a.ctx, j, canon)
		}
	}
}

// OpenChat dipanggil FE saat chat dibuka: subscribe presence + (grup) subtitle anggota.
// currentOpen — chat yg sedang dibuka (utk lewati saat naikkan unread).
func (a *App) currentOpen() string {
	a.openMu.RLock()
	defer a.openMu.RUnlock()
	return a.openChat
}

// CloseChat — UI meninggalkan percakapan (deselect) → tak ada chat terbuka.
func (a *App) CloseChat() {
	a.openMu.Lock()
	a.openChat = ""
	a.openMu.Unlock()
}

func (a *App) OpenChat(jid string) {
	if a.eng == nil {
		return
	}
	jid = a.canon(jid)
	a.openMu.Lock()
	a.openChat = jid
	a.openMu.Unlock()
	if a.store != nil { // buka chat → bersihkan badge belum-dibaca lokal
		_ = a.store.SetUnread(a.ctx, jid, 0)
	}
	a.eng.SendAvailable()
	a.eng.SubscribePresence(jid)
	if isGroupJID(jid) {
		go func() {
			// Subtitle grup = daftar nama anggota (ala WhatsApp); fallback "N anggota".
			if gi, err := a.eng.GroupInfo(a.ctx, jid); err == nil && gi != nil && len(gi.Participants) > 0 {
				names := make([]string, 0, len(gi.Participants))
				for _, p := range gi.Participants {
					n := p.Name
					if n == "" {
						n = shortJID(p.JID)
					}
					names = append(names, n)
				}
				a.emit("wa:chatinfo", map[string]string{"jid": jid, "subtitle": strings.Join(names, ", ")})
				return
			}
			if s := a.eng.GroupSubtitle(jid); s != "" {
				a.emit("wa:chatinfo", map[string]string{"jid": jid, "subtitle": s})
			}
		}()
	}
}

// domReady dipanggil setelah halaman web (WebKit/JSC) selesai dimuat — di sinilah
// WebKit dipastikan sudah memasang handler-nya, jadi kita pulihkan SA_ONSTACK lagi.
func (a *App) DomReady(ctx context.Context) {
	runtime.ResetSignalHandlers()
}

func (a *App) Shutdown(ctx context.Context) {
	if a.eng != nil {
		a.eng.Stop()
	}
	if a.store != nil {
		a.store.Close()
	}
}

// --- helper bersama ---

func hm(t time.Time) string {
	if t.IsZero() || t.Unix() == 0 {
		return ""
	}
	return t.Format("15.04")
}

// relTime = waktu ringkas utk daftar chat: hari ini→jam, kemarin→"Kemarin",
// dalam seminggu→nama hari, lebih lawas→tanggal. (ala WhatsApp)
func relTime(t time.Time) string {
	if t.IsZero() || t.Unix() == 0 {
		return ""
	}
	now := time.Now()
	yA, mA, dA := now.Date()
	yB, mB, dB := t.Date()
	today := time.Date(yA, mA, dA, 0, 0, 0, 0, now.Location())
	msgDay := time.Date(yB, mB, dB, 0, 0, 0, 0, t.Location())
	days := int(today.Sub(msgDay).Hours() / 24)
	switch {
	case days <= 0:
		return t.Format("15.04")
	case days == 1:
		return "Kemarin"
	case days < 7:
		return []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}[int(msgDay.Weekday())]
	default:
		return t.Format("02/01/06")
	}
}

func isGroupJID(jid string) bool { return strings.HasSuffix(jid, "@g.us") }

func shortJID(jid string) string {
	if i := strings.IndexByte(jid, '@'); i > 0 {
		return jid[:i]
	}
	return jid
}
