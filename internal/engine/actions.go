package engine

// actions.go — aksi kelola chat via app-state (tersinkron ke semua perangkat):
// pin, mute, arsip, bintang pesan, tandai chat dibaca/belum.

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// ChatSettings membaca pin/arsip/bisukan TERKINI dari store app-state whatsmeow
// (tabel ChatSettings; di-PutPinned saat mutasi diproses). Sumber sahih utk
// kondisi YANG SUDAH ADA: events.Pin hanya fire saat BERUBAH, tak utk sematan
// lama → query langsung di sini yang memunculkannya. ok=false bila tak diketahui.
func (e *Engine) ChatSettings(jid string) (pinned, archived, muted, ok bool) {
	if e == nil || e.Client == nil || e.Client.Store == nil || e.Client.Store.ChatSettings == nil {
		return
	}
	j, err := types.ParseJID(jid)
	if err != nil {
		return
	}
	s, err := e.Client.Store.ChatSettings.GetChatSettings(context.Background(), j)
	if err != nil || !s.Found {
		return
	}
	return s.Pinned, s.Archived, s.MutedUntil.After(time.Now()), true
}

// OnChatAction memanggil fn saat pin/mute/arsip berubah dari perangkat lain
// (mis. di-pin dari HP) — agar sidebar ikut tersinkron. action: "pin"|"mute"|"archive".
func (e *Engine) OnChatAction(fn func(jid, action string, on bool)) {
	e.Client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Pin:
			fn(v.JID.String(), "pin", v.Action.GetPinned())
		case *events.Mute:
			fn(v.JID.String(), "mute", v.Action.GetMuted())
		case *events.Archive:
			fn(v.JID.String(), "archive", v.Action.GetArchived())
		}
	})
}

// Pin menyematkan / melepas sematan chat.
func (e *Engine) Pin(ctx context.Context, jid string, pin bool) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	return e.Client.SendAppState(ctx, appstate.BuildPin(j, pin))
}

// Mute membisukan chat untuk durasi tertentu (dur 0 = tak terbatas saat mute=true).
func (e *Engine) Mute(ctx context.Context, jid string, mute bool, dur time.Duration) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	return e.Client.SendAppState(ctx, appstate.BuildMute(j, mute, dur))
}

// Archive mengarsip / mengeluarkan chat dari arsip. lastID/lastFromMe dipakai
// membangun kunci pesan terakhir yang diperlukan patch.
func (e *Engine) Archive(ctx context.Context, jid string, archive bool, lastTS time.Time, lastID string, lastFromMe bool) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	var key = makeKey(jid, lastID, lastFromMe)
	if lastID == "" {
		key = nil
	}
	return e.Client.SendAppState(ctx, appstate.BuildArchive(j, archive, lastTS, key))
}

// Star membintangi / melepas bintang satu pesan.
func (e *Engine) Star(ctx context.Context, chat, sender, msgID string, fromMe, starred bool) error {
	cj, err := types.ParseJID(chat)
	if err != nil {
		return err
	}
	sj := cj
	if fromMe && e.Client.Store.ID != nil {
		sj = *e.Client.Store.ID
	} else if sender != "" {
		if p, err := types.ParseJID(sender); err == nil {
			sj = p
		}
	}
	return e.Client.SendAppState(ctx, appstate.BuildStar(cj, sj, types.MessageID(msgID), fromMe, starred))
}

// MarkChatRead menandai seluruh chat dibaca / belum dibaca (tingkat chat, beda
// dengan MarkRead per-pesan yang juga mengirim read-receipt).
func (e *Engine) MarkChatRead(ctx context.Context, jid string, read bool, lastTS time.Time, lastID string, lastFromMe bool) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	var key = makeKey(jid, lastID, lastFromMe)
	if lastID == "" {
		key = nil
	}
	return e.Client.SendAppState(ctx, appstate.BuildMarkChatAsRead(j, read, lastTS, key))
}
