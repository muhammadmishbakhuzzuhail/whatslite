package app

// app_calls.go — API panggilan (signaling-only: log + tolak). whatsmeow tak
// punya media call, jadi tak ada menjawab/menelepon.

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"whatsapp-lite/internal/storage"
)

// GetCalls mengembalikan log panggilan terbaru.
func (a *App) GetCalls() []storage.Call {
	if a.store == nil {
		return []storage.Call{}
	}
	out, err := a.store.ListCalls(a.ctx)
	if err != nil || out == nil {
		return []storage.Call{}
	}
	return out
}

// RejectCall menolak panggilan masuk (callID) dari jid, lalu tandai log "rejected".
func (a *App) RejectCall(jid, callID string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.RejectCall(a.ctx, a.canon(jid), callID); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	if a.store != nil {
		_ = a.store.SetCallStatus(a.ctx, callID, "rejected")
	}
	runtime.EventsEmit(a.ctx, "wa:callupdate", "")
}
