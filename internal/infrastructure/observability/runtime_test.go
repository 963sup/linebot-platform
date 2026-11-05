package observability

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	traceapi "go.opentelemetry.io/otel/trace"
)

func instrumentedRuntime(t *testing.T) (*Runtime, *metric.ManualReader, *tracetest.InMemoryExporter) {
	t.Helper()
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSyncer(exporter),
	)
	runtime, err := newRuntime(nil, tp, mp, []func(context.Context) error{mp.Shutdown, tp.Shutdown})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	return runtime, reader, exporter
}

func TestRuntimeOffHasNoExternalDependencyAndShutdownIsIdempotent(t *testing.T) {
	runtime, err := NewRuntime(context.Background(), Config{
		Mode: ModeOff, ServiceName: "test", TraceSampleRatio: 0.05, MetricInterval: time.Minute,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestInboundTelemetryUsesBoundedFieldsAndIgnoresRemoteParent(t *testing.T) {
	runtime, reader, exporter := instrumentedRuntime(t)
	handler := runtime.WrapInboundHTTP("webhook", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	request := httptest.NewRequest(http.MethodPost,
		"https://service.test/private-user?token=private-secret", strings.NewReader("private-body"))
	request.Header.Set("Authorization", "Bearer private-token")
	request.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	handler.ServeHTTP(httptest.NewRecorder(), request)

	spans := exporter.GetSpans()
	if len(spans) != 1 || spans[0].Parent.TraceID().String() == "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("public request accepted untrusted trace parent: %+v", spans)
	}
	encoded := spans[0].Name
	for _, attr := range spans[0].Attributes {
		encoded += fmt.Sprint(attr)
	}
	for _, sensitive := range []string{"private-user", "private-secret", "private-body", "private-token"} {
		if strings.Contains(encoded, sensitive) {
			t.Fatalf("span leaked %q", sensitive)
		}
	}
	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &metrics); err != nil {
		t.Fatal(err)
	}
	names := metricNames(metrics)
	for _, want := range []string{"linebot.http.server.requests", "linebot.http.server.errors", "linebot.http.server.duration"} {
		if !names[want] {
			t.Fatalf("missing metric %s in %v", want, names)
		}
	}
}

func TestTrustedTaskSpanContinuesVerifiedRemoteParent(t *testing.T) {
	runtime, _, exporter := instrumentedRuntime(t)
	header := http.Header{}
	header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	ctx, end := runtime.StartTrustedTaskSpan(context.Background(), header, "line_reply_task")
	if got := traceapi.SpanContextFromContext(ctx); got.TraceID().String() != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("trusted task trace ID = %s", got.TraceID())
	}
	end()
	spans := exporter.GetSpans()
	if len(spans) != 1 || spans[0].Parent.SpanID().String() != "00f067aa0ba902b7" {
		t.Fatalf("trusted task parent was not continued: %+v", spans)
	}
}

func TestOutboundTelemetryPropagatesTraceWithoutLeakingRequest(t *testing.T) {
	runtime, _, exporter := instrumentedRuntime(t)
	transport := runtime.WrapHTTP("line")(fakeTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("traceparent") == "" {
			t.Fatal("missing traceparent")
		}
		return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader("private-response"))}, nil
	}))
	ctx, parent := runtime.tracer.Start(context.Background(), "parent")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://example.test/private-path?token=private-query", strings.NewReader("private-body"))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer private-token")
	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	parent.End()
	for _, span := range exporter.GetSpans() {
		encoded := span.Name
		for _, attr := range span.Attributes {
			encoded += fmt.Sprint(attr)
		}
		for _, sensitive := range []string{"private-path", "private-query", "private-body", "private-token", "private-response"} {
			if strings.Contains(encoded, sensitive) {
				t.Fatalf("span leaked %q", sensitive)
			}
		}
	}
}

func metricNames(resources metricdata.ResourceMetrics) map[string]bool {
	names := make(map[string]bool)
	for _, scope := range resources.ScopeMetrics {
		for _, metric := range scope.Metrics {
			names[metric.Name] = true
		}
	}
	return names
}
