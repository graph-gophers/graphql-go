package errors

import (
	"context"
	"testing"
)

func TestDefaultPanicHandler(t *testing.T) {
	handler := &DefaultPanicHandler{}
	qErr := handler.MakePanicError(context.Background(), "foo")
	if qErr == nil {
		t.Fatal("Panic error must not be nil")
	}
	const (
		expectedMessage = "panic occurred: foo"
		expectedError   = "graphql: " + expectedMessage
	)
	if qErr.Error() != expectedError {
		t.Errorf("Unexpected panic error message: %q != %q", qErr.Error(), expectedError)
	}
	if qErr.Message != expectedMessage {
		t.Errorf("Unexpected panic QueryError.Message: %q != %q", qErr.Message, expectedMessage)
	}
}
