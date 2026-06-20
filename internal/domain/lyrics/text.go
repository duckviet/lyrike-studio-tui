package lyrics

import "strings"

type Text struct {
	value string
}

func NewText(input string) (Text, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return Text{}, newValidationError(CodeEmptyText, 0, "text", input, "lyric text must not be empty")
	}
	return Text{value: value}, nil
}

func (t Text) String() string {
	return t.value
}
