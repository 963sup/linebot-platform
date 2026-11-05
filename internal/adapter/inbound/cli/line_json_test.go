package cli

import (
	"encoding/json"
	"testing"
)

func TestDecodeLINEJSONPreservesLargeInteger(t *testing.T) {
	value, err := decodeLINEJSON([]byte(`{"audienceGroupId":9007199254740993}`))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"audienceGroupId":9007199254740993}` {
		t.Fatalf("large integer lost precision: %s", encoded)
	}
	object := value.(map[string]any)
	if got := audienceNumber(object, "audienceGroupId"); got != 9007199254740993 {
		t.Fatalf("audienceNumber() = %d", got)
	}
}
