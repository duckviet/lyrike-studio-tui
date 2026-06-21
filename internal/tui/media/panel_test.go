package media

import (
	"strings"
	"testing"
)

func TestTransportRendersPlaybackControls(t *testing.T) {
	t.Parallel()

	panel := NewPanel().WithTransport(TransportPlaying, 12_340, 65_000)

	got := panel.View(80, 1)

	for _, want := range []string{"▶", "00:12.34 / 01:05.00", "[space] play/pause", "[←/→] seek", "[l] loop"} {
		if !strings.Contains(got, want) {
			t.Fatalf("View() = %q, want it to contain %q", got, want)
		}
	}
}
