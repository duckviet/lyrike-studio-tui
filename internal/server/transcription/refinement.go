package transcription

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

const systemPrompt = `You are a careful post-processor for ASR-generated song lyrics.

Processing priorities, in order:
1. Fix clear recognition errors.
2. Fix basic spelling and punctuation issues.
3. Split lines in a readable way.
4. Preserve timestamps as much as possible. Use format [mm:ss.xx] (e.g., [00:12.34]).
5. Keep plainLyrics and syncedLyrics consistent in content.

Rules:
- If unsure whether a word or line is correct, keep it unchanged.
- Do not use outside knowledge to replace the lyrics.
- Do not add new content.
- Do not add structure tags unless clearly supported by the input.
- Do not remove square brackets from timestamps.

Return valid JSON with exactly two fields: "syncedLyrics" and "plainLyrics".`

// Refiner improves ASR-generated lyrics with a small chat model.
type Refiner struct {
	client *openai.Client
	apiKey string
}

// NewRefiner builds a Refiner backed by the OpenAI chat completions API.
func NewRefiner(apiKey string) *Refiner {
	c := openai.NewClient(option.WithAPIKey(apiKey))
	return &Refiner{
		client: &c,
		apiKey: apiKey,
	}
}

// RefineLyrics asks the model to clean up synced and plain lyrics.
// It returns the original lyrics unchanged when the API key is empty.
func (r *Refiner) RefineLyrics(synced, plain, trackName, artistName string, duration float64) (RefineResult, error) {
	if r.apiKey == "" {
		return RefineResult{
			SyncedLyrics: synced,
			PlainLyrics:  plain,
			IsAIRefined:  false,
			Model:        "",
		}, nil
	}

	userPrompt := fmt.Sprintf(
		"Song information:\n"+
			"- Title: %s\n"+
			"- Artist: %s\n"+
			"- Duration: %gs\n\n"+
			"SYNCED LYRICS (RAW):\n%s\n\n"+
			"PLAIN LYRICS (RAW):\n%s",
		trackName, artistName, duration, synced, plain,
	)

	ctx := context.Background()
	rf := shared.NewResponseFormatJSONObjectParam()
	resp, err := r.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
		Temperature: openai.Float(0.2),
		MaxTokens:   openai.Int(4000),
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &rf,
		},
	})
	if err != nil {
		return RefineResult{
			SyncedLyrics: synced,
			PlainLyrics:  plain,
			IsAIRefined:  false,
			Model:        "",
			Error:        err.Error(),
		}, nil
	}

	if len(resp.Choices) == 0 {
		return RefineResult{
			SyncedLyrics: synced,
			PlainLyrics:  plain,
			IsAIRefined:  false,
			Model:        "",
		}, nil
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	var parsed struct {
		SyncedLyrics string `json:"syncedLyrics"`
		PlainLyrics  string `json:"plainLyrics"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return RefineResult{
			SyncedLyrics: synced,
			PlainLyrics:  plain,
			IsAIRefined:  false,
			Model:        "",
			Error:        err.Error(),
		}, nil
	}

	out := RefineResult{
		SyncedLyrics: synced,
		PlainLyrics:  plain,
		IsAIRefined:  true,
		Model:        "gpt-4o-mini",
	}
	if parsed.SyncedLyrics != "" {
		out.SyncedLyrics = parsed.SyncedLyrics
	}
	if parsed.PlainLyrics != "" {
		out.PlainLyrics = parsed.PlainLyrics
	}
	return out, nil
}
