package tui

import (
	"fmt"
	"math"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
	"github.com/duckviet/lyrike-studio-tui/internal/integrations/backend"
	"github.com/duckviet/lyrike-studio-tui/internal/playback"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/waveform"
)

// DemoModel returns a Model pre-populated with a small demo lyrics document.
func DemoModel() (Model, error) {
	doc, err := demoDocument()
	if err != nil {
		return Model{}, fmt.Errorf("demo document: %w", err)
	}
	dur, _ := playback.NewDuration(212000)
	player, _ := playback.NewFakePlayer(dur)
	client := backend.NewClient("http://127.0.0.1:8080")
	m := NewModel(doc, client, player, "", "")
	m.trackName = "Demo Track"
	m.artistName = "Demo Artist"
	m.media = m.media.WithMetadata(m.trackName, m.artistName)

	peaks := make([]float64, 120)
	for i := range peaks {
		peaks[i] = 0.2 + 0.7*math.Sin(float64(i)*0.15)*math.Sin(float64(i)*0.15)
	}
	m.waveform = waveform.NewPanelWithPeaks(peaks, 212000)
	return m, nil
}

func DemoFixtureModel(baseURL string) (Model, error) {
	doc, err := demoDocument()
	if err != nil {
		return Model{}, fmt.Errorf("demo document: %w", err)
	}
	dur, _ := playback.NewDuration(212000)
	player, _ := playback.NewFakePlayer(dur)
	client := backend.NewClient(baseURL)
	
	// Start playing immediately in the fake player for the demo test
	_, _ = player.Play()
	
	m := NewModel(doc, client, player, "", "")
	m.trackName = "Demo Track (Fixture)"
	m.artistName = "Demo Artist"
	m.media = m.media.WithMetadata(m.trackName, m.artistName)

	peaks := make([]float64, 120)
	for i := range peaks {
		peaks[i] = 0.2 + 0.7*math.Sin(float64(i)*0.15)*math.Sin(float64(i)*0.15)
	}
	m.waveform = waveform.NewPanelWithPeaks(peaks, 212000)
	return m, nil
}

func demoDocument() (lyrics.Document, error) {
	lines := []struct {
		ms   int64
		text string
	}{
		{0, "First line of lyrics"},
		{5000, "Second line of lyrics"},
		{10000, "Third line of lyrics"},
	}

	var docLines []lyrics.Line
	for _, entry := range lines {
		ts, err := lyrics.NewTimestamp(entry.ms)
		if err != nil {
			return lyrics.Document{}, fmt.Errorf("timestamp %d: %w", entry.ms, err)
		}
		te, err := lyrics.NewTimestamp(entry.ms + 3_000)
		if err != nil {
			return lyrics.Document{}, fmt.Errorf("end timestamp %d: %w", entry.ms+3_000, err)
		}
		txt, err := lyrics.NewText(entry.text)
		if err != nil {
			return lyrics.Document{}, fmt.Errorf("text %q: %w", entry.text, err)
		}
		line, err := lyrics.NewLine(ts, te, txt)
		if err != nil {
			return lyrics.Document{}, fmt.Errorf("line %q: %w", entry.text, err)
		}
		docLines = append(docLines, line)
	}

	doc, err := lyrics.NewDocument(docLines)
	if err != nil {
		return lyrics.Document{}, fmt.Errorf("document: %w", err)
	}
	return doc, nil
}
