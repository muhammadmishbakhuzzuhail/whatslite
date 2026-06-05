package app

import (
	"bytes"
	"testing"
)

func TestUserPart(t *testing.T) {
	cases := map[string]string{
		"6281234567890@s.whatsapp.net":    "6281234567890",
		"6281234567890:12@s.whatsapp.net": "6281234567890",
		"12345@lid":                       "12345",
		"6281234567890":                   "6281234567890",
		"":                                "",
	}
	for in, want := range cases {
		if got := userPart(in); got != want {
			t.Errorf("userPart(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDecodeDataURI(t *testing.T) {
	// "hi" base64 = aGk=
	mime, data, err := decodeDataURI("data:text/plain;base64,aGk=")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if mime != "text/plain" {
		t.Errorf("mime = %q, want text/plain", mime)
	}
	if !bytes.Equal(data, []byte("hi")) {
		t.Errorf("data = %q, want hi", data)
	}

	// tanpa header mime → default octet-stream
	mime2, _, err := decodeDataURI("aGk=")
	if err != nil {
		t.Fatalf("err2: %v", err)
	}
	if mime2 != "application/octet-stream" {
		t.Errorf("mime2 = %q, want application/octet-stream", mime2)
	}

	if _, _, err := decodeDataURI("data:text/plain;base64,@@@not-base64@@@"); err == nil {
		t.Error("expected error for invalid base64")
	}
}
