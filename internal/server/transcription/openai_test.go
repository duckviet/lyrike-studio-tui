package transcription

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func TestMapWordsToSegments(t *testing.T) {
	segments := []apiSegment{
		{Text: "Hello world", Start: 0.0, End: 2.0},
		{Text: "Again here", Start: 2.0, End: 4.0},
	}
	words := []apiWord{
		{Word: " Hello ", Start: 0.0, End: 0.9},
		{Word: "world", Start: 0.9, End: 1.9},
		{Word: "Again", Start: 2.0, End: 2.5},
		{Word: "here", Start: 2.6, End: 3.5},
	}

	got := mapWordsToSegments(segments, words)

	if len(got) != len(segments) {
		t.Fatalf("expected %d segments, got %d", len(segments), len(got))
	}
	if len(got[0].Words) != 2 {
		t.Fatalf("expected segment 0 to have 2 words, got %d: %+v", len(got[0].Words), got[0].Words)
	}
	if got[0].Words[0].Word != "Hello" {
		t.Errorf("expected first word of segment 0 to be 'Hello', got %q", got[0].Words[0].Word)
	}
	if got[0].Words[1].Word != "world" {
		t.Errorf("expected second word of segment 0 to be 'world', got %q", got[0].Words[1].Word)
	}
	if len(got[1].Words) != 2 {
		t.Fatalf("expected segment 1 to have 2 words, got %d: %+v", len(got[1].Words), got[1].Words)
	}
	if got[1].Words[0].Word != "Again" {
		t.Errorf("expected first word of segment 1 to be 'Again', got %q", got[1].Words[0].Word)
	}
	if got[1].Words[1].Word != "here" {
		t.Errorf("expected second word of segment 1 to be 'here', got %q", got[1].Words[1].Word)
	}
}

// captureTransport records the outgoing request and returns a stub response.
type captureTransport struct {
	req  *http.Request
	body []byte
}

func (ct *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ct.req = req
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	ct.body = body

	respBody := `{
		"id": "test-completion",
		"object": "chat.completion",
		"created": 1,
		"model": "gpt-4o-mini",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "{\"syncedLyrics\":\"refined synced\",\"plainLyrics\":\"refined plain\"}"
				},
				"finish_reason": "stop"
			}
		]
	}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(respBody)),
	}, nil
}

func TestRefinePrompt(t *testing.T) {
	transport := &captureTransport{}
	client := openai.NewClient(
		option.WithAPIKey("test-key"),
		option.WithHTTPClient(&http.Client{Transport: transport}),
	)
	refiner := NewRefiner("test-key")
	refiner.client = &client

	synced := "[00:00.00] Hello"
	plain := "Hello"
	trackName := "Test Track"
	artistName := "Test Artist"
	duration := 123.45

	result, err := refiner.RefineLyrics(synced, plain, trackName, artistName, duration)
	if err != nil {
		t.Fatalf("RefineLyrics returned unexpected error: %v", err)
	}

	if !result.IsAIRefined {
		t.Errorf("expected IsAIRefined to be true")
	}
	if result.Model != "gpt-4o-mini" {
		t.Errorf("expected model gpt-4o-mini, got %q", result.Model)
	}
	if result.SyncedLyrics != "refined synced" {
		t.Errorf("expected synced lyrics 'refined synced', got %q", result.SyncedLyrics)
	}
	if result.PlainLyrics != "refined plain" {
		t.Errorf("expected plain lyrics 'refined plain', got %q", result.PlainLyrics)
	}

	if transport.req == nil {
		t.Fatal("no HTTP request was captured")
	}
	if transport.req.Method != http.MethodPost {
		t.Errorf("expected POST, got %s", transport.req.Method)
	}
	if !strings.HasSuffix(transport.req.URL.Path, "/chat/completions") {
		t.Errorf("expected path to end with /chat/completions, got %s", transport.req.URL.Path)
	}

	var payload map[string]any
	if err := json.Unmarshal(transport.body, &payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}

	if payload["model"] != "gpt-4o-mini" {
		t.Errorf("expected model gpt-4o-mini, got %v", payload["model"])
	}
	if payload["temperature"] != 0.2 {
		t.Errorf("expected temperature 0.2, got %v", payload["temperature"])
	}

	rf, ok := payload["response_format"].(map[string]any)
	if !ok || rf["type"] != "json_object" {
		t.Errorf("expected response_format.type json_object, got %v", payload["response_format"])
	}

	messages, ok := payload["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %v", payload["messages"])
	}

	sys, ok := messages[0].(map[string]any)
	if !ok || sys["role"] != "system" {
		t.Fatalf("expected first message role system, got %v", messages[0])
	}
	sysContent, _ := sys["content"].(string)
	if !strings.Contains(sysContent, "Return valid JSON with exactly two fields") {
		t.Errorf("system prompt missing expected JSON instruction: %q", sysContent)
	}
	if !strings.Contains(sysContent, "Preserve timestamps as much as possible") {
		t.Errorf("system prompt missing expected timestamp instruction: %q", sysContent)
	}

	user, ok := messages[1].(map[string]any)
	if !ok || user["role"] != "user" {
		t.Fatalf("expected second message role user, got %v", messages[1])
	}
	userContent, _ := user["content"].(string)
	for _, want := range []string{
		"Title: " + trackName,
		"Artist: " + artistName,
		"Duration: 123.45s",
		"SYNCED LYRICS (RAW):",
		synced,
		"PLAIN LYRICS (RAW):",
		plain,
	} {
		if !strings.Contains(userContent, want) {
			t.Errorf("user prompt missing %q; got:\n%s", want, userContent)
		}
	}
}
