package codec

import (
	"encoding/json"
	"testing"

	workspace "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
)

func TestDecodeTypedCommandAtAdapterBoundary(t *testing.T) {
	request, err := Decode(workspace.OperationCommand{
		Operation:  "events.insert",
		Parameters: []workspace.Parameter{{Name: "calendarId", Value: "primary"}},
		Options: []workspace.Option{
			{Name: "sendNotifications", Values: []string{"true"}},
			{Name: "privateExtendedProperty", Values: []string{"owner=one", "team=two"}},
		},
		Payload: []byte(`{"summary":"event"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	calendarID, err := Required[string](request.Parameters, "calendarId")
	if err != nil || calendarID != "primary" {
		t.Fatalf("parameter mapping failed: %q, %v", calendarID, err)
	}
	notify, err := Optional[bool](request.Options, "sendNotifications")
	if err != nil || !notify {
		t.Fatalf("scalar option mapping failed: %t, %v", notify, err)
	}
	properties, err := Optional[[]string](request.Options, "privateExtendedProperty")
	if err != nil || len(properties) != 2 || properties[1] != "team=two" {
		t.Fatalf("repeated option mapping failed: %#v, %v", properties, err)
	}
	body, err := Body[map[string]string](request.Body)
	if err != nil || (*body)["summary"] != "event" {
		t.Fatalf("body mapping failed: %#v, %v", body, err)
	}
}

func TestDecodeRejectsInvalidPayloadAndEncodeReturnsJSON(t *testing.T) {
	if _, err := Decode(workspace.OperationCommand{Operation: "forms.create", Payload: []byte(`{`)}); err == nil {
		t.Fatal("invalid JSON payload accepted")
	}
	result, err := Encode(map[string]string{"id": "event-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(result.Payload) || string(result.Payload) != `{"id":"event-1"}` {
		t.Fatalf("provider result is not canonical JSON: %q", result.Payload)
	}
}
