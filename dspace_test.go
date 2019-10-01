package main

import (
	"testing"
)

func Test_findItemsByMeta(t *testing.T) {
	const dspaceBaseURL = "https://dspace.ut.ee"
	m := &meta{
		Key:   "dc.contributor.author",
		Value: "Nakshatharan, S. Sunjai",
	}
	items, err := findItemsByMeta(dspaceBaseURL, m)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("want at least 1 result, got 0")
	}
}
