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
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"whatsapp-lite/internal/engine"
	"whatsapp-lite/internal/storage"
)

// App = konteks aplikasi Wails. Method publik di sini otomatis ter-bind ke JS
// sebagai window.go.main.App.<Method>. Event ke UI lewat runtime.EventsEmit.
type App struct {
	ctx      context.Context
	eng      *engine.Engine
	store    *storage.Store
	mediaDir string // cache file media (bukan di DB) → ringan

	labelsMu sync.RWMutex      // melindungi labels
	labels   map[string]string // label kontak lokal (jid → nama), cache dari DB

	retentionDays int // 0 = simpan selamanya; >0 = prune pesan lebih tua (kec. berbintang/disematkan)

	wq chan func() // antrian tulis-DB serial (off the whatsmeow socket loop)
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
	eng, err := engine.New(ctx, filepath.Join(dataDir, "whatsapp-lite.db"), os.Getenv("WALITE_DEBUG") != "")
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

	// Retensi pesan: default 90 hari. Prune + VACUUM sekali saat boot (off-loop)
	// → app.db tetap ramping walau riwayat besar. Berbintang/disematkan aman.
	// Proxy (opsional): terapkan SEBELUM Connect (dipicu FE setelah ini).
	if px := store.GetMeta(ctx, "proxy", ""); px != "" {
		_ = eng.SetProxy(px)
	}

	a.retentionDays = atoiDef(store.GetMeta(ctx, "retention_days", "90"), 90)
	a.bg(func() {
		if cut := a.retentionCutoff(); cut > 0 {
			if n, _ := a.store.PruneMessages(a.ctx, cut); n > 0 {
				_ = a.store.Vacuum(a.ctx)
				runtime.EventsEmit(a.ctx, "wa:sync", "")
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
				runtime.EventsEmit(a.ctx, "wa:sync", "")
			}
		}
		sweep()
		t := time.NewTicker(60 * time.Second)
		defer t.Stop()
		for range t.C {
			sweep()
		}
	}()

	a.startScheduler() // pesan terjadwal + pengingat (ticker sendiri)

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
			runtime.EventsEmit(a.ctx, "wa:message", chat)
		})
	})
	// History sync → simpan semua percakapan & riwayat dalam SATU transaksi
	// (ribuan baris → 1 fsync). Off-socket-loop lewat antrian writer.
	eng.OnHistorySync(func(convs []engine.HistoryConversation, pushnames map[string]string) {
		a.bg(func() {
			// Retensi saat ingest: jangan simpan pesan lebih tua dari cutoff →
			// app.db tak pernah bengkak dari history-sync besar.
			cutoff := a.retentionCutoff()
			// Tulis berkelompok (~2000 pesan/tx) — bukan satu transaksi raksasa →
			// puncak memori + pertumbuhan WAL terbatas saat blob history besar.
			const batch = 2000
			bchats := make([]storage.HistoryChat, 0, 64)
			bmsgs := make([]storage.Message, 0, batch)
			flush := func() {
				if len(bchats) == 0 && len(bmsgs) == 0 {
					return
				}
				_ = store.SaveHistory(a.ctx, bchats, bmsgs, nil)
				bchats = bchats[:0]
				bmsgs = bmsgs[:0]
			}
			for _, c := range convs {
				cj := eng.CanonicalJID(c.JID) // satukan @lid↔nomor → 1 chat
				bchats = append(bchats, storage.HistoryChat{
					JID: cj, Name: c.Name, TS: c.Timestamp, Unread: c.Unread, Pinned: c.Pinned, Archived: c.Archived,
				})
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
			runtime.EventsEmit(a.ctx, "wa:sync", "")
		})
	})

	eng.OnConnected(func() {
		runtime.EventsEmit(a.ctx, "wa:ready", eng.SelfJID())
		// Tarik buku alamat (nama tersimpan) — tanpa ini nama tampil nomor.
		go eng.SyncContacts()
		// Umumkan online + langganan presence semua chat → indikator "mengetik"
		// muncul di sidebar (bukan hanya chat yg terbuka). Throttle ringan.
		go func() {
			eng.SendAvailable()
			jids, err := store.ListChatJIDs(a.ctx)
			if err != nil {
				return
			}
			for _, j := range jids {
				if isGroupJID(j) || strings.HasSuffix(j, "@newsletter") {
					continue // grup kirim presence tanpa subscribe; saluran tak relevan
				}
				eng.SubscribePresence(j)
				time.Sleep(40 * time.Millisecond) // hindari banjir IQ
			}
		}()
		// Satukan chat ganda @lid↔nomor dari data build lama (lid_map sudah persist).
		a.bg(func() {
			a.dedupChats(eng, store)
			runtime.EventsEmit(a.ctx, "wa:sync", "")
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
			runtime.EventsEmit(a.ctx, "wa:sync", "")
		}()
	})
	// Buku alamat / pushname berubah (app-state sync) → refresh nama di UI.
	eng.OnContactsSynced(func() {
		runtime.EventsEmit(a.ctx, "wa:sync", "")
	})
	// Pesan ditarik/dihapus-untuk-semua (oleh siapa pun) → tandai placeholder.
	eng.OnRevoke(func(chat, msgID, sender string) {
		chat = eng.CanonicalJID(chat) // samakan dgn OnMessage (row tersimpan di JID kanonik)
		a.bg(func() {
			_ = store.MarkDeleted(a.ctx, chat, msgID)
			runtime.EventsEmit(a.ctx, "wa:message", chat)
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
			runtime.EventsEmit(a.ctx, "wa:sync", "")
		})
	})
	eng.OnLoggedOut(func() { runtime.EventsEmit(a.ctx, "wa:loggedout", "") })

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
		a.Notify(name, "Panggilan masuk")
		runtime.EventsEmit(a.ctx, "wa:call", map[string]interface{}{
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
		runtime.EventsEmit(a.ctx, "wa:presence", map[string]string{"jid": jid, "text": txt})
	})
	eng.OnChatPresence(func(chat, sender string, composing bool) {
		chat = eng.CanonicalJID(chat) // samakan dgn id chat di UI (cegah @lid mismatch)
		who := ""
		if composing && isGroupJID(chat) && sender != "" {
			who = a.displayName(sender) // grup → "Budi sedang mengetik…"
		}
		runtime.EventsEmit(a.ctx, "wa:typing", map[string]interface{}{"chat": chat, "on": composing, "who": who})
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
		runtime.EventsEmit(a.ctx, "wa:receipt", map[string]interface{}{
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
			runtime.EventsEmit(a.ctx, "wa:message", chat)
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
			runtime.EventsEmit(a.ctx, "wa:message", chat)
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
			runtime.EventsEmit(a.ctx, "wa:poll", pollID)
		})
	})
	// Pin/unpin dari perangkat atau anggota lain → perbarui banner tersemat.
	eng.OnPinInChat(func(chat, msgID string, pinned bool) {
		chat = eng.CanonicalJID(chat)
		a.bg(func() { // jangan tulis DB di socket-loop → cegah websocket drop
			if a.store != nil {
				_ = a.store.SetPinnedInChat(a.ctx, chat, msgID, pinned)
			}
			runtime.EventsEmit(a.ctx, "wa:message", chat)
		})
	})
}

// dedupChats menyatukan baris chat ganda di mana @lid dan nomor menunjuk orang
// yang sama. Iterasi semua jid chat, hitung bentuk kanonik (engine.CanonicalJID);
// bila berbeda → merge baris @lid ke baris nomor. Idempoten; dijalankan setelah
// lid_map terisi (history-sync / connect). Murah: hanya jalan saat ada @lid.
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
func (a *App) OpenChat(jid string) {
	if a.eng == nil {
		return
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
				runtime.EventsEmit(a.ctx, "wa:chatinfo", map[string]string{"jid": jid, "subtitle": strings.Join(names, ", ")})
				return
			}
			if s := a.eng.GroupSubtitle(jid); s != "" {
				runtime.EventsEmit(a.ctx, "wa:chatinfo", map[string]string{"jid": jid, "subtitle": s})
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
