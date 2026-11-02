package cloudtasks

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	testQueue = "projects/test-project/locations/asia-east1/queues/line-queue"
	testKey = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

func TestQueueConfiguration(t *testing.T) {
	for _, n := range []string{"", "projects/x", "projects/x/locations/y/queues/"} {
		if validQueueName(n) {
			t.Fatal("accepted invalid Cloud Tasks queue")
		}
	}
	if !validQueueName(testQueue) {
		t.Fatal("expected valid queue")
	}
}

func TestEnqueueStableIdAndOIDC(t *testing.T) {
	var calls int
	q := &Queue{
		name: testQueue,
		worker: "https://worker.example.test/tasks/line-reply",
		audience: "https://worker.example.test",
		email: "tasks-invoker@example.iam.gserviceaccount.com",
		create: func(_ context.Context, req *cloudtaskspb.CreateTaskRequest) (*cloudtaskspb.Task, error) {
			calls++
			if req.Parent != testQueue || req.Task.Name != testQueue+"/tasks/"+testKey {
				t.Fatal("incorrect deterministic task name")
			}
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
		},
	}
	if err := q.Enqueue(context.Background(), testKey); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("expected exactly one CreateTask")
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
