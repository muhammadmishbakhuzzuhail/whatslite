package engine

// profile.go — ganti foto profil SENDIRI. whatsmeow tak punya wrapper khusus
// (hanya SetGroupPhoto), TAPI protokolnya identik: IQ set "w:profile:picture"
// dengan Target = JID kita sendiri. Kita kirim lewat DangerousInternals().SendIQ
// (alias DangerousInfoQuery diekspos) → tanpa perlu fork whatsmeow.

import (
	"context"
	"errors"

	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

// SetOwnPhoto menyetel foto profil sendiri (avatar = JPEG; nil = hapus).
func (e *Engine) SetOwnPhoto(ctx context.Context, avatar []byte) error {
	if e.Client.Store.ID == nil {
		return errors.New("belum login")
	}
	var content interface{}
	if avatar != nil {
		content = []waBinary.Node{{
			Tag:     "picture",
			Attrs:   waBinary.Attrs{"type": "image"},
			Content: avatar,
		}}
	}
	_, err := e.Client.DangerousInternals().SendIQ(ctx, whatsmeow.DangerousInfoQuery{
		Namespace: "w:profile:picture",
		Type:      "set", // untyped const → infoQueryType (string)
		To:        types.ServerJID,
		Target:    e.Client.Store.ID.ToNonAD(),
		Content:   content,
	})
	return err
}
