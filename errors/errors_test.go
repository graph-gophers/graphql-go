package errors

import (
	"io"
	"testing"
)

// Is is simplified facsimile of the go 1.13 errors.Is to ensure QueryError is compatible
func Is(err, target error) bool {
	for err != nil {
		if target == err {
			return true
		}

		switch e := err.(type) {
		case interface{ Unwrap() error }:
			err = e.Unwrap()
		default:
			break
		}
	}
	return false
}

func TestErrorf(t *testing.T) {
	cause := io.EOF

	t.Run("wrap error", func(t *testing.T) {
		err := Errorf("boom: %v", cause)
		if !Is(err, cause) {
			t.Fatalf("expected errors.Is to return true")
		}
	})

	t.Run("handles nil", func(t *testing.T) {
		var err *QueryError
		if Is(err, cause) {
			t.Fatalf("expected errors.Is to return false")
		}
	})

	t.Run("handle no arguments", func(t *testing.T) {
		err := Errorf("boom")
		if Is(err, cause) {
			t.Fatalf("expected errors.Is to return false")
		}
	})

	t.Run("handle non-error argument arguments", func(t *testing.T) {
		err := Errorf("boom: %v", "shaka")
		if Is(err, cause) {
			t.Fatalf("expected errors.Is to return false")
		}
	})
}
