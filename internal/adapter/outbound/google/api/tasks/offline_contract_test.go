package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/codec"
	workspace "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
	"google.golang.org/api/option"
	googletasks "google.golang.org/api/tasks/v1"
)

func testCommand(t *testing.T, request codec.Request) workspace.OperationCommand {
	t.Helper()
	command, err := codec.Command(request)
	if err != nil {
		t.Fatal(err)
	}
	return command
}

func TestAllTasksSDKOperationsUsePinnedRoutes(t *testing.T) {
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
	client, err := googletasks.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL+"/"))
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
		params    map[string]any
	}{
		{operation: "tasklists.delete", want: "DELETE /tasks/v1/users/@me/lists/test", body: nil, params: map[string]any{"tasklist": json.RawMessage(`"test"`)}},
		{operation: "tasklists.get", want: "GET /tasks/v1/users/@me/lists/test", body: nil, params: map[string]any{"tasklist": json.RawMessage(`"test"`)}},
		{operation: "tasklists.insert", want: "POST /tasks/v1/users/@me/lists", body: json.RawMessage(`{}`), params: map[string]any{}},
		{operation: "tasklists.list", want: "GET /tasks/v1/users/@me/lists", body: nil, params: map[string]any{}},
		{operation: "tasklists.patch", want: "PATCH /tasks/v1/users/@me/lists/test", body: json.RawMessage(`{}`), params: map[string]any{"tasklist": json.RawMessage(`"test"`)}},
		{operation: "tasklists.update", want: "PUT /tasks/v1/users/@me/lists/test", body: json.RawMessage(`{}`), params: map[string]any{"tasklist": json.RawMessage(`"test"`)}},
		{operation: "tasks.clear", want: "POST /tasks/v1/lists/test/clear", body: nil, params: map[string]any{"tasklist": json.RawMessage(`"test"`)}},
		{operation: "tasks.delete", want: "DELETE /tasks/v1/lists/test/tasks/test", body: nil, params: map[string]any{"tasklist": json.RawMessage(`"test"`), "task": json.RawMessage(`"test"`)}},
		{operation: "tasks.get", want: "GET /tasks/v1/lists/test/tasks/test", body: nil, params: map[string]any{"tasklist": json.RawMessage(`"test"`), "task": json.RawMessage(`"test"`)}},
		{operation: "tasks.insert", want: "POST /tasks/v1/lists/test/tasks", body: json.RawMessage(`{}`), params: map[string]any{"tasklist": json.RawMessage(`"test"`)}},
		{operation: "tasks.list", want: "GET /tasks/v1/lists/test/tasks", body: nil, params: map[string]any{"tasklist": json.RawMessage(`"test"`)}},
		{operation: "tasks.move", want: "POST /tasks/v1/lists/test/tasks/test/move", body: nil, params: map[string]any{"tasklist": json.RawMessage(`"test"`), "task": json.RawMessage(`"test"`)}},
		{operation: "tasks.patch", want: "PATCH /tasks/v1/lists/test/tasks/test", body: json.RawMessage(`{}`), params: map[string]any{"tasklist": json.RawMessage(`"test"`), "task": json.RawMessage(`"test"`)}},
		{operation: "tasks.update", want: "PUT /tasks/v1/lists/test/tasks/test", body: json.RawMessage(`{}`), params: map[string]any{"tasklist": json.RawMessage(`"test"`), "task": json.RawMessage(`"test"`)}},
	}
	for _, tc := range cases {
		t.Run(tc.operation, func(t *testing.T) {
			got = ""
			req := testCommand(t, codec.Request{
				Operation: tc.operation, Parameters: tc.params, Body: tc.body,
			})
			_, err := adapter.ExecuteTasks(context.Background(), req)
			if err != nil {
				t.Fatalf("SDK request: %v", err)
			}
			if got != tc.want {
				t.Fatalf("SDK route=%q, want %q", got, tc.want)
			}
		})
	}
}
