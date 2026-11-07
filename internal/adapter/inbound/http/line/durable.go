package line

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// DurableHandler acknowledges accepted events only after the outbox is
// committed and enqueueing was attempted successfully. No user content or
// LINE reply token is included in Cloud Tasks requests or structured logs.
type DurableHandler struct {
	secret      string
	destination string
	ingestor    *messagingdelivery.Ingestor
	policy      messagingdelivery.ReplyPolicy
	logger      *slog.Logger
}

func NewDurableHandler(secret, destination string, ingestor *messagingdelivery.Ingestor, policy messagingdelivery.ReplyPolicy, logger *slog.Logger) http.Handler {
	return &DurableHandler{secret: secret, destination: destination, ingestor: ingestor, policy: policy, logger: logger}
}

func (h *DurableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.secret == "" || h.destination == "" || h.ingestor == nil {
		http.Error(w, "durable LINE integration is not configured", http.StatusServiceUnavailable)
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
			if h.logger != nil {
				h.logger.Warn("LINE webhook rejected", "reason", "invalid_signature")
			}
			http.Error(w, "invalid signature", http.StatusUnauthorized)
		default:
			http.Error(w, "invalid webhook", http.StatusBadRequest)
		}
		return
	}
	if parsed.Destination != h.destination {
		if h.logger != nil {
			h.logger.Warn("LINE webhook rejected", "reason", "destination_mismatch")
		}
		http.Error(w, "destination mismatch", http.StatusUnauthorized)
		return
	}
	accepted, ignoredStandby, ignoredUnsupported, stale := 0, 0, 0, 0
	for _, event := range parsed.Events {
		e, ok := event.(webhook.MessageEvent)
		if !ok || e.ReplyToken == "" {
			ignoredUnsupported++
			continue
		}
		if e.Mode == webhook.EventMode_STANDBY {
			ignoredStandby++
			continue
		}
		if e.Mode != webhook.EventMode_ACTIVE || e.DeliveryContext == nil {
			http.Error(w, "invalid webhook event envelope", http.StatusBadRequest)
			return
		}
		m, ok := e.Message.(webhook.TextMessageContent)
		if !ok {
			ignoredUnsupported++
			continue
		}
		if e.WebhookEventId == "" {
			http.Error(w, "missing webhook event ID", http.StatusBadRequest)
			return
		}
		reply := h.policy.ResponseForText(m.Text)
		if err := h.ingestor.Ingest(r.Context(), h.destination, e.WebhookEventId, e.ReplyToken, time.UnixMilli(e.Timestamp), reply); err != nil {
			if errors.Is(err, messagingdelivery.ErrEventTooOld) {
				// ACK stale redelivery instead of inviting endless retries.
				stale++
				continue
			}
			if h.logger != nil {
				h.logger.Error("durable LINE event persistence or dispatch failed", "operation", "ingest")
			}
			// LINE redelivery and the scheduled dispatcher repair a committed
			// outbox when task scheduling fails. Existing work remains deduped.
			http.Error(w, "event temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		accepted++
	}
	if h.logger != nil {
		h.logger.Info("LINE webhook processed",
			"accepted", accepted,
			"ignored_standby", ignoredStandby,
			"ignored_unsupported", ignoredUnsupported,
			"stale", stale)
	}
	w.WriteHeader(http.StatusOK)
}
