// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package engine

// communities.go — WhatsApp Communities. Komunitas = grup dgn IsParent=true;
// di dalamnya ada sub-grup tertaut. whatsmeow: GetJoinedGroups (filter IsParent)
// + GetSubGroups untuk anggota grupnya.

import (
	"context"

	"go.mau.fi/whatsmeow/types"
)

// CommunitySubGroup = satu grup di dalam komunitas.
type CommunitySubGroup struct {
	JID       string
	Name      string
	IsDefault bool // grup pengumuman default
}

// CommunityInfo = komunitas + sub-grupnya.
type CommunityInfo struct {
	JID         string
	Name        string
	Description string
	Groups      []CommunitySubGroup
}

// ListCommunities mengembalikan komunitas yang diikuti beserta sub-grupnya.
func (e *Engine) ListCommunities(ctx context.Context) ([]CommunityInfo, error) {
	gs, err := e.Client.GetJoinedGroups(ctx)
	if err != nil {
		return nil, err
	}
	out := []CommunityInfo{}
	for _, g := range gs {
		if !g.IsParent {
			continue
		}
		ci := CommunityInfo{
			JID:         g.JID.String(),
			Name:        g.Name,
			Description: g.Topic,
		}
		subs, err := e.Client.GetSubGroups(ctx, g.JID)
		if err == nil {
			for _, s := range subs {
				ci.Groups = append(ci.Groups, CommunitySubGroup{
					JID: s.JID.String(), Name: s.Name, IsDefault: s.IsDefaultSubGroup,
				})
			}
		}
		out = append(out, ci)
	}
	return out, nil
}

// LeaveCommunity keluar dari komunitas (keluar grup induk).
func (e *Engine) LeaveCommunity(ctx context.Context, jid string) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	return e.Client.LeaveGroup(ctx, j)
}
