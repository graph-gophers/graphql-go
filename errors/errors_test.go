package errors

import (
	"errors"
	"io"
	"testing"
)

func TestErrorf(t *testing.T) {
	cause := io.EOF

	t.Run("wrap error", func(t *testing.T) {
		err := Errorf("boom: %v", cause)
		if !errors.Is(err, cause) {
			t.Fatalf("expected errors.Is to return true")
		}
	})

	t.Run("handles nil", func(t *testing.T) {
		var err *QueryError
		if errors.Is(err, cause) {
			t.Fatalf("expected errors.Is to return false")
		}
	})

	t.Run("handle no arguments", func(t *testing.T) {
		err := Errorf("boom")
		if errors.Is(err, cause) {
			t.Fatalf("expected errors.Is to return false")
		}
	})

	t.Run("handle non-error argument arguments", func(t *testing.T) {
		err := Errorf("boom: %v", "shaka")
		if errors.Is(err, cause) {
			t.Fatalf("expected errors.Is to return false")
		}
	})
}
