package app

// app_groups.go — operasi grup untuk UI: info lengkap (anggota/admin/deskripsi),
// buat grup, keluar, ubah subjek, tambah/hapus/promosikan anggota.

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

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
	Participants []GroupMemberDTO `json:"participants"`
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
	out := &GroupInfoDTO{JID: gi.JID, Name: gi.Name, Topic: gi.Topic, Owner: gi.Owner}
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
