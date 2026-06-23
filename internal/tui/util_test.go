package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestClamp(t *testing.T) {
	cases := []struct {
		name string
		v    int
		lo   int
		hi   int
		want int
	}{
		{"within range", 5, 0, 10, 5},
		{"below lower bound", -3, 0, 10, 0},
		{"above upper bound", 15, 0, 10, 10},
		{"equal to lower bound", 0, 0, 10, 0},
		{"equal to upper bound", 10, 0, 10, 10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := clamp(tc.v, tc.lo, tc.hi); got != tc.want {
				t.Fatalf("clamp(%d, %d, %d) = %d, want %d", tc.v, tc.lo, tc.hi, got, tc.want)
			}
		})
	}
}

func TestTruncateANSI(t *testing.T) {
	red := "\x1b[31m"
	reset := "\x1b[0m"
	styled := red + "hello world" + reset

	cases := []struct {
		name string
		s    string
		w    int
	}{
		{"fits exactly", styled, 11},
		{"truncate wide ansi string", styled, 5},
		{"zero width", styled, 0},
		{"negative width", styled, -1},
		{"plain string", "hello world", 5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := truncate(tc.s, tc.w)
			if tc.w <= 0 {
				if got != "" {
					t.Fatalf("truncate(%q, %d) = %q, want empty", tc.s, tc.w, got)
				}
				return
			}
			if width := ansi.StringWidth(got); width > tc.w {
				t.Fatalf("truncate(%q, %d) width = %d, want <= %d: %q", tc.s, tc.w, width, tc.w, got)
			}
			if tc.w >= ansi.StringWidth(tc.s) && got != tc.s {
				t.Fatalf("truncate(%q, %d) = %q, want unchanged", tc.s, tc.w, got)
			}
			if strings.Contains(tc.s, red) && !strings.Contains(got, red) {
				t.Fatalf("truncate dropped ANSI prefix: %q", got)
			}
		})
	}
}
