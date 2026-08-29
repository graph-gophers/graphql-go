package graphql_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/graph-gophers/graphql-go"
)

// TestTypeResolverFor_TimeResolution tests basic type resolver for time.Time
func TestTypeResolverFor_TimeResolution(t *testing.T) {
	schema := graphql.MustParseSchema(`
		scalar DateTime

		type Query {
			now: DateTime!
		}
	`, &struct {
		Now time.Time
	}{
		Now: time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC),
	}, graphql.UseFieldResolvers(),
		graphql.TypeResolverFor(func(t time.Time) (any, error) {
			return t.Format(time.RFC3339), nil
		}),
	)

	result := schema.Exec(context.Background(), `{ now }`, "", nil)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	var data struct {
		Now string `json:"now"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	expected := "2026-01-02T15:04:05Z"
	if data.Now != expected {
		t.Errorf("expected %q, got %q", expected, data.Now)
	}
}

// TestTypeResolverFor_MultipleResolvers tests registering multiple type resolvers
func TestTypeResolverFor_MultipleResolvers(t *testing.T) {
	schema := graphql.MustParseSchema(`
		scalar DateTime
		scalar UserID

		type Query {
			id: UserID!
			time: DateTime!
		}
	`, &struct {
		ID   uint
		Time time.Time
	}{
		ID:   123,
		Time: time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC),
	}, graphql.UseFieldResolvers(),
		graphql.TypeResolverFor(func(t time.Time) (any, error) {
			return t.Format(time.RFC3339), nil
		}),
		graphql.TypeResolverFor(func(id uint) (any, error) {
			return strconv.FormatUint(uint64(id), 10), nil
		}),
	)

	result := schema.Exec(context.Background(), `{ id time }`, "", nil)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	var data struct {
		ID   string `json:"id"`
		Time string `json:"time"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if data.ID != "123" {
		t.Errorf("expected id %q, got %q", "123", data.ID)
	}
	if data.Time != "2026-01-02T15:04:05Z" {
		t.Errorf("expected time %q, got %q", "2026-01-02T15:04:05Z", data.Time)
	}
}

// TestTypeResolverFor_BackwardCompatibility ensures existing behavior unchanged
func TestTypeResolverFor_BackwardCompatibility(t *testing.T) {
	schema := graphql.MustParseSchema(`
		type Query {
			value: Int!
		}
	`, &struct {
		Value int32
	}{
		Value: 42,
	}, graphql.UseFieldResolvers(),
	)

	result := schema.Exec(context.Background(), `{ value }`, "", nil)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	var data struct {
		Value int `json:"value"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if data.Value != 42 {
		t.Errorf("expected 42, got %d", data.Value)
	}
}

// TestTypeResolverFor_PointerHandling tests pointer fields with type resolvers
func TestTypeResolverFor_PointerHandling(t *testing.T) {
	birthdate := time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)

	schema := graphql.MustParseSchema(`
		scalar DateTime

		type Query {
			birthdate: DateTime
		}
	`, &struct {
		Birthdate *time.Time
	}{
		Birthdate: &birthdate,
	}, graphql.UseFieldResolvers(),
		graphql.TypeResolverFor(func(t time.Time) (any, error) {
			return t.Format(time.RFC3339), nil
		}),
	)

	result := schema.Exec(context.Background(), `{ birthdate }`, "", nil)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	var data struct {
		Birthdate *string `json:"birthdate"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if data.Birthdate == nil || *data.Birthdate != "2026-01-02T15:04:05Z" {
		t.Errorf("expected time value, got %v", data.Birthdate)
	}
}

// TestTypeResolverFor_NilPointer tests nil pointer handling
func TestTypeResolverFor_NilPointer(t *testing.T) {
	schema := graphql.MustParseSchema(`
		scalar DateTime

		type Query {
			birthdate: DateTime
		}
	`, &struct {
		Birthdate *time.Time
	}{
		Birthdate: nil,
	}, graphql.UseFieldResolvers(),
		graphql.TypeResolverFor(func(t time.Time) (any, error) {
			return t.Format(time.RFC3339), nil
		}),
	)

	result := schema.Exec(context.Background(), `{ birthdate }`, "", nil)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	var data struct {
		Birthdate *string `json:"birthdate"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if data.Birthdate != nil {
		t.Errorf("expected nil, got %v", data.Birthdate)
	}
}

// TestTypeResolverFor_SchemaLocal tests that resolvers are schema-local
func TestTypeResolverFor_SchemaLocal(t *testing.T) {
	time1 := time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)

	schema1 := graphql.MustParseSchema(`
		scalar DateTime
		type Query {
			time: DateTime!
		}
	`, &struct {
		Time time.Time
	}{
		Time: time1,
	}, graphql.UseFieldResolvers(),
		graphql.TypeResolverFor(func(t time.Time) (any, error) {
			return t.Format("2006-01-02"), nil
		}),
	)

	schema2 := graphql.MustParseSchema(`
		scalar DateTime
		type Query {
			time: DateTime!
		}
	`, &struct {
		Time time.Time
	}{
		Time: time1,
	}, graphql.UseFieldResolvers(),
		graphql.TypeResolverFor(func(t time.Time) (any, error) {
			return t.Format(time.RFC3339), nil
		}),
	)

	result1 := schema1.Exec(context.Background(), `{ time }`, "", nil)
	result2 := schema2.Exec(context.Background(), `{ time }`, "", nil)

	if len(result1.Errors) > 0 || len(result2.Errors) > 0 {
		t.Fatalf("unexpected errors")
	}

	var data1, data2 struct {
		Time string `json:"time"`
	}
	json.Unmarshal(result1.Data, &data1)
	json.Unmarshal(result2.Data, &data2)

	if data1.Time != "2026-01-02" {
		t.Errorf("schema1: expected date format, got %q", data1.Time)
	}
	if data2.Time != "2026-01-02T15:04:05Z" {
		t.Errorf("schema2: expected RFC3339 format, got %q", data2.Time)
	}
}

// TestTypeResolverFor_ErrorPropagation tests error handling
func TestTypeResolverFor_ErrorPropagation(t *testing.T) {
	schema := graphql.MustParseSchema(`
		scalar Custom
		type Query {
			value: Custom!
		}
	`, &struct {
		Value uint
	}{
		Value: 123,
	}, graphql.UseFieldResolvers(),
		graphql.TypeResolverFor(func(id uint) (any, error) {
			return nil, fmt.Errorf("custom error")
		}),
	)

	result := schema.Exec(context.Background(), `{ value }`, "", nil)
	if len(result.Errors) == 0 {
		t.Fatalf("expected errors")
	}

	if !strings.Contains(result.Errors[0].Message, "custom error") {
		t.Errorf("error message mismatch: %s", result.Errors[0].Message)
	}
}

// TestTypeResolverFor_ConcurrentExecution tests thread safety
func TestTypeResolverFor_ConcurrentExecution(t *testing.T) {
	schema := graphql.MustParseSchema(`
		scalar Custom
		type Query {
			value: Custom!
		}
	`, &struct {
		Value uint
	}{
		Value: 123,
	}, graphql.UseFieldResolvers(),
		graphql.TypeResolverFor(func(id uint) (any, error) {
			return strconv.FormatUint(uint64(id), 10), nil
		}),
	)

	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := schema.Exec(context.Background(), `{ value }`, "", nil)
			if len(result.Errors) > 0 {
				errChan <- fmt.Errorf("query failed: %v", result.Errors)
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Error(err)
	}
}
