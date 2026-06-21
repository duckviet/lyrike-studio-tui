package lyrics

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestParseLRC_whenValidPlainAndEnhancedInput(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		"[00:01.20]First line",
		"[00:03.00]Second line",
		"[00:04.50]<00:04.50>Hello <00:05.10>world",
	}, "\n")

	doc, err := ParseLRC(input)
	if err != nil {
		t.Fatalf("ParseLRC() error = %v", err)
	}

	lines := doc.Lines()
	if len(lines) != 3 {
		t.Fatalf("len(lines) = %d, want 3", len(lines))
	}
	assertLine(t, lines[0], 1200, 3000, "First line")
	assertLine(t, lines[1], 3000, 4500, "Second line")
	assertLine(t, lines[2], 4500, 14500, "Hello world") // last line: start + 10s

	words := lines[2].Words()
	if len(words) != 2 {
		t.Fatalf("len(words) = %d, want 2", len(words))
	}
	if words[0].Timestamp().Milliseconds() != 4500 || words[0].Text().String() != "Hello" {
		t.Fatalf("first enhanced word = %d/%q, want 4500/Hello", words[0].Timestamp().Milliseconds(), words[0].Text().String())
	}
	if words[1].Timestamp().Milliseconds() != 5100 || words[1].Text().String() != "world" {
		t.Fatalf("second enhanced word = %d/%q, want 5100/world", words[1].Timestamp().Milliseconds(), words[1].Text().String())
	}
}

func TestParseLRC_whenInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		code ErrorCode
	}{
		{name: "bad timestamp seconds", in: "[00:61.00]bad", code: CodeInvalidTimestamp},
		{name: "malformed line", in: "plain text without timestamp", code: CodeMalformedLine},
		{name: "duplicate timestamp", in: "[00:01.00]first\n[00:01.00]second", code: CodeInvalidSegment},
		{name: "unsorted timestamp", in: "[00:02.00]second\n[00:01.00]first", code: CodeInvalidSegment},
		{name: "malformed enhanced marker", in: "[00:01.00]<bad>word", code: CodeMalformedEnhancedMarker},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseLRC(tt.in)
			if err == nil {
				t.Fatal("ParseLRC() error = nil, want validation error")
			}

			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("ParseLRC() error type = %T, want *ValidationError", err)
			}
			if validationErr.Code != tt.code {
				t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, tt.code)
			}
		})
	}
}

func TestFormatLRC_whenDocumentContainsPlainAndEnhancedLines(t *testing.T) {
	t.Parallel()

	startA := MustParseTimestampForTest(t, "00:01.20")
	endA := MustParseTimestampForTest(t, "00:02.00")
	first, err := NewLine(startA, endA, MustTextForTest(t, "First line"))
	if err != nil {
		t.Fatalf("NewLine(first) error = %v", err)
	}
	word, err := NewWordTiming(MustParseTimestampForTest(t, "00:02.50"), MustTextForTest(t, "word"))
	if err != nil {
		t.Fatalf("NewWordTiming() error = %v", err)
	}
	startB := MustParseTimestampForTest(t, "00:02.00")
	endB := MustParseTimestampForTest(t, "00:04.00")
	second, err := NewEnhancedLine(startB, endB, MustTextForTest(t, "word"), []WordTiming{word})
	if err != nil {
		t.Fatalf("NewEnhancedLine(second) error = %v", err)
	}
	doc, err := NewDocument([]Line{first, second})
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}

	got := FormatLRC(doc)
	want := "[00:01.20]First line\n[00:02.00]<00:02.50>word"
	if got != want {
		t.Fatalf("FormatLRC() = %q, want %q", got, want)
	}
}

func TestFormatLRCWithEnd_outputsStartEndFormat(t *testing.T) {
	t.Parallel()

	start := MustParseTimestampForTest(t, "00:01.20")
	end := MustParseTimestampForTest(t, "00:03.00")
	line, err := NewLine(start, end, MustTextForTest(t, "Hello"))
	if err != nil {
		t.Fatalf("NewLine() error = %v", err)
	}
	doc, err := NewDocument([]Line{line})
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}

	got := FormatLRCWithEnd(doc)
	want := "[00:01.20-00:03.00]Hello"
	if got != want {
		t.Fatalf("FormatLRCWithEnd() = %q, want %q", got, want)
	}
}

func TestNewLine_rejectsInvalidSegment(t *testing.T) {
	t.Parallel()

	start := MustParseTimestampForTest(t, "00:05.00")
	end := MustParseTimestampForTest(t, "00:03.00")
	_, err := NewLine(start, end, MustTextForTest(t, "bad"))
	if err == nil {
		t.Fatal("NewLine() error = nil, want error for start >= end")
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	if validationErr.Code != CodeInvalidSegment {
		t.Fatalf("Code = %q, want %q", validationErr.Code, CodeInvalidSegment)
	}
}

func TestNewDocument_rejectsOverlappingSegments(t *testing.T) {
	t.Parallel()

	line1, _ := NewLine(MustParseTimestampForTest(t, "00:01.00"), MustParseTimestampForTest(t, "00:05.00"), MustTextForTest(t, "A"))
	line2, _ := NewLine(MustParseTimestampForTest(t, "00:03.00"), MustParseTimestampForTest(t, "00:07.00"), MustTextForTest(t, "B"))

	_, err := NewDocument([]Line{line1, line2})
	if err == nil {
		t.Fatal("NewDocument() error = nil, want overlap error")
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	if validationErr.Code != CodeOverlappingSegment {
		t.Fatalf("Code = %q, want %q", validationErr.Code, CodeOverlappingSegment)
	}
}

func TestManualParseSurface(t *testing.T) {
	input := "[00:02.00]second\n[00:03.10]<00:03.10>Hello <00:03.60>world"

	doc, err := ParseLRC(input)
	if err != nil {
		t.Fatalf("ParseLRC(valid) error = %v", err)
	}
	lines := doc.Lines()
	if len(lines) != 2 {
		t.Fatalf("len(lines) = %d, want 2", len(lines))
	}

	fmt.Printf("valid typed lines: %T %s=%q; %s=%q\n", doc, lines[0].Start().String(), lines[0].Text().String(), lines[1].Start().String(), lines[1].Text().String())

	_, err = ParseLRC("[00:99.00]bad timestamp")
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("ParseLRC(invalid) error = %T, want *ValidationError", err)
	}
	fmt.Printf("invalid typed error: %T code=%s line=%d\n", validationErr, validationErr.Code, validationErr.Line)
}

func TestNewText_allowsEmptyString(t *testing.T) {
	t.Parallel()

	text, err := NewText("")
	if err != nil {
		t.Fatalf("NewText('') error = %v, want nil", err)
	}
	if text.String() != "" {
		t.Fatalf("text = %q, want empty string", text.String())
	}
}

func assertLine(t *testing.T, line Line, wantStartMS int64, wantEndMS int64, wantText string) {
	t.Helper()

	if line.Start().Milliseconds() != wantStartMS {
		t.Fatalf("line start = %d, want %d", line.Start().Milliseconds(), wantStartMS)
	}
	if line.End().Milliseconds() != wantEndMS {
		t.Fatalf("line end = %d, want %d", line.End().Milliseconds(), wantEndMS)
	}
	if line.Text().String() != wantText {
		t.Fatalf("line text = %q, want %q", line.Text().String(), wantText)
	}
}

func MustParseTimestampForTest(t *testing.T, input string) Timestamp {
	t.Helper()

	timestamp, err := ParseTimestamp(input)
	if err != nil {
		t.Fatalf("ParseTimestamp(%q) error = %v", input, err)
	}
	return timestamp
}

func MustTextForTest(t *testing.T, input string) Text {
	t.Helper()

	text, err := NewText(input)
	if err != nil {
		t.Fatalf("NewText(%q) error = %v", input, err)
	}
	return text
}
