package engine

import "testing"

func TestFormatPhone(t *testing.T) {
	cases := map[string]string{
		"":             "",
		"6281519346661": "+62 815-1934-6661", // ID mobile (kasus utama)
		"628123456789":  "+62 812-3456-789",
		"14155552671":   "+1 415-5552-671", // US (CC 1-digit)
		"60123456789":   "+60 123-4567-89", // Malaysia
		"6512345678":    "+65 123-4567-8",  // Singapura
		"971501234567":  "+971 501-2345-67", // UEA (CC 3-digit)
		"abc123":        "+abc123",          // non-digit → mentah, aman
		"999":           "+999",             // CC tak dikenal → mentah
	}
	for in, want := range cases {
		if got := formatPhone(in); got != want {
			t.Errorf("formatPhone(%q) = %q, want %q", in, got, want)
		}
	}
}
