package lyrics

import (
	"errors"
	"strings"
	"testing"
)

func TestParseRenderSample(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		"[ti:Example Song]",
		"[ar:Example Artist]",
		"[00:01.20]First line",
		"[00:03.00]Second line",
		"[00:04.50]<00:04.50>Hello <00:05.10>world",
	}, "\n")

	doc, err := ParseLRC(input)
	if err != nil {
		t.Fatalf("ParseLRC() error = %v", err)
	}

	rendered := FormatLRC(doc)
	for _, want := range []string{
		"[00:01.20]First line",
		"[00:03.00]Second line",
		"[00:04.50]<00:04.50>Hello <00:05.10>world",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("FormatLRC() = %q, want it to contain %q", rendered, want)
		}
	}
}

func TestParseRejectsInvalidTimestamp(t *testing.T) {
	t.Parallel()

	_, err := ParseLRC("[00:99.00]invalid timestamp")
	if err == nil {
		t.Fatal("ParseLRC() error = nil, want invalid timestamp error")
	}

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("ParseLRC() error = %T, want *ValidationError", err)
	}
	if validationErr.Code != CodeInvalidTimestamp {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, CodeInvalidTimestamp)
	}
	if !strings.Contains(err.Error(), "invalid timestamp") {
		t.Fatalf("ParseLRC() error = %q, want invalid timestamp", err.Error())
	}
}
