package query

import (
	"testing"
)

func FuzzParseQuery(f *testing.F) {
    f.Fuzz(func(t *testing.T, queryStr string) {
        Parse(queryStr)
    })
}

