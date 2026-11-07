package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	linesession "github.com/963sup/linebot-platform/internal/application/usecases/line-session"
)

type miniAppVerifierStub struct{}

func (miniAppVerifierStub) Verify(context.Context, string) (linesession.Identity, error) {
	return linesession.Identity{}, nil
}

func validMiniAppRuntimeConfig() MiniAppConfig {
	return MiniAppConfig{
		Environment: EnvLocal, Role: RoleMiniApp,
		LIFFID: "123456-example", ChannelID: "123456",
		LIFFURL:     "https://miniapp.line.me/123456-example",
		EndpointURL: "http://localhost/miniapp/", PublicOrigin: "http://localhost",
	}
}

func TestMiniAppRuntimeReadinessAndCloseAreOwnedByComposition(t *testing.T) {
	closeCalls := 0
	composerCalls := 0
	composer := func(context.Context, MiniAppConfig, *slog.Logger) (miniAppComposition, error) {
		composerCalls++
		return miniAppComposition{
			verifier: miniAppVerifierStub{},
			closers: []runtimeCloser{{name: "test dependency", close: func() error {
				closeCalls++
				return nil
			}}},
		}, nil
	}
	runtime, err := newMiniAppRuntime(validMiniAppRuntimeConfig(), slog.Default(), composer)
	if err != nil {
		t.Fatal(err)
	}
	if composerCalls != 1 {
		t.Fatalf("composer calls = %d, want 1", composerCalls)
	}
	response := httptest.NewRecorder()
	runtime.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("ready status = %d", response.Code)
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	if closeCalls != 1 {
		t.Fatalf("close calls = %d, want 1", closeCalls)
	}
	response = httptest.NewRecorder()
	runtime.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("closed ready status = %d", response.Code)
	}
}

func TestMiniAppRuntimeClosesCompositionWhenHandlerConstructionFails(t *testing.T) {
	closeErr := errors.New("close failed")
	closeCalls := 0
	composer := func(context.Context, MiniAppConfig, *slog.Logger) (miniAppComposition, error) {
		return miniAppComposition{
			verifier: miniAppVerifierStub{},
			closers: []runtimeCloser{{name: "test dependency", close: func() error {
				closeCalls++
				return closeErr
			}}},
		}, nil
	}
	config := validMiniAppRuntimeConfig()
	config.PublicOrigin = "not-an-origin"
	runtime, err := newMiniAppRuntime(config, slog.Default(), composer)
	if runtime != nil || err == nil || !errors.Is(err, closeErr) {
		t.Fatalf("runtime=%v err=%v", runtime, err)
	}
	if closeCalls != 1 {
		t.Fatalf("close calls = %d, want 1", closeCalls)
	}
}
