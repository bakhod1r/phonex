package phonex

import "fmt"

type ErrorCode int

const (
	CodeInvalidCharacters ErrorCode = iota + 1
	CodeTooShort
	CodeTooLong
	CodeInvalidCountry
	CodeInvalidCountryCode
	CodeInvalidPrefix
	CodeInvalidFormat
	CodeInvalidType
	CodeMissingCountry
	CodeInvalidExtension
	CodeInvalidLength
)

type ValidationError struct {
	Code    ErrorCode
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("phonex: %s", e.Message)
}

func (e *ValidationError) Is(target error) bool {
	if t, ok := target.(*ValidationError); ok {
		return e.Code == t.Code
	}
	return false
}

var (
	ErrInvalidCharacters  = &ValidationError{Code: CodeInvalidCharacters, Message: "invalid characters"}
	ErrTooShort           = &ValidationError{Code: CodeTooShort, Message: "too short"}
	ErrTooLong            = &ValidationError{Code: CodeTooLong, Message: "too long"}
	ErrInvalidCountry     = &ValidationError{Code: CodeInvalidCountry, Message: "invalid country"}
	ErrInvalidCountryCode = &ValidationError{Code: CodeInvalidCountryCode, Message: "invalid country code"}
	ErrInvalidPrefix      = &ValidationError{Code: CodeInvalidPrefix, Message: "invalid prefix"}
	ErrInvalidFormat      = &ValidationError{Code: CodeInvalidFormat, Message: "invalid format"}
	ErrInvalidType        = &ValidationError{Code: CodeInvalidType, Message: "invalid type"}
	ErrMissingCountry     = &ValidationError{Code: CodeMissingCountry, Message: "missing country"}
	ErrInvalidExtension   = &ValidationError{Code: CodeInvalidExtension, Message: "invalid extension"}
	// ErrInvalidLength reports a digit count that falls between two lengths
	// the region allows, so the number is neither too short nor too long.
	ErrInvalidLength = &ValidationError{Code: CodeInvalidLength, Message: "invalid length for region"}
)
