package engine

// calls.go — panggilan masuk. whatsmeow hanya signaling (tak ada media/WebRTC):
// kita bisa MENERIMA notifikasi panggilan + MENOLAK, tapi tak bisa menjawab
// (bicara). Cukup untuk: notifikasi + tombol Tolak + log panggilan.

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// OnCall mendaftar handler panggilan masuk (CallOffer 1:1 + CallOfferNotice grup).
func (e *Engine) OnCall(fn func(fromJID string, video, group bool, callID string, ts time.Time)) {
	e.Client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.CallOffer:
			fn(v.From.String(), false, false, v.CallID, v.Timestamp)
		case *events.CallOfferNotice:
			fn(v.From.String(), v.Media == "video", v.Type == "group", v.CallID, v.Timestamp)
		}
	})
}

// RejectCall menolak panggilan masuk (mengirim sinyal reject ke pemanggil).
func (e *Engine) RejectCall(ctx context.Context, callFrom, callID string) error {
	jid, err := types.ParseJID(callFrom)
	if err != nil {
		return err
	}
	return e.Client.RejectCall(ctx, jid, callID)
}
