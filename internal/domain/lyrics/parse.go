package lyrics

import (
	"strings"
)

const defaultLastLineDurationMS = 10_000

func ParseLRC(input string) (Document, error) {
	normalized := strings.ReplaceAll(input, "\r\n", "\n")
	var rawLines []rawParsedLine
	var metadata []Metadata
	for index, raw := range strings.Split(normalized, "\n") {
		lineNumber := index + 1
		row := strings.TrimSpace(raw)
		if row == "" {
			continue
		}
		if isMetadataLine(row) {
			item, err := parseMetadataLine(row, lineNumber)
			if err != nil {
				return Document{}, err
			}
			metadata = append(metadata, item)
			continue
		}
		parsed, err := parseLRCLine(row, lineNumber)
		if err != nil {
			return Document{}, err
		}
		rawLines = append(rawLines, parsed)
	}

	lines, err := buildLinesWithEnd(rawLines)
	if err != nil {
		return Document{}, err
	}
	doc, err := NewDocument(lines)
	if err != nil {
		if validationErr, ok := err.(*ValidationError); ok {
			validationErr.Line = 1
		}
		return Document{}, err
	}
	return doc.WithMetadata(metadata), nil
}

type rawParsedLine struct {
	start Timestamp
	end   *Timestamp
	text  Text
	words []WordTiming
}

func buildLinesWithEnd(rawLines []rawParsedLine) ([]Line, error) {
	lines := make([]Line, 0, len(rawLines))
	for i, raw := range rawLines {
		var end Timestamp
		var err error
		if raw.end != nil {
			end = *raw.end
		} else {
			var endMS int64
			if i+1 < len(rawLines) {
				endMS = rawLines[i+1].start.Milliseconds()
			} else {
				endMS = raw.start.Milliseconds() + defaultLastLineDurationMS
			}
			end, err = NewTimestamp(endMS)
			if err != nil {
				return nil, err
			}
		}
		var line Line
		if len(raw.words) > 0 {
			line, err = NewEnhancedLine(raw.start, end, raw.text, raw.words)
		} else {
			line, err = NewLine(raw.start, end, raw.text)
		}
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func isMetadataLine(row string) bool {
	if !strings.HasPrefix(row, "[") || !strings.HasSuffix(row, "]") {
		return false
	}
	content := strings.TrimSuffix(strings.TrimPrefix(row, "["), "]")
	key, _, ok := strings.Cut(content, ":")
	if !ok {
		return false
	}
	if _, err := ParseTimestamp(content); err == nil {
		return false
	}
	return strings.TrimSpace(key) != ""
}

func parseMetadataLine(row string, lineNumber int) (Metadata, error) {
	content := strings.TrimSuffix(strings.TrimPrefix(row, "["), "]")
	key, value, ok := strings.Cut(content, ":")
	if !ok {
		return Metadata{}, newValidationError(CodeMalformedLine, lineNumber, "metadata", row, "metadata must use [key:value]")
	}
	metadata, err := NewMetadata(key, value)
	return metadata, withLine(err, lineNumber)
}

func parseLRCLine(row string, lineNumber int) (rawParsedLine, error) {
	if !strings.HasPrefix(row, "[") {
		return rawParsedLine{}, newValidationError(CodeMalformedLine, lineNumber, "line", row, "lyric line must start with [mm:ss.xx]")
	}
	closing := strings.Index(row, "]")
	if closing <= 1 {
		return rawParsedLine{}, newValidationError(CodeMalformedLine, lineNumber, "line", row, "lyric line must include a closing timestamp bracket")
	}
	timePart := row[1:closing]
	var startTS, endTS Timestamp
	var err error
	var hasEnd bool
	if strings.Contains(timePart, "-") {
		parts := strings.Split(timePart, "-")
		if len(parts) != 2 {
			return rawParsedLine{}, newValidationError(CodeMalformedLine, lineNumber, "timestamp", row, "invalid start-end range")
		}
		startTS, err = parseTimestamp(parts[0], lineNumber, "timestamp")
		if err != nil {
			return rawParsedLine{}, err
		}
		endTS, err = parseTimestamp(parts[1], lineNumber, "timestamp")
		if err != nil {
			return rawParsedLine{}, err
		}
		hasEnd = true
	} else {
		startTS, err = parseTimestamp(timePart, lineNumber, "timestamp")
		if err != nil {
			return rawParsedLine{}, err
		}
	}

	text, words, err := parseEnhancedText(row[closing+1:], lineNumber)
	if err != nil {
		return rawParsedLine{}, err
	}
	var endPtr *Timestamp
	if hasEnd {
		endPtr = &endTS
	}
	return rawParsedLine{start: startTS, end: endPtr, text: text, words: words}, nil
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

func ParseLyrics(input string) (Document, error) {
	// Try parsing as LRC first
	doc, err := ParseLRC(input)
	if err == nil {
		return doc, nil
	}

	// Fallback to parsing line-by-line as plain text
	normalized := strings.ReplaceAll(input, "\r\n", "\n")
	var lines []Line
	var timeOffset int64 = 0

	for _, raw := range strings.Split(normalized, "\n") {
		row := strings.TrimSpace(raw)
		if row == "" {
			continue
		}
		startTS, _ := NewTimestamp(timeOffset)
		endTS, _ := NewTimestamp(timeOffset + 3000)
		txt, _ := NewText(row)
		line, err := NewLine(startTS, endTS, txt)
		if err != nil {
			return Document{}, err
		}
		lines = append(lines, line)
		timeOffset += 3000
	}

	if len(lines) == 0 {
		return Document{}, newValidationError(CodeEmptyDocument, 0, "lines", "", "document must contain at least one lyric line")
	}

	return NewDocument(lines)
}
