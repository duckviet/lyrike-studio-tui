package playback

import (
	"errors"
	"fmt"
	"testing"
)

func TestFakePlayerTransitionsAndClock(t *testing.T) {
	player := newTestFakePlayer(t, 10_000)

	assertSnapshot(t, player.Snapshot(), 0, StatePaused)

	snapshot, err := player.Play()
	requireNoError(t, err)
	assertSnapshot(t, snapshot, 0, StatePlaying)

	snapshot, err = player.Tick(Duration(1_500))
	requireNoError(t, err)
	assertSnapshot(t, snapshot, 1_500, StatePlaying)

	snapshot, err = player.Pause()
	requireNoError(t, err)
	assertSnapshot(t, snapshot, 1_500, StatePaused)

	snapshot, err = player.Tick(Duration(2_000))
	requireNoError(t, err)
	assertSnapshot(t, snapshot, 1_500, StatePaused)

	snapshot, err = player.Play()
	requireNoError(t, err)
	assertSnapshot(t, snapshot, 1_500, StatePlaying)

	snapshot, err = player.Tick(Duration(20_000))
	requireNoError(t, err)
	assertSnapshot(t, snapshot, 10_000, StatePaused)
}

func TestFakePlayerSeekAndProgressAreDeterministic(t *testing.T) {
	player := newTestFakePlayer(t, 12_000)

	snapshot, err := player.Seek(Position(4_000))
	requireNoError(t, err)
	assertSnapshot(t, snapshot, 4_000, StatePaused)

	snapshot, err = player.Play()
	requireNoError(t, err)
	assertSnapshot(t, snapshot, 4_000, StatePlaying)

	snapshot, err = player.Tick(Duration(250))
	requireNoError(t, err)
	assertSnapshot(t, snapshot, 4_250, StatePlaying)

	snapshot, err = player.Seek(Position(2_000))
	requireNoError(t, err)
	assertSnapshot(t, snapshot, 2_000, StatePlaying)
}

func TestFakePlayerRejectsInvalidBounds(t *testing.T) {
	if _, err := NewDuration(0); !isCommandError(err) {
		t.Fatalf("NewDuration(0) error = %v, want CommandError", err)
	}
	if _, err := NewDuration(-1); !isCommandError(err) {
		t.Fatalf("NewDuration(-1) error = %v, want CommandError", err)
	}
	if _, err := NewPosition(-1); !isCommandError(err) {
		t.Fatalf("NewPosition(-1) error = %v, want CommandError", err)
	}
	if _, err := NewFakePlayer(Duration(0)); !isCommandError(err) {
		t.Fatalf("NewFakePlayer(Duration(0)) error = %v, want CommandError", err)
	}

	player := newTestFakePlayer(t, 5_000)
	if _, err := player.Seek(Position(-1)); !isCommandError(err) {
		t.Fatalf("Seek(-1) error = %v, want CommandError", err)
	}
	if _, err := player.Seek(Position(5_001)); !isCommandError(err) {
		t.Fatalf("Seek(past duration) error = %v, want CommandError", err)
	}
	if _, err := player.Tick(Duration(-1)); !isCommandError(err) {
		t.Fatalf("Tick(-1) error = %v, want CommandError", err)
	}
	if _, err := player.Tick(Duration(0)); !isCommandError(err) {
		t.Fatalf("Tick(0) error = %v, want CommandError", err)
	}
}

func TestManualFakePlayerSurface(t *testing.T) {
	player := newTestFakePlayer(t, 10_000)
	printManualSnapshot("start", player.Snapshot())

	snapshot, err := player.Play()
	requireNoError(t, err)
	printManualSnapshot("play", snapshot)

	snapshot, err = player.Tick(Duration(2_500))
	requireNoError(t, err)
	printManualSnapshot("tick 2500ms", snapshot)

	snapshot, err = player.Seek(Position(8_000))
	requireNoError(t, err)
	printManualSnapshot("seek 8000ms", snapshot)

	snapshot, err = player.Tick(Duration(3_000))
	requireNoError(t, err)
	printManualSnapshot("tick 3000ms", snapshot)

	snapshot, err = player.Pause()
	requireNoError(t, err)
	printManualSnapshot("pause", snapshot)
}

func newTestFakePlayer(t *testing.T, durationMillis int64) *FakePlayer {
	t.Helper()

	duration, err := NewDuration(durationMillis)
	requireNoError(t, err)

	player, err := NewFakePlayer(duration)
	requireNoError(t, err)

	return player
}

func assertSnapshot(t *testing.T, snapshot Snapshot, positionMillis int64, state State) {
	t.Helper()

	if snapshot.Position.Milliseconds() != positionMillis {
		t.Fatalf("position = %d, want %d", snapshot.Position.Milliseconds(), positionMillis)
	}
	if snapshot.State != state {
		t.Fatalf("state = %s, want %s", snapshot.State, state)
	}
}

func isCommandError(err error) bool {
	var commandErr *CommandError
	return errors.As(err, &commandErr)
}

func printManualSnapshot(action string, snapshot Snapshot) {
	fmt.Printf(
		"action=%q position=%dms duration=%dms state=%s\n",
		action,
		snapshot.Position.Milliseconds(),
		snapshot.Duration.Milliseconds(),
		snapshot.State,
	)
}

func requireNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
