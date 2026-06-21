package waveform

import (
	"fmt"
	"math"

	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
)

type Panel struct {
	Title       string
	peaks       []float64
	positionMS  int64
	durationMS  int64
	loopStartMS int64
	loopEndMS   int64

	// Viewport bounds
	viewStartMS int64
	viewEndMS   int64

	// Last known width
	width int

	hoverCol int
	lines    []lyrics.Line
	follow   bool
}

func NewPanel() Panel {
	p := Panel{
		Title:       "Waveform",
		peaks:       []float64{0.15, 0.4, 0.8, 1, 0.6, 0.3},
		durationMS:  10_000,
		viewStartMS: 0,
		viewEndMS:   10_000,
		hoverCol:    -1,
		follow:      true,
	}
	return p.updateTitle()
}

func NewPanelWithPeaks(peaks []float64, durationMS int64) Panel {
	panel := NewPanel()
	panel.peaks = append([]float64(nil), peaks...)
	panel.durationMS = max(durationMS, 1)
	panel.viewStartMS = 0
	panel.viewEndMS = panel.durationMS
	return panel.updateTitle()
}

func (p Panel) WithPosition(positionMS int64) Panel {
	p.positionMS = clamp(positionMS, 0, p.durationMS)
	if p.follow {
		p = p.centerViewportOnPosition()
	}
	return p.updateTitle()
}

func (p Panel) WithLoop(startMS int64, endMS int64) Panel {
	start := clamp(startMS, 0, p.durationMS)
	end := clamp(endMS, start, p.durationMS)
	p.loopStartMS = start
	p.loopEndMS = end
	return p.updateTitle()
}

func (p Panel) WithWidth(w int) Panel {
	p.width = w
	return p.updateTitle()
}

func (p Panel) WithHover(col int) Panel {
	p.hoverCol = col
	return p
}

func (p Panel) WithLines(lines []lyrics.Line) Panel {
	p.lines = lines
	return p
}

func (p Panel) peakForColumn(column int, width int) float64 {
	if len(p.peaks) == 0 {
		return 0
	}
	if len(p.peaks) == 1 {
		return p.peaks[0]
	}
	ms := p.SeekForColumn(column, width)

	// Map ms -> float index in peaks
	frac := float64(ms) / float64(p.durationMS)
	floatIdx := frac * float64(len(p.peaks)-1)

	idx := int(math.Floor(floatIdx))
	if idx < 0 {
		return p.peaks[0]
	}
	if idx >= len(p.peaks)-1 {
		return p.peaks[len(p.peaks)-1]
	}

	// Linear interpolation to make zoom smooth (no steps)
	nextFrac := floatIdx - float64(idx)
	return p.peaks[idx]*(1.0-nextFrac) + p.peaks[idx+1]*nextFrac
}

func (p Panel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Code {
		case '+', '=':
			p = p.zoomAt(p.columnFor(p.positionMS, p.width), 0.85)
		case '-':
			p = p.zoomAt(p.columnFor(p.positionMS, p.width), 1/0.85)
		case '0':
			p.viewStartMS = 0
			p.viewEndMS = p.durationMS
			p = p.updateTitle()
		case 'f', 'F':
			p = p.ToggleFollow()
		}
	}
	return p, nil
}

func (p Panel) updateTitle() Panel {
	followStr := ""
	if p.follow {
		followStr = " [Follow]"
	}
	if p.viewSpanMS() == p.durationMS {
		p.Title = "Waveform" + followStr
	} else {
		p.Title = fmt.Sprintf("Waveform [%s]%s", p.zoomStatus(), followStr)
	}
	return p
}

func (p Panel) ToggleFollow() Panel {
	p.follow = !p.follow
	p = p.updateTitle()
	if p.follow {
		p = p.centerViewportOnPosition()
	}
	return p
}

func clamp(value int64, minimum int64, maximum int64) int64 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
