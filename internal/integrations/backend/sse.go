package backend

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// TranscribeStream connects to GET /local-api/transcribe/stream/{video_id}
// and calls handler for each decoded event. It blocks until the stream ends,
// ctx is cancelled, or an error occurs.
func (c *Client) TranscribeStream(ctx context.Context, videoID string, handler func(TranscribeResponse)) error {
	url := c.baseURL + "/local-api/transcribe/stream/" + videoID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("transcribe stream: create request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("transcribe stream: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return fmt.Errorf("transcribe stream: %w", err)
	}

	return readSSE(ctx, resp.Body, func(data string) error {
		event, err := DecodeTranscribeResponse([]byte(data))
		if err != nil {
			return fmt.Errorf("transcribe stream: decode event: %w", err)
		}
		handler(event)
		return nil
	})
}

// readSSE parses an SSE stream and calls handler for each data payload.
// A goroutine closes the reader when ctx is cancelled to unblock the
// blocking ReadString call.
func readSSE(ctx context.Context, reader io.ReadCloser, handler func(string) error) error {
	go func() {
		<-ctx.Done()
		reader.Close()
	}()

	br := bufio.NewReaderSize(reader, 64*1024)
	var data strings.Builder

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err == io.EOF {
				if data.Len() > 0 {
					return handler(data.String())
				}
				return nil
			}
			return fmt.Errorf("sse read: %w", err)
		}

		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			if data.Len() > 0 {
				if err := handler(data.String()); err != nil {
					return err
				}
				data.Reset()
			}
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}

		field, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimPrefix(value, " ")

		if field == "data" {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(value)
		}
	}
}
