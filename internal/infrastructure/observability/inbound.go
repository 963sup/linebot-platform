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

func (w *inboundWriter) WriteHeader(status int) {
	if w.status != 0 {
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
		level := slog.LevelDebug
		if writer.status >= http.StatusInternalServerError {
			level = slog.LevelWarn
		}
		logger.LogAttrs(r.Context(), level, "HTTP inbound",
			slog.String("role", role),
			slog.String("route", route),
			slog.String("method", r.Method),
			slog.Int("status_code", writer.status),
			slog.Duration("duration", time.Since(started)))
	})
}
