package playback

import "fmt"

type Position int64

func NewPosition(milliseconds int64) (Position, error) {
	position := Position(milliseconds)
	if err := validatePosition("new_position", position); err != nil {
		return 0, err
	}

	return position, nil
}

func (p Position) Milliseconds() int64 {
	return int64(p)
}

func (p Position) String() string {
	return fmt.Sprintf("%dms", p.Milliseconds())
}

type Duration int64

func NewDuration(milliseconds int64) (Duration, error) {
	duration := Duration(milliseconds)
	if err := validateDuration("new_duration", duration); err != nil {
		return 0, err
	}

	return duration, nil
}

func (d Duration) Milliseconds() int64 {
	return int64(d)
}

func (d Duration) String() string {
	return fmt.Sprintf("%dms", d.Milliseconds())
}

type State string

const (
	StatePaused  State = "paused"
	StatePlaying State = "playing"
)

func (s State) String() string {
	return string(s)
}

type Snapshot struct {
	Position Position
	Duration Duration
	State    State
}

type Player interface {
	Snapshot() Snapshot
	Play() (Snapshot, error)
	Pause() (Snapshot, error)
	Seek(Position) (Snapshot, error)
	Tick(Duration) (Snapshot, error)
}

type ErrorReason string

const (
	ReasonNonPositiveDuration ErrorReason = "non_positive_duration"
	ReasonNegativePosition    ErrorReason = "negative_position"
	ReasonPositionOutOfBounds ErrorReason = "position_out_of_bounds"
	ReasonPlaybackAtEnd       ErrorReason = "playback_at_end"
)

type CommandError struct {
	Operation string
	Reason    ErrorReason
	Value     int64
	Limit     int64
}

func (e *CommandError) Error() string {
	if e.Limit > 0 {
		return fmt.Sprintf("playback %s failed: %s value=%d limit=%d", e.Operation, e.Reason, e.Value, e.Limit)
	}

	return fmt.Sprintf("playback %s failed: %s value=%d", e.Operation, e.Reason, e.Value)
}

func validateDuration(operation string, duration Duration) error {
	if duration <= 0 {
		return &CommandError{
			Operation: operation,
			Reason:    ReasonNonPositiveDuration,
			Value:     duration.Milliseconds(),
		}
	}

	return nil
}

func validatePosition(operation string, position Position) error {
	if position < 0 {
		return &CommandError{
			Operation: operation,
			Reason:    ReasonNegativePosition,
			Value:     position.Milliseconds(),
		}
	}

	return nil
}
