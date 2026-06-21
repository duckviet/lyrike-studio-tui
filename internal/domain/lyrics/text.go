package lyrics

import "strings"

type Text struct {
	value string
}

func NewText(input string) (Text, error) {
	value := strings.TrimSpace(input)
	return Text{value: value}, nil
}

func (t Text) String() string {
	return t.value
}
