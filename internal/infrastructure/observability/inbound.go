package observability

import (
	"log/slog"
	"net/http"
	"time"
)

// inboundWriter records HTTP status without inspecting or modifying bodies.
type inboundWriter struct {
	http.ResponseWriter
	status int
}

// Unwrap preserves optional ResponseWriter capabilities for
// http.NewResponseController (Flush, Hijack, and deadlines).
func (w *inboundWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// FlushError records the implicit HTTP 200 committed by a successful flush.
// Otherwise a later WriteHeader could be logged as the final status despite
// the underlying writer having already sent 200. Delegate to ResponseController
// to preserve support detection and ErrNotSupported for non-flushing writers.
func (w *inboundWriter) FlushError() error {
	if err := http.NewResponseController(w.ResponseWriter).Flush(); err != nil {
		return err
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return nil
}

func (w *inboundWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	// HTTP may send several informational 1xx responses before its final
	// status. Do not suppress the final 401/403/5xx response or report 103
	// as the terminal outcome. 101 switches protocols and is final.
	if status >= 100 && status < 200 && status != http.StatusSwitchingProtocols {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *inboundWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}

// WrapInboundHTTP records bounded, low-cardinality metadata. Unknown paths
// are logged as "other"; headers, URLs, query, LINE IDs and bodies are omitted.
func WrapInboundHTTP(role string, logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &inboundWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)
		if writer.status == 0 {
			writer.status = http.StatusOK
		}
		route := "other"
		switch r.URL.Path {
		case "/healthz":
			route = "health"
		case "/webhook/line":
			route = "line_webhook"
		case "/tasks/line-reply":
			route = "line_reply_task"
		case "/tasks/dispatch":
			route = "outbox_dispatch"
		}
		// An HTTP method is client-controlled. Bound the emitted value so
		// future metrics/log aggregations cannot gain unbounded cardinality.
		method := r.Method
		switch method {
		case http.MethodGet, http.MethodHead, http.MethodPost,
			http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions:
		default:
			method = "other"
		}
		level := slog.LevelDebug
		if writer.status >= http.StatusInternalServerError {
			level = slog.LevelWarn
		}
		logger.LogAttrs(r.Context(), level, "HTTP inbound",
			slog.String("role", role),
			slog.String("route", route),
			slog.String("method", method),
			slog.Int("status_code", writer.status),
			slog.Duration("duration", time.Since(started)))
	})
}
