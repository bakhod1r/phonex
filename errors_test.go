package phonex

import (
	"errors"
	"testing"
)

func TestErrorsAreComparable(t *testing.T) {
	_, err := Parse("+9989")
	if !errors.Is(err, ErrTooShort) {
		t.Errorf("error = %v, want ErrTooShort", err)
	}
	if errors.Is(err, ErrTooLong) {
		t.Error("ErrTooShort should not match ErrTooLong")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatal("error should unwrap to *ValidationError")
	}
	if ve.Code != CodeTooShort {
		t.Errorf("Code = %v, want CodeTooShort", ve.Code)
	}
	if ve.Error() == "" {
		t.Error("Error() should not be empty")
	}
}
