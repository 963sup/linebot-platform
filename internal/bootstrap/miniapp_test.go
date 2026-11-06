package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	attendanceapp "github.com/963sup/linebot-platform/internal/application/usecases/attendance"
)

type fakeMiniAppActions struct{}

func (fakeMiniAppActions) Enroll(context.Context, attendanceapp.EnrollCommand) (attendanceapp.EnrollResult, error) {
	return attendanceapp.EnrollResult{Status: attendanceapp.EnrollCreated}, nil
}
func (fakeMiniAppActions) ListOptions(context.Context, attendanceapp.OptionsCommand) (attendanceapp.OptionsResult, error) {
	return attendanceapp.OptionsResult{}, nil
}
func (fakeMiniAppActions) Punch(context.Context, attendanceapp.PunchCommand) (attendanceapp.PunchResult, error) {
	return attendanceapp.PunchResult{}, nil
}

func validMiniAppSettings() map[string]string {
	return map[string]string{
		KeyLIFFID: "2000000000-example", KeyLoginChannelID: "2000000000",
		KeyMiniAppPublicOrigin: "http://127.0.0.1:8080", KeyProjectID: "test-project",
		KeyAttendanceKeyID: "v1", KeyAttendanceKeyMap: "v1=projects/test-project/secrets/attendance-location-key/versions/1",
	}
}

func TestLoadMiniAppConfig(t *testing.T) {
	values := validMiniAppSettings()
	cfg, err := LoadMiniAppConfig(env(values))
	if err != nil || cfg.Role != RoleMiniApp || cfg.Environment != EnvLocal || cfg.LINEAPITimeout == 0 || cfg.FirestoreDatabase != "(default)" || len(cfg.LocationKeyVersions) != 1 {
		t.Fatalf("config=%#v err=%v", cfg, err)
	}
	values[KeyMiniAppPublicOrigin] = "http://localhost:8080"
	values[KeyAppEnv] = "production"
	if _, err := LoadMiniAppConfig(env(values)); err == nil {
		t.Fatal("production accepted non-HTTPS public origin")
	}
	values[KeyMiniAppPublicOrigin] = "https://miniapp.example.test"
	cfg, err = LoadMiniAppConfig(env(values))
	if err != nil || cfg.Log.Format != "json" {
		t.Fatalf("production config=%#v err=%v", cfg, err)
	}
}

func TestMiniAppConfigRejectsOriginAndMessagingSecrets(t *testing.T) {
	for _, mutate := range []func(map[string]string){
		func(v map[string]string) { v[KeyMiniAppPublicOrigin] = "https://miniapp.example.test/path" },
		func(v map[string]string) { v[KeyMiniAppPublicOrigin] = "https://miniapp.example.test/" },
		func(v map[string]string) { v[KeyChannelSecret] = "forbidden-secret" },
		func(v map[string]string) { v[KeyChannelToken] = "forbidden-token" },
		func(v map[string]string) { v[KeyQueueName] = "forbidden-queue" },
		func(v map[string]string) { v[KeyWorkerURL] = "https://forbidden.example" },
		func(v map[string]string) { v[KeyBotDestination] = "forbidden-destination" },
	} {
		values := validMiniAppSettings()
		mutate(values)
		if _, err := LoadMiniAppConfig(env(values)); err == nil || strings.Contains(err.Error(), "forbidden-") {
			t.Fatalf("unsafe config error=%v", err)
		}
	}
}

func TestMiniAppConfigRejectsInvalidAttendanceKeyMappings(t *testing.T) {
	for _, value := range []string{
		"", " v1=projects/test-project/secrets/attendance-location-key/versions/1",
		"v1=projects/test-project/secrets/attendance-location-key/versions/latest",
		"v1=projects/other-project/secrets/attendance-location-key/versions/1",
		"v1=projects/test-project/secrets/attendance-location-key/versions/1,v1=projects/test-project/secrets/attendance-location-key/versions/2",
		"v1=projects/test-project/secrets/other/versions/1", "v1==bad",
	} {
		values := validMiniAppSettings()
		values[KeyAttendanceKeyMap] = value
		if _, err := LoadMiniAppConfig(env(values)); err == nil {
			t.Fatalf("invalid key mapping accepted: %q", value)
		}
	}
	values := validMiniAppSettings()
	values[KeyAttendanceKeyID] = "v2"
	if _, err := LoadMiniAppConfig(env(values)); err == nil {
		t.Fatal("missing active key accepted")
	}
}

func TestMiniAppConfigRejectsMalformedPublicIdentifiersAndCloudPort(t *testing.T) {
	for _, mutate := range []func(map[string]string){
		func(v map[string]string) { v[KeyLIFFID] = "missing-hyphen-format!" },
		func(v map[string]string) { v[KeyLoginChannelID] = "12345" },
		func(v map[string]string) { v[KeyLoginChannelID] = "123456789012345678901" },
		func(v map[string]string) {
			v[KeyAppEnv] = "staging"
			v[KeyMiniAppPublicOrigin] = "https://miniapp.example.test:8443"
		},
		func(v map[string]string) {
			v[KeyAppEnv] = "staging"
			v[KeyMiniAppPublicOrigin] = "https://MINIAPP.example.test"
		},
		func(v map[string]string) { v[KeyMiniAppPublicOrigin] = "http://example.test" },
	} {
		values := validMiniAppSettings()
		mutate(values)
		if _, err := LoadMiniAppConfig(env(values)); err == nil {
			t.Fatalf("unsafe Mini App settings accepted: %#v", values)
		}
	}
}

func TestMiniAppRuntimeLifecycleAndPublicConfig(t *testing.T) {
	values := validMiniAppSettings()
	values[KeyAppEnv] = "staging"
	values[KeyMiniAppPublicOrigin] = "https://miniapp.example.test"
	cfg, err := LoadMiniAppConfig(env(values))
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := newMiniAppRuntime(cfg, nil, func(context.Context, MiniAppConfig, *slog.Logger) (miniAppComposition, error) {
		return miniAppComposition{actions: fakeMiniAppActions{}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	redirect := httptest.NewRecorder()
	runtime.Handler.ServeHTTP(redirect, httptest.NewRequest(http.MethodGet, "/miniapp", nil))
	if redirect.Code != http.StatusPermanentRedirect {
		t.Fatalf("redirect=%d", redirect.Code)
	}
	health := httptest.NewRecorder()
	runtime.Handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health=%d", health.Code)
	}
	runtime.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/miniapp/config.json", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), cfg.LIFFID) || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("response=%d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	ready := httptest.NewRecorder()
	runtime.Handler.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if ready.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready after close=%d", ready.Code)
	}
}

func TestMiniAppRuntimeCloseToleratesPartialInitialization(t *testing.T) {
	runtime := &MiniAppRuntime{}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMiniAppRuntimeClosesOwnedResourcesOnceInReverseOrder(t *testing.T) {
	values := validMiniAppSettings()
	cfg, err := LoadMiniAppConfig(env(values))
	if err != nil {
		t.Fatal(err)
	}
	var closed []string
	runtime, err := newMiniAppRuntime(cfg, nil, func(context.Context, MiniAppConfig, *slog.Logger) (miniAppComposition, error) {
		return miniAppComposition{
			actions: fakeMiniAppActions{},
			closers: []runtimeCloser{
				{name: "first", close: func() error { closed = append(closed, "first"); return nil }},
				{name: "second", close: func() error { closed = append(closed, "second"); return errors.New("close failed") }},
			},
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	firstErr := runtime.Close()
	secondErr := runtime.Close()
	if firstErr == nil || secondErr == nil || firstErr.Error() != secondErr.Error() {
		t.Fatalf("close errors=(%v,%v)", firstErr, secondErr)
	}
	if got := strings.Join(closed, ","); got != "second,first" {
		t.Fatalf("close order=%q", got)
	}
}
