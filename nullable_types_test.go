package graphql_test

import (
	"math"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/decode"
)

func TestNullID_ImplementsUnmarshaler(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Error(err)
		}
	}()

	// assert *NullID implements decode.Unmarshaler interface
	var _ decode.Unmarshaler = (*graphql.NullID)(nil)
}

func TestNullID_UnmarshalGraphQL(t *testing.T) {
	type args struct {
		input interface{}
	}

	good := graphql.ID("1234")
	ref := graphql.NullID{
		Value: &good,
		Set:   true,
	}

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name    string
			args    args
			wantErr string
		}{
			{
				name:    "boolean",
				args:    args{input: true},
				wantErr: "wrong type for ID: bool",
			},
			{
				name:    "int",
				args:    args{input: 1},
				wantErr: "wrong type for ID: int",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				gt := &graphql.NullID{}
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
		wantEq graphql.NullID
	}{
		{
			name: "string",
			args: args{
				input: string(good),
			},
			wantEq: ref,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gt := new(graphql.NullID)
			if err := gt.UnmarshalGraphQL(tt.args.input); err != nil {
				t.Errorf("UnmarshalGraphQL() error = %v", err)
				return
			}

			if *gt.Value != *tt.wantEq.Value {
				t.Errorf("UnmarshalGraphQL() got = %v, want = %v", *gt.Value, *tt.wantEq.Value)
			}
		})
	}
}

func TestNullInt_ImplementsUnmarshaler(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Error(err)
		}
	}()

	// assert *NullInt implements decode.Unmarshaler interface
	var _ decode.Unmarshaler = (*graphql.NullInt)(nil)
}

func TestNullInt_UnmarshalGraphQL(t *testing.T) {
	type args struct {
		input interface{}
	}

	a := float64(math.MaxInt32 + 1)
	b := float64(math.MinInt32 - 1)
	c := 1234.6
	good := int32(1234)
	ref := graphql.NullInt{
		Value: &good,
		Set:   true,
	}

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name    string
			args    args
			wantErr string
		}{
			{
				name:    "boolean",
				args:    args{input: true},
				wantErr: "wrong type for Int: bool",
			},
			{
				name: "int32 out of range (+)",
				args: args{
					input: a,
				},
				wantErr: "not a 32-bit integer",
			},
			{
				name: "int32 out of range (-)",
				args: args{
					input: b,
				},
				wantErr: "not a 32-bit integer",
			},
			{
				name: "non-integer",
				args: args{
					input: c,
				},
				wantErr: "not a 32-bit integer",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				gt := &graphql.NullInt{}
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
		wantEq graphql.NullInt
	}{
		{
			name: "int32",
			args: args{
				input: good,
			},
			wantEq: ref,
		},
		{
			name: "float64",
			args: args{
				input: float64(good),
			},
			wantEq: ref,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gt := new(graphql.NullInt)
			if err := gt.UnmarshalGraphQL(tt.args.input); err != nil {
				t.Errorf("UnmarshalGraphQL() error = %v", err)
				return
			}

			if *gt.Value != *tt.wantEq.Value {
				t.Errorf("UnmarshalGraphQL() got = %v, want = %v", *gt.Value, *tt.wantEq.Value)
			}
		})
	}
}

func TestNullFloat_ImplementsUnmarshaler(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Error(err)
		}
	}()

	// assert *NullFloat implements decode.Unmarshaler interface
	var _ decode.Unmarshaler = (*graphql.NullFloat)(nil)
}

func TestNullFloat_UnmarshalGraphQL(t *testing.T) {
	type args struct {
		input interface{}
	}

	good := float64(1234)
	ref := graphql.NullFloat{
		Value: &good,
		Set:   true,
	}

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name    string
			args    args
			wantErr string
		}{
			{
				name:    "boolean",
				args:    args{input: true},
				wantErr: "wrong type for Float: bool",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				gt := new(graphql.NullFloat)
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
		wantEq graphql.NullFloat
	}{
		{
			name: "int",
			args: args{
				input: int(good),
			},
			wantEq: ref,
		},
		{
			name: "int32",
			args: args{
				input: int32(good),
			},
			wantEq: ref,
		},
		{
			name: "float64",
			args: args{
				input: good,
			},
			wantEq: ref,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gt := new(graphql.NullFloat)
			if err := gt.UnmarshalGraphQL(tt.args.input); err != nil {
				t.Errorf("UnmarshalGraphQL() error = %v", err)
				return
			}

			if *gt.Value != *tt.wantEq.Value {
				t.Errorf("UnmarshalGraphQL() got = %v, want = %v", *gt.Value, *tt.wantEq.Value)
			}
		})
	}
}
