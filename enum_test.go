package graphql_test

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

type benchStringEnum string

func BenchmarkEnumStringStringer(b *testing.B) {
	v := reflect.ValueOf(benchStringEnum("TEST"))
	var out string
	for n := 0; n < b.N; n++ {
		if _, ok := v.Interface().(fmt.Stringer); ok {
			b.Error("this should not need fmt.Stringer")
		} else {
			out = v.String()
		}
	}

	if out != "TEST" {
		b.Errorf("unexpected output: %q", out)
	}
}

func BenchmarkEnumStringFmt(b *testing.B) {
	v := reflect.ValueOf(benchStringEnum("TEST"))
	var out string
	for n := 0; n < b.N; n++ {
		out = fmt.Sprintf("%s", v)
	}

	if out != "TEST" {
		b.Errorf("unexpected output: %q", out)
	}
}

type benchIntEnum int

func (i benchIntEnum) String() string {
	return strconv.Itoa(int(i))
}

func BenchmarkEnumIntStringer(b *testing.B) {
	v := reflect.ValueOf(benchIntEnum(1))
	var out string
	for n := 0; n < b.N; n++ {
		if s, ok := v.Interface().(fmt.Stringer); ok {
			out = s.String()
		} else {
			b.Error("this should use fmt.Stringer")
		}
	}

	if out != "1" {
		b.Errorf("unexpected output: %q", out)
	}
}

func BenchmarkEnumIntFmt(b *testing.B) {
	v := reflect.ValueOf(benchIntEnum(1))
	var out string
	for n := 0; n < b.N; n++ {
		out = fmt.Sprintf("%s", v)
	}

	if out != "1" {
		b.Errorf("unexpected output: %q", out)
	}
}
