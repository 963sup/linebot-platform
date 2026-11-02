package bootstrap

import (
	"log/slog"
	"net/http"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/http/health"
	linein "github.com/963sup/linebot-platform/internal/adapter/inbound/http/line"
	lineout "github.com/963sup/linebot-platform/internal/adapter/outbound/line"
	"github.com/963sup/linebot-platform/internal/application/usecases/conversation"
	"github.com/963sup/linebot-platform/internal/application/usecases/messaging"
	lineinfra "github.com/963sup/linebot-platform/internal/infrastructure/line"
)

func NewWebhookRouter(cfg Config, logger *slog.Logger) (http.Handler, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.Handler)
	if !cfg.LiveLINE() {
		mux.HandleFunc("/webhook/line", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "LINE integration is not configured", http.StatusServiceUnavailable)
		})
		return mux, nil
	}
	client, err := lineinfra.NewMessagingClient(cfg.ChannelToken)
	if err != nil {
		return nil, err
	}
	sender, err := lineout.NewSender(client)
	if err != nil {
		return nil, err
	}
	service, err := messaging.NewService(sender)
	if err != nil {
		return nil, err
	}
	responder, err := conversation.NewResponder(service)
	if err != nil {
		return nil, err
	}
	mux.Handle("/webhook/line", linein.NewHandler(cfg.ChannelSecret, responder, logger))
	return mux, nil
}
