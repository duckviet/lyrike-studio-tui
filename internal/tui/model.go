package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
	"github.com/duckviet/lyrike-studio-tui/internal/integrations/backend"
	"github.com/duckviet/lyrike-studio-tui/internal/playback"
	"github.com/duckviet/lyrike-studio-tui/internal/storage"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/editor"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/media"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/publish"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/waveform"
)

// focus identifies which panel owns keyboard input.
type focus int

const (
	focusMedia focus = iota
	focusWaveform
	focusEditor
	focusPublish
)

// Model is the root Bubble Tea model for the three-panel shell.
type Model struct {
	width    int
	height   int
	focus    focus
	media    media.Panel
	waveform waveform.Panel
	editor   editor.Panel
	publish  publish.Panel
	status   []string
	picker   projectPicker
	dirty    bool

	mediaDragging bool

	client         *backend.Client
	player         playback.Player
	draftStore     storage.Store
	projectID      draft.ProjectID
	videoID        string
	sourceURL      string
	trackName      string
	artistName     string
	albumName      string
	metadataEditor metadataEditor
}

// NewModel builds a shell model with the given panels.
func NewModel(doc lyrics.Document, client *backend.Client, player playback.Player, videoID string, sourceURL string) Model {
	projectID, _ := draft.NewProjectID(videoID)
	return NewModelWithDraftStore(doc, client, player, storage.NewDefaultStore(), projectID, videoID, sourceURL)
}

func NewModelWithDraftStore(doc lyrics.Document, client *backend.Client, player playback.Player, store storage.Store, projectID draft.ProjectID, videoID string, sourceURL string) Model {
	return Model{
		focus:      focusMedia,
		media:      media.NewPanel(),
		waveform:   waveform.NewPanel(),
		editor:     editor.NewPanel(doc),
		publish:    publish.NewPanel(),
		client:     client,
		player:     player,
		draftStore: store,
		projectID:  projectID,
		videoID:    videoID,
		sourceURL:  sourceURL,
	}
}

func (m Model) WithPublish(panel publish.Panel) Model {
	m.publish = panel
	return m
}

func (m Model) WithStatus(lines []string) Model {
	m.status = append([]string(nil), lines...)
	return m
}

type fetchMediaMsg struct {
	resp backend.FetchResponse
	err  error
}

type fetchPeaksMsg struct {
	resp backend.PeaksResponse
	err  error
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Init returns the initial command.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, tickCmd())
	if m.client != nil && (m.videoID != "" || m.sourceURL != "") {
		cmds = append(cmds,
			func() tea.Msg {
				req := backend.FetchRequest{VideoID: m.videoID}
				if m.sourceURL != "" {
					req.URL = m.sourceURL
				}
				resp, err := m.client.Fetch(context.Background(), req)
				return fetchMediaMsg{resp: resp, err: err}
			},
		)
	}
	return tea.Batch(cmds...)
}

// View renders the full three-panel layout.
func (m Model) View() tea.View {
	v := tea.NewView(renderLayout(m))
	v.MouseMode = tea.MouseModeAllMotion
	v.AltScreen = true
	return v
}

type challengeMsg struct {
	challenge backend.ChallengeResponse
	err       error
}

type powSolvedMsg struct {
	token string
	err   error
}

type publishResultMsg struct {
	err error
}
