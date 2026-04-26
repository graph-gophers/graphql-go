package norm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"text/scanner"
)

const maxReadIterations = 1 << 20

func readAllChunkedStrict(r io.Reader, chunk int) (string, error) {
	if chunk <= 0 {
		return "", fmt.Errorf("invalid chunk size %d", chunk)
	}

	var b strings.Builder
	buf := make([]byte, chunk)

	for range maxReadIterations {
		n, err := r.Read(buf)
		if n > 0 {
			b.Write(buf[:n])
		}

		if errors.Is(err, io.EOF) {
			return b.String(), nil
		}
		if err != nil {
			return "", err
		}

		if n == 0 {
			return "", errors.New("reader made no progress without EOF")
		}
	}

	return "", errors.New("reader did not terminate")
}

func scanAll(src string) {
	var sc scanner.Scanner
	sc.Init(strings.NewReader(src))
	sc.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings

	for {
		tok := sc.Scan()
		if tok == scanner.EOF {
			return
		}
	}
}

// go test ./internal/common/norm -run=^$ -fuzz=FuzzReader_ChunkInvariantAndBoundedOutput -fuzztime=10m
func FuzzReader_ChunkInvariantAndBoundedOutput(f *testing.F) {
	seeds := [][]byte{
		[]byte(``),
		[]byte(`{ f(arg: "plain") }`),
		[]byte(`{ f(arg: "\u{1F37A}") }`),
		[]byte(`{ f(arg: "\uD83C\uDF7A") }`),
		[]byte(`{ f(arg: "\uD83C") }`),
		[]byte(`{ f(arg: "\uDEAD") }`),
		[]byte(`{ f(arg: "\u{000000000}") }`),
		[]byte(`# comment with \u{1F37A}` + "\n" + `{ f(arg: "\u{41}") }`),
		[]byte(`{ f(arg: """\u{1F37A}""") }`),
		{0xff, '\\', 'u', '{', 'F', 'F', '}', 0x00, '\\', 'u', 'D', '8', '0', '0'},
	}

	for _, s := range seeds {
		f.Add(s, uint8(1))
		f.Add(s, uint8(31))
	}

	f.Fuzz(func(t *testing.T, raw []byte, chunkByte uint8) {
		input := string(raw)
		chunk := int(chunkByte%32) + 1

		outChunked, err := readAllChunkedStrict(NewReader(input), chunk)
		if err != nil {
			t.Fatalf("chunked read failed: %v", err)
		}

		outByteByByte, err := readAllChunkedStrict(NewReader(input), 1)
		if err != nil {
			t.Fatalf("byte-by-byte read failed: %v", err)
		}

		if outChunked != outByteByByte {
			t.Fatalf("output differs by chunking\nchunk=%d\noutChunked=%q\noutByteByByte=%q", chunk, outChunked, outByteByByte)
		}

		// Rewrites only happen from escapes beginning with `\u`, so output growth is bounded.
		if len(outChunked) > 2*len(input)+16 {
			t.Fatalf("unexpected output growth: in=%d out=%d", len(input), len(outChunked))
		}

		// Ensure the normalized stream is always scanner-safe (no panics).
		scanAll(outChunked)
	})
}

// go test ./internal/common/norm -run=^$ -fuzz=FuzzReader_FastPathParityWithoutUnicodeMarker -fuzztime=1m
func FuzzReader_FastPathParityWithoutUnicodeMarker(f *testing.F) {
	seeds := [][]byte{
		[]byte(``),
		[]byte(`query { field }`),
		[]byte(`"\\u"`),
		[]byte(`prefix u suffix`),
		{0xff, 0xfe, 'a', 'b'},
	}

	for _, s := range seeds {
		f.Add(s, uint8(7))
	}

	f.Fuzz(func(t *testing.T, raw []byte, chunkByte uint8) {
		if bytes.Contains(raw, []byte(`\u`)) {
			t.Skip()
		}

		input := string(raw)
		chunk := int(chunkByte%32) + 1

		got, err := readAllChunkedStrict(NewReader(input), chunk)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}

		want, err := readAllChunkedStrict(strings.NewReader(input), chunk)
		if err != nil {
			t.Fatalf("reference read failed: %v", err)
		}

		if got != want {
			t.Fatalf("fast-path mismatch\ngot =%q\nwant=%q", got, want)
		}
	})
}
