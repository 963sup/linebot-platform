// Package tasks serves authenticated Cloud Tasks and scheduled dispatcher
// callbacks. It never accepts user-controlled raw LINE payloads.
package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/usecases/ingestion"
)

type Verifier interface {
	Verify(context.Context, string) error
}

type Handler struct {
	processor  *ingestion.Processor
	dispatcher *ingestion.Dispatcher
	verifier   Verifier
	logger     *slog.Logger
}

func NewHandler(processor *ingestion.Processor, dispatcher *ingestion.Dispatcher, verifier Verifier, logger *slog.Logger) (http.Handler, error) {
	if processor == nil || dispatcher == nil || verifier == nil {
		return nil, errors.New("worker processor, dispatcher and OIDC verifier are required")
	}
	h := &Handler{processor: processor, dispatcher: dispatcher, verifier: verifier, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks/line-reply", h.handleReply)
	mux.HandleFunc("POST /tasks/dispatch", h.handleDispatch)
	return mux, nil
}

func (h *Handler) authorized(w http.ResponseWriter, r *http.Request) bool {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	if err := h.verifier.Verify(r.Context(), parts[1]); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func (h *Handler) handleReply(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var request struct {
		Key string `json:"key"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid task", http.StatusBadRequest)
		return
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(w, "invalid task", http.StatusBadRequest)
		return
	}
	key, err := ingestion.ParseEventKey(request.Key)
	if err != nil {
		http.Error(w, "invalid task key", http.StatusBadRequest)
		return
	}
	if err := h.processor.Process(r.Context(), key); err != nil {
		if h.logger != nil {
			h.logger.Error("durable task state update failed", "operation", "reply")
		}
		http.Error(w, "temporary worker error", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleDispatch(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(w, r) {
		return
	}
	if err := h.dispatcher.Dispatch(r.Context()); err != nil {
		if h.logger != nil {
			h.logger.Error("outbox dispatch failed", "operation", "dispatch")
		}
		http.Error(w, "temporary dispatcher error", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
