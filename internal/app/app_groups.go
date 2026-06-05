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
	for _, p := range gi.Participants {
		out.Participants = append(out.Participants, GroupMemberDTO{JID: p.JID, Name: p.Name, IsAdmin: p.IsAdmin})
	}
	return out
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
