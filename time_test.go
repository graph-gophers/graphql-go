package graphql_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	. "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/decode"
)

func TestTime_ImplementsUnmarshaler(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Error(err)
		}
	}()

	// assert *Time implements decode.Unmarshaler interface
	var _ decode.Unmarshaler = (*Time)(nil)
}

func TestTime_ImplementsGraphQLType(t *testing.T) {
	gt := new(Time)

	if gt.ImplementsGraphQLType("foobar") {
		t.Error("Type *Time must not claim to implement GraphQL type 'foobar'")
	}

	if !gt.ImplementsGraphQLType("Time") {
		t.Error("Failed asserting *Time implements GraphQL type Time")
	}
}

func TestTime_MarshalJSON(t *testing.T) {
	var err error
	var b1, b2 []byte
	ref := time.Date(2021, time.April, 20, 12, 3, 23, 0, time.UTC)

	if b1, err = json.Marshal(ref); err != nil {
		t.Error(err)
		return
	}

	if b2, err = json.Marshal(Time{Time: ref}); err != nil {
		t.Errorf("MarshalJSON() error = %v", err)
		return
	}

	if !bytes.Equal(b1, b2) {
		t.Errorf("MarshalJSON() got = %s, want = %s", b2, b1)
	}
}

func TestTime_UnmarshalGraphQL(t *testing.T) {
	type args struct {
		input interface{}
	}

	ref := time.Date(2021, time.April, 20, 12, 3, 23, 0, time.UTC)

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name    string
			args    args
			wantErr string
		}{
			{
				name:    "boolean",
				args:    args{input: true},
				wantErr: "wrong type for Time: bool",
			},
			{
				name:    "invalid format",
				args:    args{input: ref.Format(time.ANSIC)},
				wantErr: `parsing time "Tue Apr 20 12:03:23 2021" as "2006-01-02T15:04:05Z07:00": cannot parse "Tue Apr 20 12:03:23 2021" as "2006"`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				gt := new(Time)
				if err := gt.UnmarshalGraphQL(tt.args.input); err != nil {
					if err.Error() != tt.wantErr {
						t.Errorf("UnmarshalGraphQL() error = %v, want = %s", err, tt.wantErr)
					}

					return
				}

				t.Error("UnmarshalGraphQL() expected error not raised")
			})
		}
	})

	tests := []struct {
		name   string
		args   args
		wantEq time.Time
	}{
		{
			name: "time.Time",
			args: args{
				input: ref,
			},
			wantEq: ref,
		},
		{
			name: "string",
			args: args{
				input: ref.Format(time.RFC3339),
			},
			wantEq: ref,
		},
		{
			name: "bytes",
			args: args{
				input: []byte(ref.Format(time.RFC3339)),
			},
			wantEq: ref,
		},
		{
			name: "int32",
			args: args{
				input: int32(ref.Unix()),
			},
			wantEq: ref,
		},
		{
			name: "int64",
			args: args{
				input: ref.Unix(),
			},
			wantEq: ref,
		},
		{
			name: "float64",
			args: args{
				input: float64(ref.Unix()),
			},
			wantEq: ref,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gt := new(Time)
			if err := gt.UnmarshalGraphQL(tt.args.input); err != nil {
				t.Errorf("UnmarshalGraphQL() error = %v", err)
				return
			}

			if !gt.Equal(tt.wantEq) {
				t.Errorf("UnmarshalGraphQL() got = %v, want = %v", gt, tt.wantEq)
			}
		})
	}
}
