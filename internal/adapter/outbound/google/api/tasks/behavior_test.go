package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
	"google.golang.org/api/option"
	googletasks "google.golang.org/api/tasks/v1"
)

func TestTasksMoveParentAndListOptions(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/move") {
			if r.URL.Query().Get("parent") != "parent1" || r.URL.Query().Get("previous") != "sibling1" {
				t.Error("subtask move parameters lost")
			}
		}
		if r.Method == "GET" {
			if r.URL.Query().Get("showAssigned") != "true" || r.URL.Query().Get("maxResults") != "5" {
				t.Error("Tasks list filters lost")
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	svc, err := googletasks.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL+"/"))
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(svc)
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.ExecuteTasks(context.Background(), workspace.Command{
		Operation: "tasks.list", Parameters: map[string]any{"tasklist": json.RawMessage(`"list1"`)},
		Options: map[string]any{"showAssigned": json.RawMessage(`true`), "maxResults": json.RawMessage(`5`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.ExecuteTasks(context.Background(), workspace.Command{
		Operation: "tasks.move", Parameters: map[string]any{"tasklist": json.RawMessage(`"list1"`), "task": json.RawMessage(`"task1"`)},
		Options: map[string]any{"parent": json.RawMessage(`"parent1"`), "previous": json.RawMessage(`"sibling1"`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(methods) != 2 || methods[0] != "GET /tasks/v1/lists/list1/tasks" || methods[1] != "POST /tasks/v1/lists/list1/tasks/task1/move" {
		t.Fatalf("unexpected routes %v", methods)
	}
}
