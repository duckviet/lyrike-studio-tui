package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
	"github.com/duckviet/lyrike-studio-tui/internal/integrations/backend"
	"github.com/duckviet/lyrike-studio-tui/internal/playback"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/editor"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/media"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/publish"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/waveform"
)

// Update handles incoming messages and routes panel-local keys to the focused panel.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.waveform = m.waveform.WithWidth(m.width - 2)
		topHeight, _, _, _, _ := calculateLayout(m.width, m.height, len(m.status))
		m.editor = m.editor.WithHeight(topHeight - 3)
		return m, nil

	case tea.KeyPressMsg:
		if m.fetchInput.active() {
			return m.updateFetchInput(msg)
		}
		if m.picker.active() {
			return m.updateProjectPicker(msg)
		}
		if m.metadataEditor.active {
			return m.updateMetadataEditor(msg)
		}
		return m.updateKey(msg)

	case tea.PasteMsg:
		if m.fetchInput.active() {
			return m.updateFetchInput(msg)
		}
		if m.metadataEditor.active {
			return m.updateMetadataEditor(msg)
		}
		return m.updateFocusedPanel(msg)

	case tea.MouseMsg:
		_, isMotion := msg.(tea.MouseMotionMsg)
		var cmd tea.Cmd
		m, cmd = m.handleMouse(msg, isMotion)
		return m, cmd

	case fetchMediaMsg:
		if msg.err != nil {
			m.status = []string{"fetch failed: " + msg.err.Error()}
			return m, nil
		}
		m.videoID = msg.resp.VideoID
		if m.projectID == "" {
			m.projectID, _ = draft.NewProjectID(msg.resp.VideoID)
		}
		if msg.resp.SourceURL != nil {
			m.sourceURL = *msg.resp.SourceURL
		}
		if m.trackName == "" || m.trackName == "Unknown Track" || m.trackName == "No Track Loaded" {
			m.trackName = msg.resp.TrackName
		}
		if m.artistName == "" || m.artistName == "Unknown Artist" {
			m.artistName = msg.resp.ArtistName
		}
		if fp, ok := m.player.(*playback.FakePlayer); ok {
			if dur, err := playback.NewDuration(int64(msg.resp.Duration) * 1000); err == nil {
				fp.SetDuration(dur)
			}
		}
		m.media = m.media.WithMetadata(m.trackName, m.artistName, m.albumName).
			WithTransport(media.TransportPaused, 0, int64(msg.resp.Duration)*1000)
		m.status = []string{"fetch complete"}
		return m, func() tea.Msg {
			resp, err := m.client.Peaks(context.Background(), msg.resp.VideoID, backend.SourceOriginal, 2000)
			return fetchPeaksMsg{resp: resp, err: err}
		}

	case fetchPeaksMsg:
		if msg.err != nil {
			m.status = []string{"peaks failed: " + msg.err.Error()}
			return m, nil
		}
		m.waveform = waveform.NewPanelWithPeaks(msg.resp.Peaks, int64(msg.resp.Duration)*1000).WithTheme(m.theme)
		return m, nil

	case editor.SeekToMSMsg:
		if m.player != nil {
			pos, _ := playback.NewPosition(int64(msg))
			snap, _ := m.player.Seek(pos)
			state := media.TransportPaused
			if snap.State == playback.StatePlaying {
				state = media.TransportPlaying
			}
			m.media = m.media.WithTransport(state, snap.Position.Milliseconds(), snap.Duration.Milliseconds())
			m.waveform = m.waveform.WithPosition(snap.Position.Milliseconds())
			m.editor = m.editor.WithPlaybackPosition(snap.Position.Milliseconds())
		}
		return m, nil

	case tickMsg:
		var cmd tea.Cmd
		if m.player != nil {
			snapshot, err := m.player.Tick(playback.Duration(100))
			if err == nil {
				state := media.TransportPaused
				if snapshot.State == playback.StatePlaying {
					state = media.TransportPlaying
				}
				m.media = m.media.WithTransport(state, snapshot.Position.Milliseconds(), snapshot.Duration.Milliseconds())
				m.waveform = m.waveform.WithPosition(snapshot.Position.Milliseconds())
				m.editor = m.editor.WithPlaybackPosition(snapshot.Position.Milliseconds())
			}
		}
		return m, tea.Batch(tickCmd(), cmd)

	case publish.ConfirmPublishMsg:
		var err error
		m.publish, err = m.publish.Validate(msg.Lyrics)
		if err != nil {
			return m, nil
		}
		return m, m.requestAndSolveChallengeCmd()

	case publish.CancelPublishMsg:
		m.focus = focusEditor
		return m, nil

	case publish.StartPublishRetryMsg:
		m.focus = focusPublish
		return m, m.requestAndSolveChallengeCmd()

	case challengeMsg:
		if msg.err != nil {
			m.publish = m.publish.Publish(fmt.Errorf("challenge request failed: %w", msg.err))
			return m, nil
		}
		return m, m.solvePoWCmd(msg.challenge)

	case powSolvedMsg:
		if msg.err != nil {
			m.publish = m.publish.Publish(fmt.Errorf("pow solve failed: %w", msg.err))
			return m, nil
		}
		var err error
		m.publish, err = m.publish.SolveChallenge(msg.prefix, msg.nonce)
		if err != nil {
			m.publish = m.publish.Publish(err)
			return m, nil
		}
		token := backend.PublishToken(msg.prefix, msg.nonce)
		return m, m.submitPublishCmd(token)

	case publishResultMsg:
		m.publish = m.publish.Publish(msg.err)
		if msg.err == nil {
			m.status = []string{"publish success"}
		} else {
			m.status = []string{"publish failed: " + msg.err.Error()}
		}
		return m, nil

	case backend.TranscribeResponse:
		status := string(msg.Status())
		m.media = m.media.WithTranscribeStatus(status)
		m.status = []string{fmt.Sprintf("transcription status: %s", status)}

		if msg.Status() == backend.TranscriptionCompleted {
			if completedEvent, ok := msg.AsCompleted(); ok {
				doc, err := lyrics.ParseLRC(completedEvent.Synced)
				if err == nil {
					m.editor.Document = doc
					m.dirty = true
					m.editor = m.editor.WithSelected(0)
					m.status = []string{"transcription complete: lyrics loaded"}
				} else {
					m.status = []string{"transcription complete but failed to parse: " + err.Error()}
				}
			}
			m.transcribeChan = nil
			return m, nil
		}

		if msg.Status() == backend.TranscriptionFailed {
			if failedEvent, ok := msg.AsFailed(); ok {
				m.status = []string{"transcription failed: " + failedEvent.Error}
			} else {
				m.status = []string{"transcription failed"}
			}
			m.media = m.media.WithTranscribeStatus("")
			m.transcribeChan = nil
			return m, nil
		}

		return m, listenTranscribeCmd(m.transcribeChan)

	case transcribeErrorMsg:
		m.status = []string{"transcription failed: " + msg.err.Error()}
		m.media = m.media.WithTranscribeStatus("")
		m.transcribeChan = nil
		return m, nil

	case transcribeFinishedMsg:
		m.media = m.media.WithTranscribeStatus("")
		m.transcribeChan = nil
		return m, nil
	}
	return m, nil
}

func (m Model) requestAndSolveChallengeCmd() tea.Cmd {
	return func() tea.Msg {
		challenge, err := m.client.RequestChallenge(context.Background())
		return challengeMsg{challenge: challenge, err: err}
	}
}

func (m Model) solvePoWCmd(challenge backend.ChallengeResponse) tea.Cmd {
	return func() tea.Msg {
		nonce, err := publish.SolvePoW(challenge.Prefix, challenge.Target)
		return powSolvedMsg{prefix: challenge.Prefix, nonce: nonce, err: err}
	}
}

func (m Model) submitPublishCmd(token string) tea.Cmd {
	return func() tea.Msg {
		synced := lyrics.FormatLRC(m.editor.Document)
		var plainLines []string
		for _, line := range m.editor.Document.Lines() {
			plainLines = append(plainLines, line.Text().String())
		}
		plain := strings.Join(plainLines, "\n")

		track := m.trackName
		if track == "" {
			track = "Never Gonna Give You Up"
		}
		artist := m.artistName
		if artist == "" {
			artist = "Rick Astley"
		}

		payload := backend.PublishPayload{
			TrackName:    track,
			ArtistName:   artist,
			Duration:     int(m.player.Snapshot().Duration.Milliseconds() / 1000),
			PlainLyrics:  plain,
			SyncedLyrics: synced,
		}
		err := m.client.Publish(context.Background(), token, payload)
		return publishResultMsg{err: err}
	}
}

func (m Model) handleMouse(msg tea.MouseMsg, isMotion bool) (Model, tea.Cmd) {
	mouse := msg.Mouse()
	x := mouse.X
	y := mouse.Y

	topHeight, _, leftW, _, availableHeight := calculateLayout(m.width, m.height, len(m.status))

	if y < topHeight {
		// Top row panels
		if x < leftW {
			m.focus = focusMedia
			// Progress bar drag/click detection.
			progressBarAbsRow := media.ProgressBarRow + 1     // +1 for top border
			transcribeAbsRow := media.TranscribeButtonRow + 1 // +1 for top border
			if y == progressBarAbsRow {
				if !isMotion {
					// Direct click: seek immediately and start drag state
					m.mediaDragging = true
					innerX := x - 1     // subtract left border
					innerW := leftW - 2 // subtract both borders
					m = m.seekToMS(m.media.SeekForX(innerX, innerW))
				} else if m.mediaDragging {
					// Motion while drag is active: continue seeking
					innerX := x - 1
					innerW := leftW - 2
					m = m.seekToMS(m.media.SeekForX(innerX, innerW))
				}
			} else if y == transcribeAbsRow && !isMotion {
				m.mediaDragging = false
				return m.startTranscription()
			} else if !isMotion {
				m.mediaDragging = false
			}
		} else {
			if m.focus != focusPublish {
				m.focus = focusEditor
			}
			if mouse.Button == tea.MouseWheelUp || mouse.Button == tea.MouseWheelDown {
				m.editor = m.editor.HandleMouseScroll(mouse.Button)
			}
			m.mediaDragging = false
		}
		m.waveform = m.waveform.WithHover(-1)
	} else if y >= topHeight && y < availableHeight {
		// Bottom row (Waveform)
		m.mediaDragging = false
		m.focus = focusWaveform
		col := x - 1
		width := m.width - 2
		if width > 0 && col >= 0 && col < width {
			m.waveform = m.waveform.WithWidth(width).WithHover(col)

			if mouse.Button == tea.MouseWheelUp || mouse.Button == tea.MouseWheelDown ||
				mouse.Button == tea.MouseWheelLeft || mouse.Button == tea.MouseWheelRight {
				m.waveform = m.waveform.HandleMouseLocal(col, mouse.Button, mouse.Mod)
			} else if mouse.Button == tea.MouseLeft && !isMotion {
				newPos := m.waveform.SeekForColumn(col, width)
				m = m.seekToMS(newPos)
				m.status = []string{fmt.Sprintf("seek: %dms", newPos)}
			}
		} else {
			m.waveform = m.waveform.WithHover(-1)
		}
	} else {
		m.mediaDragging = false
		m.waveform = m.waveform.WithHover(-1)
	}
	// Release always ends drag
	if _, isRelease := msg.(tea.MouseReleaseMsg); isRelease {
		m.mediaDragging = false
	}
	return m, nil
}

// seekToMS seeks the player to the given millisecond position and updates all panels.
func (m Model) seekToMS(newPosMS int64) Model {
	if m.player == nil {
		return m
	}
	pos, _ := playback.NewPosition(newPosMS)
	snap, _ := m.player.Seek(pos)
	state := media.TransportPaused
	if snap.State == playback.StatePlaying {
		state = media.TransportPlaying
	}
	m.media = m.media.WithTransport(state, snap.Position.Milliseconds(), snap.Duration.Milliseconds())
	m.waveform = m.waveform.WithPosition(snap.Position.Milliseconds())
	m.editor = m.editor.WithPlaybackPosition(snap.Position.Milliseconds())
	return m
}

type transcribeErrorMsg struct {
	err error
}

type transcribeFinishedMsg struct{}

func listenTranscribeCmd(ch <-chan backend.TranscribeResponse) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return transcribeFinishedMsg{}
		}
		return ev
	}
}

func (m Model) startTranscription() (Model, tea.Cmd) {
	if m.client == nil {
		m.status = []string{"cannot transcribe: client not initialized"}
		return m, nil
	}
	if m.videoID == "" {
		m.status = []string{"cannot transcribe: no video loaded"}
		return m, nil
	}
	if m.transcribeChan != nil {
		m.status = []string{"transcription already in progress"}
		return m, nil
	}

	m.transcribeChan = make(chan backend.TranscribeResponse, 10)
	m.media = m.media.WithTranscribeStatus("queued")
	m.status = []string{"transcription queued..."}

	ch := m.transcribeChan
	videoID := m.videoID
	client := m.client

	streamCmd := func() tea.Msg {
		ctx := context.Background()
		req := backend.TranscribeRequest{
			VideoID:          videoID,
			Force:            true,
			EnableRefinement: false,
			Mode:             "normal",
		}
		resp, err := client.Transcribe(ctx, req)
		if err != nil {
			return transcribeErrorMsg{err: err}
		}

		go func() {
			_ = client.TranscribeStream(ctx, videoID, func(ev backend.TranscribeResponse) {
				ch <- ev
			})
			close(ch)
		}()

		return resp
	}

	return m, streamCmd
}
