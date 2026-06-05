package app

// app_contacts.go — info kontak, blokir, dan profil akun sendiri.

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ProfileDTO = profil akun sendiri untuk UI.
type ProfileDTO struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	About string `json:"about"`
}

// GetProfile mengembalikan profil akun yang sedang login.
func (a *App) GetProfile() ProfileDTO {
	if a.eng == nil {
		return ProfileDTO{Name: "Saya"}
	}
	self := a.eng.SelfJID()
	name := a.eng.ChatName(self)
	if name == "" {
		name = "Saya"
	}
	phone := a.eng.ReadableID(self)
	if phone == "" {
		phone = shortJID(self)
	}
	return ProfileDTO{Name: name, Phone: phone, About: a.eng.ContactAbout(a.ctx, self)}
}

// GetContactAbout mengambil teks info/status seorang kontak.
func (a *App) GetContactAbout(jid string) string {
	if a.eng == nil {
		return ""
	}
	return a.eng.ContactAbout(a.ctx, jid)
}

// Block memblokir / membuka blokir kontak.
func (a *App) Block(jid string, block bool) {
	if a.eng == nil {
		return
	}
	if err := a.eng.Block(a.ctx, jid, block); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
	}
}

// SetMyName memperbarui nama tampil akun sendiri.
func (a *App) SetMyName(name string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.SetMyName(a.ctx, name); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
	}
}

// SetMyAbout memperbarui teks info/status akun sendiri.
func (a *App) SetMyAbout(text string) {
	if a.eng == nil {
		return
	}
	if err := a.eng.SetAbout(a.ctx, text); err != nil {
		runtime.EventsEmit(a.ctx, "wa:error", err.Error())
	}
}
