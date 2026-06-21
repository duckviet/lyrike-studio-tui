package lyrics

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Timestamp struct {
	milliseconds int64
}

func NewTimestamp(milliseconds int64) (Timestamp, error) {
	if milliseconds < 0 {
		return Timestamp{}, newValidationError(CodeInvalidTimestamp, 0, "timestamp", strconv.FormatInt(milliseconds, 10), "timestamp must be non-negative")
	}
	return Timestamp{milliseconds: milliseconds}, nil
}

func ParseTimestamp(input string) (Timestamp, error) {
	return parseTimestamp(input, 0, "timestamp")
}

func (t Timestamp) Milliseconds() int64 {
	return t.milliseconds
}

func (t Timestamp) String() string {
	totalSeconds := t.milliseconds / 1000
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	millis := t.milliseconds % 1000
	if millis%10 == 0 {
		return fmt.Sprintf("%02d:%02d.%02d", minutes, seconds, millis/10)
	}
	return fmt.Sprintf("%02d:%02d.%03d", minutes, seconds, millis)
}

func parseTimestamp(input string, line int, field string) (Timestamp, error) {
	minutePart, rest, ok := strings.Cut(input, ":")
	if !ok {
		return Timestamp{}, invalidTimestamp(line, field, input)
	}
	secondPart, fractionPart, ok := strings.Cut(rest, ".")
	if !ok {
		return Timestamp{}, invalidTimestamp(line, field, input)
	}
	if minutePart == "" || len(secondPart) != 2 || (len(fractionPart) != 2 && len(fractionPart) != 3) {
		return Timestamp{}, invalidTimestamp(line, field, input)
	}
	if !allDigits(minutePart) || !allDigits(secondPart) || !allDigits(fractionPart) {
		return Timestamp{}, invalidTimestamp(line, field, input)
	}

	minutes, err := strconv.ParseInt(minutePart, 10, 64)
	if err != nil {
		return Timestamp{}, invalidTimestamp(line, field, input)
	}
	seconds, err := strconv.ParseInt(secondPart, 10, 64)
	if err != nil || seconds > 59 {
		return Timestamp{}, invalidTimestamp(line, field, input)
	}
	fraction, err := strconv.ParseInt(fractionPart, 10, 64)
	if err != nil {
		return Timestamp{}, invalidTimestamp(line, field, input)
	}
	if len(fractionPart) == 2 {
		fraction *= 10
	}
	return NewTimestamp((minutes*60+seconds)*1000 + fraction)
}

func invalidTimestamp(line int, field string, value string) *ValidationError {
	return newValidationError(CodeInvalidTimestamp, line, field, value, "invalid timestamp: must use mm:ss.xx with seconds from 00 to 59")
}

func allDigits(input string) bool {
	for _, r := range input {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
