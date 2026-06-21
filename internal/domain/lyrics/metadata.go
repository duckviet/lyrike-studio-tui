package lyrics

import (
	"strings"
)

type Metadata struct {
	Key   string
	Value string
}

func NewMetadata(key string, value string) (Metadata, error) {
	normalizedKey := strings.TrimSpace(key)
	if normalizedKey == "" {
		return Metadata{}, newValidationError(CodeMalformedLine, 0, "metadata", key, "metadata key must not be empty")
	}
	return Metadata{
		Key:   normalizedKey,
		Value: strings.TrimSpace(value),
	}, nil
}
