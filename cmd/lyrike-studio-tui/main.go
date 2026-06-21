package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
	"github.com/duckviet/lyrike-studio-tui/internal/integrations/backend"
	"github.com/duckviet/lyrike-studio-tui/internal/playback"
	"github.com/duckviet/lyrike-studio-tui/internal/playback/beep"
	"github.com/duckviet/lyrike-studio-tui/internal/playback/mpv"
	"github.com/duckviet/lyrike-studio-tui/internal/storage"
	"github.com/duckviet/lyrike-studio-tui/internal/tui"
	"github.com/duckviet/lyrike-studio-tui/internal/version"
)

func main() {
	versionRequested := flag.Bool("version", false, "print version and exit")
	demoRequested := flag.Bool("demo", false, "launch the TUI demo")
	backendFixtureRequested := flag.Bool("backend-fixture", false, "use deterministic backend fixture data in demo mode")

	backendURL := flag.String("backend", "http://127.0.0.1:8000", "URL of the FastAPI backend")
	mpvSocket := flag.String("mpv-socket", "/tmp/lyrike-mpv.sock", "Path to the mpv IPC socket")
	videoID := flag.String("video-id", "", "YouTube video ID to sync")
	sourceURL := flag.String("url", "", "Source media URL to sync")
	audioPath := flag.String("audio", "", "Path to the local audio file to play natively via beep")

	flag.Parse()

	if *versionRequested {
		fmt.Println(version.Label())
		return
	}

	if *demoRequested {
		if err := runDemo(*backendFixtureRequested); err != nil {
			fmt.Fprintf(os.Stderr, "lyrike-studio-tui: demo failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := runReal(*backendURL, *mpvSocket, *videoID, *sourceURL, *audioPath); err != nil {
		fmt.Fprintf(os.Stderr, "lyrike-studio-tui: error: %v\n", err)
		os.Exit(1)
	}
}

func runDemo(backendFixture bool) error {
	var (
		model tea.Model
		err   error
	)
	if backendFixture {
		server := startFixtureServer()
		defer server.Close()

		model, err = tui.DemoFixtureModel(server.URL)
	} else {
		model, err = tui.DemoModel()
	}
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(model).Run()
	return err
}

func runReal(backendURL, mpvSocket, videoID, sourceURL, audioPath string) error {
	client := backend.NewClient(backendURL)

	var player playback.Player
	var mpvPlayer *mpv.Player

	var statusMsg string

	// 1. Try native beep player first if audio file or cache is found
	targetAudio := audioPath
	if targetAudio == "" && videoID != "" {
		cachePath := fmt.Sprintf("/home/duckviet/lrclib-upload/backend/.cache/audio/%s/original.mp4", videoID)
		if _, err := os.Stat(cachePath); err == nil {
			targetAudio = cachePath
		}
	}

	if targetAudio != "" {
		beepPlayer, err := beep.NewPlayer(targetAudio)
		if err == nil {
			player = beepPlayer
			statusMsg = "playing audio natively: " + targetAudio
		} else {
			statusMsg = fmt.Sprintf("failed to init native beep player: %v", err)
		}
	}

	// 2. Fall back to mpv socket connection
	if player == nil && mpvSocket != "" {
		mpvPlayer = mpv.NewPlayer(mpvSocket)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err := mpvPlayer.Start(ctx); err == nil {
			player = mpvPlayer
			statusMsg = "connected to mpv via " + mpvSocket
		}
	}

	// 3. Fall back to FakePlayer
	if player == nil {
		dur, _ := playback.NewDuration(300 * 1000) // Default 5 minutes
		player, _ = playback.NewFakePlayer(dur)
		if statusMsg == "" {
			statusMsg = "no audio source or mpv socket connection failed: falling back to fake player"
		} else {
			statusMsg += " | falling back to fake player"
		}
	}

	// Try loading saved draft if videoID is set
	doc := defaultDocument()
	store := storage.NewDefaultStore()
	idStr := videoID
	if idStr == "" {
		idStr = "default"
	}
	if id, err := draft.NewDraftID(idStr); err == nil {
		if snapshot, err := store.Load(id); err == nil {
			doc = snapshot.Document
			statusMsg += " | loaded saved draft for " + idStr
		}
	}

	model := tui.NewModel(doc, client, player, videoID, sourceURL)
	if statusMsg != "" {
		model = model.WithStatus([]string{statusMsg})
	}

	_, err := tea.NewProgram(model).Run()

	// Clean up player and mpv connection
	if closer, ok := player.(io.Closer); ok {
		_ = closer.Close()
	}
	if mpvPlayer != nil {
		_ = mpvPlayer.Close()
	}
	return err
}

func defaultDocument() lyrics.Document {
	ts, _ := lyrics.NewTimestamp(0)
	te, _ := lyrics.NewTimestamp(10_000)
	txt, _ := lyrics.NewText("Type lyrics here...")
	line, _ := lyrics.NewLine(ts, te, txt)
	doc, _ := lyrics.NewDocument([]lyrics.Line{line})
	return doc
}

func startFixtureServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/local-api/fetch", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(backend.FetchFixture())
	})
	mux.HandleFunc("/local-api/peaks/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(backend.PeaksFixture())
	})
	mux.HandleFunc("/api/request-challenge", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(backend.ChallengeFixture())
	})
	mux.HandleFunc("/api/publish", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("published"))
	})
	return httptest.NewServer(mux)
}
