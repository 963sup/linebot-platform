package observability

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	metricapi "go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	traceapi "go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

const (
	ModeOff      = "off"
	ModeGCPOTLP  = "gcp-otlp"
	otlpEndpoint = "127.0.0.1:4317"
)

// Config contains only bounded, non-secret telemetry policy. OTLP is fixed to
// the loopback collector owned by the same runtime revision.
type Config struct {
	Mode             string
	ServiceName      string
	Environment      string
	TraceSampleRatio float64
	MetricInterval   time.Duration
}

// Runtime owns providers, instruments, propagation and exporter shutdown.
type Runtime struct {
	logger     *slog.Logger
	tracer     traceapi.Tracer
	propagator propagation.TextMapPropagator
	instruments
	shutdownOnce sync.Once
	shutdownErr  error
	shutdowns    []func(context.Context) error
}

type instruments struct {
	serverRequests metricapi.Int64Counter
	serverErrors   metricapi.Int64Counter
	serverDuration metricapi.Float64Histogram
	clientRequests metricapi.Int64Counter
	clientErrors   metricapi.Int64Counter
	clientDuration metricapi.Float64Histogram
}

// NewRuntime constructs an isolated provider set without installing global
// OpenTelemetry state. ModeOff performs no network or metadata lookup.
func NewRuntime(ctx context.Context, cfg Config, logger *slog.Logger) (*Runtime, error) {
	if ctx == nil {
		return nil, errors.New("telemetry context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cfg.Mode == "" {
		cfg.Mode = ModeOff
	}
	if cfg.ServiceName == "" {
		return nil, errors.New("telemetry service name is required")
	}
	if cfg.TraceSampleRatio < 0 || cfg.TraceSampleRatio > 1 {
		return nil, errors.New("telemetry trace sample ratio must be from 0 to 1")
	}
	if cfg.MetricInterval == 0 {
		cfg.MetricInterval = time.Minute
	}
	if cfg.MetricInterval < 10*time.Second || cfg.MetricInterval > 5*time.Minute {
		return nil, errors.New("telemetry metric interval must be from 10s to 5m")
	}
	if cfg.Mode == ModeOff {
		return newRuntime(logger, tracenoop.NewTracerProvider(), metricnoop.NewMeterProvider(), nil)
	}
	if cfg.Mode != ModeGCPOTLP {
		return nil, errors.New("unsupported telemetry mode")
	}

	detected, err := resource.New(ctx,
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			attribute.String("deployment.environment.name", boundedEnvironment(cfg.Environment)),
		),
	)
	if err != nil {
		return nil, errors.New("detect telemetry resource")
	}
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otlpEndpoint), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, errors.New("create trace exporter")
	}
	meterExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(otlpEndpoint), otlpmetricgrpc.WithInsecure())
	if err != nil {
		_ = traceExporter.Shutdown(ctx)
		return nil, errors.New("create metric exporter")
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(detected),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.TraceSampleRatio))),
		sdktrace.WithBatcher(traceExporter),
	)
	mp := metric.NewMeterProvider(
		metric.WithResource(detected),
		metric.WithReader(metric.NewPeriodicReader(meterExporter, metric.WithInterval(cfg.MetricInterval))),
	)
	runtime, err := newRuntime(logger, tp, mp, []func(context.Context) error{mp.Shutdown, tp.Shutdown})
	if err != nil {
		_ = mp.Shutdown(ctx)
		_ = tp.Shutdown(ctx)
		return nil, err
	}
	return runtime, nil
}

func newRuntime(logger *slog.Logger, tp traceapi.TracerProvider, mp metricapi.MeterProvider, shutdowns []func(context.Context) error) (*Runtime, error) {
	meter := mp.Meter("github.com/963sup/linebot-platform/internal/infrastructure/observability")
	serverRequests, err := meter.Int64Counter("linebot.http.server.requests")
	if err != nil {
		return nil, errors.New("create server request counter")
	}
	serverErrors, err := meter.Int64Counter("linebot.http.server.errors")
	if err != nil {
		return nil, errors.New("create server error counter")
	}
	serverDuration, err := meter.Float64Histogram("linebot.http.server.duration", metricapi.WithUnit("s"))
	if err != nil {
		return nil, errors.New("create server duration histogram")
	}
	clientRequests, err := meter.Int64Counter("linebot.http.client.requests")
	if err != nil {
		return nil, errors.New("create client request counter")
	}
	clientErrors, err := meter.Int64Counter("linebot.http.client.errors")
	if err != nil {
		return nil, errors.New("create client error counter")
	}
	clientDuration, err := meter.Float64Histogram("linebot.http.client.duration", metricapi.WithUnit("s"))
	if err != nil {
		return nil, errors.New("create client duration histogram")
	}
	return &Runtime{
		logger:     logger,
		tracer:     tp.Tracer("github.com/963sup/linebot-platform/internal/infrastructure/observability"),
		propagator: propagation.TraceContext{},
		instruments: instruments{
			serverRequests: serverRequests, serverErrors: serverErrors, serverDuration: serverDuration,
			clientRequests: clientRequests, clientErrors: clientErrors, clientDuration: clientDuration,
		},
		shutdowns: shutdowns,
	}, nil
}

func newLoggingRuntime(logger *slog.Logger) *Runtime {
	runtime, _ := newRuntime(logger, tracenoop.NewTracerProvider(), metricnoop.NewMeterProvider(), nil)
	return runtime
}

// Shutdown flushes and closes owned providers once. Callers supply a deadline.
func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.shutdownOnce.Do(func() {
		for _, shutdown := range r.shutdowns {
			r.shutdownErr = errors.Join(r.shutdownErr, shutdown(ctx))
		}
	})
	return r.shutdownErr
}

// Propagator returns the credential-free W3C trace-context propagator.
func (r *Runtime) Propagator() propagation.TextMapPropagator {
	if r == nil || r.propagator == nil {
		return propagation.TraceContext{}
	}
	return r.propagator
}

func boundedEnvironment(value string) string {
	switch value {
	case "local", "staging", "production":
		return value
	default:
		return "unknown"
	}
}
