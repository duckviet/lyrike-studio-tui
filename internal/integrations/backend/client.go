package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const defaultTimeout = 30 * time.Second

// Client is a typed HTTP client for the lrclib-upload FastAPI backend.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a backend client rooted at baseURL.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: stringsTrimTrailingSlash(baseURL),
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// NewClientWithHTTPClient allows injecting a custom http.Client for tests.
func NewClientWithHTTPClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    stringsTrimTrailingSlash(baseURL),
		httpClient: httpClient,
	}
}

// Fetch calls POST /local-api/fetch.
func (c *Client) Fetch(ctx context.Context, req FetchRequest) (FetchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return FetchResponse{}, &DecodeError{Kind: DecodeKindFetch, Err: err}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/local-api/fetch", bytes.NewReader(body))
	if err != nil {
		return FetchResponse{}, fmt.Errorf("fetch: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.do(httpReq)
	if err != nil {
		return FetchResponse{}, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return FetchResponse{}, err
	}

	return DecodeFetchResponse(mustReadBody(resp))
}

// Transcribe calls POST /local-api/transcribe.
func (c *Client) Transcribe(ctx context.Context, req TranscribeRequest) (TranscribeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return TranscribeResponse{}, &DecodeError{Kind: DecodeKindTranscribe, Err: err}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/local-api/transcribe", bytes.NewReader(body))
	if err != nil {
		return TranscribeResponse{}, fmt.Errorf("transcribe: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.do(httpReq)
	if err != nil {
		return TranscribeResponse{}, fmt.Errorf("transcribe: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return TranscribeResponse{}, expectStatus(resp, http.StatusOK)
	}

	return DecodeTranscribeResponse(mustReadBody(resp))
}

// Peaks calls GET /local-api/peaks/{video_id}.
func (c *Client) Peaks(ctx context.Context, videoID string, source Source, samples int) (PeaksResponse, error) {
	u, err := url.Parse(c.baseURL + "/local-api/peaks/" + videoID)
	if err != nil {
		return PeaksResponse{}, fmt.Errorf("peaks: parse url: %w", err)
	}
	q := u.Query()
	q.Set("source", string(source))
	q.Set("samples", strconv.Itoa(samples))
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return PeaksResponse{}, fmt.Errorf("peaks: create request: %w", err)
	}

	resp, err := c.do(httpReq)
	if err != nil {
		return PeaksResponse{}, fmt.Errorf("peaks: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return PeaksResponse{}, err
	}

	return DecodePeaksResponse(mustReadBody(resp))
}

// RequestChallenge calls POST /api/request-challenge.
func (c *Client) RequestChallenge(ctx context.Context) (ChallengeResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/request-challenge", nil)
	if err != nil {
		return ChallengeResponse{}, fmt.Errorf("request-challenge: create request: %w", err)
	}

	resp, err := c.do(httpReq)
	if err != nil {
		return ChallengeResponse{}, fmt.Errorf("request-challenge: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return ChallengeResponse{}, err
	}

	body := mustReadBody(resp)
	var challenge ChallengeResponse
	if err := json.Unmarshal(body, &challenge); err != nil {
		return ChallengeResponse{}, fmt.Errorf("request-challenge: decode: %w", err)
	}
	return challenge, nil
}

// Publish calls POST /api/publish with the solved PoW token.
func (c *Client) Publish(ctx context.Context, token string, payload PublishPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("publish: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/publish", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("publish: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Publish-Token", token)

	resp, err := c.do(httpReq)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return expectStatus(resp, http.StatusOK)
	}
	return nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}

func expectStatus(resp *http.Response, want int) error {
	if resp.StatusCode == want {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return &APIError{StatusCode: resp.StatusCode, Body: string(body)}
}

func mustReadBody(resp *http.Response) []byte {
	body, _ := io.ReadAll(resp.Body)
	return body
}

func stringsTrimTrailingSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
