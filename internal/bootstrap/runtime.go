package bootstrap

import (
	"errors"
	"log/slog"
	"net/http"

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
	Handler  http.Handler
	outbound *httpclient.Managed
	closers  []func() error
}

// NewWebhookRuntime constructs only the clients needed by the webhook process.
func NewWebhookRuntime(cfg Config, logger *slog.Logger) (*WebhookRuntime, error) {
	if cfg.EventMode == "durable" && cfg.Secrets.ChannelToken != "" {
		return nil, errors.New("durable webhook cannot contain LINE send token")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.Handler)
	runtime := &WebhookRuntime{Handler: observability.WrapInboundHTTP("webhook", logger, mux)}
	if !cfg.LiveLINE() {
		mux.HandleFunc("/webhook/line", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "LINE integration is not configured", http.StatusServiceUnavailable)
		})
		return runtime, nil
	}
	if cfg.EventMode == "durable" {
		if err := configureDurableWebhook(cfg, logger, mux, runtime); err != nil {
			runtime.Close()
			return nil, err
		}
		return runtime, nil
	}

	timeout := cfg.LINEAPITimeout
	if timeout == 0 {
		timeout = httpclient.DefaultTimeout
	}
	outboundHTTP, err := httpclient.New(httpclient.Config{Timeout: timeout},
		observability.WrapHTTP("line", logger))
	if err != nil {
		return nil, err
	}
	// The factory accepts a managed native HTTP client; its SDK client does not
	// own the transport. The composition root closes it after HTTP shutdown.
	client, err := lineinfra.NewMessagingClientWithHTTPClient(cfg.Secrets.ChannelToken, outboundHTTP.Client)
	if err != nil {
		outboundHTTP.Close()
		return nil, err
	}
	sender, err := lineout.NewSender(client)
	if err != nil {
		outboundHTTP.Close()
		return nil, err
	}
	service, err := messagingdelivery.NewDeliveryService(sender)
	if err != nil {
		outboundHTTP.Close()
		return nil, err
	}
	mux.Handle("/webhook/line", linein.NewHandler(cfg.Secrets.ChannelSecret, service, logger))
	runtime.outbound = outboundHTTP
	return runtime, nil
}

// Close releases idle SDK HTTP connections after the inbound server has
// drained its requests. It is safe to call more than once.
func (r *WebhookRuntime) Close() {
	if r == nil {
		return
	}
	r.outbound.Close()
	for i := len(r.closers) - 1; i >= 0; i-- {
		_ = r.closers[i]()
	}
}
