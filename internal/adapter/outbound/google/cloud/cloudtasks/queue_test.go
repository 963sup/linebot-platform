package cloudtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	queueinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/cloudtasks"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	testQueue = "projects/test-project/locations/asia-east1/queues/line-queue"
	testKey   = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

func TestQueueConfiguration(t *testing.T) {
	for _, n := range []string{"", "projects/x", "projects/x/locations/y/queues/"} {
		if _, ok := queueinfra.QueueProjectID(n); ok {
			t.Fatal("accepted invalid Cloud Tasks queue")
		}
	}
	if project, ok := queueinfra.QueueProjectID(testQueue); !ok || project != "test-project" {
		t.Fatal("expected valid queue")
	}
}

func TestEnqueueStableIdAndOIDC(t *testing.T) {
	var calls int
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var lastTaskName string
	q := &Queue{
		name:     testQueue,
		worker:   "https://worker.example.test/tasks/line-reply",
		audience: "https://worker.example.test",
		email:    "tasks-invoker@example.iam.gserviceaccount.com",
		now:      func() time.Time { return clock },
	}
	q.create = func(_ context.Context, req *cloudtaskspb.CreateTaskRequest) (*cloudtaskspb.Task, error) {
		calls++
		want := fmt.Sprintf("%s/tasks/%s-%d", testQueue, testKey, clock.Unix()/int64(taskDedupeWindow/time.Second))
		if req.Parent != testQueue || req.Task.Name != want {
			t.Fatalf("incorrect per-window task name: got %s, want %s", req.Task.Name, want)
		}
		if calls > 1 && lastTaskName != "" && calls == 2 && req.Task.Name != lastTaskName {
			t.Fatal("two dispatches in the same time window must deduplicate")
		}
		lastTaskName = req.Task.Name
		request := req.Task.GetHttpRequest()
		if request.GetHttpMethod() != cloudtaskspb.HttpMethod_POST || request.GetUrl() != "https://worker.example.test/tasks/line-reply" {
			t.Fatal("worker HTTP request mismatch")
		}
		if request.GetOidcToken().GetServiceAccountEmail() != q.email || request.GetOidcToken().GetAudience() != q.audience {
			t.Fatal("OIDC identity mismatch")
		}
		var body map[string]string
		if err := json.Unmarshal(request.Body, &body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 1 || body["key"] != testKey || strings.Contains(string(request.Body), "reply") {
			t.Fatal("task body must contain only the opaque key")
		}
		return nil, status.Error(codes.AlreadyExists, "duplicate")
	}
	if err := q.Enqueue(context.Background(), testKey); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(context.Background(), testKey); err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(11 * time.Second)
	if err := q.Enqueue(context.Background(), testKey); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 CreateTask calls including the new retry window, got %d", calls)
	}
	if err := q.Enqueue(context.Background(), "bad"); err == nil {
		t.Fatal("malformed task key must be rejected")
	}
	q.create = func(context.Context, *cloudtaskspb.CreateTaskRequest) (*cloudtaskspb.Task, error) {
		return nil, errors.New("ambiguous network error")
	}
	if err := q.Enqueue(context.Background(), testKey); err == nil {
		t.Fatal("must surface unknown task enqueue result")
	}
}
