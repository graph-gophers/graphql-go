package query

import (
	"fmt"
	"testing"
)

func Benchmark_Parse(b *testing.B) {
	tests := []int{1, 10, 100, 1000, 10000, 100000, 1000000}
	for _, tt := range tests {
		b.Run(fmt.Sprintf("Benchmark_Parse_%d", tt), func(b *testing.B) {
			query := "query {"
			for i := 0; i < tt; i++ {
				query += " name"
			}
			query += " }"
			for i := 0; i < b.N; i++ {
				Parse(query)
			}
		})
	}
}
