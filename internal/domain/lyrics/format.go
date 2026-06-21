package lyrics

import "strings"

// FormatLRC formats a document as standard LRC (start timestamp only).
func FormatLRC(document Document) string {
	lines := document.Lines()
	metadata := document.Metadata()
	formatted := make([]string, 0, len(metadata)+len(lines))
	for _, item := range metadata {
		formatted = append(formatted, "["+item.Key+":"+item.Value+"]")
	}
	for _, line := range lines {
		formatted = append(formatted, formatLine(line))
	}
	return strings.Join(formatted, "\n")
}

// FormatLRCWithEnd formats a document with start-end timestamps for karaoke timing.
func FormatLRCWithEnd(document Document) string {
	lines := document.Lines()
	metadata := document.Metadata()
	formatted := make([]string, 0, len(metadata)+len(lines))
	for _, item := range metadata {
		formatted = append(formatted, "["+item.Key+":"+item.Value+"]")
	}
	for _, line := range lines {
		formatted = append(formatted, formatLineWithEnd(line))
	}
	return strings.Join(formatted, "\n")
}

func formatLine(line Line) string {
	var builder strings.Builder
	builder.WriteByte('[')
	builder.WriteString(line.Start().String())
	builder.WriteByte(']')

	words := line.Words()
	if len(words) == 0 {
		builder.WriteString(line.Text().String())
		return builder.String()
	}
	for index, word := range words {
		if index > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteByte('<')
		builder.WriteString(word.Timestamp().String())
		builder.WriteByte('>')
		builder.WriteString(word.Text().String())
	}
	return builder.String()
}

func formatLineWithEnd(line Line) string {
	var builder strings.Builder
	builder.WriteByte('[')
	builder.WriteString(line.Start().String())
	builder.WriteByte('-')
	builder.WriteString(line.End().String())
	builder.WriteByte(']')
	builder.WriteString(line.Text().String())
	return builder.String()
}
