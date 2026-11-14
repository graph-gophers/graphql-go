package exec

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// FuzzWriteScalar checks that scalars are encoded exactly like json.Marshal
// encodes them, both below and above largeValueSize.
func FuzzWriteScalar(f *testing.F) {
	for _, s := range []string{
		"",
		"plain",
		"\"\\/\b\f\n\r\t\x00\x1f\x7f",
		`<a href="x">&amp;</a>`,
		"\u2028\u2029",
		"\xff\xfe \xed\xa0\x80 \xe2\x80",
		"日本語 🎉",
		` { "a" : [ 1 , "<\u2028>" ] } `,
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		large := strings.Repeat(s+"\xe2", largeValueSize/(len(s)+1)+1)
		largeJSON := "[" + strings.Repeat(s+",", largeValueSize/(len(s)+1)+1) + "0]"
		for _, v := range []any{s, large, json.RawMessage(s), json.RawMessage(largeJSON)} {
			want, wantErr := json.Marshal(v)
			var o output
			o.WriteString("[")
			err := o.writeScalar(v)
			o.WriteString("]")
			if fmt.Sprint(err) != fmt.Sprint(wantErr) {
				t.Fatalf("%T %q: got error %v, want %v", v, v, err, wantErr)
			}
			if err == nil {
				got := o.bytes()
				if string(got) != "["+string(want)+"]" {
					t.Fatalf("%T %q: got %q, want [%q]", v, v, got, want)
				}
				// Large values must be encoded straight into a buffer of the exact size.
				if len(want) >= largeValueSize && cap(got) != len(got) {
					t.Fatalf("%T %q: got capacity %d for %d bytes", v, v, cap(got), len(got))
				}
			}
		}
	})
}
