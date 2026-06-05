package app

// app_notify.go — notifikasi desktop Linux (native) via `notify-send` (D-Bus
// org.freedesktop.Notifications di balik layar). Web Notification API tak andal
// di WebKitGTK, jadi pakai jalur native. Frontend yang memutuskan KAPAN
// (chat tak aktif / window tak fokus / tak dibisukan); BE cuma menampilkan.

import (
	"os/exec"
	"strings"
)

// Notify menampilkan notifikasi desktop. No-op bila notify-send tak ada.
func (a *App) Notify(title, body string) {
	bin, err := exec.LookPath("notify-send")
	if err != nil {
		return
	}
	if title == "" {
		title = "WhatsApp Lite"
	}
	// Potong body panjang biar rapi.
	if len(body) > 160 {
		body = body[:157] + "…"
	}
	body = strings.TrimSpace(body)
	go func() {
		_ = exec.Command(bin,
			"--app-name=WhatsApp Lite",
			"--icon=whatsapp-lite",
			"--expire-time=5000",
			title, body,
		).Run()
	}()
}
