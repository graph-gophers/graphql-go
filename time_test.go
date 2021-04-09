package graphql

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var refTime = time.Date(2020, 4, 9, 8, 19, 58, 651387237, time.UTC)

var timeTestCases = []interface{}{
	refTime,
	int32(refTime.Unix()),
	refTime.Unix(),
	refTime.UnixNano(),
	float64(refTime.Unix()),
	refTime.Format(time.RFC3339),
}

func TestImplementsGraphQLType(t *testing.T) {
	time := Time{}
	assert.Equal(t, time.ImplementsGraphQLType("Time"), true)
}

func TestUnmarshalGraphQL(t *testing.T) {
	var err error
	testTime := &Time{}

	for _, timeType := range timeTestCases {
		assert.NoError(t, testTime.UnmarshalGraphQL(timeType), "Time type: %T", timeType)
		assert.Equal(t, refTime.Unix(), testTime.Unix(), "Time type: %T", timeType)
	}

	err = testTime.UnmarshalGraphQL(false)
	assert.EqualError(t, err, "wrong type for Time: bool")
}

func TestMarshalJSON(t *testing.T) {
	testTime := &Time{refTime}

	buf, err := testTime.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("\"%s\"", refTime.Format(time.RFC3339Nano)), string(buf))
}
