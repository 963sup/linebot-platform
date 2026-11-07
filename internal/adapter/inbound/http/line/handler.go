package line

import (
	"errors"
	"log/slog"
	"net/http"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

const maxBodyBytes = 1024 * 1024

// Handler parses and verifies LINE webhook requests before invoking application logic.
// This synchronous MVP has no durable deduplication; do not deploy as a production bot.
type Handler struct {
	secret   string
	delivery *messagingdelivery.DeliveryService
	policy   messagingdelivery.ReplyPolicy
	logger   *slog.Logger
}

func NewHandler(secret string, delivery *messagingdelivery.DeliveryService, policy messagingdelivery.ReplyPolicy, logger *slog.Logger) *Handler {
	return &Handler{secret: secret, delivery: delivery, policy: policy, logger: logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.secret == "" || h.delivery == nil {
		http.Error(w, "LINE integration is not configured", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	parsed, err := webhook.ParseRequest(h.secret, r)
	if err != nil {
		var tooLarge *http.MaxBytesError
		switch {
		case errors.As(err, &tooLarge):
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		case errors.Is(err, webhook.ErrInvalidSignature):
			http.Error(w, "invalid LINE signature", http.StatusUnauthorized)
		default:
			http.Error(w, "invalid LINE webhook", http.StatusBadRequest)
		}
		return
	}
	failed := false
	for _, event := range parsed.Events {
		e, ok := event.(webhook.MessageEvent)
		if !ok || e.ReplyToken == "" {
			continue
		}
		message, ok := e.Message.(webhook.TextMessageContent)
		if !ok {
			continue
		}
		if err := h.delivery.Reply(r.Context(), e.ReplyToken, []messagingdelivery.Message{h.policy.ResponseForText(message.Text)}); err != nil {
			// Do not log request bodies, reply tokens, user IDs or SDK error messages.
			if h.logger != nil {
				h.logger.Error("LINE reply failed", "operation", "reply")
			}
			failed = true
		}
	}
	if failed {
		http.Error(w, "LINE processing failed", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusOK)
}
