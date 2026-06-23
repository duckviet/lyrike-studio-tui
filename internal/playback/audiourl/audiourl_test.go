package audiourl

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDownloadToTemp_OK(t *testing.T) {
	wantBody := "fake audio body"
	wantType := "audio/mpeg"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", wantType)
		_, _ = w.Write([]byte(wantBody))
	}))
	defer srv.Close()

	path, cleanup, err := DownloadToTemp(context.Background(), srv.URL+"/audio.mp3")
	if err != nil {
		t.Fatalf("DownloadToTemp error: %v", err)
	}
	defer cleanup()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("temp file does not exist: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read temp file: %v", err)
	}
	if string(data) != wantBody {
		t.Errorf("body = %q, want %q", string(data), wantBody)
	}
	if !strings.HasSuffix(path, ".mp3") {
		t.Errorf("expected .mp3 extension from URL, got %q", path)
	}
}

func TestDownloadToTemp_ContentTypeExtension(t *testing.T) {
	wantBody := "fake audio body"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mp4")
		_, _ = w.Write([]byte(wantBody))
	}))
	defer srv.Close()

	path, cleanup, err := DownloadToTemp(context.Background(), srv.URL+"/audio")
	if err != nil {
		t.Fatalf("DownloadToTemp error: %v", err)
	}
	defer cleanup()

	if !strings.HasSuffix(path, ".m4a") {
		t.Errorf("expected .m4a extension from Content-Type, got %q", path)
	}
}

func TestDownloadToTemp_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "not found")
	}))
	defer srv.Close()

	_, cleanup, err := DownloadToTemp(context.Background(), srv.URL+"/missing")
	if cleanup != nil {
		cleanup()
	}
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention status 404, got: %v", err)
	}
}

func TestDownloadToTemp_UnsupportedScheme(t *testing.T) {
	_, cleanup, err := DownloadToTemp(context.Background(), "file:///etc/passwd")
	if cleanup != nil {
		cleanup()
	}
	if err == nil {
		t.Fatal("expected error for unsupported scheme")
	}
	if !strings.Contains(err.Error(), "unsupported url scheme") {
		t.Errorf("error should mention unsupported scheme, got: %v", err)
	}
}
