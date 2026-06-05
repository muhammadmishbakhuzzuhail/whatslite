package engine

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waWeb"
)

func TestInviteKey(t *testing.T) {
	cases := map[string]string{
		"https://whatsapp.com/channel/0029Vaabcd1234":      "0029Vaabcd1234",
		"https://whatsapp.com/channel/0029Vaabcd1234?foo=1": "0029Vaabcd1234",
		"0029Vaabcd1234":                                    "0029Vaabcd1234",
		"  0029Vaabcd1234  ":                                "0029Vaabcd1234",
		"whatsapp.com/channel/XYZ":                          "XYZ",
	}
	for in, want := range cases {
		if got := inviteKey(in); got != want {
			t.Errorf("inviteKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWebStatus(t *testing.T) {
	cases := map[waWeb.WebMessageInfo_Status]string{
		waWeb.WebMessageInfo_READ:         "read",
		waWeb.WebMessageInfo_PLAYED:       "read",
		waWeb.WebMessageInfo_DELIVERY_ACK: "delivered",
		waWeb.WebMessageInfo_SERVER_ACK:   "",
		waWeb.WebMessageInfo_PENDING:      "",
	}
	for in, want := range cases {
		if got := webStatus(in); got != want {
			t.Errorf("webStatus(%v) = %q, want %q", in, got, want)
		}
	}
}
