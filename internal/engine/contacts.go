// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package engine

// contacts.go — info kontak (about/status), blokir, dan profil sendiri.

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// SetDefaultDisappearing menyetel timer hilang-otomatis default utk chat baru.
func (e *Engine) SetDefaultDisappearing(ctx context.Context, secs int) error {
	return e.Client.SetDefaultDisappearingTimer(ctx, time.Duration(secs)*time.Second)
}

// MyQRLink mengambil tautan QR kontak sendiri (revoke=true buat ulang).
func (e *Engine) MyQRLink(ctx context.Context, revoke bool) (string, error) {
	return e.Client.GetContactQRLink(ctx, revoke)
}

// ResolveQR menukar kode QR-kontak (hasil scan) → JID + nama.
func (e *Engine) ResolveQR(ctx context.Context, code string) (string, string, error) {
	t, err := e.Client.ResolveContactQRLink(ctx, code)
	if err != nil || t == nil {
		return "", "", err
	}
	return t.JID.String(), t.PushName, nil
}

// BizProfile = profil bisnis (alamat/email/kategori) bila kontak akun bisnis.
type BizProfile struct {
	Address  string
	Email    string
	Category string
}

// BusinessProfile mengambil profil bisnis kontak (nil bila bukan bisnis).
func (e *Engine) BusinessProfile(ctx context.Context, jid string) *BizProfile {
	j, err := types.ParseJID(jid)
	if err != nil {
		return nil
	}
	bp, err := e.Client.GetBusinessProfile(ctx, j)
	if err != nil || bp == nil {
		return nil
	}
	cat := ""
	if len(bp.Categories) > 0 {
		cat = bp.Categories[0].Name
	}
	if bp.Address == "" && bp.Email == "" && cat == "" {
		return nil
	}
	return &BizProfile{Address: bp.Address, Email: bp.Email, Category: cat}
}

// WACheck = hasil cek "ada di WhatsApp?" untuk satu nomor.
type WACheck struct {
	Query      string
	JID        string
	Registered bool
}

// IsOnWhatsApp memeriksa apakah nomor-nomor terdaftar di WhatsApp.
func (e *Engine) IsOnWhatsApp(ctx context.Context, phones []string) ([]WACheck, error) {
	resp, err := e.Client.IsOnWhatsApp(ctx, phones)
	if err != nil {
		return nil, err
	}
	out := make([]WACheck, 0, len(resp))
	for _, r := range resp {
		out = append(out, WACheck{Query: r.Query, JID: r.JID.String(), Registered: r.IsIn})
	}
	return out, nil
}

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

// ContactJIDs mengembalikan JID semua kontak nyata (nomor @s.whatsapp.net) yang
// punya nama di buku-alamat/pushname. Dipakai daftar "Kontak" di sidebar.
func (e *Engine) ContactJIDs() []string {
	if e == nil || e.Client == nil || e.Client.Store == nil || e.Client.Store.Contacts == nil {
		return nil
	}
	m, err := e.Client.Store.Contacts.GetAllContacts(context.Background())
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(m))
	for jid, info := range m {
		if jid.Server != types.DefaultUserServer { // hanya nomor telepon nyata
			continue
		}
		// HANYA yang TERSIMPAN di buku-alamat (punya FullName/FirstName dari
		// sinkron app-state). PushName/BusinessName saja = sekadar terlihat di
		// grup → BUKAN kontak HP. Tanpa filter ini, semua anggota grup ikut masuk.
		if info.FullName == "" && info.FirstName == "" {
			continue
		}
		out = append(out, jid.String())
	}
	return out
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
