// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

import "testing"

func TestIsPhoneLike(t *testing.T) {
	phoneLike := []string{"+62 812-3456-7890", "08123456789", "(021) 555 1234", "+1-555-0000"}
	for _, s := range phoneLike {
		if !isPhoneLike(s) {
			t.Errorf("isPhoneLike(%q) = false, want true", s)
		}
	}
	names := []string{"Budi Santoso", "John 2", "Citra", "A.B."}
	for _, s := range names {
		if isPhoneLike(s) {
			t.Errorf("isPhoneLike(%q) = true, want false (nama)", s)
		}
	}
	// edge: kosong → tak ada digit → false (di-shield pemanggil dgn cek n!="")
	if isPhoneLike("") {
		t.Errorf("isPhoneLike(\"\") harus false")
	}
}
