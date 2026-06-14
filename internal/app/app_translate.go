// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

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
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Cache hasil terjemah (text|target → hasil) → hindari hit jaringan berulang
// (re-tap Translate / auto ?tr=1) + dodge rate-limit 429 senyap.
var (
	trMu    sync.Mutex
	trCache = map[string]string{}
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
	key := target + "|" + text
	trMu.Lock()
	if v, ok := trCache[key]; ok {
		trMu.Unlock()
		return v
	}
	trMu.Unlock()

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
	if resp.StatusCode != http.StatusOK { // 429/CAPTCHA → jangan cache, kembalikan asli
		return text
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return text
	}

	// Respons: [[["terjemahan","asli",...],...], null, "src_lang_terdeteksi", ...]
	var data []interface{}
	if err := json.Unmarshal(body, &data); err != nil || len(data) == 0 {
		return text
	}
	// Lewati round-trip sia-sia bila sumber == target (auto-detect).
	if len(data) > 2 {
		if det, ok := data[2].(string); ok && det != "" && det == target {
			trMu.Lock()
			trCache[key] = text
			trMu.Unlock()
			return text
		}
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
	out := html.UnescapeString(sb.String()) // gtx HTML-escape &/<>/' → kembalikan ke teks
	if out == "" {
		return text
	}
	trMu.Lock()
	if len(trCache) > 2000 { // cap kasar agar tak tumbuh tanpa batas
		trCache = map[string]string{}
	}
	trCache[key] = out
	trMu.Unlock()
	return out
}
