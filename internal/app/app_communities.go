// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_communities.go — WhatsApp Communities: daftar komunitas + sub-grup.

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// CommunitySubDTO = sub-grup di dalam komunitas.
type CommunitySubDTO struct {
	JID       string `json:"jid"`
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault"`
}

// CommunityDTO = komunitas + sub-grupnya.
type CommunityDTO struct {
	JID         string            `json:"jid"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Groups      []CommunitySubDTO `json:"groups"`
}

// GetCommunities mengembalikan komunitas yang diikuti beserta sub-grupnya.
func (a *App) GetCommunities() (out []CommunityDTO) {
	out = []CommunityDTO{}
	if a.eng == nil {
		return
	}
	cs, err := a.eng.ListCommunities(a.ctx)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	for _, c := range cs {
		d := CommunityDTO{JID: c.JID, Name: c.Name, Description: c.Description}
		for _, g := range c.Groups {
			d.Groups = append(d.Groups, CommunitySubDTO{JID: g.JID, Name: g.Name, IsDefault: g.IsDefault})
		}
		out = append(out, d)
	}
	return
}

// LeaveCommunity keluar dari komunitas.
func (a *App) LeaveCommunity(jid string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.LeaveCommunity(a.ctx, jid); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
	}
}
