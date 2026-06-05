package engine

// groups.go — operasi grup: daftar grup diikuti, info lengkap (anggota/admin),
// buat grup, keluar, ubah subjek, tambah/hapus/promosikan anggota.

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// GroupSummary = ringkasan grup yang diikuti.
type GroupSummary struct {
	JID  string
	Name string
}

// JoinedGroups mengambil daftar grup yang diikuti (butuh koneksi) dan meng-cache namanya.
func (e *Engine) JoinedGroups(ctx context.Context) ([]GroupSummary, error) {
	gs, err := e.Client.GetJoinedGroups(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]GroupSummary, 0, len(gs))
	for _, g := range gs {
		jid := g.JID.String()
		e.mu.Lock()
		e.groupNames[jid] = g.Name
		e.mu.Unlock()
		out = append(out, GroupSummary{JID: jid, Name: g.Name})
	}
	return out, nil
}

// GroupSubtitle = "N anggota" (butuh koneksi; panggilan ringan, tak di-cache).
func (e *Engine) GroupSubtitle(jid string) string {
	j, err := types.ParseJID(jid)
	if err != nil || !e.Client.IsConnected() {
		return ""
	}
	info, err := e.Client.GetGroupInfo(context.Background(), j)
	if err != nil || info == nil {
		return ""
	}
	return fmt.Sprintf("%d anggota", len(info.Participants))
}

// GroupMember = satu anggota grup (sudah disederhanakan untuk frontend).
type GroupMember struct {
	JID     string
	Name    string
	IsAdmin bool
}

// GroupFullInfo = detail grup untuk panel info.
type GroupFullInfo struct {
	JID          string
	Name         string
	Topic        string
	Owner        string
	Participants []GroupMember
}

// GroupInfo mengambil detail grup (subjek, deskripsi, anggota, admin).
func (e *Engine) GroupInfo(ctx context.Context, jid string) (*GroupFullInfo, error) {
	j, err := types.ParseJID(jid)
	if err != nil {
		return nil, err
	}
	info, err := e.Client.GetGroupInfo(ctx, j)
	if err != nil {
		return nil, err
	}
	out := &GroupFullInfo{
		JID: jid, Name: info.Name, Topic: info.Topic, Owner: info.OwnerJID.String(),
	}
	for _, p := range info.Participants {
		name := e.ChatName(p.JID.String())
		if name == "" {
			name = e.ReadableID(p.JID.String())
		}
		out.Participants = append(out.Participants, GroupMember{
			JID: p.JID.String(), Name: name, IsAdmin: p.IsAdmin || p.IsSuperAdmin,
		})
	}
	return out, nil
}

// CreateGroup membuat grup baru; kembalikan JID grup.
func (e *Engine) CreateGroup(ctx context.Context, name string, participants []string) (string, error) {
	parts := make([]types.JID, 0, len(participants))
	for _, p := range participants {
		if j, err := types.ParseJID(p); err == nil {
			parts = append(parts, j)
		}
	}
	info, err := e.Client.CreateGroup(ctx, whatsmeow.ReqCreateGroup{Name: name, Participants: parts})
	if err != nil {
		return "", err
	}
	return info.JID.String(), nil
}

// LeaveGroup keluar dari grup.
func (e *Engine) LeaveGroup(ctx context.Context, jid string) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	return e.Client.LeaveGroup(ctx, j)
}

// SetGroupSubject mengubah subjek (nama) grup.
func (e *Engine) SetGroupSubject(ctx context.Context, jid, name string) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	return e.Client.SetGroupName(ctx, j, name)
}

// UpdateParticipants menambah/menghapus/mempromosikan/menurunkan anggota grup.
// action: "add" | "remove" | "promote" | "demote".
func (e *Engine) UpdateParticipants(ctx context.Context, jid string, members []string, action string) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	var pc whatsmeow.ParticipantChange
	switch action {
	case "add":
		pc = whatsmeow.ParticipantChangeAdd
	case "remove":
		pc = whatsmeow.ParticipantChangeRemove
	case "promote":
		pc = whatsmeow.ParticipantChangePromote
	case "demote":
		pc = whatsmeow.ParticipantChangeDemote
	default:
		return fmt.Errorf("aksi peserta tak dikenal: %q", action)
	}
	parts := make([]types.JID, 0, len(members))
	for _, m := range members {
		if mj, err := types.ParseJID(m); err == nil {
			parts = append(parts, mj)
		}
	}
	_, err = e.Client.UpdateGroupParticipants(ctx, j, parts, pc)
	return err
}
