package lyrics

type Document struct {
	lines    []Line
	metadata []Metadata
}

func NewDocument(lines []Line) (Document, error) {
	if len(lines) == 0 {
		return Document{}, newValidationError(CodeEmptyDocument, 0, "lines", "", "document must contain at least one lyric line")
	}
	for i, line := range lines {
		if line.Start().Milliseconds() >= line.End().Milliseconds() {
			return Document{}, newValidationError(CodeInvalidSegment, 0, "segment", line.Start().String(),
				"segment start must be before end")
		}
		if i > 0 {
			prev := lines[i-1]
			if line.Start().Milliseconds() <= prev.Start().Milliseconds() {
				return Document{}, newValidationError(CodeUnsortedTimestamp, 0, "timestamp", line.Start().String(),
					"lyric start timestamps must be strictly increasing")
			}
			if line.Start().Milliseconds() < prev.End().Milliseconds() {
				return Document{}, newValidationError(CodeOverlappingSegment, 0, "segment", line.Start().String(),
					"segments must not overlap")
			}
		}
	}
	return Document{lines: append([]Line(nil), lines...)}, nil
}

func (d Document) Lines() []Line {
	return d.cloneLines()
}

func (d Document) Metadata() []Metadata {
	return append([]Metadata(nil), d.metadata...)
}

func (d Document) WithMetadata(metadata []Metadata) Document {
	return Document{
		lines:    d.cloneLines(),
		metadata: append([]Metadata(nil), metadata...),
	}
}

func (d Document) WithLineStart(index int, start Timestamp) (Document, error) {
	if err := d.validateIndex(index); err != nil {
		return Document{}, err
	}
	newLines := d.cloneLines()
	newLines[index] = newLines[index].WithStart(start)
	return d.withLines(newLines)
}

func (d Document) WithLineEnd(index int, end Timestamp) (Document, error) {
	if err := d.validateIndex(index); err != nil {
		return Document{}, err
	}
	newLines := d.cloneLines()
	newLines[index] = newLines[index].WithEnd(end)
	return d.withLines(newLines)
}

func (d Document) WithLineText(index int, text Text) (Document, error) {
	if err := d.validateIndex(index); err != nil {
		return Document{}, err
	}
	newLines := d.cloneLines()
	newLines[index] = newLines[index].WithText(text)
	return d.withLines(newLines)
}

func (d Document) WithInsertedLine(index int, line Line) (Document, error) {
	if index < 0 || index > len(d.lines) {
		return Document{}, newValidationError(CodeInvalidIndex, 0, "index", "", "line index out of range")
	}
	newLines := d.cloneLines()
	newLines = append(newLines, Line{})
	copy(newLines[index+1:], newLines[index:])
	newLines[index] = line
	return d.withLines(newLines)
}

func (d Document) WithDeletedLine(index int) (Document, error) {
	if err := d.validateIndex(index); err != nil {
		return Document{}, err
	}
	newLines := d.cloneLines()
	newLines = append(newLines[:index], newLines[index+1:]...)
	return d.withLines(newLines)
}

func (d Document) WithReorderedLine(fromIndex, toIndex int) (Document, error) {
	if err := d.validateIndex(fromIndex); err != nil {
		return Document{}, err
	}
	if err := d.validateIndex(toIndex); err != nil {
		return Document{}, err
	}
	if fromIndex == toIndex {
		return d, nil
	}
	original := d.cloneLines()
	reordered := d.cloneLines()
	moved := reordered[fromIndex]
	reordered = append(reordered[:fromIndex], reordered[fromIndex+1:]...)
	reordered = append(reordered, Line{})
	copy(reordered[toIndex+1:], reordered[toIndex:])
	reordered[toIndex] = moved

	newLines := make([]Line, len(original))
	for index, line := range reordered {
		newLines[index] = Line{
			start: original[index].start,
			end:   original[index].end,
			text:  line.text,
			words: append([]WordTiming(nil), line.words...),
		}
	}
	return d.withLines(newLines)
}

func (d Document) validateIndex(index int) error {
	if index < 0 || index >= len(d.lines) {
		return newValidationError(CodeInvalidIndex, 0, "index", "", "line index out of range")
	}
	return nil
}

func (d Document) cloneLines() []Line {
	return append([]Line(nil), d.lines...)
}

func (d Document) withLines(lines []Line) (Document, error) {
	next, err := NewDocument(lines)
	if err != nil {
		return Document{}, err
	}
	return next.WithMetadata(d.metadata), nil
}
