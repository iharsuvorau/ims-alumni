package main

import (
	"reflect"
	"testing"
)

func Test_findUsersFromTemplate(t *testing.T) {
	const content = `== BS, MS Students ==
{{Team|
{{TeamMember | heikki.saul| Heikki Saul   |student}}
{{TeamMember|mkorv|Mattias Kõrv|student (computer engineering)}}
{{TeamMember |mariokool|Mario Kool |student (computer engineering)}}
{{ TeamMember|kertmannik|Kert Männik|student (computer engineering)   }}
}}

== Staff ==
..`

	want := []string{
		"Heikki Saul",
		"Mattias Kõrv",
		"Mario Kool",
		"Kert Männik",
	}

	users := findUsersFromTemplate(content)
	for i, v := range users {
		if v.FullName != want[i] {
			t.Fatalf("want %q, got %q", want[i], v.FullName)
		}
	}
}

func Test_nameSplit(t *testing.T) {
	const dspaceURL = "https://dspace.ut.ee"

	args := []string{
		"S. Sunjai Nakshatharan",
		"Inga Põldsalu",
		"Kätlin Rohtlaid",
	}
	want := [][]string{
		{"S. Sunjai", "Nakshatharan"},
		{"Inga", "Põldsalu"},
		{"Kätlin", "Rohtlaid"},
	}

	for i, arg := range args {
		name := nameSplit(arg)
		if !reflect.DeepEqual(name, want[i]) {
			t.Fatalf("want %q, got %q", want[i], name)
		}
	}
}
