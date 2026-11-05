package cloudtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	ingestion "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
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
	key, err := ingestion.ParseEventKey(testKey)
	if err != nil {
		t.Fatal(err)
	}
	var calls int
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var lastTaskName string
	q := &Queue{
		name:     testQueue,
		worker:   "https://worker.example.test/tasks/v2/line-reply",
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
		if request.GetHttpMethod() != cloudtaskspb.HttpMethod_POST || request.GetUrl() != "https://worker.example.test/tasks/v2/line-reply" {
			t.Fatal("worker HTTP request mismatch")
		}
		if request.GetOidcToken().GetServiceAccountEmail() != q.email || request.GetOidcToken().GetAudience() != q.audience {
			t.Fatal("OIDC identity mismatch")
		}
		var body struct {
			SchemaVersion int    `json:"schema_version"`
			DeliveryID    string `json:"delivery_id"`
		}
		if err := json.Unmarshal(request.Body, &body); err != nil {
			t.Fatal(err)
		}
		if body.SchemaVersion != 2 || body.DeliveryID != testKey {
			t.Fatal("task body must contain only the versioned delivery identity")
		}
		return nil, status.Error(codes.AlreadyExists, "duplicate")
	}
	if err := q.Enqueue(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(11 * time.Second)
	if err := q.Enqueue(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 CreateTask calls including the new retry window, got %d", calls)
	}
	q.create = func(context.Context, *cloudtaskspb.CreateTaskRequest) (*cloudtaskspb.Task, error) {
		return nil, errors.New("ambiguous network error")
	}
	if err := q.Enqueue(context.Background(), key); err == nil {
		t.Fatal("must surface unknown task enqueue result")
	}
}

func TestEnqueueRejectsZeroEventKey(t *testing.T) {
	q := &Queue{create: func(context.Context, *cloudtaskspb.CreateTaskRequest) (*cloudtaskspb.Task, error) {
		t.Fatal("zero EventKey reached Cloud Tasks")
		return nil, nil
	}}
	if err := q.Enqueue(context.Background(), ingestion.EventKey{}); err == nil {
		t.Fatal("zero EventKey must be rejected")
	}
}
