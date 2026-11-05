// Package observability instruments outbound SDK requests without recording
// URLs, authorization, user IDs, request bodies or provider error payloads.
package observability

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var errOutboundTransport = errors.New("outbound transport error")

type loggingTransport struct {
	provider string
	next     http.RoundTripper
	runtime  *Runtime
}

// WrapHTTP is an optional safe telemetry decorator for owned SDK transports.
// It does not retry requests, inspect request bodies or alter headers.
func WrapHTTP(provider string, logger *slog.Logger) func(http.RoundTripper) http.RoundTripper {
	if logger == nil {
		return func(next http.RoundTripper) http.RoundTripper { return next }
	}
	return newLoggingRuntime(logger).WrapHTTP(provider)
}

// WrapHTTP instruments an owned transport without inspecting URLs, headers,
// bodies or provider error payloads.
func (o *Runtime) WrapHTTP(provider string) func(http.RoundTripper) http.RoundTripper {
	provider = boundedProvider(provider)
	return func(next http.RoundTripper) http.RoundTripper {
		if next == nil {
			next = http.DefaultTransport
		}
		if o == nil {
			return next
		}
		return &loggingTransport{provider: provider, next: next, runtime: o}
	}
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	started := time.Now()
	method := boundedMethod(req.Method)
	attrs := []attribute.KeyValue{
		attribute.String("server.address.name", t.provider),
		attribute.String("http.request.method", method),
	}
	ctx, span := t.runtime.tracer.Start(req.Context(), "HTTP "+t.provider,
		trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(attrs...))
	defer span.End()
	request := req.Clone(ctx)
	request.Header = req.Header.Clone()
	t.runtime.Propagator().Inject(ctx, propagation.HeaderCarrier(request.Header))
	resp, err := t.next.RoundTrip(request)
	code := 0
	if resp != nil {
		code = resp.StatusCode
	}
	statusClass := boundedStatusClass(code)
	resultAttrs := append(attrs, attribute.String("http.response.status_class", statusClass))
	t.runtime.clientRequests.Add(ctx, 1, metric.WithAttributes(resultAttrs...))
	t.runtime.clientDuration.Record(ctx, time.Since(started).Seconds(), metric.WithAttributes(resultAttrs...))
	if code != 0 {
		span.SetAttributes(attribute.Int("http.response.status_code", code))
	}
	if err != nil || code >= http.StatusInternalServerError {
		t.runtime.clientErrors.Add(ctx, 1, metric.WithAttributes(resultAttrs...))
		span.SetStatus(codes.Error, "outbound request failed")
		if err != nil {
			span.RecordError(errOutboundTransport)
		}
	}
	if t.runtime.logger == nil {
		return resp, err
	}
	logAttrs := []slog.Attr{
		slog.String("provider", t.provider),
		slog.String("method", method),
		slog.Int("status_code", code),
		slog.Duration("duration", time.Since(started)),
		slog.Bool("transport_error", err != nil),
	}
	if sc := span.SpanContext(); sc.IsValid() {
		logAttrs = append(logAttrs, slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
	}
	t.runtime.logger.LogAttrs(ctx, slog.LevelDebug, "SDK outbound HTTP", logAttrs...)
	return resp, err
}

func boundedProvider(provider string) string {
	switch provider {
	case "line", "google", "openai", "mcp":
		return provider
	default:
		return "other"
	}
}
