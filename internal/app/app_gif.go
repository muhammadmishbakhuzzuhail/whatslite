package app

// app_gif.go — pencarian GIF lewat Tenor DARI SISI GO. WebKitGTK sering memblok
// fetch() lintas-asal (CORS) sehingga picker GIF kosong; menarik dari backend
// menghindari itu. FE cukup menampilkan <img src=preview> (lintas-asal OK) dan
// mengunduh mp4 saat dipilih via FetchRemoteMedia.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// GifDTO = satu hasil GIF (URL preview + URL mp4 utk dikirim).
type GifDTO struct {
	ID      string `json:"id"`
	Preview string `json:"preview"`
	Mp4     string `json:"mp4"`
}

// tenorKey = key demo publik anonim Tenor v1 (tak perlu pengguna daftar).
const tenorKey = "LIVDSRZULELA"

// SearchGifs mengembalikan GIF trending (query kosong) atau hasil pencarian.
func (a *App) SearchGifs(query string) []GifDTO {
	out := []GifDTO{}
	endpoint := "https://g.tenor.com/v1/trending"
	if query != "" {
		endpoint = "https://g.tenor.com/v1/search"
	}
	u := endpoint + "?key=" + tenorKey + "&limit=24&media_filter=minimal&contentfilter=high"
	if query != "" {
		u += "&q=" + url.QueryEscape(query)
	}
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return out
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; WhatsAppLite/1.0)")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return out
	}
	var body struct {
		Results []struct {
			ID    string `json:"id"`
			Media []map[string]struct {
				URL string `json:"url"`
			} `json:"media"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return out
	}
	for _, r := range body.Results {
		if len(r.Media) == 0 {
			continue
		}
		m := r.Media[0]
		preview := m["tinygif"].URL
		if preview == "" {
			preview = m["nanogif"].URL
		}
		mp4 := m["mp4"].URL
		if mp4 == "" {
			mp4 = m["tinymp4"].URL
		}
		if preview == "" || mp4 == "" {
			continue
		}
		out = append(out, GifDTO{ID: r.ID, Preview: preview, Mp4: mp4})
	}
	return out
}

// SearchStickers mengembalikan stiker TRANSPARAN dari Tenor (searchfilter=sticker)
// — picker stiker "Online" ala Discord. Preview = format kecil transparan; Mp4
// (dipakai sbg URL unduh) = webp/gif transparan penuh utk dikirim sbg stiker.
func (a *App) SearchStickers(query string) []GifDTO {
	out := []GifDTO{}
	endpoint := "https://g.tenor.com/v1/trending"
	if query != "" {
		endpoint = "https://g.tenor.com/v1/search"
	}
	u := endpoint + "?key=" + tenorKey + "&limit=24&searchfilter=sticker&contentfilter=high"
	if query != "" {
		u += "&q=" + url.QueryEscape(query)
	}
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return out
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; WhatsAppLite/1.0)")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return out
	}
	var body struct {
		Results []struct {
			ID    string `json:"id"`
			Media []map[string]struct {
				URL string `json:"url"`
			} `json:"media"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return out
	}
	for _, r := range body.Results {
		if len(r.Media) == 0 {
			continue
		}
		m := r.Media[0]
		preview := first(m["tinygif_transparent"].URL, m["nanogif_transparent"].URL, m["tinygif"].URL)
		full := first(m["webp_transparent"].URL, m["gif_transparent"].URL, m["png_transparent"].URL, m["gif"].URL)
		if preview == "" || full == "" {
			continue
		}
		out = append(out, GifDTO{ID: r.ID, Preview: preview, Mp4: full})
	}
	return out
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
