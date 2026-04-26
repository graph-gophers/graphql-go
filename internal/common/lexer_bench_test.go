package common

import (
	"io"
	"strings"
	"testing"
	"text/scanner"

	"github.com/graph-gophers/graphql-go/internal/common/norm"
)

const benchmarkLexerNonTransformingInput = `
schema {
	query: Query
}

type Query {
	user(id: ID!): User
	users(limit: Int = 100): [User!]!
}

type User {
	id: ID!
	name: String!
	email: String!
	friends(first: Int = 10): [User!]!
}
`

func BenchmarkLexerReaderNonTransforming(b *testing.B) {
	b.Run("before_strings_reader", func(b *testing.B) {
		benchmarkScanWithReader(b, func() io.Reader {
			return strings.NewReader(benchmarkLexerNonTransformingInput)
		})
	})

	b.Run("after_norm_reader", func(b *testing.B) {
		benchmarkScanWithReader(b, func() io.Reader {
			return norm.NewReader(benchmarkLexerNonTransformingInput)
		})
	})
}

func benchmarkScanWithReader(b *testing.B, reader func() io.Reader) {
	b.Helper()
	b.ReportAllocs()

	for b.Loop() {
		sc := &scanner.Scanner{
			Mode: scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings,
		}
		sc.Init(reader())

		for tok := sc.Scan(); tok != scanner.EOF; tok = sc.Scan() {
		}
	}
}
