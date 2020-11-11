package errors

import (
	"fmt"
)

type QueryError struct {
	Message       string                 `json:"message"`
	Locations     []Location             `json:"locations,omitempty"`
	Path          []interface{}          `json:"path,omitempty"`
	Rule          string                 `json:"-"`
	ResolverError error                  `json:"-"`
	Extensions    map[string]interface{} `json:"extensions,omitempty"`
}

type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
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

// SubscriptionError can be implemented by top-level resolver object to communicate to
// the library a terminal subscription error happened while the stream is still active.
//
// After a subscription has started, this is the mechanism to inform subscriber about stream
// failure in a graceful manner.
//
// **Note** This works only on the top-level object of the resolver, when implemented
// by fields selector, this has no effect.
type SubscriptionError interface {
	// SubscriptionError is called to determined if a terminal error occurred. If the returned
	// value is nil, subscription continues normally. If the error is non-nil, the subscription is
	// assumed to have reached a terminal error, the subscription's channel is closed and the error
	// is returned to the user.
	//
	// If the non-nil error returned is a *QueryError type, it is returned as-is to the user, otherwise,
	// the non-nill error is wrapped using `Errorf("%s", err)` above.
	SubscriptionError() error
}
