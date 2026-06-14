package engine

// messages.go — pesan masuk (live + history sync), pesan keluar, dan klasifikasi
// konten pesan whatsmeow menjadi bentuk sederhana untuk frontend.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// webStatus memetakan status WebMessageInfo (history sync) → status centang kita.
func webStatus(s waWeb.WebMessageInfo_Status) string {
	switch s {
	case waWeb.WebMessageInfo_READ, waWeb.WebMessageInfo_PLAYED:
		return "read"
	case waWeb.WebMessageInfo_DELIVERY_ACK:
		return "delivered"
	}
	return ""
}

// IncomingMessage adalah pesan masuk yang sudah disederhanakan untuk frontend.
type IncomingMessage struct {
	ID        string
	Chat      string
	Sender    string
	PushName  string
	Text      string // caption (media) atau isi teks; label utk tipe non-render
	Kind      string // text | image | video | sticker  ("" = junk, lewati)
	Thumb     string // data-URI thumbnail (image/video/sticker), bila ada
	Media     string // base64 proto pesan (utk download media penuh on-demand)
	Timestamp time.Time
	FromMe    bool

	// Balasan (reply/quote) — kosong bila bukan balasan.
	QuotedID     string
	QuotedSender string
	QuotedText   string

	// Status centang (hanya bermakna utk pesan sendiri): "delivered" | "read".
	// Diisi dari history sync (WebMessageInfo.Status); "" → default 'sent'.
	Status string

	// ExpireSecs = TTL disappearing-message (detik) dari ContextInfo; 0 = tetap.
	ExpireSecs uint32
}

// OnMessage mendaftarkan callback untuk pesan masuk.
func (e *Engine) OnMessage(fn func(IncomingMessage)) {
	e.Client.AddEventHandler(func(evt interface{}) {
		m, ok := evt.(*events.Message)
		if !ok {
			return
		}
		kind, txt, thumb, media := describeMessage(m.Message)
		if kind == "" {
			return // lewati pesan protokol/reaksi/kosong (jangan jadi bubble kosong)
		}
		qid, qsender, qtext := extractQuote(m.Message)
		fn(IncomingMessage{
			ID:           m.Info.ID,
			Chat:         m.Info.Chat.String(),
			Sender:       m.Info.Sender.String(),
			PushName:     m.Info.PushName,
			Text:         txt,
			Kind:         kind,
			Thumb:        thumb,
			Media:        media,
			Timestamp:    m.Info.Timestamp,
			FromMe:       m.Info.IsFromMe,
			QuotedID:     qid,
			QuotedSender: qsender,
			QuotedText:   qtext,
			ExpireSecs:   ephemeralTTL(m.Message),
		})
	})
}

// OnRevoke memanggil fn saat sebuah pesan ditarik/dihapus-untuk-semua (oleh
// pengirim mana pun) → UI tampilkan placeholder "pesan dihapus".
func (e *Engine) OnRevoke(fn func(chat, msgID, sender string)) {
	e.Client.AddEventHandler(func(evt interface{}) {
		m, ok := evt.(*events.Message)
		if !ok {
			return
		}
		pm := m.Message.GetProtocolMessage()
		if pm.GetType() == waE2E.ProtocolMessage_REVOKE {
			fn(m.Info.Chat.String(), pm.GetKey().GetID(), m.Info.Sender.String())
		}
	})
}

// OnPinInChat memanggil fn saat sebuah pesan disematkan/dilepas (dari perangkat
// lain / anggota lain) → UI perbarui banner tersemat.
func (e *Engine) OnPinInChat(fn func(chat, msgID string, pinned bool)) {
	e.Client.AddEventHandler(func(evt interface{}) {
		m, ok := evt.(*events.Message)
		if !ok {
			return
		}
		pin := m.Message.GetPinInChatMessage()
		if pin == nil || pin.GetKey() == nil {
			return
		}
		fn(m.Info.Chat.String(), pin.GetKey().GetID(), pin.GetType() == waE2E.PinInChatMessage_PIN_FOR_ALL)
	})
}

// OnReaction memanggil fn saat reaksi masuk (emoji ""=dilepas). target=pesan
// yang direaksi, sender=pemberi reaksi.
func (e *Engine) OnReaction(fn func(chat, targetID, sender, emoji string, fromMe bool)) {
	e.Client.AddEventHandler(func(evt interface{}) {
		m, ok := evt.(*events.Message)
		if !ok {
			return
		}
		r := m.Message.GetReactionMessage()
		if r == nil || r.GetKey() == nil {
			return
		}
		sender := m.Info.Sender.String()
		if m.Info.IsFromMe {
			sender = e.SelfJID()
		}
		fn(m.Info.Chat.String(), r.GetKey().GetID(), sender, r.GetText(), m.Info.IsFromMe)
	})
}

// OnPollVote memanggil fn saat ada suara polling masuk (sudah didekripsi).
// selected = hash opsi terpilih; cocokkan via MatchPollHashes.
func (e *Engine) OnPollVote(fn func(chat, pollID, voter string, selected [][]byte)) {
	e.Client.AddEventHandler(func(evt interface{}) {
		m, ok := evt.(*events.Message)
		if !ok || m.Message.GetPollUpdateMessage() == nil {
			return
		}
		pv, err := e.Client.DecryptPollVote(context.Background(), m)
		if err != nil {
			return
		}
		key := m.Message.GetPollUpdateMessage().GetPollCreationMessageKey()
		chat := key.GetRemoteJID()
		if chat == "" {
			chat = m.Info.Chat.String()
		}
		fn(chat, key.GetID(), m.Info.Sender.String(), pv.GetSelectedOptions())
	})
}

// MatchPollHashes memetakan hash opsi terpilih → nama opsi.
func (e *Engine) MatchPollHashes(options []string, selected [][]byte) []string {
	hashes := whatsmeow.HashPollOptions(options)
	var out []string
	for i, h := range hashes {
		for _, s := range selected {
			if bytes.Equal(h, s) {
				out = append(out, options[i])
				break
			}
		}
	}
	return out
}

// OnEdit memanggil fn saat sebuah pesan disunting pengirimnya (teks baru).
func (e *Engine) OnEdit(fn func(chat, msgID, newText string)) {
	e.Client.AddEventHandler(func(evt interface{}) {
		m, ok := evt.(*events.Message)
		if !ok {
			return
		}
		pm := m.Message.GetProtocolMessage()
		if pm.GetType() != waE2E.ProtocolMessage_MESSAGE_EDIT {
			return
		}
		edited := pm.GetEditedMessage()
		if edited == nil {
			return
		}
		_, txt, _, _ := describeMessage(edited)
		if txt == "" {
			return
		}
		fn(m.Info.Chat.String(), pm.GetKey().GetID(), txt)
	})
}

// HistoryConversation = satu percakapan dari history sync (sudah disederhanakan).
type HistoryConversation struct {
	JID       string
	Name      string // subjek grup / nama kontak (bila ada)
	Timestamp int64  // aktivitas terakhir (otoritatif utk urutan sidebar)
	Unread    int
	Pinned    bool
	Archived  bool
	Messages  []IncomingMessage
}

// OnHistorySync mendaftarkan callback saat blob history sync tiba dari HP.
// Ini yang mengisi daftar chat & riwayat lengkap saat pertama login. Callback
// juga menerima peta pushname (jid → nama kontak) untuk menamai chat 1:1.
func (e *Engine) OnHistorySync(fn func([]HistoryConversation, map[string]string, bool)) {
	e.Client.AddEventHandler(func(evt interface{}) {
		hs, ok := evt.(*events.HistorySync)
		if !ok || hs.Data == nil {
			return
		}
		out := make([]HistoryConversation, 0, len(hs.Data.GetConversations()))
		for _, conv := range hs.Data.GetConversations() {
			chat := conv.GetID()
			hc := HistoryConversation{
				JID:       chat,
				Name:      conv.GetName(),
				Timestamp: int64(conv.GetConversationTimestamp()),
				Unread:    int(conv.GetUnreadCount()),
				Pinned:    conv.GetPinned() > 0,
				Archived:  conv.GetArchived(),
			}
			for _, hmsg := range conv.GetMessages() {
				wmi := hmsg.GetMessage()
				if wmi == nil {
					continue
				}
				kind, txt, thumb, media := describeMessage(wmi.GetMessage())
				if kind == "" {
					continue // skip pesan protokol/reaksi/kosong
				}
				key := wmi.GetKey()
				// Pengirim grup ada di WebMessageInfo.Participant (level atas),
				// bukan key.Participant (sering kosong di history sync).
				sender := wmi.GetParticipant()
				if sender == "" {
					sender = key.GetParticipant()
				}
				if sender == "" {
					sender = chat
				}
				qid, qsender, qtext := extractQuote(wmi.GetMessage())
				hc.Messages = append(hc.Messages, IncomingMessage{
					ID:           key.GetID(),
					Chat:         chat,
					Sender:       sender,
					PushName:     wmi.GetPushName(),
					Text:         txt,
					Kind:         kind,
					Thumb:        thumb,
					Media:        media,
					Timestamp:    time.Unix(int64(wmi.GetMessageTimestamp()), 0),
					FromMe:       key.GetFromMe(),
					QuotedID:     qid,
					QuotedSender: qsender,
					QuotedText:   qtext,
					Status:       webStatus(wmi.GetStatus()),
				})
			}
			out = append(out, hc)
		}
		names := make(map[string]string)
		for _, p := range hs.Data.GetPushnames() {
			if p.GetID() != "" && p.GetPushname() != "" {
				names[p.GetID()] = p.GetPushname()
			}
		}
		onDemand := hs.Data.GetSyncType() == waHistorySync.HistorySync_ON_DEMAND
		fn(out, names, onDemand)
	})
}

// RequestOlderHistory minta `count` pesan SEBELUM pesan tertua yang kita punya
// untuk sebuah chat (history on-demand WhatsApp). Respons tiba sbg events.History
// Sync tipe ON_DEMAND → diproses OnHistorySync. Rekomendasi count = 50.
func (e *Engine) RequestOlderHistory(chatJID, oldestID string, oldestFromMe bool, oldestTsUnix int64, count int) error {
	if e == nil || e.Client == nil {
		return fmt.Errorf("engine nil")
	}
	j, err := types.ParseJID(chatJID)
	if err != nil {
		return err
	}
	info := &types.MessageInfo{
		MessageSource: types.MessageSource{Chat: j, IsFromMe: oldestFromMe},
		ID:            oldestID,
		Timestamp:     time.Unix(oldestTsUnix, 0),
	}
	msg := e.Client.BuildHistorySyncRequest(info, count)
	_, err = e.Client.SendPeerMessage(context.Background(), msg)
	return err
}

// describeMessage mengklasifikasi pesan → (kind, text, thumb, media).
//
//	kind  : "text" | "image" | "video" | "sticker" | "voice"  ("" = junk → lewati)
//	text  : caption (media) atau isi teks, atau label utk tipe non-render
//	thumb : data-URI thumbnail (image/video/sticker) bila tersedia di pesan
//	media : base64 proto pesan (utk download media penuh on-demand)
//
// Pesan teks biasa bisa datang sebagai Conversation ATAU ExtendedTextMessage
// (mis. saat ada reply/link/format), jadi keduanya dicek. Thumbnail diambil dari
// proto (JPEGThumbnail) TANPA download tambahan. Getter protobuf aman thd nil.
func describeMessage(msg *waE2E.Message) (kind, text, thumb, media string) {
	if msg == nil {
		return "", "", "", ""
	}
	// Buka pembungkus dulu.
	switch {
	case msg.GetEphemeralMessage() != nil:
		return describeMessage(msg.GetEphemeralMessage().GetMessage())
	case msg.GetViewOnceMessage() != nil:
		return describeMessage(msg.GetViewOnceMessage().GetMessage())
	case msg.GetViewOnceMessageV2() != nil:
		return describeMessage(msg.GetViewOnceMessageV2().GetMessage())
	case msg.GetDocumentWithCaptionMessage() != nil:
		return describeMessage(msg.GetDocumentWithCaptionMessage().GetMessage())
	case msg.GetDeviceSentMessage() != nil:
		return describeMessage(msg.GetDeviceSentMessage().GetMessage())
	}
	jpeg := func(b []byte) string {
		if len(b) == 0 {
			return ""
		}
		return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(b)
	}
	// serialize proto pesan ini → utk download media on-demand nanti.
	// Buang thumbnail tertanam dulu: sudah disimpan terpisah di kolom `thumb`,
	// dan download pakai directPath/mediaKey/sha (bukan thumbnail) → proto jadi
	// ramping (hilangkan duplikasi byte thumbnail × ribuan pesan). Dipanggil
	// SETELAH thumb diekstrak (urutan argumen return: jpeg(...) lalu ser()).
	ser := func() string {
		if im := msg.GetImageMessage(); im != nil {
			im.JPEGThumbnail = nil
		}
		if vm := msg.GetVideoMessage(); vm != nil {
			vm.JPEGThumbnail = nil
		}
		if sm := msg.GetStickerMessage(); sm != nil {
			sm.PngThumbnail = nil
		}
		if dm := msg.GetDocumentMessage(); dm != nil {
			dm.JPEGThumbnail = nil
		}
		b, err := proto.Marshal(msg)
		if err != nil {
			return ""
		}
		return base64.StdEncoding.EncodeToString(b)
	}
	switch {
	case msg.GetConversation() != "":
		return "text", msg.GetConversation(), "", ""
	case msg.GetExtendedTextMessage().GetText() != "":
		return "text", msg.GetExtendedTextMessage().GetText(), "", ""
	case msg.GetImageMessage() != nil:
		return "image", msg.GetImageMessage().GetCaption(), jpeg(msg.GetImageMessage().GetJPEGThumbnail()), ser()
	case msg.GetVideoMessage() != nil:
		vm := msg.GetVideoMessage()
		if vm.GetGifPlayback() {
			return "gif", vm.GetCaption(), jpeg(vm.GetJPEGThumbnail()), ser()
		}
		return "video", vm.GetCaption(), jpeg(vm.GetJPEGThumbnail()), ser()
	case msg.GetStickerMessage() != nil:
		return "sticker", "", jpeg(msg.GetStickerMessage().GetPngThumbnail()), ser()
	case msg.GetAudioMessage() != nil:
		return "voice", fmtDur(msg.GetAudioMessage().GetSeconds()), "", ser()
	case msg.GetDocumentMessage() != nil:
		name := msg.GetDocumentMessage().GetFileName()
		if name == "" {
			name = "Dokumen"
		}
		// kind "document": text=nama, media=proto (utk unduh via /media).
		return "document", name, "", ser()
	case msg.GetLocationMessage() != nil:
		lm := msg.GetLocationMessage()
		name := lm.GetName()
		label := "📍 " + name
		if name == "" {
			label = "📍 Lokasi"
		}
		// thumb dipakai utk membawa "lat,lng" (dibaca FE utk peta/buka).
		coord := fmt.Sprintf("%f,%f", lm.GetDegreesLatitude(), lm.GetDegreesLongitude())
		return "location", label, coord, ""
	case msg.GetContactMessage() != nil:
		cm := msg.GetContactMessage()
		name := cm.GetDisplayName()
		if name == "" {
			name = "Kontak"
		}
		// thumb membawa nomor telepon (di-parse dari vCard).
		return "contact", "👤 " + name, vcardPhone(cm.GetVcard()), ""
	}
	// Polling (beberapa versi proto). Opsi di-JSON-kan ke `thumb` utk render+vote.
	if pc := pollCreation(msg); pc != nil {
		opts := make([]string, 0, len(pc.GetOptions()))
		for _, o := range pc.GetOptions() {
			opts = append(opts, o.GetOptionName())
		}
		j, _ := json.Marshal(opts)
		name := pc.GetName()
		if name == "" {
			name = "Polling"
		}
		return "poll", name, string(j), ""
	}
	return "", "", "", "" // reaksi/protokol/key-dist/dll → junk
}

// vcardPhone memetik nomor telepon pertama dari teks vCard (baris TEL).
func vcardPhone(vcard string) string {
	for _, line := range strings.Split(vcard, "\n") {
		up := strings.ToUpper(line)
		if strings.HasPrefix(up, "TEL") {
			if i := strings.LastIndex(line, ":"); i >= 0 {
				return strings.TrimSpace(line[i+1:])
			}
		}
	}
	return ""
}

// pollCreation mengembalikan PollCreationMessage dari versi proto mana pun.
func pollCreation(msg *waE2E.Message) *waE2E.PollCreationMessage {
	switch {
	case msg.GetPollCreationMessage() != nil:
		return msg.GetPollCreationMessage()
	case msg.GetPollCreationMessageV2() != nil:
		return msg.GetPollCreationMessageV2()
	case msg.GetPollCreationMessageV3() != nil:
		return msg.GetPollCreationMessageV3()
	}
	return nil
}

func fmtDur(sec uint32) string {
	if sec == 0 {
		return ""
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

// extractQuote membaca ContextInfo (balasan): id pesan dikutip, JID pengirimnya,
// dan teks/label preview kutipan. Kosong bila pesan bukan balasan.
func extractQuote(msg *waE2E.Message) (id, sender, text string) {
	ci := msgContext(msg)
	if ci == nil || ci.GetStanzaID() == "" {
		return "", "", ""
	}
	id = ci.GetStanzaID()
	sender = ci.GetParticipant()
	if q := ci.GetQuotedMessage(); q != nil {
		kind, t, _, _ := describeMessage(q)
		if t != "" {
			text = t
		} else {
			switch kind {
			case "image":
				text = "🖼️ Foto"
			case "video":
				text = "🎬 Video"
			case "sticker":
				text = "🏷️ Stiker"
			case "voice":
				text = "🎤 Pesan suara"
			}
		}
	}
	return id, sender, text
}

// ephemeralTTL membaca TTL disappearing (detik) dari ContextInfo.Expiration.
// 0 = pesan biasa (tak hilang).
func ephemeralTTL(msg *waE2E.Message) uint32 {
	if ci := msgContext(msg); ci != nil {
		return ci.GetExpiration()
	}
	return 0
}

// msgContext mengambil ContextInfo dari tipe pesan yang mendukungnya.
func msgContext(msg *waE2E.Message) *waE2E.ContextInfo {
	switch {
	case msg.GetExtendedTextMessage() != nil:
		return msg.GetExtendedTextMessage().GetContextInfo()
	case msg.GetImageMessage() != nil:
		return msg.GetImageMessage().GetContextInfo()
	case msg.GetVideoMessage() != nil:
		return msg.GetVideoMessage().GetContextInfo()
	case msg.GetStickerMessage() != nil:
		return msg.GetStickerMessage().GetContextInfo()
	case msg.GetAudioMessage() != nil:
		return msg.GetAudioMessage().GetContextInfo()
	case msg.GetDocumentMessage() != nil:
		return msg.GetDocumentMessage().GetContextInfo()
	}
	return nil
}
