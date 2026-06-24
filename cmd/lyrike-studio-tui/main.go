package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
	"github.com/duckviet/lyrike-studio-tui/internal/integrations/backend"
	"github.com/duckviet/lyrike-studio-tui/internal/playback"
	"github.com/duckviet/lyrike-studio-tui/internal/playback/beep"
	"github.com/duckviet/lyrike-studio-tui/internal/playback/mpv"
	"github.com/duckviet/lyrike-studio-tui/internal/server"
	"github.com/duckviet/lyrike-studio-tui/internal/server/cache"
	"github.com/duckviet/lyrike-studio-tui/internal/server/lrclib"
	"github.com/duckviet/lyrike-studio-tui/internal/server/transcription"
	"github.com/duckviet/lyrike-studio-tui/internal/storage"
	"github.com/duckviet/lyrike-studio-tui/internal/tui"
	"github.com/duckviet/lyrike-studio-tui/internal/version"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		if err := runServe(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "serve: %v\n", err)
			os.Exit(1)
		}
		return
	}

	versionRequested := flag.Bool("version", false, "print version and exit")
	demoRequested := flag.Bool("demo", false, "launch the TUI demo")
	backendFixtureRequested := flag.Bool("backend-fixture", false, "use deterministic backend fixture data in demo mode")

	defaultBackend := os.Getenv("BACKEND_URL")

	if defaultBackend == "" {
		defaultBackend = "http://127.0.0.1:8000"
	}
	backendURL := flag.String("backend", defaultBackend, "URL of the FastAPI backend")
	mpvSocket := flag.String("mpv-socket", "/tmp/lyrike-mpv.sock", "Path to the mpv IPC socket")
	videoID := flag.String("video-id", "", "YouTube video ID to sync")
	projectID := flag.String("project", "", "project ID for draft save/load")
	sourceURL := flag.String("url", "", "Source media URL to sync")
	audioPath := flag.String("audio", "", "Path to the local audio file to play natively via beep")
	importPath := flag.String("import", "", "Path to a lyric file (.lrc or .txt) to import at startup")

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

	if err := runReal(*backendURL, *mpvSocket, *videoID, *projectID, *sourceURL, *audioPath, *importPath); err != nil {
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
	if m, ok := model.(tui.Model); ok {
		model = m.WithTheme(tui.DefaultTheme())
	}
	_, err = tea.NewProgram(model).Run()
	return err
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	port := fs.Int("port", 0, "backend port")
	cacheDir := fs.String("cache-dir", "", "override cache directory (drafts live under <cache-dir>/drafts)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := server.LoadConfig()
	if err != nil {
		return err
	}
	if *port != 0 {
		cfg.Port = *port
	}
	if *cacheDir != "" {
		cfg.CacheDir = *cacheDir
		cfg.DraftDir = filepath.Join(*cacheDir, "drafts")
	}
	if err := server.WriteCookiesFromEnv(server.DefaultYouTubeCookiesPath); err != nil {
		return err
	}
	store := cache.NewStore(cfg.CacheDir)
	provider := transcription.NewTranscriber(cfg.OpenAIAPIKey, cfg.OpenAITranscriptionModel)
	refiner := transcription.NewRefiner(cfg.OpenAIAPIKey)
	manager := transcription.NewManager(store, provider, refiner)
	srv := server.NewServer(cfg, store, manager, lrclib.NewProxy())
	return http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), srv.Handler())
}

func newPlayer(backendURL, mpvSocket, audioPath, videoID string) (playback.Player, *mpv.Player, string) {
	var player playback.Player
	var mpvPlayer *mpv.Player
	var statusMsg string

	// 1. Try native beep player first if audio file or backend audio URL is available
	targetAudio := audioPath
	if targetAudio == "" && videoID != "" {
		targetAudio = strings.TrimRight(backendURL, "/") + "/local-api/audio/" + videoID
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
		} else {
			if statusMsg != "" {
				statusMsg += " | "
			}
			statusMsg += fmt.Sprintf("mpv connection failed: %v", err)
			_ = mpvPlayer.Close()
			mpvPlayer = nil
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

	return player, mpvPlayer, statusMsg
}

func runReal(backendURL, mpvSocket, videoID, projectIDValue, sourceURL, audioPath, importPath string) error {
	client := backend.NewClient(backendURL)

	player, mpvPlayer, statusMsg := newPlayer(backendURL, mpvSocket, audioPath, videoID)

	playerFactory := func(vid string) (playback.Player, string) {
		p, _, s := newPlayer(backendURL, mpvSocket, "", vid)
		return p, s
	}

	// Try loading draft or imported lyrics
	doc := defaultDocument()
	store := storage.NewRemoteStore(client)
	var projectID draft.ProjectID
	if projectIDValue != "" {
		id, err := draft.NewProjectID(projectIDValue)
		if err != nil {
			return err
		}
		projectID = id
	} else if videoID != "" {
		if id, err := draft.NewProjectID(videoID); err == nil {
			projectID = id
		}
	}

	var loadedSnapshot draft.Snapshot
	var loadedSnapshotErr error
	if importPath != "" {
		content, err := os.ReadFile(importPath)
		if err == nil {
			if parsedDoc, err := lyrics.ParseLyrics(string(content)); err == nil {
				doc = parsedDoc
				statusMsg += " | imported lyrics from " + importPath
			} else {
				statusMsg += " | failed to parse import file: " + err.Error()
			}
		} else {
			statusMsg += " | failed to read import file: " + err.Error()
		}
	} else if projectID != "" {
		loadedSnapshot, loadedSnapshotErr = store.Load(projectID)
		if loadedSnapshotErr == nil {
			doc = loadedSnapshot.Document
			statusMsg += " | loaded project " + projectID.String()
		} else {
			statusMsg += " | failed to load project: " + loadedSnapshotErr.Error()
		}
	}

	model := tui.NewModelWithDraftStore(doc, client, player, store, projectID, videoID, sourceURL).
		WithPlayerFactory(playerFactory).
		WithTheme(tui.DefaultTheme())
	if projectID != "" && loadedSnapshotErr == nil {
		model = model.WithProjectMetadata(loadedSnapshot.Metadata.TrackName, loadedSnapshot.Metadata.ArtistName, loadedSnapshot.Metadata.AlbumName)
	}
	if projectID == "" && importPath == "" {
		model = model.OpenProjectPickerOnStartup()
	}
	if statusMsg != "" {
		model = model.WithStatus([]string{statusMsg})
	}

	finalModel, err := tea.NewProgram(model).Run()

	if m, ok := finalModel.(tui.Model); ok {
		_ = m.Close()
	} else {
		if closer, ok := player.(io.Closer); ok {
			_ = closer.Close()
		}
		if mpvPlayer != nil {
			_ = mpvPlayer.Close()
		}
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
