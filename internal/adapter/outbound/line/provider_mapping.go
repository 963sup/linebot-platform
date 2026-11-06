package line

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

func providerObject(value any) (map[string]any, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, errors.New("cannot encode LINE response")
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return nil, errors.New("invalid LINE response")
	}
	if object == nil {
		return nil, errors.New("invalid LINE response")
	}
	return object, nil
}

// decodeLINERequest remains local to the adapter while legacy-shaped methods
// are migrated in this package. It deliberately does not expose the shape to
// application contracts.
func decodeLINERequest(request any, target any) error {
	payload, err := json.Marshal(request)
	if err != nil {
		return errors.New("invalid LINE request")
	}
	var wrapper struct {
		Value json.RawMessage `json:"Value"`
	}
	if err := json.Unmarshal(payload, &wrapper); err == nil && len(wrapper.Value) > 0 {
		payload = wrapper.Value
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return errors.New("invalid LINE request")
	}
	return nil
}

func stringValue(object map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := object[key].(string); ok {
			return value
		}
	}
	return ""
}

func int64Value(object map[string]any, keys ...string) int64 {
	for _, key := range keys {
		switch value := object[key].(type) {
		case json.Number:
			result, _ := value.Int64()
			return result
		case float64:
			return int64(value)
		case string:
			result, _ := strconv.ParseInt(value, 10, 64)
			return result
		}
	}
	return 0
}

func boolValue(object map[string]any, keys ...string) bool {
	for _, key := range keys {
		if value, ok := object[key].(bool); ok {
			return value
		}
	}
	return false
}

func timeValue(object map[string]any, keys ...string) time.Time {
	for _, key := range keys {
		switch value := object[key].(type) {
		case string:
			for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05.000Z", "20060102"} {
				if result, err := time.Parse(layout, value); err == nil {
					return result
				}
			}
		case json.Number:
			if milliseconds, err := value.Int64(); err == nil {
				return time.UnixMilli(milliseconds)
			}
		}
	}
	return time.Time{}
}
