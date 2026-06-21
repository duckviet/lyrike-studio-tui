package waveform

import (
	"fmt"
	"math"
	"strings"
)

func lerpColor(startHex, endHex string, frac float64) string {
	startHex = strings.TrimPrefix(startHex, "#")
	endHex = strings.TrimPrefix(endHex, "#")

	var r1, g1, b1, r2, g2, b2 int
	fmt.Sscanf(startHex, "%02x%02x%02x", &r1, &g1, &b1)
	fmt.Sscanf(endHex, "%02x%02x%02x", &r2, &g2, &b2)

	r := int(float64(r1) + frac*float64(r2-r1))
	g := int(float64(g1) + frac*float64(g2-g1))
	b := int(float64(b1) + frac*float64(b2-b1))

	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

var blocksUp = []rune{' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func upGlyph(frac float64) rune {
	idx := int(math.Round(frac * 7))
	if idx < 0 {
		idx = 0
	}
	if idx > 7 {
		idx = 7
	}
	return blocksUp[idx]
}

func downGlyph(frac float64) rune {
	switch {
	case frac >= 0.75:
		return '█'
	case frac >= 0.33:
		return '▀'
	case frac > 0:
		return '▔'
	default:
		return ' '
	}
}

func glyphForPeak(peak float64) rune {
	normalized := math.Max(0, math.Min(1, peak))
	switch {
	case normalized >= 0.75:
		return '█'
	case normalized >= 0.5:
		return '▓'
	case normalized >= 0.25:
		return '▒'
	case normalized > 0:
		return '░'
	default:
		return '·'
	}
}
