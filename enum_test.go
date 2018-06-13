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
	for n := 0; n < b.N; n++ {
		if s, ok := v.Interface().(fmt.Stringer); ok {
			s.String()
		} else {
			v.String()
		}
	}
}

func BenchmarkEnumStringFmt(b *testing.B) {
	v := reflect.ValueOf(benchStringEnum("TEST"))
	for n := 0; n < b.N; n++ {
		fmt.Sprintf("%s", v)
	}
}

type benchIntEnum int

func (i benchIntEnum) String() string {
	return strconv.Itoa(int(i))
}

func BenchmarkEnumIntStringer(b *testing.B) {
	v := reflect.ValueOf(benchIntEnum(1))
	for n := 0; n < b.N; n++ {
		if s, ok := v.Interface().(fmt.Stringer); ok {
			s.String()
		} else {
			v.String()
		}
	}
}

func BenchmarkEnumIntFmt(b *testing.B) {
	v := reflect.ValueOf(benchIntEnum(1))
	for n := 0; n < b.N; n++ {
		fmt.Sprintf("%s", v)
	}
}
