package engine

// send.go — pesan keluar: teks, media (foto/video/voice/dokumen), balasan
// (reply/quote), teruskan (forward), reaksi, hapus (revoke), tandai dibaca.

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// SendText mengirim pesan teks ke JID tujuan dan mengembalikan ID pesan.
func (e *Engine) SendText(ctx context.Context, to, text string) (string, error) {
	return e.sendMessage(ctx, to, &waE2E.Message{Conversation: proto.String(text)})
}

// SendTextMentions mengirim teks dengan mention (ContextInfo.MentionedJID). Teks
// harus memuat "@<nomor>" yang cocok dgn JID. Tanpa mention → SendText biasa.
func (e *Engine) SendTextMentions(ctx context.Context, to, text string, mentions []string) (string, error) {
	if len(mentions) == 0 {
		return e.SendText(ctx, to, text)
	}
	msg := &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
		Text:        proto.String(text),
		ContextInfo: &waE2E.ContextInfo{MentionedJID: mentions},
	}}
	return e.sendMessage(ctx, to, msg)
}

// Reply mengirim balasan yang mengutip pesan lain (ContextInfo). quotedSender =
// JID pengirim pesan yang dikutip; quotedText = isi/preview kutipan.
func (e *Engine) Reply(ctx context.Context, to, text, quotedID, quotedSender, quotedText string) (string, error) {
	msg := &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
		Text:        proto.String(text),
		ContextInfo: quoteContext(quotedID, quotedSender, quotedText),
	}}
	return e.sendMessage(ctx, to, msg)
}

// ForwardText meneruskan pesan teks (tandai "diteruskan").
func (e *Engine) ForwardText(ctx context.Context, to, text string) (string, error) {
	msg := &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
		Text:        proto.String(text),
		ContextInfo: &waE2E.ContextInfo{IsForwarded: proto.Bool(true), ForwardingScore: proto.Uint32(1)},
	}}
	return e.sendMessage(ctx, to, msg)
}

// ForwardMedia meneruskan pesan media memakai proto aslinya (base64 tersimpan) —
// tak perlu unggah ulang; cukup tandai diteruskan lalu kirim.
func (e *Engine) ForwardMedia(ctx context.Context, to, srcProtoB64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(srcProtoB64)
	if err != nil {
		return "", err
	}
	var msg waE2E.Message
	if err := proto.Unmarshal(raw, &msg); err != nil {
		return "", err
	}
	fwd := &waE2E.ContextInfo{IsForwarded: proto.Bool(true), ForwardingScore: proto.Uint32(1)}
	switch {
	case msg.GetImageMessage() != nil:
		msg.ImageMessage.ContextInfo = fwd
	case msg.GetVideoMessage() != nil:
		msg.VideoMessage.ContextInfo = fwd
	case msg.GetAudioMessage() != nil:
		msg.AudioMessage.ContextInfo = fwd
	case msg.GetStickerMessage() != nil:
		msg.StickerMessage.ContextInfo = fwd
	case msg.GetDocumentMessage() != nil:
		msg.DocumentMessage.ContextInfo = fwd
	}
	return e.sendMessage(ctx, to, &msg)
}

// SendMedia mengunggah `data` lalu mengirimnya sebagai pesan media. kind:
// "image" | "video" | "voice" | "document". caption/fileName opsional.
func (e *Engine) SendMedia(ctx context.Context, to, kind, mime, caption, fileName string, data []byte) (string, error) {
	mt, err := mediaTypeFor(kind)
	if err != nil {
		return "", err
	}
	up, err := e.Client.Upload(ctx, data, mt)
	if err != nil {
		return "", fmt.Errorf("upload: %w", err)
	}
	length := uint64(len(data))
	msg := &waE2E.Message{}
	switch kind {
	case "image":
		msg.ImageMessage = &waE2E.ImageMessage{
			Caption: strPtr(caption), Mimetype: proto.String(mime),
			URL: &up.URL, DirectPath: &up.DirectPath, MediaKey: up.MediaKey,
			FileEncSHA256: up.FileEncSHA256, FileSHA256: up.FileSHA256, FileLength: &length,
		}
	case "video":
		msg.VideoMessage = &waE2E.VideoMessage{
			Caption: strPtr(caption), Mimetype: proto.String(mime),
			URL: &up.URL, DirectPath: &up.DirectPath, MediaKey: up.MediaKey,
			FileEncSHA256: up.FileEncSHA256, FileSHA256: up.FileSHA256, FileLength: &length,
		}
	case "voice":
		if mime == "" {
			mime = "audio/ogg; codecs=opus"
		}
		msg.AudioMessage = &waE2E.AudioMessage{
			Mimetype: proto.String(mime), PTT: proto.Bool(true),
			URL: &up.URL, DirectPath: &up.DirectPath, MediaKey: up.MediaKey,
			FileEncSHA256: up.FileEncSHA256, FileSHA256: up.FileSHA256, FileLength: &length,
		}
	case "document":
		if fileName == "" {
			fileName = "Dokumen"
		}
		msg.DocumentMessage = &waE2E.DocumentMessage{
			Mimetype: proto.String(mime), FileName: proto.String(fileName), Title: proto.String(fileName),
			Caption: strPtr(caption),
			URL:     &up.URL, DirectPath: &up.DirectPath, MediaKey: up.MediaKey,
			FileEncSHA256: up.FileEncSHA256, FileSHA256: up.FileSHA256, FileLength: &length,
		}
	default:
		return "", fmt.Errorf("kind tak didukung: %q", kind)
	}
	return e.sendMessage(ctx, to, msg)
}

// PostTextStatus mengunggah status teks ke status@broadcast. whatsmeow otomatis
// menyusun daftar penerima (kontak) — kita cukup kirim ke StatusBroadcastJID.
func (e *Engine) PostTextStatus(ctx context.Context, text string) (string, error) {
	msg := &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
		Text: proto.String(text),
	}}
	resp, err := e.Client.SendMessage(ctx, types.StatusBroadcastJID, msg)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// SendLocation mengirim pesan lokasi (lat/lng + nama opsional).
func (e *Engine) SendLocation(ctx context.Context, to string, lat, lng float64, name string) (string, error) {
	msg := &waE2E.Message{LocationMessage: &waE2E.LocationMessage{
		DegreesLatitude:  proto.Float64(lat),
		DegreesLongitude: proto.Float64(lng),
		Name:             strPtr(name),
	}}
	return e.sendMessage(ctx, to, msg)
}

// SendPoll mengirim polling (BuildPollCreation). selectable=1 → pilihan tunggal.
func (e *Engine) SendPoll(ctx context.Context, to, name string, options []string, selectable int) (string, error) {
	if selectable < 1 {
		selectable = 1
	}
	msg := e.Client.BuildPollCreation(name, options, selectable)
	return e.sendMessage(ctx, to, msg)
}

// SetDisappearing mengatur timer pesan sementara (detik; 0 = mati).
func (e *Engine) SetDisappearing(ctx context.Context, chat string, seconds int) error {
	cj, err := types.ParseJID(chat)
	if err != nil {
		return err
	}
	return e.Client.SetDisappearingTimer(ctx, cj, time.Duration(seconds)*time.Second, time.Now())
}

// PostMediaStatus mengunggah media lalu memposnya sebagai status (image/video).
func (e *Engine) PostMediaStatus(ctx context.Context, kind, mime, caption string, data []byte) (string, error) {
	mt, err := mediaTypeFor(kind)
	if err != nil {
		return "", err
	}
	up, err := e.Client.Upload(ctx, data, mt)
	if err != nil {
		return "", fmt.Errorf("upload: %w", err)
	}
	length := uint64(len(data))
	msg := &waE2E.Message{}
	switch kind {
	case "image":
		msg.ImageMessage = &waE2E.ImageMessage{
			Caption: strPtr(caption), Mimetype: proto.String(mime),
			URL: &up.URL, DirectPath: &up.DirectPath, MediaKey: up.MediaKey,
			FileEncSHA256: up.FileEncSHA256, FileSHA256: up.FileSHA256, FileLength: &length,
		}
	case "video":
		msg.VideoMessage = &waE2E.VideoMessage{
			Caption: strPtr(caption), Mimetype: proto.String(mime),
			URL: &up.URL, DirectPath: &up.DirectPath, MediaKey: up.MediaKey,
			FileEncSHA256: up.FileEncSHA256, FileSHA256: up.FileSHA256, FileLength: &length,
		}
	default:
		return "", fmt.Errorf("status media tak didukung: %q", kind)
	}
	resp, err := e.Client.SendMessage(ctx, types.StatusBroadcastJID, msg)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// SendEdit menyunting pesan teks yang sudah terkirim (≤15 menit, pesan sendiri).
func (e *Engine) SendEdit(ctx context.Context, chat, msgID, newText string) error {
	cj, err := types.ParseJID(chat)
	if err != nil {
		return err
	}
	edit := e.Client.BuildEdit(cj, types.MessageID(msgID), &waE2E.Message{Conversation: proto.String(newText)})
	_, err = e.Client.SendMessage(ctx, cj, edit)
	return err
}

// PinMessage menyemat / melepas pesan di dalam chat (PinInChatMessage, untuk semua).
func (e *Engine) PinMessage(ctx context.Context, chat, sender, msgID string, fromMe, pin bool) error {
	cj, err := types.ParseJID(chat)
	if err != nil {
		return err
	}
	key := &waCommon.MessageKey{
		RemoteJID: proto.String(chat), FromMe: proto.Bool(fromMe), ID: proto.String(msgID),
	}
	if cj.Server == types.GroupServer && sender != "" {
		key.Participant = proto.String(sender)
	}
	t := waE2E.PinInChatMessage_PIN_FOR_ALL
	if !pin {
		t = waE2E.PinInChatMessage_UNPIN_FOR_ALL
	}
	msg := &waE2E.Message{PinInChatMessage: &waE2E.PinInChatMessage{
		Key: key, Type: t.Enum(), SenderTimestampMS: proto.Int64(time.Now().UnixMilli()),
	}}
	_, err = e.Client.SendMessage(ctx, cj, msg)
	return err
}

// React menambah/menghapus reaksi emoji pada pesan (emoji "" = hapus reaksi).
func (e *Engine) React(ctx context.Context, chat, sender, msgID, emoji string, fromMe bool) error {
	cj, sj, err := e.reactKey(chat, sender, fromMe)
	if err != nil {
		return err
	}
	_, err = e.Client.SendMessage(ctx, cj, e.Client.BuildReaction(cj, sj, types.MessageID(msgID), emoji))
	return err
}

// Revoke menghapus pesan untuk semua orang (delete-for-everyone).
func (e *Engine) Revoke(ctx context.Context, chat, sender, msgID string, fromMe bool) error {
	cj, sj, err := e.reactKey(chat, sender, fromMe)
	if err != nil {
		return err
	}
	_, err = e.Client.SendMessage(ctx, cj, e.Client.BuildRevoke(cj, sj, types.MessageID(msgID)))
	return err
}

// MarkRead menandai pesan sudah dibaca (kirim read-receipt → centang biru bagi
// lawan + bersihkan unread sisi kita).
func (e *Engine) MarkRead(ctx context.Context, chat, sender, msgID string) error {
	cj, err := types.ParseJID(chat)
	if err != nil {
		return err
	}
	sj := cj
	if sender != "" {
		if p, err := types.ParseJID(sender); err == nil {
			sj = p
		}
	}
	return e.Client.MarkRead(ctx, []types.MessageID{types.MessageID(msgID)}, time.Now(), cj, sj)
}

// --- helper ---

// sendMessage membungkus SendMessage agar pengirim cuma urus *waE2E.Message.
func (e *Engine) sendMessage(ctx context.Context, to string, msg *waE2E.Message) (string, error) {
	jid, err := types.ParseJID(to)
	if err != nil {
		return "", fmt.Errorf("parse jid %q: %w", to, err)
	}
	resp, err := e.Client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// reactKey menyiapkan (chatJID, senderJID) untuk reaksi/revoke. Untuk pesan
// sendiri, sender = JID kita.
func (e *Engine) reactKey(chat, sender string, fromMe bool) (types.JID, types.JID, error) {
	cj, err := types.ParseJID(chat)
	if err != nil {
		return types.JID{}, types.JID{}, err
	}
	sj := cj
	if fromMe && e.Client.Store.ID != nil {
		sj = *e.Client.Store.ID
	} else if sender != "" {
		if p, err := types.ParseJID(sender); err == nil {
			sj = p
		}
	}
	return cj, sj, nil
}

func mediaTypeFor(kind string) (whatsmeow.MediaType, error) {
	switch kind {
	case "image":
		return whatsmeow.MediaImage, nil
	case "video":
		return whatsmeow.MediaVideo, nil
	case "voice":
		return whatsmeow.MediaAudio, nil
	case "document":
		return whatsmeow.MediaDocument, nil
	}
	return "", fmt.Errorf("kind media tak dikenal: %q", kind)
}

func quoteContext(id, sender, text string) *waE2E.ContextInfo {
	ci := &waE2E.ContextInfo{
		StanzaID:      proto.String(id),
		QuotedMessage: &waE2E.Message{Conversation: proto.String(text)},
	}
	if sender != "" {
		ci.Participant = proto.String(sender)
	}
	return ci
}

// makeKey membangun waCommon.MessageKey untuk patch app-state (archive/star/read).
func makeKey(chat, id string, fromMe bool) *waCommon.MessageKey {
	return &waCommon.MessageKey{
		RemoteJID: proto.String(chat),
		FromMe:    proto.Bool(fromMe),
		ID:        proto.String(id),
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return proto.String(s)
}
