package beep

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/duckviet/lyrike-studio-tui/internal/playback"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/wav"
)

var (
	speakerOnce sync.Once
)

type Player struct {
	filePath    string
	tempWavPath string
	wavFile     *os.File
	streamer    beep.StreamSeekCloser
	format      beep.Format
	ctrl        *beep.Ctrl
	duration    playback.Duration
}

// Compile-time check to ensure Player implements playback.Player and io.Closer
var _ playback.Player = (*Player)(nil)
var _ io.Closer = (*Player)(nil)

func NewPlayer(filePath string) (*Player, error) {
	p := &Player{
		filePath: filePath,
	}

	targetPath := filePath
	// If the file is not a WAV file, transcode it to a temporary WAV using ffmpeg.
	if !strings.HasSuffix(strings.ToLower(filePath), ".wav") {
		tempFile, err := os.CreateTemp("", "lyrike-*.wav")
		if err != nil {
			return nil, fmt.Errorf("create temp file: %w", err)
		}
		tempWavPath := tempFile.Name()
		tempFile.Close() // Close fd so ffmpeg can write to it

		// Transcode synchronously
		cmd := exec.Command("ffmpeg", "-y", "-i", filePath, "-acodec", "pcm_s16le", "-ac", "2", "-ar", "44100", tempWavPath)
		if err := cmd.Run(); err != nil {
			os.Remove(tempWavPath)
			return nil, fmt.Errorf("ffmpeg transcoding failed: %w", err)
		}

		p.tempWavPath = tempWavPath
		targetPath = tempWavPath
	}

	wavFile, err := os.Open(targetPath)
	if err != nil {
		p.cleanup()
		return nil, fmt.Errorf("open wav file: %w", err)
	}
	p.wavFile = wavFile

	streamer, format, err := wav.Decode(wavFile)
	if err != nil {
		p.cleanup()
		return nil, fmt.Errorf("decode wav: %w", err)
	}
	p.streamer = streamer
	p.format = format

	// Calculate duration
	durMS := format.SampleRate.D(streamer.Len()).Milliseconds()
	dur, _ := playback.NewDuration(durMS)
	p.duration = dur

	// Initialize speaker once using the track's sample rate.
	var initErr error
	speakerOnce.Do(func() {
		initErr = speaker.Init(format.SampleRate, format.SampleRate.N(time.Millisecond*120))
	})
	if initErr != nil {
		p.cleanup()
		return nil, fmt.Errorf("init speaker: %w", initErr)
	}

	p.ctrl = &beep.Ctrl{Streamer: streamer, Paused: true}
	speaker.Play(p.ctrl)

	return p, nil
}

func (p *Player) Snapshot() playback.Snapshot {
	if p.streamer == nil {
		return playback.Snapshot{}
	}

	speaker.Lock()
	posSamples := p.streamer.Position()
	isPaused := p.ctrl.Paused
	speaker.Unlock()

	posMS := p.format.SampleRate.D(posSamples).Milliseconds()

	state := playback.StatePaused
	if !isPaused {
		state = playback.StatePlaying
	}

	// Auto-pause at the end of track
	if posSamples >= p.streamer.Len() && !isPaused {
		speaker.Lock()
		p.ctrl.Paused = true
		speaker.Unlock()
		state = playback.StatePaused
	}

	return playback.Snapshot{
		Position: playback.Position(posMS),
		Duration: p.duration,
		State:    state,
	}
}

func (p *Player) Play() (playback.Snapshot, error) {
	if p.ctrl != nil {
		speaker.Lock()
		// If we're at the end, wrap around to start
		if p.streamer.Position() >= p.streamer.Len() {
			_ = p.streamer.Seek(0)
		}
		p.ctrl.Paused = false
		speaker.Unlock()
	}
	return p.Snapshot(), nil
}

func (p *Player) Pause() (playback.Snapshot, error) {
	if p.ctrl != nil {
		speaker.Lock()
		p.ctrl.Paused = true
		speaker.Unlock()
	}
	return p.Snapshot(), nil
}

func (p *Player) Seek(pos playback.Position) (playback.Snapshot, error) {
	if p.streamer == nil {
		return p.Snapshot(), fmt.Errorf("no active stream")
	}

	samples := p.format.SampleRate.N(time.Duration(pos.Milliseconds()) * time.Millisecond)
	if samples < 0 {
		samples = 0
	}
	if samples > p.streamer.Len() {
		samples = p.streamer.Len()
	}

	speaker.Lock()
	err := p.streamer.Seek(samples)
	speaker.Unlock()

	if err != nil {
		return p.Snapshot(), fmt.Errorf("seek failed: %w", err)
	}

	return p.Snapshot(), nil
}

func (p *Player) Tick(dur playback.Duration) (playback.Snapshot, error) {
	// The speaker runs concurrently, so we just snapshot the state.
	return p.Snapshot(), nil
}

func (p *Player) Close() error {
	p.cleanup()
	return nil
}

func (p *Player) cleanup() {
	speaker.Lock()
	if p.ctrl != nil {
		p.ctrl.Paused = true
	}
	speaker.Unlock()

	speaker.Clear()

	if p.streamer != nil {
		p.streamer.Close()
		p.streamer = nil
	}

	if p.wavFile != nil {
		p.wavFile.Close()
		p.wavFile = nil
	}

	if p.tempWavPath != "" {
		os.Remove(p.tempWavPath)
		p.tempWavPath = ""
	}
}
