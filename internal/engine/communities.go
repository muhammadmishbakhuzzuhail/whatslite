// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package engine

// communities.go — WhatsApp Communities. Komunitas = grup dgn IsParent=true;
// di dalamnya ada sub-grup tertaut. Sub-grup yang kita ikuti dikenali dari
// LinkedParentJID pada GetJoinedGroups → tak perlu GetSubGroups per komunitas.

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
//
// SATU panggilan jaringan (GetJoinedGroups). Dulu juga memanggil GetSubGroups
// PER komunitas (= N IQ jaringan tambahan tiap daftar dibangun) — berat. Karena
// GetJoinedGroups sudah memuat SEMUA grup yang kita ikuti termasuk sub-grup,
// kita kelompokkan sub-grup ke induknya via LinkedParentJID secara lokal.
// Konsekuensi: hanya sub-grup yang DIIKUTI yang tampil (cukup untuk daftar; tap
// sub-grup = buka chat-nya). Sub-grup yang belum diikuti tak ditampilkan.
func (e *Engine) ListCommunities(ctx context.Context) ([]CommunityInfo, error) {
	gs, err := e.Client.GetJoinedGroups(ctx)
	if err != nil {
		return nil, err
	}
	children := map[string][]CommunitySubGroup{}
	parents := make([]*types.GroupInfo, 0, 4)
	for _, g := range gs {
		if g.IsParent {
			parents = append(parents, g)
			continue
		}
		if p := g.LinkedParentJID; !p.IsEmpty() {
			pj := p.String()
			children[pj] = append(children[pj], CommunitySubGroup{
				JID: g.JID.String(), Name: g.Name, IsDefault: g.IsDefaultSubGroup,
			})
		}
	}
	out := make([]CommunityInfo, 0, len(parents))
	for _, g := range parents {
		out = append(out, CommunityInfo{
			JID:         g.JID.String(),
			Name:        g.Name,
			Description: g.Topic,
			Groups:      children[g.JID.String()],
		})
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
