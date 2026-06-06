package app

// app_gif.go — pencarian GIF & stiker lewat Tenor DARI SISI GO. WebKitGTK sering
// memblok fetch() lintas-asal (CORS) sehingga picker kosong; menarik dari backend
// menghindari itu. FE menampilkan <img src=preview> lalu mengunduh media penuh
// saat dipilih via FetchRemoteMedia.
//
// Pagination: Tenor mengembalikan kursor `next` (string). FE kirim balik sbg
// `pos` utk halaman berikutnya → infinite scroll (bukan lagi cap 24 sekali muat).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GifDTO = satu hasil (URL preview + URL media penuh utk dikirim).
type GifDTO struct {
	ID      string `json:"id"`
	Preview string `json:"preview"`
	Mp4     string `json:"mp4"`
}

// GifPage = satu halaman hasil + kursor `next` (kosong = habis).
type GifPage struct {
	Items []GifDTO `json:"items"`
	Next  string   `json:"next"`
}

// tenorKey = key demo publik anonim Tenor v1 (tak perlu pengguna daftar).
const tenorKey = "LIVDSRZULELA"
const tenorLimit = 50 // maks per halaman (Tenor)

var tenorHTTP = &http.Client{Timeout: 15 * time.Second}

// tenorResp = bentuk respons Tenor v1 yang kita pakai (results + next).
type tenorResp struct {
	Next    string `json:"next"`
	Results []struct {
		ID    string `json:"id"`
		Media []map[string]struct {
			URL string `json:"url"`
		} `json:"media"`
	} `json:"results"`
}

// tenorFetch menjalankan satu query Tenor (trending bila query kosong) dgn
// extra param (mis. searchfilter=sticker) + kursor pos. Mengembalikan respons mentah.
func (a *App) tenorFetch(query, pos, extra string) (*tenorResp, bool) {
	endpoint := "https://g.tenor.com/v1/trending"
	if query != "" {
		endpoint = "https://g.tenor.com/v1/search"
	}
	u := endpoint + "?key=" + tenorKey + "&limit=" + itoa(tenorLimit) + "&contentfilter=high" + extra
	if query != "" {
		u += "&q=" + url.QueryEscape(query)
	}
	if pos != "" {
		u += "&pos=" + url.QueryEscape(pos)
	}
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; WhatsAppLite/1.0)")
	resp, err := tenorHTTP.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}
	var body tenorResp
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, false
	}
	return &body, true
}

// SearchGifs mengembalikan satu halaman GIF (trending / hasil cari) + kursor next.
func (a *App) SearchGifs(query, pos string) GifPage {
	if k := a.klipyKey(); k != "" { // KLIPY (Tenor v1 sunset Jun 2026) bila key di-set
		return a.klipySearch(k, "gifs", query, pos)
	}
	page := GifPage{Items: []GifDTO{}}
	body, ok := a.tenorFetch(query, pos, "&media_filter=minimal")
	if !ok {
		return page
	}
	page.Next = body.Next
	for _, r := range body.Results {
		if len(r.Media) == 0 {
			continue
		}
		m := r.Media[0]
		preview := first(m["tinygif"].URL, m["nanogif"].URL)
		mp4 := first(m["mp4"].URL, m["tinymp4"].URL)
		if preview == "" || mp4 == "" {
			continue
		}
		page.Items = append(page.Items, GifDTO{ID: r.ID, Preview: preview, Mp4: mp4})
	}
	return page
}

// SearchStickers mengembalikan satu halaman stiker TRANSPARAN (searchfilter=sticker)
// + kursor next. Preview = format kecil transparan; Mp4 (URL unduh) = webp/gif
// transparan penuh utk dikirim sbg stiker.
func (a *App) SearchStickers(query, pos string) GifPage {
	// Trending → Stickerly (library besar, PNG/webp 512² transparan).
	if strings.TrimSpace(query) == "" {
		if p := a.stickerlyTrending(pos); len(p.Items) > 0 {
			return p
		}
	}
	// Cari kata kunci / fallback → KLIPY lalu Tenor.
	if k := a.klipyKey(); k != "" {
		return a.klipySearch(k, "stickers", query, pos)
	}
	page := GifPage{Items: []GifDTO{}}
	body, ok := a.tenorFetch(query, pos, "&searchfilter=sticker")
	if !ok {
		return page
	}
	page.Next = body.Next
	for _, r := range body.Results {
		if len(r.Media) == 0 {
			continue
		}
		m := r.Media[0]
		// HANYA format transparan (stiker tanpa background).
		preview := first(m["tinygif_transparent"].URL, m["nanogif_transparent"].URL)
		full := first(m["webp_transparent"].URL, m["gif_transparent"].URL, m["png_transparent"].URL)
		if preview == "" || full == "" {
			continue
		}
		page.Items = append(page.Items, GifDTO{ID: r.ID, Preview: preview, Mp4: full})
	}
	return page
}

// first mengembalikan argumen non-kosong pertama.
func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// itoa kecil tanpa import strconv di banyak tempat.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// ============================ KLIPY ============================
// Tenor v1 dijadwalkan SHUTDOWN ~Jun 2026; Giphy berbayar. KLIPY = alternatif
// gratis (key gratis, library besar, Tenor-compatible) yg dipakai WhatsApp.
// Key dibaca dari env KLIPY_API_KEY ATAU file <dataDir>/klipy.key (TIDAK
// di-commit ke repo). Kosong → fallback Tenor (di atas).
func (a *App) klipyKey() string {
	if k := strings.TrimSpace(os.Getenv("KLIPY_API_KEY")); k != "" {
		return k
	}
	if a.mediaDir != "" {
		if b, err := os.ReadFile(filepath.Join(filepath.Dir(a.mediaDir), "klipy.key")); err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}

// klipySearch: GET api.klipy.com/api/v1/{key}/{kind}/{search|trending}. kind =
// "gifs" | "stickers". pos = nomor halaman (string). Respons defensif: kumpulkan
// semua URL di tiap item lalu pilih mp4/gif/webp (bentuk `files` tak terdokumentasi
// publik → parse rekursif, tahan perubahan skema).
func (a *App) klipySearch(key, kind, query, pos string) GifPage {
	page := GifPage{Items: []GifDTO{}}
	p := 1
	if n, err := strconv.Atoi(pos); err == nil && n > 0 {
		p = n
	}
	mode := "trending"
	if strings.TrimSpace(query) != "" {
		mode = "search"
	}
	u := "https://api.klipy.com/api/v1/" + url.PathEscape(key) + "/" + kind + "/" + mode +
		"?per_page=50&page=" + itoa(p)
	if mode == "search" {
		u += "&q=" + url.QueryEscape(query)
	}
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return page
	}
	resp, err := tenorHTTP.Do(req)
	if err != nil {
		return page
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return page
	}
	var body struct {
		Data struct {
			Data    []map[string]any `json:"data"`
			HasNext bool             `json:"has_next"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return page
	}
	if body.Data.HasNext {
		page.Next = itoa(p + 1)
	}
	for i, it := range body.Data.Data {
		// Bentuk: file.{hd|md|sm|xs}.{gif|webp|jpg|mp4|webm|png}.url
		f, _ := it["file"].(map[string]any)
		pick := func(fmt string, tiers ...string) string {
			for _, t := range tiers {
				tm, _ := f[t].(map[string]any)
				fm, _ := tm[fmt].(map[string]any)
				if u, ok := fm["url"].(string); ok && u != "" {
					return u
				}
			}
			return ""
		}
		var preview, full string
		if kind == "stickers" { // transparan
			preview = first(pick("webp", "sm", "md", "xs"), pick("gif", "sm", "md", "xs"))
			full = first(pick("webp", "hd", "md", "sm"), pick("png", "hd", "md"), pick("gif", "hd", "md"))
		} else {
			preview = first(pick("gif", "sm", "md", "xs"), pick("webp", "sm", "md")) // thumbnail kecil
			full = first(pick("mp4", "hd", "md", "sm"), pick("gif", "hd", "md"))     // dikirim
		}
		if full == "" {
			full = preview
		}
		if preview == "" || full == "" {
			continue
		}
		id, _ := it["id"].(string)
		if id == "" {
			id = mode + itoa(p) + "_" + itoa(i)
		}
		page.Items = append(page.Items, GifDTO{ID: id, Preview: preview, Mp4: full})
	}
	return page
}

// ============================ STICKERLY (unofficial) ============================
// Sumber stiker BESAR (PNG/webp 512² transparan) via API tak-resmi sticker.ly.
// Endpoint `recommend` balikin ratusan pack sekaligus → di-flatten + cache 10mnt,
// disajikan per-halaman ke picker. Catatan: tak resmi → bisa berubah/putus
// sewaktu-waktu (fallback KLIPY/Tenor tetap ada).
const stickerlyUA = "androidapp.stickerly/3.14.1 (Redmi Note 8; U; Android 11; en; brand/Redmi;)"

var (
	stickerlyMu    sync.Mutex
	stickerlyCache []GifDTO
	stickerlyAt    time.Time
)

// stickerlyTrending mengembalikan satu halaman (60) dari daftar stiker trending
// Stickerly yg sudah di-flatten + cache.
func (a *App) stickerlyTrending(pos string) GifPage {
	const per = 60
	p := 1
	if n, err := strconv.Atoi(pos); err == nil && n > 0 {
		p = n
	}
	list := a.stickerlyList()
	page := GifPage{Items: []GifDTO{}}
	start := (p - 1) * per
	if start >= len(list) {
		return page
	}
	end := start + per
	if end > len(list) {
		end = len(list)
	}
	page.Items = list[start:end]
	if end < len(list) {
		page.Next = itoa(p + 1)
	}
	return page
}

// stickerlyList → daftar stiker (cache 10mnt). Tiap stiker: url = prefix+file.
func (a *App) stickerlyList() []GifDTO {
	stickerlyMu.Lock()
	if len(stickerlyCache) > 0 && time.Since(stickerlyAt) < 10*time.Minute {
		defer stickerlyMu.Unlock()
		return stickerlyCache
	}
	stickerlyMu.Unlock()

	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.sticker.ly/v4/stickerPack/recommend", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", stickerlyUA)
	resp, err := tenorHTTP.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var body struct {
		Result struct {
			StickerPacks []struct {
				ResourceURLPrefix string   `json:"resourceUrlPrefix"`
				ResourceFiles     []string `json:"resourceFiles"`
			} `json:"stickerPacks"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil
	}
	var out []GifDTO
	for pi, pk := range body.Result.StickerPacks {
		for fi, f := range pk.ResourceFiles {
			u := pk.ResourceURLPrefix + f
			out = append(out, GifDTO{ID: "sly_" + itoa(pi) + "_" + itoa(fi), Preview: u, Mp4: u})
		}
	}
	if len(out) > 0 {
		stickerlyMu.Lock()
		stickerlyCache = out
		stickerlyAt = time.Now()
		stickerlyMu.Unlock()
	}
	return out
}
