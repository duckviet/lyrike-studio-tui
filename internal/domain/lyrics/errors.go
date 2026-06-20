package lyrics

import "fmt"

type ErrorCode string

const (
	CodeEmptyDocument           ErrorCode = "empty_document"
	CodeEmptyText               ErrorCode = "empty_text"
	CodeDuplicateTimestamp      ErrorCode = "duplicate_timestamp"
	CodeInvalidTimestamp        ErrorCode = "invalid_timestamp"
	CodeMalformedEnhancedMarker ErrorCode = "malformed_enhanced_marker"
	CodeMalformedLine           ErrorCode = "malformed_line"
	CodeUnsortedTimestamp       ErrorCode = "unsorted_timestamp"
)

type ValidationError struct {
	Code    ErrorCode
	Line    int
	Field   string
	Value   string
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return "lyrics validation error"
	}
	if e.Line > 0 {
		return fmt.Sprintf("lyrics validation error at line %d: %s", e.Line, e.Message)
	}
	return fmt.Sprintf("lyrics validation error: %s", e.Message)
}

func newValidationError(code ErrorCode, line int, field string, value string, message string) *ValidationError {
	return &ValidationError{
		Code:    code,
		Line:    line,
		Field:   field,
		Value:   value,
		Message: message,
	}
}
