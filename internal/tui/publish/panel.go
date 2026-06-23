package publish

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type State string

const (
	StateConfirm  State = "confirm"
	StateValidate State = "validate"
	StatePoW      State = "pow"
	StatePublish  State = "publish"
	StateDone     State = "done"
	StateFailed   State = "failed"
)

type Panel struct {
	Title      string
	state      State
	token      string
	err        error
	retry      int
	lyrics     string
	trackName  string
	artistName string
}

func NewPanel() Panel {
	return Panel{
		Title: "Publish",
		state: StateValidate,
	}
}

func (p Panel) WithMetadata(track, artist string) Panel {
	p.trackName = track
	p.artistName = artist
	return p
}

func (p Panel) Confirm(lyrics string) Panel {
	p.lyrics = lyrics
	p.state = StateConfirm
	p.err = nil
	return p
}

func (p Panel) State() State {
	return p.state
}

func (p Panel) Token() string {
	return p.token
}

func (p Panel) Err() error {
	return p.err
}

func (p Panel) RetryCount() int {
	return p.retry
}

func (p Panel) Validate(lyrics string) (Panel, error) {
	if lyrics == "" {
		p.state = StateFailed
		p.err = errors.New("publish validation failed: lyrics are empty")
		return p, p.err
	}
	p.lyrics = lyrics
	p.err = nil
	p.state = StatePoW
	return p, nil
}

func (p Panel) SolveChallenge(prefix string, nonce string) (Panel, error) {
	if p.state != StatePoW {
		p.err = fmt.Errorf("publish challenge unavailable from state %s", p.state)
		p.state = StateFailed
		return p, p.err
	}
	p.token = prefix + nonce
	p.err = nil
	p.state = StatePublish
	return p, nil
}

func (p Panel) Publish(err error) Panel {
	if err != nil {
		p.err = err
		p.state = StateFailed
		return p
	}
	p.err = nil
	p.state = StateDone
	return p
}

func (p Panel) Retry() Panel {
	p.retry++
	p.err = nil
	if p.lyrics == "" {
		p.state = StateValidate
		return p
	}
	p.state = StatePoW
	return p
}

type ConfirmPublishMsg struct {
	Lyrics string
}

type CancelPublishMsg struct{}

type StartPublishRetryMsg struct {
	Lyrics string
}

func (p Panel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, nil
	}
	if p.state == StateConfirm {
		if key.Code == 'y' || key.Code == 'Y' {
			return p, func() tea.Msg {
				return ConfirmPublishMsg{Lyrics: p.lyrics}
			}
		}
		if key.Code == tea.KeyEscape {
			return p, func() tea.Msg {
				return CancelPublishMsg{}
			}
		}
	}
	if p.state == StateFailed && key.Code == 'r' {
		p = p.Retry()
		return p, func() tea.Msg {
			return StartPublishRetryMsg{Lyrics: p.lyrics}
		}
	}
	return p, nil
}

func (p Panel) View(width, height int) string {
	var sb strings.Builder
	sb.WriteString("Publishing Lyrics to LRCLIB\n\n")
	switch p.state {
	case StateConfirm:
		sb.WriteString("  Are you sure you want to publish lyrics?\n\n")
		sb.WriteString(fmt.Sprintf("  Track:  %s\n", p.trackName))
		sb.WriteString(fmt.Sprintf("  Artist: %s\n\n", p.artistName))
		sb.WriteString("  Press 'y' to confirm and publish, or Esc to cancel.")
	case StateValidate:
		sb.WriteString("  [ ] Validating lyrics...\n")
	case StatePoW:
		sb.WriteString("  [x] Lyrics validated\n")
		sb.WriteString("  [>] Requesting challenge and solving proof-of-work...\n")
	case StatePublish:
		sb.WriteString("  [x] Lyrics validated\n")
		sb.WriteString("  [x] Proof-of-work solved\n")
		sb.WriteString("  [>] Submitting to LRCLIB backend...\n")
	case StateDone:
		sb.WriteString("  [x] Lyrics validated\n")
		sb.WriteString("  [x] Proof-of-work solved\n")
		sb.WriteString("  [x] Published successfully!\n\n")
		sb.WriteString("Press Enter to return to editor.")
	case StateFailed:
		sb.WriteString(fmt.Sprintf("  [!] Error: %v\n\n", p.err))
		sb.WriteString("Press 'r' to retry, or Esc to return to editor.")
	}
	return sb.String()
}

// SolvePoW solves the proof-of-work challenge.
func SolvePoW(prefix string, targetHex string) (string, error) {
	target, err := hex.DecodeString(targetHex)
	if err != nil {
		return "", err
	}
	n := len(target)

	var nonce int64
	for {
		s := prefix + strconv.FormatInt(nonce, 10)
		h := sha256.Sum256([]byte(s))
		if verifyNonce(h[:], target, n) {
			return strconv.FormatInt(nonce, 10), nil
		}
		nonce++
	}
}

func verifyNonce(hash []byte, target []byte, n int) bool {
	if len(hash) != len(target) {
		return false
	}
	for i := 0; i < n-1; i++ {
		if hash[i] > target[i] {
			return false
		}
		if hash[i] < target[i] {
			return true
		}
	}
	return true
}
