package graphql

import (
	"testing"
	"time"
)

var timeTestCases = []interface{}{
	time.Now(),
	int32(1),
	time.Now().Unix(),
	time.Now().UnixNano(),
	float64(-1),
	"2006-01-02T15:04:05Z",
}

func TestImplementsGraphQLType(t *testing.T) {
	time := Time{}
	if !time.ImplementsGraphQLType("Time") {
		t.Fail()
	}
}

func TestUnmarshalGraphQL(t *testing.T) {
	var err error
	testTime := &Time{}

	for _, timeType := range timeTestCases {
		if err = testTime.UnmarshalGraphQL(timeType); err != nil {
			t.Fatalf("Failed to unmarshal %#v to Time: %s", timeType, err.Error())
		}
	}

	if err = testTime.UnmarshalGraphQL(false); err == nil {
		t.Fatalf("Unmarshaling of %T to Time should be failed.", false)
	}
}

func TestMarshalJSON(t *testing.T) {
	exampleTime := "\"0001-01-01T00:00:00Z\""
	testTime := &Time{}

	buf, err := testTime.MarshalJSON()
	if err != nil {
		t.Fatalf("Failed to marshal time to JSON: %s", err.Error())
	}
	if string(buf) != exampleTime {
		t.Fatalf("Failed to marshal Time to JSON, expected %s, but instead got %s", exampleTime, buf)
	}
}
