package lyrics

type WordTiming struct {
	timestamp Timestamp
	text      Text
}

type Line struct {
	start Timestamp
	end   Timestamp
	text  Text
	words []WordTiming
}

func NewWordTiming(timestamp Timestamp, text Text) (WordTiming, error) {
	return WordTiming{timestamp: timestamp, text: text}, nil
}

func NewLine(start Timestamp, end Timestamp, text Text) (Line, error) {
	if start.Milliseconds() >= end.Milliseconds() {
		return Line{}, newValidationError(CodeInvalidSegment, 0, "segment", "",
			"segment start must be before end")
	}
	return Line{start: start, end: end, text: text}, nil
}

func NewEnhancedLine(start Timestamp, end Timestamp, text Text, words []WordTiming) (Line, error) {
	line, err := NewLine(start, end, text)
	if err != nil {
		return Line{}, err
	}
	if len(words) == 0 {
		return line, nil
	}
	for i := 1; i < len(words); i++ {
		if words[i].timestamp.Milliseconds() <= words[i-1].timestamp.Milliseconds() {
			return Line{}, newValidationError(CodeUnsortedTimestamp, 0, "word_timestamp", words[i].timestamp.String(), "enhanced word timestamps must be strictly increasing")
		}
	}
	line.words = append([]WordTiming(nil), words...)
	return line, nil
}

func (l Line) Start() Timestamp {
	return l.start
}

func (l Line) End() Timestamp {
	return l.end
}

// Timestamp returns the start timestamp. Kept as alias for backward compat in formatters.
func (l Line) Timestamp() Timestamp {
	return l.start
}

func (l Line) Text() Text {
	return l.text
}

func (l Line) Words() []WordTiming {
	return append([]WordTiming(nil), l.words...)
}

func (l Line) WithStart(start Timestamp) Line {
	return Line{start: start, end: l.end, text: l.text, words: append([]WordTiming(nil), l.words...)}
}

func (l Line) WithEnd(end Timestamp) Line {
	return Line{start: l.start, end: end, text: l.text, words: append([]WordTiming(nil), l.words...)}
}

func (l Line) WithText(text Text) Line {
	return Line{start: l.start, end: l.end, text: text, words: append([]WordTiming(nil), l.words...)}
}

func (w WordTiming) Timestamp() Timestamp {
	return w.timestamp
}

func (w WordTiming) Text() Text {
	return w.text
}
