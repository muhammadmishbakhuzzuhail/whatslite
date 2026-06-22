// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// ui_test.go — unit test fungsi MURNI di paket gioui (logika tanpa GL/engine):
// filter daftar chat, format tanggal/ukuran, pisah @mention, siklus privasi, dll.
package gioui

import (
	"image/color"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/x/richtext"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/app"
)

func TestComputeShown(t *testing.T) {
	u := &UI{chats: []app.ChatDTO{
		{ID: "1", Name: "Andi", Preview: "halo", Pinned: true},
		{ID: "2", Name: "Keluarga", Preview: "makan", Group: true, Unread: true, Badge: 2},
		{ID: "3", Name: "Sarah", Preview: "oke kabar"},
		{ID: "4", Name: "Tim X", Preview: "upload", Group: true},
	}}
	cases := []struct {
		name   string
		filter int
		query  string
		want   []int // index ke u.chats
	}{
		{"semua", 0, "", []int{0, 1, 2, 3}},
		{"belum-dibaca", 1, "", []int{1}},        // Unread/Badge
		{"favorit-pinned", 2, "", []int{0}},      // Pinned
		{"grup", 3, "", []int{1, 3}},             // Group
		{"cari-nama", 0, "sar", []int{2}},        // case-insensitive nama
		{"cari-preview", 0, "kabar", []int{2}},   // preview
		{"cari+filter-grup", 3, "tim", []int{3}}, // filter AND query
		{"cari-kosong", 0, "zzz", []int{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			u.filterSel = c.filter
			u.searchEd.SetText(c.query)
			u.computeShown()
			if len(u.shown) != len(c.want) {
				t.Fatalf("len(shown)=%v want %v (shown=%v)", len(u.shown), len(c.want), u.shown)
			}
			for i := range c.want {
				if u.shown[i] != c.want[i] {
					t.Errorf("shown[%d]=%d want %d", i, u.shown[i], c.want[i])
				}
			}
		})
	}
}

func TestDayKeyLabel(t *testing.T) {
	now := time.Now()
	a := now.Unix()
	b := now.Add(3 * time.Hour).Unix() // jam beda, hari sama
	if dayKey(a) != dayKey(b) {
		t.Errorf("dayKey berbeda utk jam beda di hari sama")
	}
	if dayKey(0) != 0 {
		t.Errorf("dayKey(0) harus 0")
	}
	if dayKey(now.AddDate(0, 0, -1).Unix()) == dayKey(a) {
		t.Errorf("dayKey hari beda harus beda")
	}
	if got := dayLabel(now.Unix()); got != "Hari ini" {
		t.Errorf("dayLabel now = %q want Hari ini", got)
	}
	if got := dayLabel(now.AddDate(0, 0, -1).Unix()); got != "Kemarin" {
		t.Errorf("dayLabel kemarin = %q want Kemarin", got)
	}
	if got := dayLabel(0); got != "" {
		t.Errorf("dayLabel(0) = %q want empty", got)
	}
}

func TestFmtBytesSubs(t *testing.T) {
	for _, c := range []struct {
		n    int64
		want string
	}{{500, "500 B"}, {2048, "2 KB"}, {3 << 20, "3 MB"}} {
		if got := fmtBytes(c.n); got != c.want {
			t.Errorf("fmtBytes(%d)=%q want %q", c.n, got, c.want)
		}
	}
	for _, c := range []struct {
		n    int
		want string
	}{{42, "42 pengikut"}, {12000, "12 rb pengikut"}, {2_000_000, "2 jt pengikut"}} {
		if got := fmtSubs(c.n); got != c.want {
			t.Errorf("fmtSubs(%d)=%q want %q", c.n, got, c.want)
		}
	}
}

func TestDocExt(t *testing.T) {
	for _, c := range []struct{ mime, want string }{
		{"application/pdf", "PDF"},
		{"application/msword", "DOC"},
		{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "XLS"},
		{"application/zip", "ZIP"},
		{"", ""},
		{"image/png", "FILE"},
	} {
		if got := docExt(c.mime); got != c.want {
			t.Errorf("docExt(%q)=%q want %q", c.mime, got, c.want)
		}
	}
}

func TestDecodeDataURI(t *testing.T) {
	// "hi" → base64 "aGk="
	if got := decodeDataURI("data:text/plain;base64,aGk="); string(got) != "hi" {
		t.Errorf("decodeDataURI round-trip = %q want hi", got)
	}
	if decodeDataURI("https://example.com/x.png") != nil {
		t.Errorf("non-data-uri harus nil")
	}
	if decodeDataURI("data:text/plain;base64,!!!notbase64") != nil {
		t.Errorf("base64 invalid harus nil")
	}
	if decodeDataURI("") != nil {
		t.Errorf("empty harus nil")
	}
}

func TestNextPrivacyPrivValue(t *testing.T) {
	if nextPrivacy("all") != "contacts" || nextPrivacy("contacts") != "none" || nextPrivacy("none") != "all" || nextPrivacy("") != "all" {
		t.Errorf("siklus nextPrivacy salah")
	}
	if privValue("all") != "Semua orang" || privValue("none") != "Tidak ada" {
		t.Errorf("privValue terjemahan salah")
	}
	if privValue("xyz") != "xyz" {
		t.Errorf("privValue tak dikenal harus passthrough")
	}
}

func TestPhoneQuery(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"+62 812-3456-7890", "6281234567890"},
		{"08123456789", "08123456789"},
		{"(021) 555 1234", "0215551234"},
		{"halo budi", ""},  // huruf → bukan nomor
		{"12345", ""},      // <8 digit
		{"", ""},           // kosong
		{"  +1 555  ", ""}, // <8 digit setelah strip
	} {
		if got := phoneQuery(c.in); got != c.want {
			t.Errorf("phoneQuery(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestNextRetention(t *testing.T) {
	// siklus penuh 30→90→180→365→0→30
	seq := []int{30, 90, 180, 365, 0}
	for i, d := range seq {
		want := seq[(i+1)%len(seq)]
		if got := nextRetention(d); got != want {
			t.Errorf("nextRetention(%d)=%d want %d", d, got, want)
		}
	}
	if nextRetention(7) != 30 { // nilai tak dikenal → 30
		t.Errorf("nextRetention(unknown) want 30")
	}
}

func TestRetentionDesc(t *testing.T) {
	if got := retentionDesc(&SettingsCtl{Retention: 0}); got != "Simpan pesan selamanya" {
		t.Errorf("retensi 0 = %q", got)
	}
	if got := retentionDesc(&SettingsCtl{Retention: 90}); got != "Hapus pesan setelah 90 hari" {
		t.Errorf("retensi 90 = %q", got)
	}
	if got := retentionDesc(nil); got != "Hapus pesan setelah 90 hari" {
		t.Errorf("retensi nil(demo) = %q want 90 hari", got)
	}
}

func TestMentionSpans(t *testing.T) {
	base := richtext.SpanStyle{Color: color.NRGBA{R: 1}}
	acc := richtext.SpanStyle{Color: color.NRGBA{G: 1}}
	mentions := []app.MentionDTO{{Name: "Budi Santoso"}, {Name: "Citra"}}
	spans := mentionSpans("Halo @Budi Santoso dan @Citra ya", mentions, base, acc)
	// rangkai ulang Content harus == teks asli
	var joined string
	accCount := 0
	for _, s := range spans {
		joined += s.Content
		if s.Color == acc.Color {
			accCount++
		}
	}
	if joined != "Halo @Budi Santoso dan @Citra ya" {
		t.Errorf("span join = %q (tak sama teks asli)", joined)
	}
	if accCount != 2 {
		t.Errorf("accCount=%d want 2 (dua mention)", accCount)
	}
	// tanpa mention → satu span normal
	if got := mentionSpans("plain text", nil, base, acc); len(got) != 1 || got[0].Content != "plain text" {
		t.Errorf("tanpa mention harus 1 span normal, dapat %v", got)
	}
}

func TestAvatarColorDeterministic(t *testing.T) {
	if avatarColor("Andi") != avatarColor("Andi") {
		t.Errorf("avatarColor tak deterministik")
	}
}

func TestWithAlpha(t *testing.T) {
	c := withAlpha(color.NRGBA{R: 10, G: 20, B: 30, A: 255}, 40)
	if c.R != 10 || c.G != 20 || c.B != 30 || c.A != 40 {
		t.Errorf("withAlpha salah: %+v", c)
	}
}

func TestIsMediaType(t *testing.T) {
	for _, ty := range []string{"image", "video", "gif", "document", "voice", "audio", "ptt", "sticker"} {
		if !isMediaType(ty) {
			t.Errorf("isMediaType(%q) = false, mau true", ty)
		}
	}
	for _, ty := range []string{"text", "", "poll", "location", "contact"} {
		if isMediaType(ty) {
			t.Errorf("isMediaType(%q) = true, mau false", ty)
		}
	}
}

func TestSaveName(t *testing.T) {
	cases := map[string]struct {
		m    app.MessageDTO
		want string
	}{
		"image":   {app.MessageDTO{ID: "m1", Type: "image"}, "whatslite-m1.jpg"},
		"video":   {app.MessageDTO{ID: "m2", Type: "video"}, "whatslite-m2.mp4"},
		"voice":   {app.MessageDTO{ID: "m3", Type: "ptt"}, "whatslite-m3.ogg"},
		"doctxt":  {app.MessageDTO{ID: "m4", Type: "document", Text: "rapat.pdf"}, "whatslite-rapat.pdf"},
		"docnone": {app.MessageDTO{ID: "m5", Type: "document"}, "whatslite-m5"},
		"sticker": {app.MessageDTO{ID: "m6", Type: "sticker"}, "whatslite-m6.webp"},
	}
	for name, c := range cases {
		if got := saveName(c.m); got != c.want {
			t.Errorf("%s: saveName = %q, mau %q", name, got, c.want)
		}
	}
}

func TestFabHidden(t *testing.T) {
	cases := []struct {
		name string
		pos  layout.Position
		n    int
		want bool
	}{
		{"kosong", layout.Position{}, 0, true},
		{"belum-terukur", layout.Position{Count: 0, First: 0}, 20, true},
		{"di-dasar", layout.Position{First: 11, Count: 9, OffsetLast: 0}, 20, true},
		{"tergulir-naik", layout.Position{First: 0, Count: 9, OffsetLast: 50}, 20, false},
		{"dasar-tapi-overflow", layout.Position{First: 11, Count: 9, OffsetLast: 30}, 20, false},
	}
	for _, c := range cases {
		if got := fabHidden(c.pos, c.n); got != c.want {
			t.Errorf("%s: fabHidden = %v, mau %v", c.name, got, c.want)
		}
	}
}

func TestEmojiOnlyCount(t *testing.T) {
	cases := map[string]int{
		"👍":            1,
		"🔥🔥":           2,
		"😀😃😄":          3,
		"😀😃😄😁":         0, // >3 → tak diperbesar
		"halo":         0,
		"oke 👍":        0, // ada teks biasa
		"":             0,
		"   ":          0,
		"👍🏽":           1, // skin tone modifier
		"👨‍👩‍👧":          1, // ZWJ family = satu emoji
	}
	for in, want := range cases {
		if got := emojiOnlyCount(in); got != want {
			t.Errorf("emojiOnlyCount(%q) = %d, mau %d", in, got, want)
		}
	}
}
