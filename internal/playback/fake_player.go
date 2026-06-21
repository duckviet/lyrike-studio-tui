package playback

type FakePlayer struct {
	duration Duration
	position Position
	state    State
}

func NewFakePlayer(duration Duration) (*FakePlayer, error) {
	if err := validateDuration("new_fake_player", duration); err != nil {
		return nil, err
	}

	return &FakePlayer{
		duration: duration,
		state:    StatePaused,
	}, nil
}

func (p *FakePlayer) SetDuration(duration Duration) {
	p.duration = duration
}

func (p *FakePlayer) Snapshot() Snapshot {
	return Snapshot{
		Position: p.position,
		Duration: p.duration,
		State:    p.state,
	}
}

func (p *FakePlayer) Play() (Snapshot, error) {
	if p.position == Position(p.duration) {
		return p.Snapshot(), &CommandError{
			Operation: "play",
			Reason:    ReasonPlaybackAtEnd,
			Value:     p.position.Milliseconds(),
			Limit:     p.duration.Milliseconds(),
		}
	}

	p.state = StatePlaying
	return p.Snapshot(), nil
}

func (p *FakePlayer) Pause() (Snapshot, error) {
	p.state = StatePaused
	return p.Snapshot(), nil
}

func (p *FakePlayer) Seek(position Position) (Snapshot, error) {
	if err := validatePosition("seek", position); err != nil {
		return p.Snapshot(), err
	}
	if position > Position(p.duration) {
		return p.Snapshot(), &CommandError{
			Operation: "seek",
			Reason:    ReasonPositionOutOfBounds,
			Value:     position.Milliseconds(),
			Limit:     p.duration.Milliseconds(),
		}
	}

	p.position = position
	if p.position == Position(p.duration) {
		p.state = StatePaused
	}

	return p.Snapshot(), nil
}

func (p *FakePlayer) Tick(delta Duration) (Snapshot, error) {
	if err := validateDuration("tick", delta); err != nil {
		return p.Snapshot(), err
	}
	if p.state != StatePlaying {
		return p.Snapshot(), nil
	}

	remaining := p.duration - Duration(p.position)
	if delta >= remaining {
		p.position = Position(p.duration)
		p.state = StatePaused
		return p.Snapshot(), nil
	}

	p.position += Position(delta)
	return p.Snapshot(), nil
}
