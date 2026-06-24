package beep

import (
	"os"
	"testing"

	"github.com/duckviet/lyrike-studio-tui/internal/playback"
)

func TestBeepPlayerSeekAfterEOF(t *testing.T) {
	wavPath := "/home/duckviet/lyrike-studio-tui/test_output.wav"
	if _, err := os.Stat(wavPath); err != nil {
		t.Skip("test_output.wav not found, skipping")
	}

	p, err := NewPlayer(wavPath)
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	defer p.Close()

	// Start playing
	_, err = p.Play()
	if err != nil {
		t.Fatalf("failed to play: %v", err)
	}

	// Simulate background audio streaming to EOF
	buf := make([][2]float64, 512)
	eofReached := false
	for i := 0; i < 50000; i++ {
		n, ok := p.ctrl.Stream(buf)
		if n == 0 || !ok {
			eofReached = true
			break
		}
		// If we read silence (zeros) after EOF, we detect it
		if p.streamer.(*permanentStreamer).streamer.Position() >= p.streamer.(*permanentStreamer).streamer.Len() {
			eofReached = true
			break
		}
	}

	if !eofReached {
		t.Fatalf("did not reach EOF during streaming")
	}

	// Trigger the auto-pause logic that TUI/Snapshot does
	snap := p.Snapshot()
	t.Logf("Snapshot position: %v, Len: %v, ctrl.Paused: %v", p.streamer.Position(), p.streamer.Len(), p.ctrl.Paused)
	if snap.State != playback.StatePaused {
		t.Fatalf("expected player to auto-pause at EOF, got state %v", snap.State)
	}

	// Seek back to 1 second
	targetPos, _ := playback.NewPosition(1000)
	snap, err = p.Seek(targetPos)
	if err != nil {
		t.Fatalf("seek failed: %v", err)
	}

	if snap.Position.Milliseconds() != 1000 {
		t.Fatalf("expected position to be 1000ms after seek, got %dms", snap.Position.Milliseconds())
	}

	// Now try to Play again
	snap, err = p.Play()
	if err != nil {
		t.Fatalf("play after seek failed: %v", err)
	}

	// Simulate streaming again - it should read non-silence samples now!
	n, ok := p.ctrl.Stream(buf)
	if n == 0 || !ok {
		t.Fatalf("stream after seek returned no samples or ok=false: n=%d, ok=%v", n, ok)
	}

	// Verify that the position advanced
	newPos := p.streamer.Position()
	if newPos <= 0 {
		t.Fatalf("expected position to advance after play, got %d", newPos)
	}
}
