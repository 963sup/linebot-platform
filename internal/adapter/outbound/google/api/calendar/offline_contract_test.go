package calendar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	googlecalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func TestAllCalendarSDKOperationsUsePinnedRoutes(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Method + " " + r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete || r.Method == http.MethodPost && r.URL.Path == "/calendar/v3/channels/stop" {
			// A generated API method may return a structured or empty result.
			_, _ = w.Write([]byte("{}"))
			return
		}
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	client, err := googlecalendar.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL+"/calendar/v3/"))
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := New(client)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		operation string
		want      string
		body      json.RawMessage
		params    map[string]json.RawMessage
	}{
		{operation: "acl.delete", want: "DELETE /calendar/v3/calendars/test/acl/test", body: nil, params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`), "ruleId": json.RawMessage(`"test"`)}},
		{operation: "acl.get", want: "GET /calendar/v3/calendars/test/acl/test", body: nil, params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`), "ruleId": json.RawMessage(`"test"`)}},
		{operation: "acl.insert", want: "POST /calendar/v3/calendars/test/acl", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "acl.list", want: "GET /calendar/v3/calendars/test/acl", body: nil, params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "acl.patch", want: "PATCH /calendar/v3/calendars/test/acl/test", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`), "ruleId": json.RawMessage(`"test"`)}},
		{operation: "acl.update", want: "PUT /calendar/v3/calendars/test/acl/test", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`), "ruleId": json.RawMessage(`"test"`)}},
		{operation: "acl.watch", want: "POST /calendar/v3/calendars/test/acl/watch", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "calendarList.delete", want: "DELETE /calendar/v3/users/me/calendarList/test", body: nil, params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "calendarList.get", want: "GET /calendar/v3/users/me/calendarList/test", body: nil, params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "calendarList.insert", want: "POST /calendar/v3/users/me/calendarList", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{}},
		{operation: "calendarList.list", want: "GET /calendar/v3/users/me/calendarList", body: nil, params: map[string]json.RawMessage{}},
		{operation: "calendarList.patch", want: "PATCH /calendar/v3/users/me/calendarList/test", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "calendarList.update", want: "PUT /calendar/v3/users/me/calendarList/test", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "calendarList.watch", want: "POST /calendar/v3/users/me/calendarList/watch", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{}},
		{operation: "calendars.clear", want: "POST /calendar/v3/calendars/test/clear", body: nil, params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "calendars.delete", want: "DELETE /calendar/v3/calendars/test", body: nil, params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "calendars.get", want: "GET /calendar/v3/calendars/test", body: nil, params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "calendars.insert", want: "POST /calendar/v3/calendars", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{}},
		{operation: "calendars.patch", want: "PATCH /calendar/v3/calendars/test", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "calendars.transferOwnership", want: "POST /calendar/v3/calendars/test/transferOwnership", body: nil, params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`), "newDataOwner": json.RawMessage(`"test"`), "useAdminAccess": json.RawMessage(`true`)}},
		{operation: "calendars.update", want: "PUT /calendar/v3/calendars/test", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "channels.stop", want: "POST /calendar/v3/channels/stop", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{}},
		{operation: "colors.get", want: "GET /calendar/v3/colors", body: nil, params: map[string]json.RawMessage{}},
		{operation: "events.delete", want: "DELETE /calendar/v3/calendars/test/events/test", body: nil, params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`), "eventId": json.RawMessage(`"test"`)}},
		{operation: "events.get", want: "GET /calendar/v3/calendars/test/events/test", body: nil, params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`), "eventId": json.RawMessage(`"test"`)}},
		{operation: "events.import", want: "POST /calendar/v3/calendars/test/events/import", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "events.insert", want: "POST /calendar/v3/calendars/test/events", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "events.instances", want: "GET /calendar/v3/calendars/test/events/test/instances", body: nil, params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`), "eventId": json.RawMessage(`"test"`)}},
		{operation: "events.list", want: "GET /calendar/v3/calendars/test/events", body: nil, params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "events.move", want: "POST /calendar/v3/calendars/test/events/test/move", body: nil, params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`), "eventId": json.RawMessage(`"test"`), "destination": json.RawMessage(`"test"`)}},
		{operation: "events.patch", want: "PATCH /calendar/v3/calendars/test/events/test", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`), "eventId": json.RawMessage(`"test"`)}},
		{operation: "events.quickAdd", want: "POST /calendar/v3/calendars/test/events/quickAdd", body: nil, params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`), "text": json.RawMessage(`"test"`)}},
		{operation: "events.update", want: "PUT /calendar/v3/calendars/test/events/test", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`), "eventId": json.RawMessage(`"test"`)}},
		{operation: "events.watch", want: "POST /calendar/v3/calendars/test/events/watch", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{"calendarId": json.RawMessage(`"test"`)}},
		{operation: "freebusy.query", want: "POST /calendar/v3/freeBusy", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{}},
		{operation: "settings.get", want: "GET /calendar/v3/users/me/settings/test", body: nil, params: map[string]json.RawMessage{"setting": json.RawMessage(`"test"`)}},
		{operation: "settings.list", want: "GET /calendar/v3/users/me/settings", body: nil, params: map[string]json.RawMessage{}},
		{operation: "settings.watch", want: "POST /calendar/v3/users/me/settings/watch", body: json.RawMessage(`{}`), params: map[string]json.RawMessage{}},
	}
	for _, tc := range cases {
		t.Run(tc.operation, func(t *testing.T) {
			got = ""
			req := outbound.WorkspaceRequest{
				Operation: tc.operation, Parameters: tc.params, Body: tc.body,
			}
			_, err := adapter.ExecuteCalendar(context.Background(), req)
			if err != nil {
				t.Fatalf("SDK request: %v", err)
			}
			if got != tc.want {
				t.Fatalf("SDK route=%q, want %q", got, tc.want)
			}
		})
	}
}
