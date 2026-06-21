package mpv

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/duckviet/lyrike-studio-tui/internal/playback"
)

func TestPlayer_MissingMpvGuidance(t *testing.T) {
	t.Parallel()

	player := NewPlayer("/nonexistent/mpv/socket")
	err := player.Start(context.Background())
	if err == nil {
		t.Fatalf("Start() error = nil, want missing-mpv error")
	}
	if !errors.Is(err, ErrMpvUnavailable) {
		t.Fatalf("error is not ErrMpvUnavailable: %v", err)
	}
	if !containsAll(err.Error(), "mpv", "--input-ipc-server") {
		t.Fatalf("error message missing guidance: %v", err)
	}
}

func TestPlayer_ObserveTimePosFromSocket(t *testing.T) {
	t.Parallel()

	server, socketPath := newFakeMPVServer(t)
	defer server.Close()

	player := NewPlayer(socketPath)
	if err := player.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer player.Close()
	<-server.ready

	server.sendPropertyChange("time-pos", 12.34)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var snapshot playback.Snapshot
	for {
		if ctx.Err() != nil {
			t.Fatalf("timed out waiting for time-pos update")
		}
		snapshot = player.Snapshot()
		if snapshot.Position.Milliseconds() == 12_340 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if snapshot.State != playback.StatePlaying {
		t.Fatalf("State = %q, want playing", snapshot.State)
	}
}

func TestPlayer_PlayPauseCommands(t *testing.T) {
	t.Parallel()

	server, socketPath := newFakeMPVServer(t)
	defer server.Close()

	player := NewPlayer(socketPath)
	if err := player.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer player.Close()
	<-server.ready

	server.sendPropertyChange("time-pos", 5.0)
	server.setPaused(false)

	snap, err := player.Play()
	if err != nil {
		t.Fatalf("Play() error = %v", err)
	}
	if snap.State != playback.StatePlaying {
		t.Fatalf("State = %q, want playing", snap.State)
	}

	server.setPaused(true)
	snap, err = player.Pause()
	if err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	if snap.State != playback.StatePaused {
		t.Fatalf("State = %q, want paused", snap.State)
	}
}

func TestPlayer_Seek(t *testing.T) {
	t.Parallel()

	server, socketPath := newFakeMPVServer(t)
	defer server.Close()

	player := NewPlayer(socketPath)
	if err := player.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer player.Close()
	<-server.ready

	server.sendPropertyChange("time-pos", 0.0)

	target, err := playback.NewPosition(30_000)
	if err != nil {
		t.Fatalf("NewPosition() error = %v", err)
	}
	if _, err := player.Seek(target); err != nil {
		t.Fatalf("Seek() error = %v", err)
	}

	if server.lastSeekMillis != 30_000 {
		t.Fatalf("last seek = %d, want 30000", server.lastSeekMillis)
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsHelper(s, sub))
}

func containsHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// fakeServer is a minimal mpv JSON IPC server for tests.
type fakeServer struct {
	listener       net.Listener
	conn           net.Conn
	lastSeekMillis int64
	paused         bool
	t              *testing.T
	ready          chan struct{}
}

func newFakeMPVServer(t *testing.T) (*fakeServer, string) {
	t.Helper()
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "mpv.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix socket: %v", err)
	}

	server := &fakeServer{
		listener: listener,
		paused:   false,
		t:        t,
		ready:    make(chan struct{}),
	}

	go server.acceptLoop()
	return server, socketPath
}

func (s *fakeServer) acceptLoop() {
	conn, err := s.listener.Accept()
	if err != nil {
		return
	}
	s.conn = conn
	close(s.ready)
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		s.handleCommand(scanner.Bytes())
	}
}

func (s *fakeServer) handleCommand(raw []byte) {
	var msg struct {
		Command   []any   `json:"command"`
		RequestID int64   `json:"request_id"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}
	if len(msg.Command) == 0 {
		return
	}

	cmd := msg.Command[0].(string)
	switch cmd {
	case "observe_property":
		// Acknowledge observation.
		s.write(map[string]any{"request_id": msg.RequestID, "error": "success"})
	case "set_property":
		prop := msg.Command[1].(string)
		switch prop {
		case "pause":
			s.paused = msg.Command[2].(bool)
		case "time-pos":
			s.lastSeekMillis = int64(msg.Command[2].(float64) * 1000)
		}
		s.write(map[string]any{"request_id": msg.RequestID, "error": "success"})
	case "get_property":
		prop := msg.Command[1].(string)
		var value any
		switch prop {
		case "time-pos":
			value = float64(s.lastSeekMillis) / 1000
		case "pause":
			value = s.paused
		}
		s.write(map[string]any{"request_id": msg.RequestID, "error": "success", "data": value})
	default:
		s.write(map[string]any{"request_id": msg.RequestID, "error": "success"})
	}
}

func (s *fakeServer) write(v map[string]any) {
	if s.conn == nil {
		return
	}
	b, _ := json.Marshal(v)
	b = append(b, '\n')
	_, _ = s.conn.Write(b)
}

func (s *fakeServer) sendPropertyChange(name string, value float64) {
	if s.conn == nil {
		return
	}
	b, _ := json.Marshal(map[string]any{
		"event":        "property-change",
		"name":         name,
		"data":         value,
	})
	b = append(b, '\n')
	_, _ = s.conn.Write(b)
}

func (s *fakeServer) setPaused(paused bool) {
	s.paused = paused
}

func (s *fakeServer) Close() {
	if s.conn != nil {
		_ = s.conn.Close()
	}
	_ = s.listener.Close()
}

// Avoid unused import warning on os.
var _ = os.Open
