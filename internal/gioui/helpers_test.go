// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// helpers_test.go — uji fungsi murni tautan (deterministik, tanpa render).
// (fmtBytes/docExt/emojiOnlyCount sudah diuji di ui_test.go.)
package gioui

import "testing"

func TestFirstURL(t *testing.T) {
	cases := map[string]string{
		"Cek https://example.com/a ya":      "https://example.com/a",
		"awal http://x.io akhir":            "http://x.io",
		"trailing https://t.co/abc.":        "https://t.co/abc", // buang titik ekor
		"tanda https://t.co/x)!":            "https://t.co/x",
		"tanpa tautan sama sekali":          "",
		"ftp://nope.com bukan http":         "",
		"https://www.tiktok.com/@u/video/1": "https://www.tiktok.com/@u/video/1",
	}
	for in, want := range cases {
		if got := firstURL(in); got != want {
			t.Errorf("firstURL(%q) = %q, mau %q", in, got, want)
		}
	}
}

func TestURLHost(t *testing.T) {
	cases := map[string]string{
		"https://www.tiktok.com/@u/v": "tiktok.com",
		"http://example.com":          "example.com",
		"https://sub.domain.co/x?y=1": "sub.domain.co",
		"https://www.youtube.com":     "youtube.com",
	}
	for in, want := range cases {
		if got := urlHost(in); got != want {
			t.Errorf("urlHost(%q) = %q, mau %q", in, got, want)
		}
	}
}
