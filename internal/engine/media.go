package engine

// media.go — foto profil (thumbnail) + unduh media penuh on-demand. Keduanya
// menghasilkan data-URI siap pakai untuk frontend.

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// picHTTP = klien khusus fetch foto CDN dgn timeout → koneksi macet tak bikin
// goroutine (mis. RequestPhotos × 8) menggantung selamanya.
var picHTTP = &http.Client{Timeout: 20 * time.Second}

// ProfilePictureRaw mengambil foto profil sebagai BYTES (utk di-cache ke FILE).
// (nil, nil) = tak ada foto (negatif). Dipanggil LAZY (avatar terlihat saja).
func (e *Engine) ProfilePictureRaw(jid string) ([]byte, error) {
	j, err := types.ParseJID(jid)
	if err != nil || !e.Client.IsConnected() {
		return nil, err
	}
	// Saluran (newsletter): foto ada di metadata thread, bukan IQ picture biasa.
	if j.Server == types.NewsletterServer {
		return e.newsletterPicRaw(j)
	}
	info, err := e.Client.GetProfilePictureInfo(context.Background(), j, &whatsmeow.GetProfilePictureParams{Preview: true})
	if err != nil || info == nil || info.URL == "" {
		return nil, nil // tak ada foto
	}
	resp, err := picHTTP.Get(info.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // maks 2MB
	if err != nil {
		return nil, err
	}
	return b, nil
}

// newsletterPicRaw mengambil foto profil saluran (newsletter) sbg bytes. Foto
// saluran tak lewat IQ picture; ada di NewsletterThreadMetadata (Picture/Preview).
func (e *Engine) newsletterPicRaw(j types.JID) ([]byte, error) {
	meta, err := e.Client.GetNewsletterInfo(context.Background(), j)
	if err != nil || meta == nil {
		return nil, err
	}
	url := ""
	if meta.ThreadMeta.Picture != nil && meta.ThreadMeta.Picture.URL != "" {
		url = meta.ThreadMeta.Picture.URL
	} else if meta.ThreadMeta.Preview.URL != "" {
		url = meta.ThreadMeta.Preview.URL
	}
	if url == "" {
		return nil, nil // tak ada foto
	}
	resp, err := picHTTP.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 2<<20))
}

// DownloadMediaRaw mengunduh media penuh dari proto pesan (base64) → (bytes, mime).
// Dipakai untuk meng-cache ke FILE (ringan) lalu disajikan via asset-server.
func (e *Engine) DownloadMediaRaw(protoB64 string) ([]byte, string, error) {
	raw, err := base64.StdEncoding.DecodeString(protoB64)
	if err != nil {
		return nil, "", err
	}
	var msg waE2E.Message
	if err := proto.Unmarshal(raw, &msg); err != nil {
		return nil, "", err
	}
	var dl whatsmeow.DownloadableMessage
	mime := "application/octet-stream"
	switch {
	case msg.GetImageMessage() != nil:
		dl = msg.GetImageMessage()
		mime = msg.GetImageMessage().GetMimetype()
	case msg.GetStickerMessage() != nil:
		dl = msg.GetStickerMessage()
		mime = "image/webp"
	case msg.GetVideoMessage() != nil:
		dl = msg.GetVideoMessage()
		mime = msg.GetVideoMessage().GetMimetype()
	case msg.GetAudioMessage() != nil:
		dl = msg.GetAudioMessage()
		mime = msg.GetAudioMessage().GetMimetype()
	default:
		return nil, "", fmt.Errorf("no downloadable media")
	}
	var data []byte
	err = retry(context.Background(), 3, func() (e2 error) { data, e2 = e.Client.Download(context.Background(), dl); return })
	if err != nil {
		return nil, "", err
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	return data, mime, nil
}

// DownloadMedia (lama) → data-URI. Dipertahankan untuk pemanggil non-file.
func (e *Engine) DownloadMedia(protoB64 string) (string, error) {
	data, mime, err := e.DownloadMediaRaw(protoB64)
	if err != nil {
		return "", err
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}
