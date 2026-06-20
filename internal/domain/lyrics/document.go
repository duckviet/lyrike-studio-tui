package lyrics

type Document struct {
	lines []Line
}

func NewDocument(lines []Line) (Document, error) {
	if len(lines) == 0 {
		return Document{}, newValidationError(CodeEmptyDocument, 0, "lines", "", "lyrics document must contain at least one line")
	}
	for i := 1; i < len(lines); i++ {
		previous := lines[i-1].Timestamp().Milliseconds()
		current := lines[i].Timestamp().Milliseconds()
		if current == previous {
			return Document{}, newValidationError(CodeDuplicateTimestamp, 0, "timestamp", lines[i].Timestamp().String(), "lyric timestamps must be unique")
		}
		if current < previous {
			return Document{}, newValidationError(CodeUnsortedTimestamp, 0, "timestamp", lines[i].Timestamp().String(), "lyric timestamps must be strictly increasing")
		}
	}
	return Document{lines: append([]Line(nil), lines...)}, nil
}

func (d Document) Lines() []Line {
	return append([]Line(nil), d.lines...)
}
