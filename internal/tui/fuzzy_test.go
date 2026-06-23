package tui

import (
	"reflect"
	"testing"
)

func TestFuzzyScore(t *testing.T) {
	tests := []struct {
		pattern string
		text    string
		match   bool
	}{
		{"", "anything", true},
		{"abc", "abc", true},
		{"abc", "axbxc", true},
		{"abc", "cba", false},
		{"ab", "alpha bravo", true},
		{"xyz", "xy", false},
	}

	for _, tc := range tests {
		_, gotMatch := fuzzyScore(tc.pattern, tc.text)
		if gotMatch != tc.match {
			t.Errorf("fuzzyScore(%q, %q) match = %v, want %v", tc.pattern, tc.text, gotMatch, tc.match)
		}
	}
}

func TestFuzzyRank(t *testing.T) {
	items := []string{
		"apple pie",
		"banana cake",
		"apricot bread",
		"blueberry tart",
	}

	// Pattern "ap" should match "apple pie" and "apricot bread"
	got := fuzzyRank(items, "ap", func(s string) string { return s })
	want := []string{"apple pie", "apricot bread"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("fuzzyRank results = %v, want %v", got, want)
	}
}
