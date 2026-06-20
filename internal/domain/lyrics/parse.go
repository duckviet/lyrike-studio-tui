package lyrics

import (
	"strings"
)

func ParseLRC(input string) (Document, error) {
	normalized := strings.ReplaceAll(input, "\r\n", "\n")
	var lines []Line
	for index, raw := range strings.Split(normalized, "\n") {
		lineNumber := index + 1
		row := strings.TrimSpace(raw)
		if row == "" {
			continue
		}
		line, err := parseLRCLine(row, lineNumber)
		if err != nil {
			return Document{}, err
		}
		if err := validateLineOrder(lines, line, lineNumber); err != nil {
			return Document{}, err
		}
		lines = append(lines, line)
	}
	doc, err := NewDocument(lines)
	if err != nil {
		if validationErr, ok := err.(*ValidationError); ok {
			validationErr.Line = 1
		}
		return Document{}, err
	}
	return doc, nil
}

func parseLRCLine(row string, lineNumber int) (Line, error) {
	if !strings.HasPrefix(row, "[") {
		return Line{}, newValidationError(CodeMalformedLine, lineNumber, "line", row, "lyric line must start with [mm:ss.xx]")
	}
	closing := strings.Index(row, "]")
	if closing <= 1 {
		return Line{}, newValidationError(CodeMalformedLine, lineNumber, "line", row, "lyric line must include a closing timestamp bracket")
	}
	timestamp, err := parseTimestamp(row[1:closing], lineNumber, "timestamp")
	if err != nil {
		return Line{}, err
	}

	text, words, err := parseEnhancedText(row[closing+1:], lineNumber)
	if err != nil {
		return Line{}, err
	}
	if len(words) > 0 {
		line, err := NewEnhancedLine(timestamp, text, words)
		return line, withLine(err, lineNumber)
	}
	line, err := NewLine(timestamp, text)
	return line, withLine(err, lineNumber)
}

func parseEnhancedText(input string, lineNumber int) (Text, []WordTiming, error) {
	if !strings.Contains(input, "<") {
		text, err := NewText(input)
		return text, nil, withLine(err, lineNumber)
	}

	var plain strings.Builder
	var words []WordTiming
	remaining := input
	for remaining != "" {
		markerStart := strings.Index(remaining, "<")
		if markerStart < 0 {
			appendPlain(&plain, remaining)
			break
		}
		appendPlain(&plain, remaining[:markerStart])
		markerEnd := strings.Index(remaining[markerStart:], ">")
		if markerEnd < 0 {
			return Text{}, nil, newValidationError(CodeMalformedEnhancedMarker, lineNumber, "enhanced_marker", remaining[markerStart:], "enhanced marker must use <mm:ss.xx>")
		}
		markerEnd += markerStart
		timestamp, err := parseTimestamp(remaining[markerStart+1:markerEnd], lineNumber, "enhanced_marker")
		if err != nil {
			return Text{}, nil, newValidationError(CodeMalformedEnhancedMarker, lineNumber, "enhanced_marker", remaining[markerStart:markerEnd+1], "enhanced marker must use <mm:ss.xx>")
		}
		nextMarker := strings.Index(remaining[markerEnd+1:], "<")
		wordText := remaining[markerEnd+1:]
		if nextMarker >= 0 {
			wordText = remaining[markerEnd+1 : markerEnd+1+nextMarker]
		}
		text, err := NewText(wordText)
		if err != nil {
			return Text{}, nil, withLine(err, lineNumber)
		}
		word, err := NewWordTiming(timestamp, text)
		if err != nil {
			return Text{}, nil, withLine(err, lineNumber)
		}
		words = append(words, word)
		appendPlain(&plain, text.String())
		if nextMarker < 0 {
			break
		}
		remaining = remaining[markerEnd+1+nextMarker:]
	}

	text, err := NewText(plain.String())
	return text, words, withLine(err, lineNumber)
}

func appendPlain(builder *strings.Builder, input string) {
	value := strings.TrimSpace(input)
	if value == "" {
		return
	}
	if builder.Len() > 0 {
		builder.WriteByte(' ')
	}
	builder.WriteString(value)
}

func withLine(err error, lineNumber int) error {
	if validationErr, ok := err.(*ValidationError); ok {
		validationErr.Line = lineNumber
		return validationErr
	}
	return err
}

func validateLineOrder(lines []Line, next Line, lineNumber int) error {
	if len(lines) == 0 {
		return nil
	}
	previous := lines[len(lines)-1].Timestamp().Milliseconds()
	current := next.Timestamp().Milliseconds()
	if current == previous {
		return newValidationError(CodeDuplicateTimestamp, lineNumber, "timestamp", next.Timestamp().String(), "lyric timestamps must be unique")
	}
	if current < previous {
		return newValidationError(CodeUnsortedTimestamp, lineNumber, "timestamp", next.Timestamp().String(), "lyric timestamps must be strictly increasing")
	}
	return nil
}
