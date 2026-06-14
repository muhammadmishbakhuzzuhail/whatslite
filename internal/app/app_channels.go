// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_channels.go — WhatsApp Channels (newsletter). Daftar saluran diikuti,
// feed read-only, ikuti/berhenti via tautan.

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/engine"
)

// ChannelDTO = saluran utk daftar di sidebar.
type ChannelDTO struct {
	JID         string `json:"jid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Subscribers int    `json:"subscribers"`
	Picture     string `json:"picture"`
	Verified    bool   `json:"verified"`
	Muted       bool   `json:"muted"`
	Role        string `json:"role"`
}

// ChannelMsgDTO = satu pesan saluran (read-only feed).
type ChannelMsgDTO struct {
	ID       string `json:"id"`
	ServerID int64  `json:"serverId"`
	Type     string `json:"type"`
	Text     string `json:"text"`
	Thumb    string `json:"thumb"`
	Time     string `json:"time"`
	Views    int    `json:"views"`
}

// CreateChannel membuat saluran baru; kembalikan JID (atau "").
func (a *App) CreateChannel(name, desc string) string {
	if a.eng == nil || name == "" {
		return ""
	}
	jid, err := a.eng.CreateChannel(a.ctx, name, desc, nil)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	runtime.EventsEmit(a.ctx, "wa:sync", "")
	return jid
}

// PostChannel mengirim teks ke saluran (hanya owner/admin). whatsmeow mengarah-
// kan SendMessage ke JID @newsletter secara otomatis.
func (a *App) PostChannel(jid, text string) {
	if a.eng == nil || text == "" {
		return
	}
	if _, err := a.eng.SendText(a.ctx, jid, text); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	runtime.EventsEmit(a.ctx, "wa:sync", "")
}

// GetChannels mengembalikan saluran yang diikuti.
func (a *App) GetChannels() (out []ChannelDTO) {
	out = []ChannelDTO{}
	if a.eng == nil {
		return
	}
	cs, err := a.eng.ListChannels(a.ctx)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	for _, c := range cs {
		out = append(out, channelDTO(c))
	}
	return
}

// GetRecommendedChannels mengambil saluran rekomendasi GLOBAL (jelajah), bukan
// hanya yang diikuti. Experimental (API undocumented) — error disurfacing ke UI.
func (a *App) GetRecommendedChannels(query string) (out []ChannelDTO) {
	out = []ChannelDTO{}
	if a.eng == nil {
		return
	}
	cs, err := a.eng.RecommendedChannels(a.ctx, query)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	for _, c := range cs {
		out = append(out, channelDTO(c))
	}
	return
}

// FollowChannelByJID mengikuti saluran hasil jelajah (berdasarkan JID).
func (a *App) FollowChannelByJID(jid string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.FollowChannelByJID(a.ctx, jid); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	runtime.EventsEmit(a.ctx, "wa:sync", "")
}

// GetChannelMessages mengambil feed (read-only) satu saluran.
func (a *App) GetChannelMessages(jid string) (out []ChannelMsgDTO) {
	out = []ChannelMsgDTO{}
	if a.eng == nil {
		return
	}
	ms, err := a.eng.ChannelMessages(a.ctx, jid, 50)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	// Tandai sudah dilihat (view-receipt) — off-loop, best-effort.
	sids := make([]int64, 0, len(ms))
	for _, m := range ms {
		if m.ServerID > 0 {
			sids = append(sids, m.ServerID)
		}
	}
	if len(sids) > 0 {
		a.bg(func() { _ = a.eng.MarkChannelViewed(a.ctx, jid, sids) })
	}
	// terbaru dulu dari API → balik jadi lama→baru (ala feed chat).
	for i := len(ms) - 1; i >= 0; i-- {
		m := ms[i]
		out = append(out, ChannelMsgDTO{
			ID: m.ID, ServerID: m.ServerID, Type: m.Kind, Text: m.Text, Thumb: m.Thumb,
			Time: hm(m.Timestamp), Views: m.Views,
		})
	}
	return
}

// FollowChannel mengikuti saluran via tautan undangan; kembalikan info-nya.
func (a *App) FollowChannel(link string) *ChannelDTO {
	if a.eng == nil {
		return nil
	}
	c, err := a.eng.FollowChannel(a.ctx, link)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return nil
	}
	d := channelDTO(c)
	return &d
}

// UnfollowChannel berhenti mengikuti saluran.
func (a *App) UnfollowChannel(jid string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.UnfollowChannel(a.ctx, jid); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
	}
}

// ReactChannel mengirim reaksi emoji pada post saluran.
func (a *App) ReactChannel(channelJID, msgID string, serverID int64, emoji string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.ReactChannel(a.ctx, channelJID, msgID, serverID, emoji); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
	}
}

// MuteChannel membisukan / membunyikan saluran.
func (a *App) MuteChannel(jid string, mute bool) {
	if a.eng == nil {
		return
	}
	if err := a.eng.MuteChannel(a.ctx, jid, mute); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
	}
}

func channelDTO(c engine.ChannelInfo) ChannelDTO {
	return ChannelDTO{
		JID: c.JID, Name: c.Name, Description: c.Description,
		Subscribers: c.Subscribers, Picture: c.Picture, Verified: c.Verified,
		Muted: c.Muted, Role: c.Role,
	}
}
