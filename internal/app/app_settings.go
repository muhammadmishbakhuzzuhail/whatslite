package app

// app_settings.go — setelan persisten ringan (app_meta). Saat ini: retensi pesan.

import (
	"encoding/base64"
	"strconv"

	qrcode "github.com/skip2/go-qrcode"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func atoiDef(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

// SetDefaultDisappearing menyetel timer hilang-otomatis default (detik; 0 = off).
func (a *App) SetDefaultDisappearing(seconds int) {
	if a.eng != nil && !a.emitErr(a.eng.SetDefaultDisappearing(a.ctx, seconds)) {
		runtime.EventsEmit(a.ctx, "wa:sync", "")
	}
}

// MyQR mengembalikan QR kontak sendiri sebagai PNG data-URI (revoke=buat ulang).
func (a *App) MyQR(revoke bool) string {
	if a.eng == nil {
		return ""
	}
	link, err := a.eng.MyQRLink(a.ctx, revoke)
	if err != nil || link == "" {
		if err != nil {
			runtime.EventsEmit(a.ctx, "wa:error", err.Error())
		}
		return ""
	}
	png, e := qrcode.Encode(link, qrcode.Medium, 320)
	if e != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}

// GetRetention mengembalikan jumlah hari retensi pesan (0 = selamanya).
func (a *App) GetRetention() int { return a.retentionDays }

// SetRetention menyetel retensi (hari; 0 = selamanya), simpan, lalu prune+VACUUM.
func (a *App) SetRetention(days int) {
	if days < 0 {
		days = 0
	}
	a.retentionDays = days
	if a.store == nil {
		return
	}
	_ = a.store.SetMeta(a.ctx, "retention_days", strconv.Itoa(days))
	a.bg(func() {
		if cut := a.retentionCutoff(); cut > 0 {
			if n, _ := a.store.PruneMessages(a.ctx, cut); n > 0 {
				_ = a.store.Vacuum(a.ctx)
			}
		}
		runtime.EventsEmit(a.ctx, "wa:sync", "")
	})
}
