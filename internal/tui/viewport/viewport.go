package viewport

type Model struct {
	Width      int
	Height     int
	YOffset    int
	TotalLines int
}

func New(width, height int) Model {
	return Model{
		Width:  width,
		Height: height,
	}
}

func (m Model) WithHeight(height int) Model {
	m.Height = height
	return m.Clamp()
}

func (m Model) WithTotalLines(totalLines int) Model {
	m.TotalLines = totalLines
	return m.Clamp()
}

func (m Model) Clamp() Model {
	maxOffset := m.TotalLines - m.Height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.YOffset > maxOffset {
		m.YOffset = maxOffset
	}
	if m.YOffset < 0 {
		m.YOffset = 0
	}
	return m
}

func (m Model) EnsureVisible(index int) Model {
	m = m.Clamp()
	if m.Height <= 0 {
		return m
	}
	if index < m.YOffset {
		m.YOffset = index
	} else if index >= m.YOffset+m.Height {
		m.YOffset = index - m.Height + 1
	}
	return m.Clamp()
}

func (m Model) ScrollUp() Model {
	m.YOffset = m.YOffset - 1
	return m.Clamp()
}

func (m Model) ScrollDown() Model {
	m.YOffset = m.YOffset + 1
	return m.Clamp()
}
