package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestWorkspaceFactoriesRequireAuthorizedTransport(t *testing.T) {
	if _, err := NewTasks(context.Background(), nil); err == nil {
		t.Fatal("Google Tasks must require a caller-owned authorized HTTP client")
	}
	if _, err := NewCalendar(context.Background(), nil); err == nil {
		t.Fatal("Google Calendar must require a caller-owned authorized HTTP client")
	}
	if _, err := NewDrive(context.Background(), nil); err == nil {
		t.Fatal("Google Drive must require a caller-owned authorized HTTP client")
	}
	if _, err := NewSheets(context.Background(), nil); err == nil {
		t.Fatal("Google Sheets must require a caller-owned authorized HTTP client")
	}
	if _, err := NewTasks(nil, &http.Client{}); err == nil {
		t.Fatal("nil context must fail")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewCalendar(ctx, &http.Client{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancelled context, got %v", err)
	}
}

func TestTasksAndCalendarNativeAPIAreCallableOffline(t *testing.T) {
	// This HTTP client simulates an already-authorized user token. Real code
	// must obtain it from a secure OAuth flow and enforce the narrowest scopes.
	var gotTasks, gotCalendar atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fake-oauth-user-token" {
			t.Errorf("missing simulated OAuth bearer token")
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/users/@me/lists"):
			gotTasks.Store(true)
			_, _ = w.Write([]byte(`{"items":[{"id":"list-1","title":"Personal"}]}`))
		case strings.Contains(r.URL.Path, "/calendars/primary/events"):
			gotCalendar.Store(true)
			_, _ = w.Write([]byte(`{"items":[{"id":"event-1","summary":"Event"}]}`))
		default:
			t.Errorf("unexpected Google API request path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	base := server.Client()
	client := &http.Client{Transport: bearerTransport{token: "fake-oauth-user-token", next: base.Transport}}
	tasksSvc, err := NewTasks(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	tasksSvc.BasePath = server.URL + "/"
	tasklists, err := tasksSvc.Tasklists.List().Do()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasklists.Items) != 1 || tasklists.Items[0].Title != "Personal" {
		t.Fatalf("unexpected tasks response: %#v", tasklists.Items)
	}

	calSvc, err := NewCalendar(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	calSvc.BasePath = server.URL + "/"
	events, err := calSvc.Events.List("primary").Do()
	if err != nil {
		t.Fatal(err)
	}
	if len(events.Items) != 1 || events.Items[0].Summary != "Event" {
		t.Fatalf("unexpected calendar events: %#v", events.Items)
	}
	if !gotTasks.Load() || !gotCalendar.Load() {
		t.Fatal("both Google APIs must be exercised")
	}

	sheetsSvc, err := NewSheets(context.Background(), client)
	if err != nil || sheetsSvc.Spreadsheets == nil {
		t.Fatalf("sheets client unavailable: %v", err)
	}
	driveSvc, err := NewDrive(context.Background(), client)
	if err != nil || driveSvc.Files == nil {
		t.Fatalf("drive client unavailable: %v", err)
	}
}

type bearerTransport struct {
	token string
	next  http.RoundTripper
}

func (b bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	copy := req.Clone(req.Context())
	copy.Header.Set("Authorization", "Bearer "+b.token)
	return b.next.RoundTrip(copy)
}
