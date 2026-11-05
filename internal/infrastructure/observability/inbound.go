package observability

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// StartTrustedTaskSpan extracts W3C context and starts the worker server span.
// Callers must invoke it only after successful OIDC verification.
func (o *Runtime) StartTrustedTaskSpan(ctx context.Context, header http.Header, operation string) (context.Context, func()) {
	switch operation {
	case "line_reply_task", "outbox_dispatch":
	default:
		operation = "other"
	}
	parent := o.propagator.Extract(ctx, propagation.HeaderCarrier(header))
	spanCtx, span := o.tracer.Start(parent, "TASK "+operation,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attribute.String("http.route.name", operation)))
	return spanCtx, func() { span.End() }
}

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
	return newLoggingRuntime(logger).WrapInboundHTTP(role, next)
}

// WrapInboundHTTP records bounded metrics, starts a safe server span and logs
// the same low-cardinality dimensions. Public webhook requests do not control
// trace parentage; authenticated worker traffic may continue Cloud Tasks spans.
func (o *Runtime) WrapInboundHTTP(role string, next http.Handler) http.Handler {
	role = boundedRole(role)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		route := boundedRoute(r.URL.Path)
		method := boundedMethod(r.Method)
		// Never accept client-controlled trace parentage here. Private task
		// handlers extract trace context only after OIDC verification succeeds.
		parent := r.Context()
		attrs := []attribute.KeyValue{
			attribute.String("service.role", role),
			attribute.String("http.route.name", route),
			attribute.String("http.request.method", method),
		}
		ctx, span := o.tracer.Start(parent, "HTTP "+route,
			trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attrs...))
		defer span.End()
		r = r.WithContext(ctx)
		writer := &inboundWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)
		if writer.status == 0 {
			writer.status = http.StatusOK
		}
		statusClass := boundedStatusClass(writer.status)
		resultAttrs := append(attrs,
			attribute.String("http.response.status_class", statusClass))
		o.serverRequests.Add(ctx, 1, metric.WithAttributes(resultAttrs...))
		o.serverDuration.Record(ctx, time.Since(started).Seconds(), metric.WithAttributes(resultAttrs...))
		level := slog.LevelDebug
		if writer.status >= http.StatusInternalServerError {
			level = slog.LevelWarn
			o.serverErrors.Add(ctx, 1, metric.WithAttributes(resultAttrs...))
			span.SetStatus(codes.Error, "server error")
		}
		span.SetAttributes(attribute.Int("http.response.status_code", writer.status))
		if o.logger == nil {
			return
		}
		logAttrs := []slog.Attr{
			slog.String("role", role),
			slog.String("route", route),
			slog.String("method", method),
			slog.Int("status_code", writer.status),
			slog.Duration("duration", time.Since(started)),
		}
		if sc := span.SpanContext(); sc.IsValid() {
			logAttrs = append(logAttrs, slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
		}
		o.logger.LogAttrs(ctx, level, "HTTP inbound", logAttrs...)
	})
}

func boundedRole(role string) string {
	switch role {
	case "webhook", "worker":
		return role
	default:
		return "other"
	}
}

func boundedRoute(path string) string {
	switch path {
	case "/healthz":
		return "health"
	case "/readyz":
		return "ready"
	case "/webhook/line":
		return "line_webhook"
	case "/tasks/v2/line-reply", "/tasks/line-reply":
		return "line_reply_task"
	case "/tasks/v2/dispatch", "/tasks/dispatch":
		return "outbox_dispatch"
	default:
		return "other"
	}
}

func boundedMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost,
		http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions:
		return method
	default:
		return "other"
	}
}

func boundedStatusClass(status int) string {
	if status < 100 || status > 599 {
		return "other"
	}
	return strconv.Itoa(status/100) + "xx"
}
