package app

// app_translate.go — terjemah pesan on-demand (fitur "Terjemahkan" di menu chat,
// ala Twitter). Memakai endpoint Google Translate publik (gtx) — tanpa API key.
//
// CATATAN PRIVASI: teks pesan dikirim ke server Google. Untuk pesan E2E ini
// berarti isi keluar dari perangkat. Pengganti privat (offline) di masa depan:
// Bergamot/Argos (NMT on-device) — ganti isi Translate() saja, kontrak tetap.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Translate menerjemahkan text ke bahasa target (kode ISO: "id","en","es",…).
// Sumber dideteksi otomatis. Kembalikan teks asli bila gagal.
func (a *App) Translate(text, target string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}
	if target == "" {
		target = "en"
	}
	endpoint := "https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl=" +
		url.QueryEscape(target) + "&dt=t&q=" + url.QueryEscape(text)

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return text
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return text
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return text
	}

	// Respons: [[["terjemahan","asli",...],...], ...]
	var data []interface{}
	if err := json.Unmarshal(body, &data); err != nil || len(data) == 0 {
		return text
	}
	segs, ok := data[0].([]interface{})
	if !ok {
		return text
	}
	var sb strings.Builder
	for _, s := range segs {
		p, ok := s.([]interface{})
		if !ok || len(p) == 0 {
			continue
		}
		if t, ok := p[0].(string); ok {
			sb.WriteString(t)
		}
	}
	out := sb.String()
	if out == "" {
		return text
	}
	return out
}
