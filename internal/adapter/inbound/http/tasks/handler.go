// Package tasks serves authenticated Cloud Tasks and scheduled dispatcher
// callbacks. It never accepts user-controlled raw LINE payloads.
package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	ingestion "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
)

type Verifier interface {
	Verify(context.Context, string) error
}

// TrustedTraceStarter starts task-server telemetry only after the caller's
// Google-signed identity has been verified. Implementations must treat name as
// a bounded operation identifier, never as user input.
type TrustedTraceStarter interface {
	StartTrustedTaskSpan(context.Context, http.Header, string) (context.Context, func())
}

type noOpTraceStarter struct{}

func (noOpTraceStarter) StartTrustedTaskSpan(ctx context.Context, _ http.Header, _ string) (context.Context, func()) {
	return ctx, func() {}
}

type Handler struct {
	processor  *ingestion.Processor
	dispatcher *ingestion.Dispatcher
	verifier   Verifier
	logger     *slog.Logger
	tracing    TrustedTraceStarter
}

func NewHandler(processor *ingestion.Processor, dispatcher *ingestion.Dispatcher, verifier Verifier, logger *slog.Logger) (http.Handler, error) {
	return NewHandlerWithTelemetry(processor, dispatcher, verifier, noOpTraceStarter{}, logger)
}

func NewHandlerWithTelemetry(processor *ingestion.Processor, dispatcher *ingestion.Dispatcher, verifier Verifier, tracing TrustedTraceStarter, logger *slog.Logger) (http.Handler, error) {
	if processor == nil || dispatcher == nil || verifier == nil || tracing == nil {
		return nil, errors.New("worker processor, dispatcher, OIDC verifier and trusted telemetry are required")
	}
	h := &Handler{
		processor: processor, dispatcher: dispatcher, verifier: verifier, logger: logger,
		tracing: tracing,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks/v2/line-reply", h.handleReply)
	mux.HandleFunc("POST /tasks/v2/dispatch", h.handleDispatch)
	// Legacy routes are drain-only compatibility for tasks/schedules created
	// before the V2 cutover. New configuration and task creation require V2.
	mux.HandleFunc("POST /tasks/line-reply", h.handleReply)
	mux.HandleFunc("POST /tasks/dispatch", h.handleDispatch)
	return mux, nil
}

func (h *Handler) authorized(w http.ResponseWriter, r *http.Request, operation string) (context.Context, func(), bool) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, nil, false
	}
	if err := h.verifier.Verify(r.Context(), parts[1]); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, nil, false
	}
	// Trace context is accepted only after the Google-signed OIDC identity is
	// verified. Invalid callers cannot choose parentage in trusted traces.
	ctx, end := h.tracing.StartTrustedTaskSpan(r.Context(), r.Header, operation)
	return ctx, end, true
}

func (h *Handler) handleReply(w http.ResponseWriter, r *http.Request) {
	ctx, end, ok := h.authorized(w, r, "line_reply_task")
	if !ok {
		return
	}
	defer end()
	r = r.WithContext(ctx)
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		http.Error(w, "application/json required", http.StatusUnsupportedMediaType)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	decode := decodeDeliveryTask
	if r.URL.Path == "/tasks/line-reply" {
		decode = decodeLegacyDeliveryTask
	}
	deliveryID, err := decode(r.Body)
	if err != nil {
		http.Error(w, "invalid task", http.StatusBadRequest)
		return
	}
	key, err := ingestion.ParseEventKey(deliveryID)
	if err != nil {
		http.Error(w, "invalid task key", http.StatusBadRequest)
		return
	}
	if err := h.processor.Process(r.Context(), key); err != nil {
		if errors.Is(err, ingestion.ErrDeliveryInFlight) {
			w.Header().Set("Retry-After", "2")
		}
		if h.logger != nil {
			h.logger.Error("durable task state update failed", "operation", "reply")
		}
		http.Error(w, "temporary worker error", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeLegacyDeliveryTask(reader io.Reader) (string, error) {
	decoder := json.NewDecoder(reader)
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return "", errors.New("task must be an object")
	}
	seen := false
	var deliveryID string
	for decoder.More() {
		nameToken, err := decoder.Token()
		if err != nil {
			return "", err
		}
		name, ok := nameToken.(string)
		if !ok || name != "key" || seen {
			return "", errors.New("duplicate or unknown legacy task field")
		}
		seen = true
		if err := decoder.Decode(&deliveryID); err != nil {
			return "", err
		}
	}
	if _, err := decoder.Token(); err != nil {
		return "", err
	}
	if token, err := decoder.Token(); err != io.EOF || token != nil || !seen || deliveryID == "" {
		return "", errors.New("invalid legacy task")
	}
	return deliveryID, nil
}

func decodeDeliveryTask(reader io.Reader) (string, error) {
	decoder := json.NewDecoder(reader)
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return "", errors.New("task must be an object")
	}
	seen := map[string]bool{}
	var version int
	var deliveryID string
	for decoder.More() {
		nameToken, err := decoder.Token()
		if err != nil {
			return "", err
		}
		name, ok := nameToken.(string)
		if !ok || seen[name] {
			return "", errors.New("duplicate or invalid task field")
		}
		seen[name] = true
		switch name {
		case "schema_version":
			err = decoder.Decode(&version)
		case "delivery_id":
			err = decoder.Decode(&deliveryID)
		default:
			return "", errors.New("unknown task field")
		}
		if err != nil {
			return "", err
		}
	}
	if _, err := decoder.Token(); err != nil {
		return "", err
	}
	if token, err := decoder.Token(); err != io.EOF || token != nil {
		return "", errors.New("trailing task data")
	}
	if version != 2 || deliveryID == "" {
		return "", errors.New("invalid task version or delivery ID")
	}
	return deliveryID, nil
}

func (h *Handler) handleDispatch(w http.ResponseWriter, r *http.Request) {
	ctx, end, ok := h.authorized(w, r, "outbox_dispatch")
	if !ok {
		return
	}
	defer end()
	r = r.WithContext(ctx)
	if err := h.dispatcher.Dispatch(r.Context()); err != nil {
		if h.logger != nil {
			h.logger.Error("outbox dispatch failed", "operation", "dispatch")
		}
		http.Error(w, "temporary dispatcher error", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
