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
	linein "github.com/963sup/linebot-platform/internal/adapter/inbound/http/line"
	lineout "github.com/963sup/linebot-platform/internal/adapter/outbound/line"
	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
	lineinfra "github.com/963sup/linebot-platform/internal/infrastructure/line"
	"github.com/963sup/linebot-platform/internal/infrastructure/observability"
)

// WebhookRuntime is the composition root for an HTTP process. It owns exactly
// the resources it constructs. No optional provider is initialized by default.
type WebhookRuntime struct {
	Handler   http.Handler
	outbound  *httpclient.Managed
	closers   []runtimeCloser
	ready     atomic.Bool
	closeOnce sync.Once
	closeErr  error
}

func closeRuntimeClosers(closers []runtimeCloser) error {
	var closeErr error
	for i := len(closers) - 1; i >= 0; i-- {
		if closers[i].close == nil {
			continue
		}
		if err := closers[i].close(); err != nil {
			closeErr = errors.Join(closeErr, resourceCloseError{resource: closers[i].name, cause: err})
		}
	}
	return closeErr
}

// NewWebhookRuntime constructs only the clients needed by the webhook process.
func NewWebhookRuntime(cfg Config, logger *slog.Logger) (*WebhookRuntime, error) {
	if cfg.EventMode == "durable" && cfg.Secrets.MessagingAPIChannelAccessToken != "" {
		return nil, errors.New("durable webhook cannot contain LINE send token")
	}
	replyPolicy, err := messagingdelivery.NewReplyPolicy(cfg.MiniAppEntryURL)
	if err != nil {
		return nil, err
	}
	startup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	telemetry, err := newTelemetryRuntime(startup, cfg, logger, "linebot-webhook")
	cancel()
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.Handler)
	runtime := &WebhookRuntime{
		Handler: telemetry.WrapInboundHTTP("webhook", mux),
		closers: []runtimeCloser{{name: "telemetry", close: telemetryCloser(telemetry)}},
	}
	mux.Handle("/readyz", health.ReadyHandler(func(context.Context) error {
		if !runtime.ready.Load() {
			return errors.New("webhook runtime is not ready")
		}
		return nil
	}))
	if !cfg.LiveLINE() {
		mux.HandleFunc("/webhook/line", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "LINE integration is not configured", http.StatusServiceUnavailable)
		})
		runtime.ready.Store(true)
		return runtime, nil
	}
	if cfg.EventMode == "durable" {
		if err := configureDurableWebhook(cfg, logger, mux, runtime); err != nil {
			return nil, errors.Join(err, runtime.Close())
		}
		runtime.ready.Store(true)
		return runtime, nil
	}

	timeout := cfg.LINEAPITimeout
	if timeout == 0 {
		timeout = httpclient.DefaultTimeout
	}
	outboundHTTP, err := httpclient.New(httpclient.Config{Timeout: timeout}, telemetry.WrapHTTP("line"))
	if err != nil {
		return nil, errors.Join(err, runtime.Close())
	}
	runtime.outbound = outboundHTTP
	// The factory accepts a managed native HTTP client; its SDK client does not
	// own the transport. The composition root closes it after HTTP shutdown.
	client, err := lineinfra.NewMessagingClientWithHTTPClient(cfg.Secrets.MessagingAPIChannelAccessToken, outboundHTTP.Client)
	if err != nil {
		return nil, errors.Join(err, runtime.Close())
	}
	sender, err := lineout.NewSender(client)
	if err != nil {
		return nil, errors.Join(err, runtime.Close())
	}
	service, err := messagingdelivery.NewDeliveryService(sender)
	if err != nil {
		return nil, errors.Join(err, runtime.Close())
	}
	mux.Handle("/webhook/line", linein.NewHandler(cfg.Secrets.MessagingAPIChannelSecret, service, replyPolicy, logger))
	runtime.ready.Store(true)
	return runtime, nil
}

func newTelemetryRuntime(ctx context.Context, cfg Config, logger *slog.Logger, serviceName string) (*observability.Runtime, error) {
	return observability.NewRuntime(ctx, observability.Config{
		Mode: cfg.Telemetry.Mode, ServiceName: serviceName,
		Environment: string(cfg.Environment), TraceSampleRatio: cfg.Telemetry.TraceSampleRatio,
		MetricInterval: cfg.Telemetry.MetricInterval,
	}, logger)
}

func telemetryCloser(runtime *observability.Runtime) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return runtime.Shutdown(ctx)
	}
}

// Close releases idle SDK HTTP connections after the inbound server has
// drained its requests. It is safe to call more than once and returns the
// stable aggregate of any owned SDK close failures.
func (r *WebhookRuntime) Close() error {
	if r == nil {
		return nil
	}
	r.ready.Store(false)
	r.closeOnce.Do(func() {
		r.outbound.Close()
		r.closeErr = closeRuntimeClosers(r.closers)
	})
	return r.closeErr
}

type runtimeCloser struct {
	name  string
	close func() error
}

// resourceCloseError keeps shutdown logs actionable without rendering raw SDK
// error text. The underlying cause remains available to errors.Is/As.
type resourceCloseError struct {
	resource string
	cause    error
}

func (e resourceCloseError) Error() string { return "close " + e.resource + " failed" }
func (e resourceCloseError) Unwrap() error { return e.cause }
