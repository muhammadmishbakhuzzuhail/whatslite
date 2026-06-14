package engine

// presence.go — event koneksi/logout + presence (online/last seen), indikator
// mengetik, dan tanda terima (receipt) untuk centang baca.

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// OnConnected mendaftarkan callback saat terhubung & terautentikasi.
func (e *Engine) OnConnected(fn func()) {
	e.Client.AddEventHandler(func(evt interface{}) {
		if _, ok := evt.(*events.Connected); ok {
			fn()
		}
	})
}

// OnLoggedOut mendaftarkan callback saat akun ter-logout (mis. dari perangkat lain / 401).
func (e *Engine) OnLoggedOut(fn func()) {
	e.Client.AddEventHandler(func(evt interface{}) {
		if _, ok := evt.(*events.LoggedOut); ok {
			fn()
		}
	})
}

// OnStreamReplaced: sesi diambil alih klien lain dgn kunci sama (mis. proses
// kembar). whatsmeow berhenti — JANGAN reconnect (akan saling rebut). Beritahu
// pengguna saja.
func (e *Engine) OnStreamReplaced(fn func()) {
	e.Client.AddEventHandler(func(evt interface{}) {
		if _, ok := evt.(*events.StreamReplaced); ok {
			fn()
		}
	})
}

// OnTemporaryBan: blokir sementara (ConnectFailure TempBanned). Mundur jangan
// reconnect-loop. Kembalikan alasan + sisa durasi (0 bila tak diketahui).
func (e *Engine) OnTemporaryBan(fn func(reason string, expire time.Duration)) {
	e.Client.AddEventHandler(func(evt interface{}) {
		if tb, ok := evt.(*events.TemporaryBan); ok {
			fn(tb.Code.String(), tb.Expire)
		}
	})
}

// SendAvailable umumkan status online kita (perlu agar bisa terima presence balik).
func (e *Engine) SendAvailable() {
	_ = e.Client.SendPresence(context.Background(), types.PresenceAvailable)
}

// SubscribePresence berlangganan presence (online/last seen) satu kontak.
func (e *Engine) SubscribePresence(jid string) {
	if j, err := types.ParseJID(jid); err == nil {
		_ = e.Client.SubscribePresence(context.Background(), j)
	}
}

// SendTyping mengirim indikator "sedang mengetik" (composing) / berhenti (paused).
// recording=true → media audio → lawan bicara lihat "merekam suara…".
func (e *Engine) SendTyping(jid string, composing, recording bool) {
	j, err := types.ParseJID(jid)
	if err != nil {
		return
	}
	state := types.ChatPresencePaused
	media := types.ChatPresenceMediaText
	if composing {
		state = types.ChatPresenceComposing
		if recording {
			media = types.ChatPresenceMediaAudio
		}
	}
	_ = e.Client.SendChatPresence(context.Background(), j, state, media)
}

// OnPresence: kontak online / terakhir dilihat.
func (e *Engine) OnPresence(fn func(jid string, online bool, lastSeen time.Time)) {
	e.Client.AddEventHandler(func(evt interface{}) {
		if p, ok := evt.(*events.Presence); ok {
			fn(p.From.String(), !p.Unavailable, p.LastSeen)
		}
	})
}

// OnChatPresence: lawan bicara sedang mengetik (composing) / berhenti.
// sender = pengirim presence (utk grup → tampilkan siapa yg mengetik).
// recording = sedang merekam suara (media="audio") → "merekam suara…".
func (e *Engine) OnChatPresence(fn func(chat, sender string, composing, recording bool)) {
	e.Client.AddEventHandler(func(evt interface{}) {
		if cp, ok := evt.(*events.ChatPresence); ok {
			composing := cp.State == types.ChatPresenceComposing
			recording := composing && cp.Media == types.ChatPresenceMediaAudio
			fn(cp.MessageSource.Chat.String(), cp.MessageSource.Sender.String(), composing, recording)
		}
	})
}

// OnReceipt: pesan kita dibaca/terkirim (utk centang). status: "delivered"|"read".
// sender = penerima yang mengirim tanda terima (di grup: per anggota). ids =
// pesan terdampak. ts = waktu tanda terima.
func (e *Engine) OnReceipt(fn func(chat, sender string, ids []string, status string, ts time.Time)) {
	e.Client.AddEventHandler(func(evt interface{}) {
		r, ok := evt.(*events.Receipt)
		if !ok {
			return
		}
		status := ""
		switch r.Type {
		case types.ReceiptTypeRead, types.ReceiptTypeReadSelf, types.ReceiptTypePlayed:
			status = "read"
		case types.ReceiptTypeDelivered:
			status = "delivered"
		default:
			return
		}
		ids := make([]string, 0, len(r.MessageIDs))
		for _, id := range r.MessageIDs {
			ids = append(ids, string(id))
		}
		fn(r.MessageSource.Chat.String(), r.MessageSource.Sender.String(), ids, status, r.Timestamp)
	})
}
