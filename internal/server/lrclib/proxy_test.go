package lrclib

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestChallengeProxy(t *testing.T) {
	wantBody := `{"prefix":"abc","target":"0000"}`
	wantStatus := http.StatusOK

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/request-challenge" {
			t.Errorf("expected path /api/request-challenge, got %s", r.URL.Path)
		}
		if got := r.Header.Get("User-Agent"); got != userAgent {
			t.Errorf("expected User-Agent %q, got %q", userAgent, got)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(wantStatus)
		if _, err := w.Write([]byte(wantBody)); err != nil {
			t.Fatalf("upstream write: %v", err)
		}
	}))
	defer srv.Close()

	p := newTestProxy(srv.URL+"/api", &http.Client{Timeout: 5 * time.Second})
	body, headers, status, err := p.RequestChallenge(context.Background())
	if err != nil {
		t.Fatalf("RequestChallenge: %v", err)
	}
	defer body.Close()

	if status != wantStatus {
		t.Errorf("expected status %d, got %d", wantStatus, status)
	}
	if got := headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", got)
	}

	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(got) != wantBody {
		t.Errorf("expected body %q, got %q", wantBody, string(got))
	}
}

func TestPublishProxy(t *testing.T) {
	wantToken := "prefix:nonce"
	wantBody := `{"trackName":"Test","artistName":"Artist"}`
	wantStatus := http.StatusCreated

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/publish" {
			t.Errorf("expected path /api/publish, got %s", r.URL.Path)
		}
		if got := r.Header.Get("User-Agent"); got != userAgent {
			t.Errorf("expected User-Agent %q, got %q", userAgent, got)
		}
		if got := r.Header.Get("X-Publish-Token"); got != wantToken {
			t.Errorf("expected X-Publish-Token %q, got %q", wantToken, got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", got)
		}

		gotBody, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("upstream read body: %v", err)
		}
		if string(gotBody) != wantBody {
			t.Errorf("expected body %q, got %q", wantBody, string(gotBody))
		}

		w.WriteHeader(wantStatus)
		if _, err := w.Write([]byte(`{"ok":true}`)); err != nil {
			t.Fatalf("upstream write: %v", err)
		}
	}))
	defer srv.Close()

	p := newTestProxy(srv.URL+"/api", &http.Client{Timeout: 5 * time.Second})
	body, _, status, err := p.Publish(context.Background(), wantToken, strings.NewReader(wantBody))
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	defer body.Close()

	if status != wantStatus {
		t.Errorf("expected status %d, got %d", wantStatus, status)
	}
}

func TestProxyTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := newTestProxy(srv.URL+"/api", &http.Client{Timeout: 50 * time.Millisecond})
	_, _, _, err := p.RequestChallenge(context.Background())
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

// newTestProxy builds a Proxy pointed at the given base URL.
func newTestProxy(baseURL string, client *http.Client) *Proxy {
	return &Proxy{
		client:  client,
		baseURL: baseURL,
	}
}
