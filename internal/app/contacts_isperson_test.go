// SPDX-License-Identifier: GPL-3.0-or-later
package app

import "testing"

func TestIsPersonJID(t *testing.T) {
	person := []string{"628123@s.whatsapp.net", "12345@lid"}
	notPerson := []string{"", "status@broadcast", "120363@g.us", "abc@newsletter", "x@broadcast"}
	for _, j := range person {
		if !isPersonJID(j) {
			t.Errorf("isPersonJID(%q)=false, mau true", j)
		}
	}
	for _, j := range notPerson {
		if isPersonJID(j) {
			t.Errorf("isPersonJID(%q)=true, mau false", j)
		}
	}
}
