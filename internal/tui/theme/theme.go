package theme

import (
	"image/color"
	"strings"

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
	Border   color.Color // box and rule borders (used sparingly — active only)
	Surface  color.Color // slightly elevated background for inactive panes
	Surface2 color.Color // more elevated background for modals/overlays
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
		Accent:   lipgloss.Color("#7263E1"),
		Accent2:  lipgloss.Color("#FF3366"),
		Fg:       lipgloss.NoColor{},
		Muted:    lipgloss.Color("#666666"),
		Subdued:  lipgloss.Color("#4A4A4A"),
		Border:   lipgloss.Color("#7263E1"), // accent color — border is rare, so make it count
		Surface:  lipgloss.Color("#161616"), // inactive pane bg — just darker than terminal bg
		Surface2: lipgloss.Color("#222222"), // modal/overlay bg — slightly elevated
		Unplayed: lipgloss.Color("#333333"),
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

	// Panes: active gets a border, inactive gets a background — not both.
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

	// PaneTitle variants — dim title text reinforces inactive state without border
	PaneTitleActive   lipgloss.Style
	PaneTitleInactive lipgloss.Style
}

// NewTheme builds all styles from a palette.
func NewTheme(name string, p Palette) Theme {
	t := Theme{Name: name, P: p}
	border := lipgloss.NormalBorder()

	// ── Footer ──────────────────────────────────────────────────────────────
	t.FooterKey = lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
	t.FooterDesc = lipgloss.NewStyle().Foreground(p.Muted)
	t.StatusOK = lipgloss.NewStyle().Foreground(p.Good)
	t.StatusErr = lipgloss.NewStyle().Foreground(p.Bad).Bold(true)

	// ── Modal ────────────────────────────────────────────────────────────────
	t.ModalBorder = lipgloss.NewStyle().
		Background(p.Surface2).
		Padding(1, 2).
		Border(lipgloss.ThickBorder(), false, false, false, true). // left-only
		BorderForeground(p.Accent)
	t.ModalTitle = lipgloss.NewStyle().Foreground(p.Accent).Bold(true).Background(p.Surface2)
	t.Prompt = lipgloss.NewStyle().Foreground(p.Accent2).Bold(true).Background(p.Surface2)
	t.SelItem = lipgloss.NewStyle().Foreground(p.Fg).Background(p.Surface2)
	t.SelItemSel = lipgloss.NewStyle().Foreground(p.SelFg).Background(p.SelBg).Bold(true)
	t.SelDesc = lipgloss.NewStyle().Foreground(p.Muted).Background(p.Surface2)

	// ── Panes ────────────────────────────────────────────────────────────────
	t.PaneActive = lipgloss.NewStyle().
		Border(border).
		BorderForeground(p.Accent).
		Background(p.Surface).
		Padding(PanePaddingY, PanePaddingX)

	t.PaneInactive = lipgloss.NewStyle().
		Border(border).
		BorderForeground(p.Surface).
		Background(p.Surface).
		Padding(PanePaddingY, PanePaddingX)

	t.Rule = lipgloss.NewStyle().Foreground(p.Subdued).Background(p.Surface)

	t.PaneTitleActive = lipgloss.NewStyle().Foreground(p.Accent).Bold(true).Background(p.Surface)
	t.PaneTitleInactive = lipgloss.NewStyle().Foreground(p.Muted).Background(p.Surface)

	// ── Semantic ─────────────────────────────────────────────────────────────
	t.Good = lipgloss.NewStyle().Foreground(p.Good).Background(p.Surface)
	t.Warn = lipgloss.NewStyle().Foreground(p.Warn).Background(p.Surface)
	t.Bad = lipgloss.NewStyle().Foreground(p.Bad).Background(p.Surface)
	t.Dim = lipgloss.NewStyle().Foreground(p.Muted).Background(p.Surface)

	// ── Text roles ───────────────────────────────────────────────────────────
	t.Title = lipgloss.NewStyle().Foreground(p.Accent).Bold(true).Background(p.Surface)
	t.Subtitle = lipgloss.NewStyle().Foreground(p.Subdued).Italic(true).Background(p.Surface)
	t.Text = lipgloss.NewStyle().Foreground(p.Fg).Background(p.Surface)
	t.Status = lipgloss.NewStyle().Foreground(p.Accent2).Background(p.Surface)
	t.Key = lipgloss.NewStyle().Foreground(p.Accent).Background(p.Surface)
	t.Desc = lipgloss.NewStyle().Foreground(p.Subdued).Background(p.Surface)
	t.Button = lipgloss.NewStyle().Background(p.Accent).Foreground(p.SelFg).Bold(true)
	t.Played = lipgloss.NewStyle().Foreground(p.Accent).Background(p.Surface)
	t.Unplayed = lipgloss.NewStyle().Foreground(p.Unplayed).Background(p.Surface)
	t.Knob = lipgloss.NewStyle().Foreground(p.Accent2).Background(p.Surface)
	t.ActiveLine = lipgloss.NewStyle().Foreground(p.Accent).Bold(true).Background(p.Surface)
	t.SelectedLine = lipgloss.NewStyle().Foreground(p.Accent2).Background(p.Surface)
	t.ActiveItem = lipgloss.NewStyle().Background(p.Accent).Foreground(p.SelFg).Bold(true)
	t.InactiveItem = lipgloss.NewStyle().Background(p.Surface).Foreground(p.Muted)
	t.ErrorText = lipgloss.NewStyle().Foreground(p.Bad).Background(p.Surface)
	t.Value = lipgloss.NewStyle().Foreground(p.SelFg).Background(p.Surface)
	t.Hint = lipgloss.NewStyle().Foreground(p.Muted).Background(p.Surface)

	return t
}

// Modal returns the style s with its background set to the modal surface color.
func (t Theme) Modal(s lipgloss.Style) lipgloss.Style {
	return s.Background(t.P.Surface2)
}

// PaintModal wraps a string so that its background matches the modal surface.
func (t Theme) PaintModal(s string) string {
	return PaintBackground(s, t.P.Surface2)
}

// PaintPane wraps a string so that its background matches the pane surface.
func (t Theme) PaintPane(s string) string {
	return PaintBackground(s, t.P.Surface)
}

// PaintBackground ensures that any style reset within the rendered string
// immediately re-asserts the theme's background color, preventing unstyled
// segments/spaces from falling back to default black.
func PaintBackground(s string, bg color.Color) string {
	if bg == nil {
		return s
	}
	res := lipgloss.NewStyle().Background(bg).Render(" ")
	if len(res) <= 4 {
		return s
	}
	var bgSeq string
	if strings.HasSuffix(res, "\x1b[m") {
		bgSeq = res[:len(res)-4]
	} else if strings.HasSuffix(res, "\x1b[0m") {
		bgSeq = res[:len(res)-5]
	} else {
		return s
	}

	s = strings.ReplaceAll(s, "\x1b[m", "\x1b[m"+bgSeq)
	s = strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+bgSeq)
	return bgSeq + s + "\x1b[m"
}

// DefaultTheme returns the built-in lyrike-studio-tui theme.
func DefaultTheme() Theme {
	return NewTheme("default", DefaultPalette())
}