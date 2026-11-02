// Package cloudtasks provides stable-id Cloud Tasks dispatch to a private
// OIDC-authenticated Cloud Run worker.
package cloudtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	sdk "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/963sup/linebot-platform/internal/application/usecases/ingestion"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type createTask func(context.Context, *cloudtaskspb.CreateTaskRequest) (*cloudtaskspb.Task, error)

type Queue struct {
	name     string
	worker   string
	audience string
	email    string
	create   createTask
}

// NewQueue configures a native Cloud Tasks client. Only a previously
// authorized OIDC service account may invoke the private worker; IAM must
// separately grant Cloud Tasks task creation and Cloud Run invoker roles.
func NewQueue(client *sdk.Client, queueName, workerURL, audience, invokerEmail string) (*Queue, error) {
	if client == nil {
		return nil, errors.New("Cloud Tasks client is required")
	}
	if !validQueueName(queueName) {
		return nil, errors.New("invalid Cloud Tasks queue name")
	}
	u, err := url.Parse(workerURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil ||
		u.RawQuery != "" || u.Fragment != "" || u.Path != "/tasks/line-reply" {
		return nil, errors.New("worker URL must be HTTPS and end with /tasks/line-reply")
	}
	aud, err := url.Parse(audience)
	if err != nil || aud.Scheme != "https" || aud.Host == "" || aud.User != nil ||
		aud.RawQuery != "" || aud.Fragment != "" || aud.Path != "" && aud.Path != "/" {
		return nil, errors.New("OIDC audience must be an HTTPS service origin")
	}
	if strings.TrimSpace(invokerEmail) == "" {
		return nil, errors.New("OIDC invoker service account email is required")
	}
	return &Queue{
		name: queueName, worker: workerURL, audience: audience, email: invokerEmail,
		create: func(ctx context.Context, req *cloudtaskspb.CreateTaskRequest) (*cloudtaskspb.Task, error) {
			return client.CreateTask(ctx, req)
		},
	}, nil
}

func validQueueName(n string) bool {
	parts := strings.Split(n, "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "queues" {
		return false
	}
	for _, i := range []int{1, 3, 5} {
		if parts[i] == "" || parts[i] == "." || parts[i] == ".." {
			return false
		}
	}
	return true
}

// Enqueue includes only an opaque digest; no reply token, user ID or user text.
func (q *Queue) Enqueue(ctx context.Context, key string) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if !ingestion.ValidKey(key) {
		return errors.New("invalid task event key")
	}
	payload, err := json.Marshal(struct {
		Key string `json:"key"`
	}{Key: key})
	if err != nil {
		return err
	}
	task := &cloudtaskspb.Task{
		Name: fmt.Sprintf("%s/tasks/%s", q.name, key),
		MessageType: &cloudtaskspb.Task_HttpRequest{HttpRequest: &cloudtaskspb.HttpRequest{
			HttpMethod: cloudtaskspb.HttpMethod_POST,
			Url:        q.worker,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       payload,
			AuthorizationHeader: &cloudtaskspb.HttpRequest_OidcToken{
				OidcToken: &cloudtaskspb.OidcToken{
					ServiceAccountEmail: q.email,
					Audience:            q.audience,
				},
			},
		}},
	}
	_, err = q.create(ctx, &cloudtaskspb.CreateTaskRequest{Parent: q.name, Task: task})
	if status.Code(err) == codes.AlreadyExists {
		return nil // deterministic ID: LINE redelivery / scheduled reconcile
	}
	return err // ambiguous RPC results are retried via durable outbox
}
