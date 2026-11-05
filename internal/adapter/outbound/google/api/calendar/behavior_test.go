package calendar

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	googlecalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func TestCalendarEventsPreserveOptionsAndSanitizeErrors(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/calendar/v3/calendars/primary/events" {
			t.Errorf("unexpected route %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("sendUpdates") != "none" || r.URL.Query().Get("conferenceDataVersion") != "1" {
			t.Error("Calendar event notification and conference options lost")
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if !strings.Contains(string(raw), `"summary":"event"`) {
			t.Errorf("event JSON body missing summary")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"event1","summary":"event"}`))
	}))
	defer server.Close()
	svc, err := googlecalendar.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL+"/calendar/v3/"))
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := New(svc)
	if err != nil {
		t.Fatal(err)
	}
	req := testCommand(t, codec.Request{
		Operation:  "events.insert",
		Parameters: map[string]any{"calendarId": json.RawMessage(`"primary"`)},
		Options: map[string]any{
			"sendUpdates":           json.RawMessage(`"none"`),
			"conferenceDataVersion": json.RawMessage(`1`),
		},
		Body: json.RawMessage(`{"summary":"event","start":{"date":"2026-11-05"},"end":{"date":"2026-11-06"}}`),
	})
	result, err := adapter.ExecuteCalendar(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(result.Payload) || !strings.Contains(string(result.Payload), `"event1"`) {
		t.Fatalf("response lost: %v", result.Payload)
	}
	if calls.Load() != 1 {
		t.Fatalf("called provider %d times", calls.Load())
	}
	invalid := req
	invalid = testCommand(t, codec.Request{Operation: "events.insert", Parameters: map[string]any{"calendarId": json.RawMessage(`"primary"`)}, Options: map[string]any{"unknownOption": json.RawMessage(`true`)}, Body: json.RawMessage(`{}`)})
	if _, err := adapter.ExecuteCalendar(context.Background(), invalid); err == nil {
		t.Fatal("unrecognized query option accepted")
	}
	invalid = testCommand(t, codec.Request{Operation: "events.insert", Parameters: map[string]any{"calendarId": json.RawMessage(`false`)}, Body: json.RawMessage(`{}`)})
	if _, err := adapter.ExecuteCalendar(context.Background(), invalid); err == nil {
		t.Fatal("invalid calendar identifier accepted")
	}
	if calls.Load() != 1 {
		t.Fatal("invalid request reached Google")
	}
}

func TestCalendarProviderFailureDoesNotExposeResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"private-schedule-title","code":403}}`))
	}))
	defer server.Close()
	svc, err := googlecalendar.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL+"/calendar/v3/"))
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := New(svc)
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.ExecuteCalendar(context.Background(), testCommand(t, codec.Request{
		Operation: "events.list", Parameters: map[string]any{"calendarId": json.RawMessage(`"primary"`)},
	}))
	var external *codec.RemoteError
	if !errors.As(err, &external) || external.Status != 403 {
		t.Fatalf("expected 403 classification, got %v", err)
	}
	if strings.Contains(err.Error(), "private-schedule-title") {
		t.Fatal("provider response body leaked")
	}
}
