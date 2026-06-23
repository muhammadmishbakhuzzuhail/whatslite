// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// linkpreview_test.go — uji parsing host tautan (deterministik, tanpa jaringan).
package app

import "testing"

func TestHostOf(t *testing.T) {
	cases := map[string]string{
		"https://www.tiktok.com/@u/video/123": "www.tiktok.com",
		"http://example.com/path?q=1#frag":    "example.com",
		"https://youtu.be/abc":                "youtu.be",
		"https://sub.site.co.id":              "sub.site.co.id",
		"no-scheme.com/x":                     "no-scheme.com",
	}
	for in, want := range cases {
		if got := hostOf(in); got != want {
			t.Errorf("hostOf(%q) = %q, mau %q", in, got, want)
		}
	}
}
