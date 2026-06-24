package tui

import (
	"context"
	"io"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
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

// PlayerFactory creates a playback.Player for the given video ID and returns
// the player plus a status message describing how it was initialized.
type PlayerFactory func(videoID string) (playback.Player, string)

// Model is the root Bubble Tea model for the three-panel shell.
type Model struct {
	width        int
	height       int
	focus        focus
	theme        Theme
	media        media.Panel
	waveform     waveform.Panel
	editor       editor.Panel
	publish      publish.Panel
	status       string
	statusErr    bool
	overlay      overlayKind
	picker       selector
	pickerTarget draft.ProjectID
	confirm      confirmView
	help         helpView
	fetchInput   fetchInput
	dirty        bool

	mediaDragging bool

	client         *backend.Client
	player         playback.Player
	playerFactory  PlayerFactory
	draftStore     storage.Store
	projectID      draft.ProjectID
	videoID        string
	sourceURL      string
	trackName      string
	artistName     string
	albumName      string
	metadataEditor metadataEditor
	transcribeChan chan backend.TranscribeResponse
}

// NewModel builds a shell model with the given panels.
func NewModel(doc lyrics.Document, client *backend.Client, player playback.Player, videoID string, sourceURL string) Model {
	projectID, _ := draft.NewProjectID(videoID)
	return NewModelWithDraftStore(doc, client, player, storage.NewDefaultStore(), projectID, videoID, sourceURL)
}

func NewModelWithDraftStore(doc lyrics.Document, client *backend.Client, player playback.Player, store storage.Store, projectID draft.ProjectID, videoID string, sourceURL string) Model {
	th := DefaultTheme()
	ti := textinput.New()
	ti.Prompt = th.Prompt.Render("❯ ")
	styles := ti.Styles()
	styles.Cursor.Blink = false
	styles.Focused.Text = styles.Focused.Text.Background(th.P.Surface2)
	styles.Focused.Placeholder = styles.Focused.Placeholder.Background(th.P.Surface2)
	styles.Blurred.Text = styles.Blurred.Text.Background(th.P.Surface2)
	styles.Blurred.Placeholder = styles.Blurred.Placeholder.Background(th.P.Surface2)
	ti.SetStyles(styles)

	return Model{
		focus:      focusMedia,
		theme:      th,
		media:      media.NewPanel().WithTheme(th),
		waveform:   waveform.NewPanel().WithTheme(th),
		editor:     editor.NewPanel(doc).WithTheme(th),
		publish:    publish.NewPanel(),
		picker:     newSelector(th),
		confirm:    confirmView{th: th},
		help:       newHelpView(th),
		fetchInput: fetchInput{input: ti},
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

func (m Model) WithPlayerFactory(factory PlayerFactory) Model {
	m.playerFactory = factory
	return m
}

func (m Model) WithStatus(lines []string) Model {
	m.status = strings.Join(lines, " | ")
	m.statusErr = false
	return m
}

func (m *Model) setStatus(status string) {
	m.status = status
	m.statusErr = false
}

func (m *Model) setErrorStatus(status string) {
	m.status = status
	m.statusErr = true
}

func (m Model) WithTheme(th Theme) Model {
	m.theme = th
	m.media = m.media.WithTheme(th)
	m.waveform = m.waveform.WithTheme(th)
	m.editor = m.editor.WithTheme(th)
	m.picker.th = th
	m.picker.input.Prompt = th.Prompt.Render("❯ ")
	m.confirm.th = th
	m.fetchInput.input.Prompt = th.Prompt.Render("❯ ")
	m.help.th = th
	return m
}

func (m Model) Close() error {
	if closer, ok := m.player.(io.Closer); ok {
		return closer.Close()
	}
	return nil
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
	prefix string
	nonce  string
	err    error
}

type publishResultMsg struct {
	err error
}
