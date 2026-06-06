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
	Announce     bool // true = hanya admin boleh kirim
	Locked       bool // true = hanya admin boleh ubah info grup
	JoinApproval bool // true = anggota baru butuh persetujuan admin
	AdminAddOnly bool // true = hanya admin boleh menambah anggota
}

// GroupJoinRequest = satu permintaan bergabung (menunggu persetujuan admin).
type GroupJoinRequest struct {
	JID  string
	Name string
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
		Announce: info.IsAnnounce, Locked: info.IsLocked, JoinApproval: info.IsJoinApprovalRequired,
		AdminAddOnly: info.MemberAddMode == types.GroupMemberAddModeAdmin,
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

// SetGroupDescription mengubah deskripsi (topic) grup.
func (e *Engine) SetGroupDescription(ctx context.Context, jid, topic string) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	return e.Client.SetGroupTopic(ctx, j, "", "", topic)
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

// JoinGroupViaLink bergabung ke grup via kode/tautan undangan; kembalikan JID.
func (e *Engine) JoinGroupViaLink(ctx context.Context, code string) (string, error) {
	j, err := e.Client.JoinGroupWithLink(ctx, code)
	if err != nil {
		return "", err
	}
	return j.String(), nil
}

// GroupPreviewFromLink mengembalikan nama grup dari tautan (pratinjau sblm gabung).
func (e *Engine) GroupPreviewFromLink(ctx context.Context, code string) (string, error) {
	gi, err := e.Client.GetGroupInfoFromLink(ctx, code)
	if err != nil {
		return "", err
	}
	return gi.Name, nil
}

// GroupInviteLink mengambil tautan undangan grup (reset=true buat tautan baru).
func (e *Engine) GroupInviteLink(ctx context.Context, jid string, reset bool) (string, error) {
	j, err := types.ParseJID(jid)
	if err != nil {
		return "", err
	}
	return e.Client.GetGroupInviteLink(ctx, j, reset)
}

// SetGroupAnnounce: true = hanya admin boleh kirim pesan.
func (e *Engine) SetGroupAnnounce(ctx context.Context, jid string, on bool) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	return e.Client.SetGroupAnnounce(ctx, j, on)
}

// SetGroupLocked: true = hanya admin boleh ubah info grup.
func (e *Engine) SetGroupLocked(ctx context.Context, jid string, on bool) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	return e.Client.SetGroupLocked(ctx, j, on)
}

// SetGroupJoinApproval: true = anggota baru butuh persetujuan admin.
func (e *Engine) SetGroupJoinApproval(ctx context.Context, jid string, on bool) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	return e.Client.SetGroupJoinApprovalMode(ctx, j, on)
}

// SetGroupAddMode: adminOnly=true → hanya admin tambah anggota.
func (e *Engine) SetGroupAddMode(ctx context.Context, jid string, adminOnly bool) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	mode := types.GroupMemberAddModeAllMember
	if adminOnly {
		mode = types.GroupMemberAddModeAdmin
	}
	return e.Client.SetGroupMemberAddMode(ctx, j, mode)
}

// GroupRequests daftar permintaan bergabung yang menunggu persetujuan.
func (e *Engine) GroupRequests(ctx context.Context, jid string) ([]GroupJoinRequest, error) {
	j, err := types.ParseJID(jid)
	if err != nil {
		return nil, err
	}
	reqs, err := e.Client.GetGroupRequestParticipants(ctx, j)
	if err != nil {
		return nil, err
	}
	out := make([]GroupJoinRequest, 0, len(reqs))
	for _, r := range reqs {
		name := e.ChatName(r.JID.String())
		if name == "" {
			name = e.ReadableID(r.JID.String())
		}
		out = append(out, GroupJoinRequest{JID: r.JID.String(), Name: name})
	}
	return out, nil
}

// UpdateGroupRequest menyetujui/menolak permintaan bergabung. approve=false → tolak.
func (e *Engine) UpdateGroupRequest(ctx context.Context, jid string, members []string, approve bool) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	parts := make([]types.JID, 0, len(members))
	for _, m := range members {
		if mj, err := types.ParseJID(m); err == nil {
			parts = append(parts, mj)
		}
	}
	action := whatsmeow.ParticipantChangeReject
	if approve {
		action = whatsmeow.ParticipantChangeApprove
	}
	_, err = e.Client.UpdateGroupRequestParticipants(ctx, j, parts, action)
	return err
}

// SetGroupPhoto mengganti foto grup (avatar = JPEG bytes; nil = hapus).
func (e *Engine) SetGroupPhoto(ctx context.Context, jid string, avatar []byte) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	_, err = e.Client.SetGroupPhoto(ctx, j, avatar)
	return err
}
