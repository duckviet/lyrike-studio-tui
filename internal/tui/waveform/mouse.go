package waveform

import (
	tea "charm.land/bubbletea/v2"
)

func (p Panel) HandleMouseLocal(localX int, button tea.MouseButton, mod tea.KeyMod) Panel {
	if p.width <= 0 {
		return p
	}

	switch button {
	case tea.MouseWheelUp, tea.MouseWheelDown:
		if mod&tea.ModCtrl != 0 || mod&tea.ModAlt != 0 {
			factor := 0.85
			if button == tea.MouseWheelDown {
				factor = 1.0 / 0.85
			}
			p = p.zoomAt(localX, factor)
		} else {
			panBy := p.viewSpanMS() / 8
			if button == tea.MouseWheelDown {
				p = p.pan(panBy)
			} else {
				p = p.pan(-panBy)
			}
		}
	case tea.MouseWheelLeft, tea.MouseWheelRight:
		panBy := p.viewSpanMS() / 8
		if button == tea.MouseWheelRight {
			p = p.pan(panBy)
		} else {
			p = p.pan(-panBy)
		}
	}
	return p
}
