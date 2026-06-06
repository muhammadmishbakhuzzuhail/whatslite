package app

// app_groups.go — operasi grup untuk UI: info lengkap (anggota/admin/deskripsi),
// buat grup, keluar, ubah subjek, tambah/hapus/promosikan anggota.

import (
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// inviteCode mengekstrak kode dari tautan undangan grup penuh atau kode mentah.
func inviteCode(link string) string {
	s := strings.TrimSpace(link)
	for _, p := range []string{"https://chat.whatsapp.com/", "http://chat.whatsapp.com/", "chat.whatsapp.com/"} {
		s = strings.TrimPrefix(s, p)
	}
	return strings.TrimSpace(s)
}

// JoinGroupLink bergabung ke grup via tautan undangan; kembalikan JID (atau "").
func (a *App) JoinGroupLink(link string) string {
	if a.eng == nil {
		return ""
	}
	jid, err := a.eng.JoinGroupViaLink(a.ctx, inviteCode(link))
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	runtime.EventsEmit(a.ctx, "wa:sync", "")
	return jid
}

// PreviewGroupLink mengembalikan nama grup dari tautan (pratinjau sebelum gabung).
func (a *App) PreviewGroupLink(link string) string {
	if a.eng == nil {
		return ""
	}
	name, err := a.eng.GroupPreviewFromLink(a.ctx, inviteCode(link))
	if err != nil {
		return ""
	}
	return name
}

// GroupMemberDTO / GroupInfoDTO = detail grup untuk panel info.
type GroupMemberDTO struct {
	JID     string `json:"jid"`
	Name    string `json:"name"`
	IsAdmin bool   `json:"isAdmin"`
}

type GroupInfoDTO struct {
	JID          string           `json:"jid"`
	Name         string           `json:"name"`
	Topic        string           `json:"topic"`
	Owner        string           `json:"owner"`
	AmAdmin      bool             `json:"amAdmin"` // saya admin grup ini?
	Announce     bool             `json:"announce"`
	Locked       bool             `json:"locked"`
	JoinApproval bool             `json:"joinApproval"`
	AdminAddOnly bool             `json:"adminAddOnly"`
	Participants []GroupMemberDTO `json:"participants"`
}

// GroupRequestDTO = permintaan bergabung yang menunggu persetujuan.
type GroupRequestDTO struct {
	JID  string `json:"jid"`
	Name string `json:"name"`
}

// GetGroupInfo mengambil detail grup (subjek, deskripsi, anggota).
func (a *App) GetGroupInfo(jid string) *GroupInfoDTO {
	if a.eng == nil {
		return nil
	}
	gi, err := a.eng.GroupInfo(a.ctx, jid)
	if err != nil || gi == nil {
		return nil
	}
	out := &GroupInfoDTO{
		JID: gi.JID, Name: gi.Name, Topic: gi.Topic, Owner: gi.Owner,
		Announce: gi.Announce, Locked: gi.Locked, JoinApproval: gi.JoinApproval, AdminAddOnly: gi.AdminAddOnly,
	}
	self := userPart(a.eng.SelfJID())
	for _, p := range gi.Participants {
		out.Participants = append(out.Participants, GroupMemberDTO{JID: p.JID, Name: p.Name, IsAdmin: p.IsAdmin})
		if p.IsAdmin && self != "" && userPart(p.JID) == self {
			out.AmAdmin = true
		}
	}
	return out
}

// GroupInviteLink mengambil tautan undangan grup.
func (a *App) GroupInviteLink(jid string, reset bool) string {
	if a.eng == nil {
		return ""
	}
	link, err := a.eng.GroupInviteLink(a.ctx, jid, reset)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	return link
}

// SetGroupPhoto mengganti foto grup dari data-URI gambar.
func (a *App) SetGroupPhoto(jid, dataURI string) {
	if a.eng == nil {
		return
	}
	_, data, err := decodeDataURI(dataURI)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	if err := a.eng.SetGroupPhoto(a.ctx, jid, data); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	runtime.EventsEmit(a.ctx, "wa:sync", "")
}

// CreateGroup membuat grup baru; kembalikan JID grup (atau "").
func (a *App) CreateGroup(name string, participants []string) string {
	if a.eng == nil {
		return ""
	}
	jid, err := a.eng.CreateGroup(a.ctx, name, participants)
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return ""
	}
	runtime.EventsEmit(a.ctx, "wa:sync", "")
	return jid
}

// LeaveGroup keluar dari grup.
func (a *App) LeaveGroup(jid string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.LeaveGroup(a.ctx, jid); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	if a.store != nil {
		_ = a.store.SetArchived(a.ctx, jid, true)
	}
	runtime.EventsEmit(a.ctx, "wa:sync", "")
}

// SetGroupSubject mengubah subjek (nama) grup.
func (a *App) SetGroupSubject(jid, name string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.SetGroupSubject(a.ctx, jid, name); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	if a.store != nil {
		_ = a.store.SetChatName(a.ctx, jid, name)
	}
	runtime.EventsEmit(a.ctx, "wa:sync", "")
}

// SetGroupDescription mengubah deskripsi grup.
func (a *App) SetGroupDescription(jid, topic string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.SetGroupDescription(a.ctx, jid, topic); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return
	}
	runtime.EventsEmit(a.ctx, "wa:sync", "")
}

// UpdateGroupParticipants menambah/menghapus/mempromosikan/menurunkan anggota.
// action: "add" | "remove" | "promote" | "demote".
func (a *App) UpdateGroupParticipants(jid string, members []string, action string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.UpdateParticipants(a.ctx, jid, members, action); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
	}
}

// --- Setelan admin grup ---
func (a *App) emitErr(err error) bool {
	if err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		return true
	}
	return false
}

// SetGroupAnnounce: true = hanya admin boleh kirim.
func (a *App) SetGroupAnnounce(jid string, on bool) {
	if a.eng != nil && !a.emitErr(a.eng.SetGroupAnnounce(a.ctx, jid, on)) {
		runtime.EventsEmit(a.ctx, "wa:sync", "")
	}
}

// SetGroupLocked: true = hanya admin boleh ubah info.
func (a *App) SetGroupLocked(jid string, on bool) {
	if a.eng != nil && !a.emitErr(a.eng.SetGroupLocked(a.ctx, jid, on)) {
		runtime.EventsEmit(a.ctx, "wa:sync", "")
	}
}

// SetGroupJoinApproval: true = anggota baru butuh persetujuan.
func (a *App) SetGroupJoinApproval(jid string, on bool) {
	if a.eng != nil && !a.emitErr(a.eng.SetGroupJoinApproval(a.ctx, jid, on)) {
		runtime.EventsEmit(a.ctx, "wa:sync", "")
	}
}

// SetGroupAddMode: adminOnly=true → hanya admin tambah anggota.
func (a *App) SetGroupAddMode(jid string, adminOnly bool) {
	if a.eng != nil && !a.emitErr(a.eng.SetGroupAddMode(a.ctx, jid, adminOnly)) {
		runtime.EventsEmit(a.ctx, "wa:sync", "")
	}
}

// GetGroupRequests daftar permintaan bergabung menunggu persetujuan.
func (a *App) GetGroupRequests(jid string) []GroupRequestDTO {
	out := []GroupRequestDTO{}
	if a.eng == nil {
		return out
	}
	reqs, err := a.eng.GroupRequests(a.ctx, jid)
	if err != nil {
		return out
	}
	for _, r := range reqs {
		out = append(out, GroupRequestDTO{JID: r.JID, Name: r.Name})
	}
	return out
}

// UpdateGroupRequest menyetujui/menolak permintaan bergabung.
func (a *App) UpdateGroupRequest(jid string, members []string, approve bool) {
	if a.eng != nil && !a.emitErr(a.eng.UpdateGroupRequest(a.ctx, jid, members, approve)) {
		runtime.EventsEmit(a.ctx, "wa:sync", "")
	}
}
