package transcription

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Transcriber transcribes audio files using the OpenAI Whisper API.
type Transcriber struct {
	client *openai.Client
	model  string
}

// NewTranscriber builds a Transcriber for the given model (e.g. whisper-1).
func NewTranscriber(apiKey, model string) *Transcriber {
	c := openai.NewClient(option.WithAPIKey(apiKey))
	return &Transcriber{
		client: &c,
		model:  model,
	}
}

// apiSegment and apiWord mirror the verbose_json fields we need from the
// OpenAI audio transcription response.
type apiSegment struct {
	Text  string  `json:"text"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type apiWord struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// Transcribe reads the audio file at audioPath and returns a TranscriptionResult.
func (t *Transcriber) Transcribe(audioPath string) (TranscriptionResult, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return TranscriptionResult{}, fmt.Errorf("open audio file: %w", err)
	}
	defer f.Close()

	ctx := context.Background()
	resp, err := t.client.Audio.Transcriptions.New(ctx, openai.AudioTranscriptionNewParams{
		Model:                  openai.AudioModel(t.model),
		File:                   f,
		ResponseFormat:         openai.AudioResponseFormatVerboseJSON,
		TimestampGranularities: []string{"word", "segment"},
	})
	if err != nil {
		return TranscriptionResult{}, fmt.Errorf("openai transcription failed: %w", err)
	}

	rawJSON := resp.RawJSON()

	var raw map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		return TranscriptionResult{}, fmt.Errorf("decode raw verbose_json: %w", err)
	}

	var verbose struct {
		Language *string      `json:"language"`
		Text     string       `json:"text"`
		Segments []apiSegment `json:"segments"`
		Words    []apiWord    `json:"words"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &verbose); err != nil {
		return TranscriptionResult{}, fmt.Errorf("decode verbose_json: %w", err)
	}

	language := ""
	if verbose.Language != nil {
		language = *verbose.Language
	}

	return TranscriptionResult{
		Provider:  fmt.Sprintf("openai-%s", t.model),
		Language:  language,
		Segments:  mapWordsToSegments(verbose.Segments, verbose.Words),
		PlainText: verbose.Text,
		Raw:       raw,
	}, nil
}

// mapWordsToSegments distributes global words into segments by timeframe.
// A word belongs to a segment when word.End > seg.Start && word.Start < seg.End.
func mapWordsToSegments(segments []apiSegment, words []apiWord) []TranscribedSegment {
	out := make([]TranscribedSegment, 0, len(segments))
	wordIdx := 0
	for _, seg := range segments {
		segWords := make([]TranscribedWord, 0)
		for wordIdx < len(words) {
			w := words[wordIdx]
			if w.End <= seg.Start {
				wordIdx++
				continue
			}
			if w.Start >= seg.End {
				break
			}
			segWords = append(segWords, TranscribedWord{
				Word:  strings.TrimSpace(w.Word),
				Start: w.Start,
				End:   w.End,
			})
			wordIdx++
		}
		out = append(out, TranscribedSegment{
			Text:  strings.TrimSpace(seg.Text),
			Start: seg.Start,
			End:   seg.End,
			Words: segWords,
		})
	}
	return out
}
