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

	users := findUsersFromTemplate(reTeamMember, content)
	for i, v := range users {
		if v.FullName != want[i] {
			t.Fatalf("want %q, got %q", want[i], v.FullName)
		}
	}
}

func Test_nameSplit(t *testing.T) {
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

func Test_getUserThesesIMS(t *testing.T) {
	const dspaceBaseURL = "https://dspace.ut.ee"
	const mwBaseURL = "https://ims.ut.ee"

	tests := []struct {
		name                string
		firstName, lastName string
		wantErr             bool
		resultsCount        int
	}{
		{
			name:         "A",
			firstName:    "S. Sunjai",
			lastName:     "Nakshatharan",
			wantErr:      false,
			resultsCount: 1,
		},
		{
			name:         "B",
			firstName:    "Iti-Jantra",
			lastName:     "Metsamaa",
			wantErr:      false,
			resultsCount: 0,
		},
		{
			name:         "C",
			firstName:    "Madis Kaspar",
			lastName:     "Nigol",
			wantErr:      false,
			resultsCount: 1,
		},
	}

	advisors, err := getAdvisorsIMS(mwBaseURL)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := getUserThesesIMS(dspaceBaseURL, tt.firstName, tt.lastName, advisors)
			if err != nil && !tt.wantErr {
				t.Fatal(err)
			}
			if l := len(items); l != tt.resultsCount {
				t.Fatalf("want %v items, got %v", tt.resultsCount, l)
			}
		})
	}
}

func Test_getAdvisorsIMS(t *testing.T) {
	const mwBaseURL = "https://ims.ut.ee"
	advisors, err := getAdvisorsIMS(mwBaseURL)
	if err != nil {
		t.Fatal(err)
	}
	// for _, v := range advisors {
	// 	t.Logf("%s", v.FullName)
	// }
	if len(advisors) == 0 {
		t.Fatal("expect more advisors, got 0")
	}
}
