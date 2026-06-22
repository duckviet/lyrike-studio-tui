package mpv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/duckviet/lyrike-studio-tui/internal/playback"
)

// ErrMpvUnavailable is returned when mpv cannot be reached via its IPC socket.
var ErrMpvUnavailable = errors.New("mpv: unavailable")

// Player is a Unix-socket mpv JSON IPC adapter that implements playback.Player.
type Player struct {
	socketPath string
	conn       net.Conn
	mu         sync.RWMutex
	snapshot   playback.Snapshot
	pending    map[int64]chan ipcResponse
	nextID     int64
	stop       chan struct{}
	wg         sync.WaitGroup
	closed     bool
}

// NewPlayer creates an unconnected mpv adapter for socketPath.
func NewPlayer(socketPath string) *Player {
	return &Player{
		socketPath: socketPath,
		nextID:     1,
	}
}

// Start connects to the mpv IPC socket and starts observing time-pos.
func (p *Player) Start(ctx context.Context) error {
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "unix", p.socketPath)
	if err != nil {
		return fmt.Errorf(
			"%w: cannot connect to mpv IPC socket %q: %v. "+
				"Start mpv first: mpv <file> --input-ipc-server=%s",
			ErrMpvUnavailable, p.socketPath, err, p.socketPath,
		)
	}
	p.conn = conn
	p.pending = make(map[int64]chan ipcResponse)
	p.stop = make(chan struct{})

	p.wg.Add(1)
	go p.readLoop()

	if err := p.observeTimePos(); err != nil {
		_ = p.Close()
		return fmt.Errorf("mpv: observe time-pos: %w", err)
	}

	duration, err := p.getDurationSeconds()
	if err == nil {
		p.updateDuration(duration)
	}

	paused, err := p.getPause()
	if err == nil {
		p.updateState(paused)
	}

	return nil
}

// Close shuts down the IPC connection and background goroutine.
func (p *Player) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	if p.stop != nil {
		close(p.stop)
	}
	conn := p.conn
	p.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
	p.wg.Wait()
	return nil
}

// Snapshot returns the latest observed playback state.
func (p *Player) Snapshot() playback.Snapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.snapshot
}

// Play resumes playback.
func (p *Player) Play() (playback.Snapshot, error) {
	if _, err := p.sendCommand([]any{"set_property", "pause", false}); err != nil {
		return p.Snapshot(), fmt.Errorf("mpv play: %w", err)
	}
	p.updateState(false)
	return p.Snapshot(), nil
}

// Pause pauses playback.
func (p *Player) Pause() (playback.Snapshot, error) {
	if _, err := p.sendCommand([]any{"set_property", "pause", true}); err != nil {
		return p.Snapshot(), fmt.Errorf("mpv pause: %w", err)
	}
	p.updateState(true)
	return p.Snapshot(), nil
}

// Seek jumps to position.
func (p *Player) Seek(position playback.Position) (playback.Snapshot, error) {
	seconds := float64(position.Milliseconds()) / 1000.0
	if _, err := p.sendCommand([]any{"set_property", "time-pos", seconds}); err != nil {
		return p.Snapshot(), fmt.Errorf("mpv seek: %w", err)
	}
	p.updatePosition(position)
	return p.Snapshot(), nil
}

// Tick is a no-op for real mpv because the authoritative clock is observed
// from the IPC socket. It returns the current snapshot.
func (p *Player) Tick(_ playback.Duration) (playback.Snapshot, error) {
	return p.Snapshot(), nil
}

func (p *Player) sendCommand(command []any) (any, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("%w: player is closed", ErrMpvUnavailable)
	}
	id := p.nextID
	p.nextID++
	respCh := make(chan ipcResponse, 1)
	p.pending[id] = respCh
	conn := p.conn
	p.mu.Unlock()

	msg := map[string]any{
		"command":    command,
		"request_id": id,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal command: %w", err)
	}
	data = append(data, '\n')

	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("%w: write command: %v", ErrMpvUnavailable, err)
	}

	select {
	case resp := <-respCh:
		if resp.err != "" && resp.err != "success" {
			return resp.data, fmt.Errorf("mpv command error: %s", resp.err)
		}
		return resp.data, nil
	case <-p.stop:
		return nil, fmt.Errorf("%w: player stopped", ErrMpvUnavailable)
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("mpv command timed out")
	}
}
