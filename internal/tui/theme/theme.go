package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

const (
	PanePaddingY = 0
	PanePaddingX = 1
)

// Palette is the small set of semantic colors every style is derived from.
type Palette struct {
	Accent   color.Color // primary accent: titles, active keys, active borders
	Accent2  color.Color // secondary accent: prompts, status, selected items
	Fg       color.Color // body text (NoColor = terminal default)
	Muted    color.Color // hints, dim text
	Subdued  color.Color // secondary descriptions
	Border   color.Color // box and rule borders
	Surface  color.Color // dark background for inactive items
	Unplayed color.Color // unplayed waveform/progress segments
	Good     color.Color // success
	Warn     color.Color // warning
	Bad      color.Color // error
	SelFg    color.Color // selected item foreground
	SelBg    color.Color // selected item background
}

// DefaultPalette returns the built-in lyrike-studio-tui palette.
func DefaultPalette() Palette {
	return Palette{
		Accent:   lipgloss.Color("#7D56F4"),
		Accent2:  lipgloss.Color("#FF3366"),
		Fg:       lipgloss.NoColor{},
		Muted:    lipgloss.Color("#888888"),
		Subdued:  lipgloss.Color("#666666"),
		Border:   lipgloss.Color("#555555"),
		Surface:  lipgloss.Color("#2C2C2C"),
		Unplayed: lipgloss.Color("#444444"),
		Good:     lipgloss.Color("#10B981"),
		Warn:     lipgloss.Color("#F59E0B"),
		Bad:      lipgloss.Color("#EF4444"),
		SelFg:    lipgloss.Color("#FFFFFF"),
		SelBg:    lipgloss.Color("#FF3366"),
	}
}

// Theme bundles a palette with the precomputed styles the views render with.
type Theme struct {
	Name string
	P    Palette

	FooterKey  lipgloss.Style
	FooterDesc lipgloss.Style
	StatusOK   lipgloss.Style
	StatusErr  lipgloss.Style

	ModalBorder lipgloss.Style
	ModalTitle  lipgloss.Style
	Prompt      lipgloss.Style
	SelItem     lipgloss.Style
	SelItemSel  lipgloss.Style
	SelDesc     lipgloss.Style

	PaneActive   lipgloss.Style
	PaneInactive lipgloss.Style
	Rule         lipgloss.Style

	Good lipgloss.Style
	Warn lipgloss.Style
	Bad  lipgloss.Style
	Dim  lipgloss.Style

	Title        lipgloss.Style
	Subtitle     lipgloss.Style
	Text         lipgloss.Style
	Status       lipgloss.Style
	Key          lipgloss.Style
	Desc         lipgloss.Style
	Button       lipgloss.Style
	Played       lipgloss.Style
	Unplayed     lipgloss.Style
	Knob         lipgloss.Style
	ActiveLine   lipgloss.Style
	SelectedLine lipgloss.Style
	ActiveItem   lipgloss.Style
	InactiveItem lipgloss.Style
	ErrorText    lipgloss.Style
	Value        lipgloss.Style
	Hint         lipgloss.Style
}

// NewTheme builds all styles from a palette.
func NewTheme(name string, p Palette) Theme {
	t := Theme{Name: name, P: p}
	border := lipgloss.NormalBorder()
	rounded := lipgloss.RoundedBorder()

	t.FooterKey = lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
	t.FooterDesc = lipgloss.NewStyle().Foreground(p.Muted)
	t.StatusOK = lipgloss.NewStyle().Foreground(p.Good)
	t.StatusErr = lipgloss.NewStyle().Foreground(p.Bad).Bold(true)

	t.ModalBorder = lipgloss.NewStyle().Border(border).BorderForeground(p.Accent).Padding(1, 2)
	t.ModalTitle = lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
	t.Prompt = lipgloss.NewStyle().Foreground(p.Accent2).Bold(true)
	t.SelItem = lipgloss.NewStyle().Foreground(p.Fg)
	t.SelItemSel = lipgloss.NewStyle().Foreground(p.SelFg).Background(p.SelBg).Bold(true)
	t.SelDesc = lipgloss.NewStyle().Foreground(p.Muted)

	t.PaneActive = lipgloss.NewStyle().Border(rounded).BorderForeground(p.Accent).Padding(PanePaddingY, PanePaddingX)
	t.PaneInactive = lipgloss.NewStyle().Border(rounded).BorderForeground(p.Border).Padding(PanePaddingY, PanePaddingX)
	t.Rule = lipgloss.NewStyle().Foreground(p.Border)

	t.Good = lipgloss.NewStyle().Foreground(p.Good)
	t.Warn = lipgloss.NewStyle().Foreground(p.Warn)
	t.Bad = lipgloss.NewStyle().Foreground(p.Bad)
	t.Dim = lipgloss.NewStyle().Foreground(p.Muted)

	t.Title = lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
	t.Subtitle = lipgloss.NewStyle().Foreground(p.Subdued).Italic(true)
	t.Text = lipgloss.NewStyle().Foreground(p.Fg)
	t.Status = lipgloss.NewStyle().Foreground(p.Accent2)
	t.Key = lipgloss.NewStyle().Foreground(p.Accent)
	t.Desc = lipgloss.NewStyle().Foreground(p.Subdued)
	t.Button = lipgloss.NewStyle().Background(p.Accent).Foreground(p.SelFg).Bold(true)
	t.Played = lipgloss.NewStyle().Foreground(p.Accent)
	t.Unplayed = lipgloss.NewStyle().Foreground(p.Unplayed)
	t.Knob = lipgloss.NewStyle().Foreground(p.Accent2)
	t.ActiveLine = lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
	t.SelectedLine = lipgloss.NewStyle().Foreground(p.Accent2)
	t.ActiveItem = lipgloss.NewStyle().Background(p.Accent).Foreground(p.SelFg).Bold(true)
	t.InactiveItem = lipgloss.NewStyle().Background(p.Surface).Foreground(p.Muted)
	t.ErrorText = lipgloss.NewStyle().Foreground(p.Bad)
	t.Value = lipgloss.NewStyle().Foreground(p.SelFg)
	t.Hint = lipgloss.NewStyle().Foreground(p.Muted)

	return t
}

// DefaultTheme returns the built-in lyrike-studio-tui theme.
func DefaultTheme() Theme {
	return NewTheme("default", DefaultPalette())
}
