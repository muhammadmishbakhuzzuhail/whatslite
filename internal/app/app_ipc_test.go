// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

import (
	"encoding/json"
	"testing"
)

// TestDispatchIPC menguji dispatcher refleksi: method valid terpanggil + hasil
// benar, method tak dikenal & denylist ditolak. Pakai Version() (tanpa
// engine/store) agar tak butuh sesi WhatsApp.
func TestDispatchIPC(t *testing.T) {
	a := NewApp() // ctx/eng/store nil — Version() aman

	// Method valid, tanpa argumen → hasil "dev".
	res, err := a.dispatchIPC("Version", nil)
	if err != nil {
		t.Fatalf("Version err: %v", err)
	}
	if res != "dev" {
		t.Fatalf("Version = %v, mau \"dev\"", res)
	}

	// Method tak dikenal → error.
	if _, err := a.dispatchIPC("NoSuchMethod", nil); err == nil {
		t.Fatal("method tak dikenal seharusnya error")
	}

	// Denylist (siklus hidup) → ditolak.
	if _, err := a.dispatchIPC("Startup", []json.RawMessage{json.RawMessage(`null`)}); err == nil {
		t.Fatal("Startup (denylist) seharusnya ditolak")
	}

	// Jumlah argumen salah → error (Version butuh 0).
	if _, err := a.dispatchIPC("Version", []json.RawMessage{json.RawMessage(`"x"`)}); err == nil {
		t.Fatal("argumen berlebih seharusnya error")
	}
}
