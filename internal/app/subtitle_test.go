// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

import (
	"testing"
	"time"
)

// ChatSubtitle: prioritas mengetik/merekam DI ATAS presence; typing off → presence.
func TestChatSubtitle(t *testing.T) {
	a := &App{}
	const jid = "x@s.whatsapp.net"

	if got := a.ChatSubtitle(jid); got != "" {
		t.Errorf("kosong awal = %q want \"\"", got)
	}

	a.setPresence(jid, "online")
	if got := a.ChatSubtitle(jid); got != "online" {
		t.Errorf("presence = %q want online", got)
	}

	a.setTyping(jid, typingStateT{on: true})
	if got := a.ChatSubtitle(jid); got != "mengetik…" {
		t.Errorf("typing = %q want mengetik…", got)
	}

	a.setTyping(jid, typingStateT{on: true, who: "Budi"})
	if got := a.ChatSubtitle(jid); got != "Budi sedang mengetik…" {
		t.Errorf("typing grup = %q", got)
	}

	a.setTyping(jid, typingStateT{on: true, rec: true})
	if got := a.ChatSubtitle(jid); got != "merekam audio…" {
		t.Errorf("recording = %q want merekam audio…", got)
	}

	a.setTyping(jid, typingStateT{on: true, rec: true, who: "Citra"})
	if got := a.ChatSubtitle(jid); got != "Citra sedang merekam audio…" {
		t.Errorf("recording grup = %q", got)
	}

	// typing berhenti → kembali ke presence.
	a.setTyping(jid, typingStateT{on: false})
	if got := a.ChatSubtitle(jid); got != "online" {
		t.Errorf("typing off = %q want online (presence)", got)
	}
}

// typing yang basi (lewat TTL) → tak ditampilkan, jatuh ke presence.
func TestChatSubtitleTypingExpiry(t *testing.T) {
	a := &App{}
	const jid = "y@s.whatsapp.net"
	a.setPresence(jid, "online")
	// suntik status mengetik yg sudah kedaluwarsa.
	a.presMu.Lock()
	a.typing = map[string]typingStateT{jid: {on: true, at: time.Now().Add(-2 * typingTTL)}}
	a.presMu.Unlock()
	if got := a.ChatSubtitle(jid); got != "online" {
		t.Errorf("typing basi = %q want online (jatuh ke presence)", got)
	}
}
