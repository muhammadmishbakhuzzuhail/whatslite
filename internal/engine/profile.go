// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

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

// SetOwnPhoto menyetel foto profil sendiri (full + preview JPEG; nil = hapus).
// WhatsApp resmi mengirim DUA node (image 640² + preview ~96²); hanya image saja
// kadang tak memunculkan avatar kecil → sertakan keduanya.
func (e *Engine) SetOwnPhoto(ctx context.Context, full, preview []byte) error {
	if e.Client.Store.ID == nil {
		return errors.New("belum login")
	}
	var content interface{}
	if full != nil {
		nodes := []waBinary.Node{{Tag: "picture", Attrs: waBinary.Attrs{"type": "image"}, Content: full}}
		if preview != nil {
			nodes = append(nodes, waBinary.Node{Tag: "picture", Attrs: waBinary.Attrs{"type": "preview"}, Content: preview})
		}
		content = nodes
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
