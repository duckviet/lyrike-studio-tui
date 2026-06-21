package draft

import (
	"errors"
	"testing"
)

func TestNewProjectID_acceptsFilenameSafeIDs(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want ProjectID
	}{
		{name: "simple", raw: "album", want: "album"},
		{name: "trims whitespace", raw: "  live-session_01  ", want: "live-session_01"},
		{name: "allows dot", raw: "demo.v2", want: "demo.v2"},
		{name: "allows unicode letters", raw: "dự-án-1", want: "dự-án-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewProjectID(tt.raw)
			if err != nil {
				t.Fatalf("NewProjectID() error = %v, want nil", err)
			}
			if got != tt.want {
				t.Fatalf("NewProjectID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewProjectID_rejectsUnsafeIDs(t *testing.T) {
	tests := []string{
		"",
		"   ",
		".",
		"..",
		"../demo",
		"demo/session",
		"demo\\session",
		"demo session",
	}

	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			_, err := NewProjectID(raw)
			if !errors.Is(err, ErrInvalidProjectID) {
				t.Fatalf("NewProjectID() error = %v, want ErrInvalidProjectID", err)
			}
		})
	}
}
