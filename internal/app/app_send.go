package app

// app_send.go — aksi pesan: kirim media, balas (reply), teruskan (forward),
// reaksi, hapus, bintang, tandai dibaca, indikator mengetik.

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"whatsapp-lite/internal/storage"
)

// SendMedia mengirim media (data-URI) lalu menyimpannya lokal & memberi tahu UI.
// kind: "image" | "video" | "voice" | "document". viewOnce → sekali lihat.
func (a *App) SendMedia(jid, kind, caption, fileName, dataURI string, viewOnce bool) string {
	if a.eng == nil {
		return ""
	}
	mime, data, err := decodeDataURI(dataURI)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", "media tak valid: "+err.Error())
		return ""
	}
	id, err := a.eng.SendMedia(a.ctx, jid, kind, mime, caption, fileName, data, viewOnce)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	// Simpan lokal: thumb = data-URI agar UI langsung menampilkan kiriman.
	_ = a.store.SaveMessage(a.ctx, storage.Message{
		ID: id, ChatJID: jid, Text: caption, Kind: kind, Thumb: dataURI,
		Timestamp: time.Now(), FromMe: true,
	})
	runtime.EventsEmit(a.ctx, "wa:message", jid)
	return id
}

// SendTextMentioned mengirim teks dgn @mention (daftar JID di-notif).
func (a *App) SendTextMentioned(jid, text string, mentions []string) string {
	if a.eng == nil {
		return ""
	}
	id, err := a.eng.SendTextMentions(a.ctx, jid, text, mentions)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	_ = a.store.SaveMessage(a.ctx, storage.Message{
		ID: id, ChatJID: jid, Text: text, Timestamp: time.Now(), FromMe: true,
	})
	runtime.EventsEmit(a.ctx, "wa:message", jid)
	return id
}

// Reply mengirim balasan yang mengutip pesan lain.
func (a *App) Reply(jid, text, quotedID, quotedSender, quotedText string) string {
	if a.eng == nil {
		return ""
	}
	id, err := a.eng.Reply(a.ctx, jid, text, quotedID, quotedSender, quotedText)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	_ = a.store.SaveMessage(a.ctx, storage.Message{
		ID: id, ChatJID: jid, Text: text, Timestamp: time.Now(), FromMe: true,
	})
	runtime.EventsEmit(a.ctx, "wa:message", jid)
	return id
}

// Forward meneruskan satu pesan (dari srcChat/msgID) ke chat tujuan toJID.
func (a *App) Forward(srcChat, msgID, toJID string) string {
	if a.eng == nil || a.store == nil {
		return ""
	}
	m, err := a.store.GetMessage(a.ctx, srcChat, msgID)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	var id string
	if m.Media != "" {
		id, err = a.eng.ForwardMedia(a.ctx, toJID, m.Media)
	} else {
		id, err = a.eng.ForwardText(a.ctx, toJID, m.Text)
	}
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	_ = a.store.SaveMessage(a.ctx, storage.Message{
		ID: id, ChatJID: toJID, Text: m.Text, Kind: m.Kind, Thumb: m.Thumb,
		Timestamp: time.Now(), FromMe: true,
	})
	runtime.EventsEmit(a.ctx, "wa:message", toJID)
	return id
}

// EditMessage menyunting teks pesan terkirim (pesan sendiri).
func (a *App) EditMessage(chat, msgID, newText string) {
	if a.eng == nil || a.store == nil {
		return
	}
	if err := a.eng.SendEdit(a.ctx, chat, msgID, newText); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	_ = a.store.EditText(a.ctx, chat, msgID, newText)
	runtime.EventsEmit(a.ctx, "wa:message", chat)
}

// React menambah/menghapus reaksi emoji pada pesan (emoji "" = hapus).
func (a *App) React(chat, msgID, sender, emoji string, fromMe bool) {
	if a.eng == nil {
		return
	}
	if err := a.eng.React(a.ctx, chat, sender, msgID, emoji, fromMe); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
	}
}

// DeleteMessage menghapus pesan. everyone=true → revoke di server + tandai
// "dihapus" (placeholder tetap, ala WhatsApp). everyone=false → hapus-untuk-saya
// (hilang dari lokal).
func (a *App) DeleteMessage(chat, msgID, sender string, fromMe, everyone bool) {
	if a.eng == nil || a.store == nil {
		return
	}
	if everyone {
		if err := a.eng.Revoke(a.ctx, chat, sender, msgID, fromMe); err != nil {
			runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		}
		_ = a.store.MarkDeleted(a.ctx, chat, msgID)
	} else {
		_ = a.store.DeleteLocalMessage(a.ctx, chat, msgID)
	}
	runtime.EventsEmit(a.ctx, "wa:message", chat)
}

// StarMessage membintangi / melepas bintang pesan.
func (a *App) StarMessage(chat, msgID, sender string, fromMe, star bool) {
	if a.eng == nil {
		return
	}
	if a.store != nil {
		_ = a.store.SetStarred(a.ctx, chat, msgID, star)
	}
	if err := a.eng.Star(a.ctx, chat, sender, msgID, fromMe, star); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
	}
}

// GetStarred mengembalikan pesan berbintang (lintas chat) untuk panel "Berbintang".
func (a *App) GetStarred() (out []SearchHitDTO) {
	out = []SearchHitDTO{}
	if a.store == nil {
		return
	}
	ms, err := a.store.ListStarred(a.ctx, 200)
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
		out = append(out, SearchHitDTO{
			ChatJID: m.ChatJID, ChatName: name, MsgID: m.ID,
			Text: a.resolveMentions(m.Text), Time: hm(m.Timestamp), Group: isGroupJID(m.ChatJID),
		})
	}
	return out
}

// PinMessage menyemat / melepas pesan di dalam chat (untuk semua).
func (a *App) PinMessage(chat, msgID, sender string, fromMe, pin bool) {
	if a.eng == nil || a.store == nil {
		return
	}
	if err := a.eng.PinMessage(a.ctx, chat, sender, msgID, fromMe, pin); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	_ = a.store.SetPinnedInChat(a.ctx, chat, msgID, pin)
	runtime.EventsEmit(a.ctx, "wa:message", chat)
}

// GetPinned mengembalikan pesan yang disematkan di chat (terbaru dulu).
func (a *App) GetPinned(chat string) []MessageDTO {
	if a.store == nil {
		return []MessageDTO{}
	}
	ms, err := a.store.ListPinned(a.ctx, chat)
	if err != nil {
		return []MessageDTO{}
	}
	return a.toDTO(ms)
}

// ReceiptDTO = satu baris tanda terima (penerima + waktu) di modal Info.
type ReceiptDTO struct {
	Name string `json:"name"`
	Time string `json:"time"`
}

// MsgInfoDTO = detail satu pesan (modal "Info").
type MsgInfoDTO struct {
	ID          string       `json:"id"`
	Type        string       `json:"type"`
	Status      string       `json:"status"` // sent | delivered | read
	FromMe      bool         `json:"fromMe"`
	Sent        string       `json:"sent"`   // waktu kirim lengkap
	Sender      string       `json:"sender"` // nama pengirim (grup)
	SenderID    string       `json:"senderId"`
	ReadBy      []ReceiptDTO `json:"readBy"`      // penerima yang sudah baca
	DeliveredTo []ReceiptDTO `json:"deliveredTo"` // tersampaikan, belum baca
}

// GetMessageInfo mengembalikan detail satu pesan untuk modal Info.
func (a *App) GetMessageInfo(chat, msgID string) *MsgInfoDTO {
	if a.store == nil {
		return nil
	}
	m, err := a.store.GetMessage(a.ctx, chat, msgID)
	if err != nil {
		return nil
	}
	status := ""
	if m.FromMe {
		status = m.Status
		if status == "" {
			status = "sent"
		}
	}
	name := m.PushName
	if name == "" && m.Sender != "" && a.eng != nil {
		if n := a.eng.ChatName(m.Sender); n != "" {
			name = n
		} else {
			name = shortJID(m.Sender)
		}
	}
	kind := m.Kind
	if kind == "" {
		kind = "text"
	}
	info := &MsgInfoDTO{
		ID: m.ID, Type: kind, Status: status, FromMe: m.FromMe,
		Sent: m.Timestamp.Format("2 Jan 2006, 15:04"), Sender: name, SenderID: m.Sender,
		ReadBy: []ReceiptDTO{}, DeliveredTo: []ReceiptDTO{},
	}
	if m.FromMe {
		if rs, err := a.store.ListReceipts(a.ctx, chat, msgID); err == nil {
			for _, r := range rs {
				rn := ""
				if a.eng != nil {
					rn = a.eng.ChatName(r.Recipient)
				}
				if rn == "" {
					rn = shortJID(r.Recipient)
				}
				row := ReceiptDTO{Name: rn, Time: r.Timestamp.Format("2 Jan, 15:04")}
				if r.Status == "read" {
					info.ReadBy = append(info.ReadBy, row)
				} else {
					info.DeliveredTo = append(info.DeliveredTo, row)
				}
			}
		}
	}
	return info
}

// SendSticker mengirim stiker (webp data-URI).
func (a *App) SendSticker(jid, dataURI string) string {
	if a.eng == nil {
		return ""
	}
	_, data, err := decodeDataURI(dataURI)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	id, err := a.eng.SendSticker(a.ctx, jid, data)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	_ = a.store.SaveMessage(a.ctx, storage.Message{
		ID: id, ChatJID: jid, Kind: "sticker", Thumb: dataURI, Timestamp: time.Now(), FromMe: true,
	})
	runtime.EventsEmit(a.ctx, "wa:message", jid)
	return id
}

// SendGif mengirim GIF (mp4 data-URI) sebagai video GifPlayback.
func (a *App) SendGif(jid, dataURI string) string {
	if a.eng == nil {
		return ""
	}
	mime, data, err := decodeDataURI(dataURI)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	id, err := a.eng.SendGif(a.ctx, jid, mime, data)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	_ = a.store.SaveMessage(a.ctx, storage.Message{
		ID: id, ChatJID: jid, Kind: "video", Thumb: dataURI, Timestamp: time.Now(), FromMe: true,
	})
	runtime.EventsEmit(a.ctx, "wa:message", jid)
	return id
}

// SendContact mengirim kartu kontak (membangun vCard dari nama + nomor).
func (a *App) SendContact(jid, displayName, phone string) string {
	if a.eng == nil {
		return ""
	}
	num := ""
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			num += string(r)
		}
	}
	vcard := "BEGIN:VCARD\nVERSION:3.0\nFN:" + displayName +
		"\nTEL;type=CELL;type=VOICE;waid=" + num + ":+" + num + "\nEND:VCARD"
	id, err := a.eng.SendContact(a.ctx, jid, displayName, vcard)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	_ = a.store.SaveMessage(a.ctx, storage.Message{
		ID: id, ChatJID: jid, Text: "👤 " + displayName, Timestamp: time.Now(), FromMe: true,
	})
	runtime.EventsEmit(a.ctx, "wa:message", jid)
	return id
}

// SendLocation mengirim lokasi (lat/lng + nama opsional).
func (a *App) SendLocation(jid string, lat, lng float64, name string) string {
	if a.eng == nil {
		return ""
	}
	id, err := a.eng.SendLocation(a.ctx, jid, lat, lng, name)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	_ = a.store.SaveMessage(a.ctx, storage.Message{
		ID: id, ChatJID: jid, Text: "📍 " + nonEmpty(name, "Lokasi"), Timestamp: time.Now(), FromMe: true,
	})
	runtime.EventsEmit(a.ctx, "wa:message", jid)
	return id
}

// SendPoll mengirim polling (pertanyaan + opsi).
func (a *App) SendPoll(jid, question string, options []string, selectable int) string {
	if a.eng == nil {
		return ""
	}
	id, err := a.eng.SendPoll(a.ctx, jid, question, options, selectable)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	_ = a.store.SaveMessage(a.ctx, storage.Message{
		ID: id, ChatJID: jid, Text: "📊 " + question, Timestamp: time.Now(), FromMe: true,
	})
	runtime.EventsEmit(a.ctx, "wa:message", jid)
	return id
}

// VotePoll mengirim suara untuk polling masuk.
func (a *App) VotePoll(chat, pollSender, pollID string, options []string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.VotePoll(a.ctx, chat, pollSender, pollID, options); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
	}
}

// SetDisappearing mengatur timer pesan sementara untuk chat (detik; 0 = mati).
func (a *App) SetDisappearing(jid string, seconds int) {
	if a.eng == nil {
		return
	}
	if err := a.eng.SetDisappearing(a.ctx, jid, seconds); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
	}
}

// MarkRead menandai chat dibaca (read-receipt + bersihkan unread lokal).
func (a *App) MarkRead(chat, sender, msgID string) {
	if a.eng == nil || a.store == nil {
		return
	}
	if err := a.eng.MarkRead(a.ctx, chat, sender, msgID); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
	}
	_ = a.store.SetUnread(a.ctx, chat, 0)
}

// SendTyping mengirim indikator mengetik (composing) / berhenti (paused).
func (a *App) SendTyping(jid string, composing bool) {
	if a.eng != nil {
		a.eng.SendTyping(jid, composing)
	}
}

// nonEmpty mengembalikan s bila tak kosong, selainnya def.
func nonEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// decodeDataURI memecah "data:<mime>;base64,<payload>" → (mime, bytes).
func decodeDataURI(uri string) (string, []byte, error) {
	mime := "application/octet-stream"
	payload := uri
	if strings.HasPrefix(uri, "data:") {
		if i := strings.IndexByte(uri, ','); i >= 0 {
			header := uri[5:i] // mis. "image/jpeg;base64"
			payload = uri[i+1:]
			if semi := strings.IndexByte(header, ';'); semi >= 0 {
				mime = header[:semi]
			} else if header != "" {
				mime = header
			}
		}
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	return mime, data, err
}
