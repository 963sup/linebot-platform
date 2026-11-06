package line

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	insightsreporting "github.com/963sup/linebot-platform/internal/application/usecases/insights-reporting"
)

type malformedProviderPayload struct{}

func (malformedProviderPayload) MarshalJSON() ([]byte, error) {
	return []byte("{"), nil
}

func TestProviderObjectRejectsInvalidPayloads(t *testing.T) {
	var typedNilPointer *struct{}
	var typedNilMap map[string]any
	var typedNilSlice []string

	tests := []struct {
		name  string
		value any
	}{
		{name: "nil", value: nil},
		{name: "typed nil pointer", value: typedNilPointer},
		{name: "typed nil map", value: typedNilMap},
		{name: "typed nil slice", value: typedNilSlice},
		{name: "malformed JSON", value: malformedProviderPayload{}},
		{name: "scalar", value: "not an object"},
		{name: "array", value: []string{"not", "an", "object"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if object, err := providerObject(test.value); err == nil {
				t.Fatalf("providerObject() = %#v, nil; want error", object)
			}
		})
	}
}

func TestProviderObjectAcceptsObject(t *testing.T) {
	object, err := providerObject(map[string]any{"count": 2})
	if err != nil {
		t.Fatal(err)
	}
	if got := int64Value(object, "count"); got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
}

func TestInsightMappersPropagateProviderObjectErrors(t *testing.T) {
	value := malformedProviderPayload{}
	if _, err := demographicsReport(value); err == nil {
		t.Fatal("demographicsReport() error = nil")
	}
	if _, err := countReport(value, time.Time{}); err == nil {
		t.Fatal("countReport() error = nil")
	}
	if _, err := statisticsResult("statistics", nil, value, nil, time.Time{}); err == nil {
		t.Fatal("statisticsResult() error = nil")
	}
}

func TestNumericDimensionsAreSortedByKey(t *testing.T) {
	object := map[string]any{
		"zeta":  json.Number("3"),
		"alpha": float64(1),
		"text":  "excluded",
		"mu":    json.Number("2"),
	}
	want := []insightsreporting.DimensionValue{
		{Dimension: "alpha", Count: 1},
		{Dimension: "mu", Count: 2},
		{Dimension: "zeta", Count: 3},
	}
	if got := numericDimensions(object); !reflect.DeepEqual(got, want) {
		t.Fatalf("numericDimensions() = %#v, want %#v", got, want)
	}
}
