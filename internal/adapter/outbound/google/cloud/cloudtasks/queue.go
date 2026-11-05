// Package cloudtasks provides bounded task-name deduplication to a private
// OIDC-authenticated Cloud Run worker.
package cloudtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	sdk "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	ingestion "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	queueinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/cloudtasks"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type createTask func(context.Context, *cloudtaskspb.CreateTaskRequest) (*cloudtaskspb.Task, error)

var ErrQueueUnavailable = errors.New("Cloud Tasks queue unavailable")

// Cloud Tasks reserves completed/deleted task names for up to 24 hours.
// Use a new deterministic name per time window, not one permanent name for an
// event. The Firestore single-use Claim remains the actual send idempotency gate.
const taskDedupeWindow = 10 * time.Second

type Queue struct {
	name       string
	worker     string
	audience   string
	email      string
	create     createTask
	now        func() time.Time
	propagator propagation.TextMapPropagator
}

var _ ingestion.EventQueue = (*Queue)(nil)

// NewQueue configures a native Cloud Tasks client. Only a previously
// authorized OIDC service account may invoke the private worker; IAM must
// separately grant Cloud Tasks task creation and Cloud Run invoker roles.
func NewQueue(client *sdk.Client, queueName, workerURL, audience, invokerEmail string) (*Queue, error) {
	if client == nil {
		return nil, errors.New("Cloud Tasks client is required")
	}
	if _, ok := queueinfra.QueueProjectID(queueName); !ok {
		return nil, errors.New("invalid Cloud Tasks queue name")
	}
	u, err := url.Parse(workerURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil ||
		u.RawQuery != "" || u.Fragment != "" || u.Path != "/tasks/v2/line-reply" {
		return nil, errors.New("worker URL must be HTTPS and end with /tasks/v2/line-reply")
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
		now: time.Now, propagator: propagation.TraceContext{},
	}, nil
}

// Enqueue includes only an opaque digest; no reply token, user ID or user text.
func (q *Queue) Enqueue(ctx context.Context, key ingestion.EventKey) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !key.Valid() {
		return errors.New("invalid task event key")
	}
	value := key.String()
	payload, err := json.Marshal(struct {
		SchemaVersion int    `json:"schema_version"`
		DeliveryID    string `json:"delivery_id"`
	}{SchemaVersion: 2, DeliveryID: value})
	if err != nil {
		return err
	}
	clock := time.Now
	if q.now != nil {
		clock = q.now
	}
	bucket := clock().UTC().Unix() / int64(taskDedupeWindow/time.Second)
	task := &cloudtaskspb.Task{
		Name: fmt.Sprintf("%s/tasks/%s-%d", q.name, value, bucket),
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
	propagator := q.propagator
	if propagator == nil {
		propagator = propagation.TraceContext{}
	}
	propagator.Inject(ctx, taskHeaderCarrier(task.GetHttpRequest().Headers))
	_, err = q.create(ctx, &cloudtaskspb.CreateTaskRequest{Parent: q.name, Task: task})
	if status.Code(err) == codes.AlreadyExists {
		return nil // duplicate within the same scheduling window
	}
	return classifyQueueError(err) // durable outbox may offer the work again; this call is never blindly retried here
}

func classifyQueueError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded {
		return context.DeadlineExceeded
	}
	return fmt.Errorf("%w: create task", ErrQueueUnavailable)
}

// taskHeaderCarrier deliberately exposes only Cloud Tasks' string header map;
// baggage is not configured, so no application or user data is propagated.
type taskHeaderCarrier map[string]string

func (c taskHeaderCarrier) Get(key string) string { return c[key] }
func (c taskHeaderCarrier) Set(key, value string) { c[key] = value }
func (c taskHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	return keys
}
