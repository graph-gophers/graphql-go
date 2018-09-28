package errors

import (
	"fmt"
)

type GraphQLError interface {
	PrepareExtErr() *QueryError
}

type QueryError struct {
	Message       string        `json:"message"`
	Locations     []Location    `json:"locations,omitempty"`
	Path          []interface{} `json:"path,omitempty"`
	Rule          string        `json:"-"`
	ResolverError error         `json:"-"`
	Extensions    Extensions    `json:"extensions"`
}

type Extensions struct {
	Code             int    `json:"code,omitempty"`
	DeveloperMessage string `json:"developerMessage,omitempty"`
	MoreInfo         string `json:"moreInfo,omitempty"`
	Timestamp        string `json:"timestamp,omitempty"`
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
	if err.Extensions.Code != 0 {
		str += fmt.Sprintf(" code: %d", err.Extensions.Code)
	}
	if err.Extensions.DeveloperMessage != "" {
		str += fmt.Sprintf(" developerMessage: %s", err.Extensions.DeveloperMessage)
	}
	if err.Extensions.MoreInfo != "" {
		str += fmt.Sprintf(" moreInfo: %s", err.Extensions.MoreInfo)
	}
	if err.Extensions.Timestamp != "" {
		str += fmt.Sprintf(" timestamp: %s", err.Extensions.Timestamp)
	}
	for _, loc := range err.Locations {
		str += fmt.Sprintf(" (line %d, column %d)", loc.Line, loc.Column)
	}
	return str
}

var _ error = &QueryError{}

func (err *QueryError) AddErrCode(code int) *QueryError {
	err.Extensions.Code = code
	return err
}

func (err *QueryError) AddDevMsg(msg string) *QueryError {
	err.Extensions.DeveloperMessage = msg
	return err
}

func (err *QueryError) AddMoreInfo(moreInfo string) *QueryError {
	err.Extensions.MoreInfo = moreInfo
	return err
}

func (err *QueryError) AddErrTimestamp(errTime string) *QueryError {
	err.Extensions.Timestamp = errTime
	return err
}
