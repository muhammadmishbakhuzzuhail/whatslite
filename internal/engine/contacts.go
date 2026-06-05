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

// Blocklist mengembalikan daftar JID yang diblokir.
func (e *Engine) Blocklist(ctx context.Context) ([]string, error) {
	bl, err := e.Client.GetBlocklist(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(bl.JIDs))
	for _, j := range bl.JIDs {
		out = append(out, j.String())
	}
	return out, nil
}

// PrivacyMap mengembalikan setelan privasi sebagai map name→value.
func (e *Engine) PrivacyMap(ctx context.Context) map[string]string {
	ps := e.Client.GetPrivacySettings(ctx)
	return map[string]string{
		"lastseen":     string(ps.LastSeen),
		"profile":      string(ps.Profile),
		"status":       string(ps.Status),
		"readreceipts": string(ps.ReadReceipts),
		"groupadd":     string(ps.GroupAdd),
		"online":       string(ps.Online),
	}
}

// SetPrivacy mengubah satu setelan privasi.
func (e *Engine) SetPrivacy(ctx context.Context, name, value string) error {
	_, err := e.Client.SetPrivacySetting(ctx, types.PrivacySettingType(name), types.PrivacySetting(value))
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
