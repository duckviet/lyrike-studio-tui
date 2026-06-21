package waveform

import (
	"fmt"
	"math"
	"strings"
)

const minSpanMS = 1000 // 1 second minimum zoom

func (p Panel) viewSpanMS() int64 {
	span := p.viewEndMS - p.viewStartMS
	if span < 1 {
		return 1
	}
	return span
}

func (p Panel) SeekForColumn(column int, width int) int64 {
	if width <= 1 {
		return p.viewStartMS
	}
	c := clamp(int64(column), 0, int64(width-1))
	return p.viewStartMS + c*p.viewSpanMS()/int64(width-1)
}

func (p Panel) columnFor(ms int64, width int) int {
	if width <= 1 || p.viewSpanMS() <= 0 {
		return 0
	}
	rel := ms - p.viewStartMS
	return int(rel * int64(width-1) / p.viewSpanMS())
}

func (p Panel) zoomAt(mouseCol int, factor float64) Panel {
	if p.width <= 1 {
		return p
	}
	if mouseCol < 0 {
		mouseCol = 0
	} else if mouseCol > p.width-1 {
		mouseCol = p.width - 1
	}

	// Time at mouse position remains fixed
	anchorMS := p.SeekForColumn(mouseCol, p.width)
	anchorFrac := float64(mouseCol) / float64(p.width-1)

	newSpan := int64(float64(p.viewSpanMS()) * factor)
	newSpan = clamp(newSpan, minSpanMS, p.durationMS)

	newStart := anchorMS - int64(anchorFrac*float64(newSpan))
	newEnd := newStart + newSpan

	if newStart < 0 {
		newStart = 0
		newEnd = newSpan
	}
	if newEnd > p.durationMS {
		newEnd = p.durationMS
		newStart = newEnd - newSpan
	}
	p.viewStartMS = clamp(newStart, 0, p.durationMS)
	p.viewEndMS = clamp(newEnd, p.viewStartMS+1, p.durationMS)
	return p.updateTitle()
}

func (p Panel) pan(deltaMS int64) Panel {
	p.follow = false
	span := p.viewSpanMS()
	newStart := p.viewStartMS + deltaMS
	if newStart < 0 {
		newStart = 0
	}
	if newStart+span > p.durationMS {
		newStart = p.durationMS - span
	}
	p.viewStartMS = clamp(newStart, 0, p.durationMS)
	p.viewEndMS = clamp(newStart+span, p.viewStartMS+1, p.durationMS)
	return p.updateTitle()
}

func (p Panel) zoomStatus() string {
	if p.durationMS <= 0 {
		return "100%"
	}

	const mapSize = 8
	startFrac := float64(p.viewStartMS) / float64(p.durationMS)
	spanFrac := float64(p.viewSpanMS()) / float64(p.durationMS)

	startIdx := int(math.Round(startFrac * mapSize))
	spanLen := int(math.Round(spanFrac * mapSize))
	if spanLen < 1 {
		spanLen = 1
	}
	endIdx := startIdx + spanLen
	if endIdx > mapSize {
		endIdx = mapSize
		startIdx = endIdx - spanLen
	}

	var sb strings.Builder
	for i := 0; i < mapSize; i++ {
		if i >= startIdx && i < endIdx {
			sb.WriteRune('█')
		} else {
			sb.WriteRune('░')
		}
	}

	startStr := formatTimelineMS(p.viewStartMS)
	endStr := formatTimelineMS(p.viewEndMS)
	sec := p.viewSpanMS() / 1000

	return fmt.Sprintf("%s %s–%s (%ds)", sb.String(), startStr, endStr, sec)
}

func (p Panel) centerViewportOnPosition() Panel {
	span := p.viewSpanMS()
	newStart := p.positionMS - span/2
	newEnd := newStart + span

	if newStart < 0 {
		newStart = 0
		newEnd = span
	}
	if newEnd > p.durationMS {
		newEnd = p.durationMS
		newStart = newEnd - span
	}

	p.viewStartMS = clamp(newStart, 0, p.durationMS)
	p.viewEndMS = clamp(newEnd, p.viewStartMS+1, p.durationMS)
	return p
}
