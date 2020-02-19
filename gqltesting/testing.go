package gqltesting

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/nsf/jsondiff"

	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/errors"
)

// Test is a GraphQL test case to be used with RunTest(s).
type Test struct {
	Context        context.Context
	Schema         *graphql.Schema
	Query          string
	OperationName  string
	Variables      map[string]interface{}
	ExpectedResult string
	ExpectedErrors []*errors.QueryError
}

// RunTests runs the given GraphQL test cases as subtests.
func RunTests(t *testing.T, tests []*Test) {
	t.Helper()
	if len(tests) == 1 {
		RunTest(t, tests[0])
		return
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			t.Helper()
			RunTest(t, test)
		})
	}
}

// RunTest runs a single GraphQL test case.
func RunTest(t *testing.T, test *Test) {
	t.Helper()
	if test.Context == nil {
		test.Context = context.Background()
	}
	result := test.Schema.Exec(test.Context, test.Query, test.OperationName, test.Variables)

	checkErrors(t, test.ExpectedErrors, result.Errors)

	if test.ExpectedResult == "" {
		if result.Data != nil {
			t.Fatalf("got: %s", result.Data)
			t.Fatalf("want: null")
		}
		return
	}

	// opts := jsondiff.DefaultConsoleOptions()
	opts := jsondiff.Options{
		Added:   jsondiff.Tag{Begin: "+++", End: "+++"},
		Removed: jsondiff.Tag{Begin: "---", End: "---"},
		Changed: jsondiff.Tag{Begin: "|||", End: "|||"},
		Indent:  "    ",
	}
	diff, output := jsondiff.Compare([]byte(test.ExpectedResult), result.Data, &opts)
	if diff != jsondiff.FullMatch {
		t.Log("Did not get expected result:\n", output)
		t.Log("Got:", string(result.Data))
		t.Fail()
	}
}

func formatJSON(data []byte) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	formatted, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return formatted, nil
}

func checkErrors(t *testing.T, want, got []*errors.QueryError) {
	t.Helper()
	sortErrors(want)
	sortErrors(got)

	if !reflect.DeepEqual(got, want) {
		t.Log("unexpected error:")
		t.Log("  Got: \n", formatErrors(got))
		t.Log("  Want: \n", formatErrors(want))
		t.Fatal()
	}
}

func formatErrors(errs []*errors.QueryError) string {
	var errorStr string
	for _, err := range errs {
		if err == nil {
			errorStr = errorStr + "(nil)\n"
		} else {
			errorStr = errorStr + formatError(*err)
		}
	}
	return errorStr
}

func formatError(err errors.QueryError) string {
	return fmt.Sprintf(
		`%s
Path: %v
Rule: %s
Resolver: %s
Extensions: %+v
`,
		err.Error(),
		err.Path,
		err.Rule,
		err.ResolverError,
		err.Extensions)
}

func sortErrors(errors []*errors.QueryError) {
	if len(errors) <= 1 {
		return
	}
	sort.Slice(errors, func(i, j int) bool {
		return fmt.Sprintf("%s", errors[i].Path) < fmt.Sprintf("%s", errors[j].Path)
	})
}
