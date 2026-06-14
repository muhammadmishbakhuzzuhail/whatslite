package app

// app_status.go — Status / Stories. Pesan status tiba via OnMessage dgn
// chat=status@broadcast (difilter dari daftar chat) lalu tersimpan biasa.
// Di sini dikelompokkan per pengirim (24 jam terakhir) untuk panel Status.

import (
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/muhammadmishbakhuzzuhail/whatsapp-lite/internal/storage"
)

// StatusItemDTO = satu unggahan status (teks / media).
type StatusItemDTO struct {
	ID    string `json:"id"`
	Type  string `json:"type"` // text | image | video | sticker
	Text  string `json:"text"`
	Thumb string `json:"thumb"` // data-URI pratinjau (image/video/sticker)
	Time  string `json:"time"`
	Ts    int64  `json:"ts"`
}

// StatusGroupDTO = semua status satu kontak (atau milik sendiri).
type StatusGroupDTO struct {
	Jid   string          `json:"jid"`
	Name  string          `json:"name"`
	Time  string          `json:"time"` // waktu update terbaru
	Mine  bool            `json:"mine"`
	Count int             `json:"count"`
	Items []StatusItemDTO `json:"items"` // urut lama→baru (utk tap-through viewer)
}

// GetStatuses mengembalikan status 24 jam terakhir, dikelompokkan per pengirim.
// Urutan: milik sendiri dulu, lalu kontak lain berdasar update terbaru.
func (a *App) GetStatuses() (out []StatusGroupDTO) {
	out = []StatusGroupDTO{}
	if a.store == nil {
		return
	}
	since := time.Now().Add(-24 * time.Hour)
	ms, err := a.store.ListStatuses(a.ctx, since) // terbaru dulu
	if err != nil {
		return
	}
	self := ""
	if a.eng != nil {
		self = userPart(a.eng.SelfJID())
	}
	groups := map[string]*StatusGroupDTO{}
	var order []string // urut kemunculan = update terbaru dulu (ms terbaru dulu)
	for _, m := range ms {
		key := m.Sender
		g := groups[key]
		if g == nil {
			mine := self != "" && userPart(m.Sender) == self
			name := ""
			if mine {
				name = "Status saya"
			} else {
				if a.eng != nil {
					name = a.eng.ChatName(m.Sender)
				}
				if name == "" && m.PushName != "" {
					name = m.PushName
				}
				if name == "" && a.eng != nil {
					name = a.eng.ReadableID(m.Sender)
				}
				if name == "" {
					name = shortJID(m.Sender)
				}
			}
			g = &StatusGroupDTO{Jid: m.Sender, Name: name, Time: relTime(m.Timestamp), Mine: mine}
			groups[key] = g
			order = append(order, key)
		}
		g.Items = append(g.Items, StatusItemDTO{
			ID: m.ID, Type: m.Kind, Text: m.Text, Thumb: m.Thumb,
			Time: hm(m.Timestamp), Ts: m.Timestamp.Unix(),
		})
	}
	// Items per grup saat ini baru→lama; viewer ingin lama→baru → balik.
	for _, k := range order {
		g := groups[k]
		for i, j := 0, len(g.Items)-1; i < j; i, j = i+1, j-1 {
			g.Items[i], g.Items[j] = g.Items[j], g.Items[i]
		}
		g.Count = len(g.Items)
	}
	// Milik sendiri ke depan.
	var mine, others []StatusGroupDTO
	for _, k := range order {
		g := groups[k]
		if g.Mine {
			mine = append(mine, *g)
		} else {
			others = append(others, *g)
		}
	}
	out = append(out, mine...)
	out = append(out, others...)
	return out
}

// GetStatusViewers mengembalikan siapa saja yang sudah melihat status kita.
// Data dari tanda terima (receipt) di status@broadcast — terisi live sejak app
// jalan; kosong bila belum ada yang melihat / belum tersinkron.
func (a *App) GetStatusViewers(statusID string) []ReceiptDTO {
	out := []ReceiptDTO{}
	if a.store == nil {
		return out
	}
	rs, err := a.store.ListReceipts(a.ctx, "status@broadcast", statusID)
	if err != nil {
		return out
	}
	for _, r := range rs {
		name := ""
		if a.eng != nil {
			name = a.eng.ChatName(r.Recipient)
		}
		if name == "" {
			name = shortJID(r.Recipient)
		}
		out = append(out, ReceiptDTO{Name: name, Time: r.Timestamp.Format("2 Jan, 15:04")})
	}
	return out
}

// PostTextStatus mengunggah status teks (bg ARGB + font opsional). Best-effort.
func (a *App) PostTextStatus(text string, bgArgb int64, font int) string {
	if a.eng == nil || strings.TrimSpace(text) == "" {
		return ""
	}
	id, err := a.eng.PostTextStatus(a.ctx, text, uint32(bgArgb), uint32(font))
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	return id
}

// PostMediaStatus mengunggah status media (gambar/video) dari data-URI.
func (a *App) PostMediaStatus(kind, caption, dataURI string) string {
	if a.eng == nil {
		return ""
	}
	mime, data, err := decodeDataURI(dataURI)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	id, err := a.eng.PostMediaStatus(a.ctx, kind, mime, caption, data)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	return id
}

// ReactStatus memberi reaksi emoji pada status seseorang.
func (a *App) ReactStatus(posterJid, statusID, emoji string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.React(a.ctx, "status@broadcast", a.canon(posterJid), statusID, emoji, false); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
	}
}

// ReplyStatus membalas status (kirim ke chat 1:1 pemilik, mengutip status itu).
func (a *App) ReplyStatus(posterJid, statusID, statusText, text string) {
	if a.eng == nil || strings.TrimSpace(text) == "" {
		return
	}
	to := a.canon(posterJid)
	id, err := a.eng.ReplyToStatus(a.ctx, to, text, statusID, a.canon(posterJid), statusText)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	if a.store != nil {
		_ = a.store.SaveMessage(a.ctx, storage.Message{
			ID: id, ChatJID: to, Text: text, Timestamp: time.Now(), FromMe: true,
			QuotedID: statusID, QuotedText: statusText,
		})
	}
	runtime.EventsEmit(a.ctx, "wa:message", to)
}

// userPart mengambil bagian pengguna JID (sebelum ':' device & '@' server).
func userPart(jid string) string {
	if i := strings.IndexByte(jid, '@'); i >= 0 {
		jid = jid[:i]
	}
	if i := strings.IndexByte(jid, ':'); i >= 0 {
		jid = jid[:i]
	}
	return jid
}
