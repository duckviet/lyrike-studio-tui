package lrclib

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	upstreamBaseURL = "https://lrclib.net/api"
	userAgent       = "LyricsStudio/1.0.0"
	requestTimeout  = 30 * time.Second
)

// Proxy forwards challenge and publish requests to LRCLIB.NET.
type Proxy struct {
	client  *http.Client
	baseURL string
}

// NewProxy returns a Proxy that talks to the real LRCLIB.NET API.
func NewProxy() *Proxy {
	return &Proxy{
		client:  &http.Client{Timeout: requestTimeout},
		baseURL: upstreamBaseURL,
	}
}

// RequestChallenge proxies POST https://lrclib.net/api/request-challenge.
// The caller must close the returned io.ReadCloser.
func (p *Proxy) RequestChallenge(ctx context.Context) (io.ReadCloser, http.Header, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/request-challenge", http.NoBody)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("create request-challenge request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("request-challenge failed: %w", err)
	}
	return resp.Body, resp.Header, resp.StatusCode, nil
}

// Publish proxies POST https://lrclib.net/api/publish with the given publish token.
// The caller must close the returned io.ReadCloser.
func (p *Proxy) Publish(ctx context.Context, token string, body io.Reader) (io.ReadCloser, http.Header, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/publish", body)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("create publish request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-Publish-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("publish failed: %w", err)
	}
	return resp.Body, resp.Header, resp.StatusCode, nil
}
