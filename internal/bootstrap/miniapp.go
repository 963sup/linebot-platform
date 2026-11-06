package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/http/health"
	miniappin "github.com/963sup/linebot-platform/internal/adapter/inbound/http/miniapp"
	firestoreout "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	lineout "github.com/963sup/linebot-platform/internal/adapter/outbound/line"
	attendanceapp "github.com/963sup/linebot-platform/internal/application/usecases/attendance"
	firestoreinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/firestore"
	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
	"github.com/963sup/linebot-platform/internal/infrastructure/observability"
)

type MiniAppRuntime struct {
	Handler   http.Handler
	ready     atomic.Bool
	closeOnce sync.Once
	closeErr  error
	closers   []runtimeCloser
}

type miniAppComposition struct {
	actions attendanceapp.Actions
	wrap    func(http.Handler) http.Handler
	closers []runtimeCloser
}

type miniAppComposer func(context.Context, MiniAppConfig, *slog.Logger) (miniAppComposition, error)

// NewMiniAppRuntime composes the production Attendance Mini App. Cloud clients
// are initialized before readiness is published and are owned by the runtime.
func NewMiniAppRuntime(cfg MiniAppConfig, logger *slog.Logger) (*MiniAppRuntime, error) {
	return newMiniAppRuntime(cfg, logger, composeMiniApp)
}

func newMiniAppRuntime(cfg MiniAppConfig, logger *slog.Logger, compose miniAppComposer) (*MiniAppRuntime, error) {
	if cfg.Role != RoleMiniApp || cfg.LIFFID == "" || cfg.LoginChannelID == "" || cfg.PublicOrigin == "" ||
		cfg.ProjectID == "" || cfg.FirestoreDatabase == "" || cfg.LocationActiveKeyID == "" || len(cfg.LocationKeyVersions) == 0 {
		return nil, errors.New("validated miniapp configuration is required")
	}
	if compose == nil {
		return nil, errors.New("miniapp composition is required")
	}
	startup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	composition, err := compose(startup, cfg, logger)
	if err != nil {
		return nil, err
	}
	runtime := &MiniAppRuntime{closers: composition.closers}
	miniappHandler, err := miniappin.New(miniappin.Config{LIFFID: cfg.LIFFID, AllowedOrigin: cfg.PublicOrigin}, composition.actions)
	if err != nil {
		return nil, errors.Join(err, runtime.Close())
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.Handler)
	mux.Handle("/readyz", health.ReadyHandler(func(context.Context) error {
		if !runtime.ready.Load() {
			return errors.New("miniapp runtime is not ready")
		}
		return nil
	}))
	mux.Handle("/miniapp/", miniappHandler)
	mux.Handle("/miniapp", miniappHandler)
	if composition.wrap == nil {
		composition.wrap = func(handler http.Handler) http.Handler { return handler }
	}
	runtime.Handler = composition.wrap(mux)
	runtime.ready.Store(true)
	return runtime, nil
}

func composeMiniApp(ctx context.Context, cfg MiniAppConfig, logger *slog.Logger) (miniAppComposition, error) {
	var result miniAppComposition
	fail := func(err error) (miniAppComposition, error) {
		return miniAppComposition{}, errors.Join(err, closeRuntimeClosers(result.closers))
	}
	telemetry, err := observability.NewRuntime(ctx, observability.Config{
		Mode: cfg.Telemetry.Mode, ServiceName: "linebot-miniapp",
		Environment: string(cfg.Environment), TraceSampleRatio: cfg.Telemetry.TraceSampleRatio,
		MetricInterval: cfg.Telemetry.MetricInterval,
	}, logger)
	if err != nil {
		return result, err
	}
	result.wrap = func(handler http.Handler) http.Handler { return telemetry.WrapInboundHTTP("miniapp", handler) }
	result.closers = append(result.closers, runtimeCloser{name: "telemetry", close: telemetryCloser(telemetry)})

	timeout := cfg.LINEAPITimeout
	if timeout == 0 {
		timeout = httpclient.DefaultTimeout
	}
	outbound, err := httpclient.New(httpclient.Config{Timeout: timeout}, telemetry.WrapHTTP("line-login"))
	if err != nil {
		return fail(err)
	}
	result.closers = append(result.closers, runtimeCloser{name: "line HTTP", close: func() error { outbound.Close(); return nil }})
	verifier, err := lineout.NewIDTokenVerifier(outbound.Client, cfg.LoginChannelID)
	if err != nil {
		return fail(err)
	}
	keyring, err := loadAttendanceKeyring(ctx, cfg.LocationActiveKeyID, cfg.LocationKeyVersions)
	if err != nil {
		return fail(err)
	}
	firestoreClient, err := firestoreinfra.NewClientWithDatabase(ctx, cfg.ProjectID, cfg.FirestoreDatabase)
	if err != nil {
		return fail(err)
	}
	result.closers = append(result.closers, runtimeCloser{name: "Firestore", close: firestoreClient.Close})
	store, err := firestoreout.NewAttendanceStore(firestoreClient, keyring)
	if err != nil {
		return fail(err)
	}
	service, err := attendanceapp.NewService(attendanceapp.Dependencies{
		Verifier: verifier, Enrollment: store, Reader: store,
		Attendance: store, Clock: systemClock{},
	})
	if err != nil {
		return fail(err)
	}
	result.actions = service
	return result, nil
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

func (r *MiniAppRuntime) Close() error {
	if r == nil {
		return nil
	}
	r.ready.Store(false)
	r.closeOnce.Do(func() { r.closeErr = closeRuntimeClosers(r.closers) })
	return r.closeErr
}
