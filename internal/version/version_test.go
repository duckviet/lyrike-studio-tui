package version

import "testing"

func TestLabel_whenVersionIsRequested(t *testing.T) {
	t.Parallel()

	got := Label()

	if got != "lyrike-studio-tui 0.1.0-dev" {
		t.Fatalf("Label() = %q, want %q", got, "lyrike-studio-tui 0.1.0-dev")
	}
}
