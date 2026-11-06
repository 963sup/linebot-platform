package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/http/health"
	taskin "github.com/963sup/linebot-platform/internal/adapter/inbound/http/tasks"
	queueout "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/cloudtasks"
	storeout "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	lineout "github.com/963sup/linebot-platform/internal/adapter/outbound/line"
	ingestion "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	queueinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/cloudtasks"
	storeinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/firestore"
	"github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/tasksauth"
	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
	lineinfra "github.com/963sup/linebot-platform/internal/infrastructure/line"
)

const (
	workerMaxConcurrentRequests = 16
	workerDrainTimeout          = 10 * time.Second
)

// WorkerRuntime owns all three native outbound SDK resources used by the
// private Cloud Run worker. Close is called after the HTTP server has drained.
type WorkerRuntime struct {
	Handler   http.Handler
	outbound  *httpclient.Managed
	limiter   *taskin.ConcurrencyLimiter
	closers   []runtimeCloser
	ready     atomic.Bool
	closeOnce sync.Once
	closeErr  error
}

func (r *WorkerRuntime) Close() error {
	if r == nil {
		return nil
	}
	r.ready.Store(false)
	if r.limiter != nil {
		ctx, cancel := context.WithTimeout(context.Background(), workerDrainTimeout)
		err := r.limiter.Shutdown(ctx)
		cancel()
		if err != nil {
			return resourceCloseError{resource: "worker request drain", cause: err}
		}
	}
	r.closeOnce.Do(func() {
		r.outbound.Close()
		r.closeErr = closeRuntimeClosers(r.closers)
	})
	return r.closeErr
}

// NewWorkerRuntime validates OIDC identity in its handler. Cloud Run must
// additionally enforce IAM-based authentication and private ingress.
func NewWorkerRuntime(ctx context.Context, cfg WorkerConfig, logger *slog.Logger) (*WorkerRuntime, error) {
	if ctx == nil {
		return nil, errors.New("worker context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	verifier, err := tasksauth.NewVerifier(cfg.Durable.OIDCAudience, cfg.Durable.InvokerEmail)
	if err != nil {
		return nil, err
	}
	telemetry, err := newTelemetryRuntime(ctx, cfg, logger, "linebot-worker")
	if err != nil {
		return nil, err
	}
	db, err := storeinfra.NewClient(ctx, cfg.Durable.ProjectID)
	if err != nil {
		_ = telemetryCloser(telemetry)()
		return nil, err
	}
	rt := &WorkerRuntime{closers: []runtimeCloser{
		{name: "telemetry", close: telemetryCloser(telemetry)},
		{name: "Firestore client", close: db.Close},
	}}
	cleanup := func(err error) (*WorkerRuntime, error) {
		return nil, errors.Join(err, rt.Close())
	}
	queueClient, err := queueinfra.NewClient(ctx)
	if err != nil {
		return cleanup(err)
	}
	rt.closers = append(rt.closers, runtimeCloser{name: "Cloud Tasks client", close: queueClient.Close})
	store, err := storeout.NewStore(db)
	if err != nil {
		return cleanup(err)
	}
	queue, err := queueout.NewQueue(queueClient, cfg.Durable.QueueName, cfg.Durable.WorkerURL,
		cfg.Durable.OIDCAudience, cfg.Durable.InvokerEmail)
	if err != nil {
		return cleanup(err)
	}
	out, err := httpclient.New(httpclient.Config{Timeout: cfg.LINEAPITimeout}, telemetry.WrapHTTP("line"))
	if err != nil {
		return cleanup(err)
	}
	rt.outbound = out
	client, err := lineinfra.NewMessagingClientWithHTTPClient(cfg.Secrets.ChannelToken, out.Client)
	if err != nil {
		return cleanup(err)
	}
	sender, err := lineout.NewSender(client)
	if err != nil {
		return cleanup(err)
	}
	processor, err := ingestion.NewProcessor(store, sender)
	if err != nil {
		return cleanup(err)
	}
	dispatcher, err := ingestion.NewDispatcher(store, queue)
	if err != nil {
		return cleanup(err)
	}
	workHandler, err := taskin.NewHandlerWithTelemetry(processor, dispatcher, verifier, telemetry, logger)
	if err != nil {
		return cleanup(err)
	}
	limiter, err := taskin.NewConcurrencyLimiter(workerMaxConcurrentRequests)
	if err != nil {
		return cleanup(err)
	}
	rt.limiter = limiter
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.Handler)
	mux.Handle("/readyz", health.ReadyHandler(func(context.Context) error {
		if !rt.ready.Load() {
			return errors.New("worker runtime is not ready")
		}
		return nil
	}))
	mux.Handle("/tasks/", limiter.Wrap(workHandler))
	rt.Handler = telemetry.WrapInboundHTTP("worker", mux)
	rt.ready.Store(true)
	return rt, nil
}

// A distinct startup deadline prevents slow GCP credential discovery from
// indefinitely blocking the worker process.
func WorkerStartupContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}
