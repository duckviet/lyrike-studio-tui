package lyrics

type WordTiming struct {
	timestamp Timestamp
	text      Text
}

type Line struct {
	timestamp Timestamp
	text      Text
	words     []WordTiming
}

func NewWordTiming(timestamp Timestamp, text Text) (WordTiming, error) {
	if text.String() == "" {
		return WordTiming{}, newValidationError(CodeEmptyText, 0, "word", "", "enhanced lyric word must not be empty")
	}
	return WordTiming{timestamp: timestamp, text: text}, nil
}

func NewLine(timestamp Timestamp, text Text) (Line, error) {
	if text.String() == "" {
		return Line{}, newValidationError(CodeEmptyText, 0, "text", "", "lyric text must not be empty")
	}
	return Line{timestamp: timestamp, text: text}, nil
}

func NewEnhancedLine(timestamp Timestamp, text Text, words []WordTiming) (Line, error) {
	line, err := NewLine(timestamp, text)
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

func (l Line) Timestamp() Timestamp {
	return l.timestamp
}

func (l Line) Text() Text {
	return l.text
}

func (l Line) Words() []WordTiming {
	return append([]WordTiming(nil), l.words...)
}

func (w WordTiming) Timestamp() Timestamp {
	return w.timestamp
}

func (w WordTiming) Text() Text {
	return w.text
}
