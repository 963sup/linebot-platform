package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
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
	"github.com/963sup/linebot-platform/internal/infrastructure/observability"
)

// WorkerRuntime owns all three native outbound SDK resources used by the
// private Cloud Run worker. Close is called after the HTTP server has drained.
type WorkerRuntime struct {
	Handler  http.Handler
	outbound *httpclient.Managed
	closers  []func() error
}

func (r *WorkerRuntime) Close() {
	if r == nil {
		return
	}
	if r.outbound != nil {
		r.outbound.Close()
	}
	for i := len(r.closers) - 1; i >= 0; i-- {
		_ = r.closers[i]()
	}
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
	db, err := storeinfra.NewClient(ctx, cfg.Durable.ProjectID)
	if err != nil {
		return nil, err
	}
	queueClient, err := queueinfra.NewClient(ctx)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	rt := &WorkerRuntime{closers: []func() error{db.Close, queueClient.Close}}
	cleanup := func(err error) (*WorkerRuntime, error) {
		rt.Close()
		return nil, err
	}
	store, err := storeout.NewStore(db)
	if err != nil {
		return cleanup(err)
	}
	queue, err := queueout.NewQueue(queueClient, cfg.Durable.QueueName, cfg.Durable.WorkerURL,
		cfg.Durable.OIDCAudience, cfg.Durable.InvokerEmail)
	if err != nil {
		return cleanup(err)
	}
	out, err := httpclient.New(httpclient.Config{Timeout: cfg.LINEAPITimeout},
		observability.WrapHTTP("line", logger))
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
	workHandler, err := taskin.NewHandler(processor, dispatcher, verifier, logger)
	if err != nil {
		return cleanup(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.Handler)
	mux.Handle("/tasks/", workHandler)
	rt.Handler = observability.WrapInboundHTTP("worker", logger, mux)
	return rt, nil
}

// A distinct startup deadline prevents slow GCP credential discovery from
// indefinitely blocking the worker process.
func WorkerStartupContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}
