package engine

// contacts.go — info kontak (about/status), blokir, dan profil sendiri.

import (
	"context"

	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// ContactAbout mengambil teks "info"/status seorang kontak (butuh koneksi).
func (e *Engine) ContactAbout(ctx context.Context, jid string) string {
	j, err := types.ParseJID(jid)
	if err != nil || !e.Client.IsConnected() {
		return ""
	}
	infos, err := e.Client.GetUserInfo(ctx, []types.JID{j})
	if err != nil {
		return ""
	}
	if info, ok := infos[j]; ok {
		return info.Status
	}
	return ""
}

// Block memblokir / membuka blokir kontak.
func (e *Engine) Block(ctx context.Context, jid string, block bool) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	action := events.BlocklistChangeActionUnblock
	if block {
		action = events.BlocklistChangeActionBlock
	}
	_, err = e.Client.UpdateBlocklist(ctx, j, action)
	return err
}

// SetAbout memperbarui teks "info"/status akun sendiri.
func (e *Engine) SetAbout(ctx context.Context, text string) error {
	return e.Client.SetStatusMessage(ctx, text)
}

// SetMyName memperbarui push name (nama tampil) akun sendiri lewat app-state.
func (e *Engine) SetMyName(ctx context.Context, name string) error {
	return e.Client.SendAppState(ctx, appstate.BuildSettingPushName(name))
}
