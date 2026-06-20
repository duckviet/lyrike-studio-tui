package lyrics

import "strings"

func FormatLRC(document Document) string {
	lines := document.Lines()
	formatted := make([]string, 0, len(lines))
	for _, line := range lines {
		formatted = append(formatted, formatLine(line))
	}
	return strings.Join(formatted, "\n")
}

func formatLine(line Line) string {
	var builder strings.Builder
	builder.WriteByte('[')
	builder.WriteString(line.Timestamp().String())
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
