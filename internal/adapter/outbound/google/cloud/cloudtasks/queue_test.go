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
	ingestion "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	queueinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/cloudtasks"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
		return nil, errors.New("ambiguous network error with sensitive endpoint")
	}
	if err := q.Enqueue(context.Background(), key); !errors.Is(err, ErrQueueUnavailable) || strings.Contains(err.Error(), "sensitive endpoint") {
		t.Fatalf("unsafe or missing queue classification: %v", err)
	}
}

func TestEnqueuePreservesContextErrors(t *testing.T) {
	key, _ := ingestion.ParseEventKey(testKey)
	q := &Queue{
		name: testQueue, worker: "https://worker.example.test/tasks/v2/line-reply",
		audience: "https://worker.example.test", email: "tasks-invoker@example.iam.gserviceaccount.com",
		now: time.Now,
		create: func(context.Context, *cloudtaskspb.CreateTaskRequest) (*cloudtaskspb.Task, error) {
			return nil, status.Error(codes.DeadlineExceeded, "sensitive provider deadline")
		},
	}
	if err := q.Enqueue(context.Background(), key); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline classification = %v", err)
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

func TestEnqueuePropagatesOnlyTraceContext(t *testing.T) {
	key, err := ingestion.ParseEventKey(testKey)
	if err != nil {
		t.Fatal(err)
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	ctx, span := tp.Tracer("test").Start(context.Background(), "enqueue")
	defer span.End()
	q := &Queue{
		name: testQueue, worker: "https://worker.example.test/tasks/v2/line-reply",
		audience: "https://worker.example.test", email: "tasks-invoker@example.iam.gserviceaccount.com",
		propagator: propagation.TraceContext{},
		create: func(_ context.Context, req *cloudtaskspb.CreateTaskRequest) (*cloudtaskspb.Task, error) {
			headers := req.Task.GetHttpRequest().Headers
			if headers["traceparent"] == "" {
				t.Fatal("missing W3C traceparent")
			}
			if len(headers) != 2 || headers["Content-Type"] != "application/json" {
				t.Fatalf("unexpected propagated headers: %v", headers)
			}
			return &cloudtaskspb.Task{}, nil
		},
	}
	if err := q.Enqueue(ctx, key); err != nil {
		t.Fatal(err)
	}
}
