package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/local-api/fetch", nil)
	req.RemoteAddr = "10.0.0.1:1234"

	if got := GetRealIP(req); got != "10.0.0.1" {
		t.Errorf("RemoteAddr fallback: want %q, got %q", "10.0.0.1", got)
	}

	req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.2")
	if got := GetRealIP(req); got != "192.168.1.1" {
		t.Errorf("X-Forwarded-For first entry: want %q, got %q", "192.168.1.1", got)
	}

	req.Header.Set("CF-Connecting-IP", "1.2.3.4")
	if got := GetRealIP(req); got != "1.2.3.4" {
		t.Errorf("CF-Connecting-IP precedence: want %q, got %q", "1.2.3.4", got)
	}
}

func TestRateLimit429(t *testing.T) {
	lim := NewRateLimiter()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := lim.Handler(next)

	const ip = "127.0.0.1:1234"
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/local-api/transcribe", nil)
		req.RemoteAddr = ip
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rr.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/local-api/transcribe", nil)
	req.RemoteAddr = ip
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("6th request: expected 429, got %d", rr.Code)
	}
	if got := rr.Header().Get("Retry-After"); got != "60" {
		t.Errorf("Retry-After header: want %q, got %q", "60", got)
	}

	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON error body: %v", err)
	}
	if body["error"] != "rate_limit_exceeded" {
		t.Errorf("error field: want %q, got %q", "rate_limit_exceeded", body["error"])
	}
	if body["retry_after"] != "60" {
		t.Errorf("retry_after field: want %q, got %q", "60", body["retry_after"])
	}
	if body["detail"] == "" {
		t.Errorf("detail field should not be empty")
	}
}

func TestCORSPreflight(t *testing.T) {
	c := NewCORS("")
	handler := c.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be invoked for OPTIONS preflight")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/local-api/fetch", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("preflight status: want %d, got %d", http.StatusNoContent, rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("Allow-Origin: want %q, got %q", "http://localhost:5173", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Allow-Credentials: want %q, got %q", "true", got)
	}
	if got := rr.Header().Get("Access-Control-Max-Age"); got != "600" {
		t.Errorf("Max-Age: want %q, got %q", "600", got)
	}

	methods := rr.Header().Get("Access-Control-Allow-Methods")
	for _, m := range []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"} {
		if !strings.Contains(methods, m) {
			t.Errorf("Allow-Methods missing %s: got %q", m, methods)
		}
	}

	headers := rr.Header().Get("Access-Control-Allow-Headers")
	for _, h := range []string{"Origin", "X-Requested-With", "Content-Type", "Accept", "Authorization", "Cache-Control", "X-Publish-Token"} {
		if !strings.Contains(headers, h) {
			t.Errorf("Allow-Headers missing %s: got %q", h, headers)
		}
	}
}
