package errors

import (
	"fmt"
)

type QueryError struct {
	Message       string        `json:"message"`
	Locations     []Location    `json:"locations,omitempty"`
	Path          []interface{} `json:"path,omitempty"`
	Rule          string        `json:"-"`
	ResolverError error         `json:"-"`
}

type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// ErrorHandler describes a function that converts an error to a QueryError
type ErrorHandler func(error) *QueryError

// DefaultErrorHandler returns the default error handler
func DefaultErrorHandler() ErrorHandler {
	return func(err error) *QueryError {
		return Errorf("%s", err)
	}
}

func (a Location) Before(b Location) bool {
	return a.Line < b.Line || (a.Line == b.Line && a.Column < b.Column)
}

func Errorf(format string, a ...interface{}) *QueryError {
	return &QueryError{
		Message: fmt.Sprintf(format, a...),
	}
}

func (err *QueryError) Error() string {
	if err == nil {
		return "<nil>"
	}
	str := fmt.Sprintf("graphql: %s", err.Message)
	for _, loc := range err.Locations {
		str += fmt.Sprintf(" (line %d, column %d)", loc.Line, loc.Column)
	}
	return str
}

var _ error = &QueryError{}
