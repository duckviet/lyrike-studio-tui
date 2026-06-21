package mpv

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/duckviet/lyrike-studio-tui/internal/playback"
)

type ipcResponse struct {
	RequestID int64           `json:"request_id"`
	Error     string          `json:"error"`
	Data      json.RawMessage `json:"data"`
	data      any
	err       string
	Event     string `json:"event"`
	Name      string `json:"name"`
}

func (p *Player) readLoop() {
	defer func() {
		p.mu.Lock()
		for id, pending := range p.pending {
			close(pending)
			delete(p.pending, id)
		}
		p.mu.Unlock()
		p.wg.Done()
	}()
	decoder := json.NewDecoder(p.conn)
	for {
		var msg ipcResponse
		if err := decoder.Decode(&msg); err != nil {
			return
		}
		p.handleMessage(msg)
	}
}

func (p *Player) handleLine(line string) {
	var msg ipcResponse
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return
	}
	p.handleMessage(msg)
}

func (p *Player) handleMessage(msg ipcResponse) {
	msg.err = msg.Error
	if len(msg.Data) > 0 {
		_ = json.Unmarshal(msg.Data, &msg.data)
	}
	if msg.Event != "" {
		p.handleEvent(msg)
		return
	}
	if msg.RequestID == 0 {
		return
	}

	p.mu.Lock()
	pending := p.pending[msg.RequestID]
	delete(p.pending, msg.RequestID)
	p.mu.Unlock()
	if pending != nil {
		select {
		case pending <- msg:
		default:
		}
	}
}

func (p *Player) handleEvent(msg ipcResponse) {
	if msg.Event != "property-change" {
		return
	}
	switch msg.Name {
	case "time-pos":
		seconds, ok := toFloat64(msg.Data)
		if !ok {
			return
		}
		position, err := playback.NewPosition(int64(seconds * 1000))
		if err != nil {
			return
		}
		p.updatePosition(position)
	case "duration":
		seconds, ok := toFloat64(msg.Data)
		if ok {
			p.updateDuration(seconds)
		}
	case "pause":
		var paused bool
		if err := json.Unmarshal(msg.Data, &paused); err == nil {
			p.updateState(paused)
		}
	}
}

func toFloat64(data json.RawMessage) (float64, bool) {
	var number float64
	if err := json.Unmarshal(data, &number); err == nil {
		return number, true
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return 0, false
	}
	number, err := strconv.ParseFloat(text, 64)
	return number, err == nil
}

func toFloat64Value(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case string:
		number, err := strconv.ParseFloat(v, 64)
		return number, err == nil
	default:
		return 0, false
	}
}

func (p *Player) observeTimePos() error {
	_, err := p.sendCommand([]any{"observe_property", 1, "time-pos"})
	return err
}

func (p *Player) getDurationSeconds() (float64, error) {
	data, err := p.sendCommand([]any{"get_property", "duration"})
	if err != nil {
		return 0, err
	}
	seconds, ok := toFloat64Value(data)
	if !ok {
		return 0, fmt.Errorf("unexpected duration type %T", data)
	}
	return seconds, nil
}

func (p *Player) getPause() (bool, error) {
	data, err := p.sendCommand([]any{"get_property", "pause"})
	if err != nil {
		return false, err
	}
	paused, ok := data.(bool)
	if !ok {
		return false, fmt.Errorf("unexpected pause type %T", data)
	}
	return paused, nil
}

func (p *Player) updatePosition(position playback.Position) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.snapshot.Position = position
}

func (p *Player) updateDuration(seconds float64) {
	duration, _ := playback.NewDuration(int64(seconds * 1000))
	p.mu.Lock()
	defer p.mu.Unlock()
	p.snapshot.Duration = duration
}

func (p *Player) updateState(paused bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if paused {
		p.snapshot.State = playback.StatePaused
	} else {
		p.snapshot.State = playback.StatePlaying
	}
}
