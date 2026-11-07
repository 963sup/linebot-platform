package observability

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiniAppDimensionsAreBounded(t *testing.T) {
	if got := boundedRole("miniapp"); got != "miniapp" {
		t.Fatalf("role=%q", got)
	}
	if got := boundedRoute("/miniapp/config.json"); got != "miniapp_config" {
		t.Fatalf("route=%q", got)
	}
	for path, want := range map[string]string{
		"/miniapp/api/v1/session": "miniapp_session",
	} {
		if got := boundedRoute(path); got != want {
			t.Fatalf("route %q=%q, want %q", path, got, want)
		}
	}
	if got := boundedRoute("/miniapp/private-user-id"); got != "other" {
		t.Fatalf("unbounded route=%q", got)
	}
}

func TestInboundMetadataNeverLeaksQueryOrHeaders(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := WrapInboundHTTP("webhook", logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusServiceUnavailable)
	}))
	req := httptest.NewRequest(http.MethodPost, "http://local/private-user-id?token=private-secret", strings.NewReader("private-message"))
	req.Header.Set("Authorization", "Bearer private-auth")
	writer := httptest.NewRecorder()
	handler.ServeHTTP(writer, req)
	if writer.Code != http.StatusServiceUnavailable || !strings.Contains(out.String(), "503") {
		t.Fatal("missing safe HTTP telemetry")
	}
	for _, secret := range []string{"private-user-id", "private-secret", "private-message", "private-auth"} {
		if strings.Contains(out.String(), secret) {
			t.Fatalf("inbound telemetry leaked sensitive %s", secret)
		}
	}
	if !strings.Contains(out.String(), "other") {
		t.Fatal("unknown route must use a bounded label")
	}
}

func TestInboundWrapperPreservesResponseControllerFlush(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := WrapInboundHTTP("webhook", logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("ResponseController.Flush through observability wrapper: %v", err)
		}
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if !recorder.Flushed {
		t.Fatal("underlying response writer flush capability was lost")
	}
	if !strings.Contains(out.String(), "\"health\"") {
		t.Fatal("missing bounded health route telemetry")
	}
}

func TestInboundMethodHasBoundedLoggingCardinality(t *testing.T) {
	for _, tc := range []struct {
		method string
		want   string
	}{
		{http.MethodPost, http.MethodPost},
		{"CUSTOM-UNBOUNDED-PRIVATE-REQUEST", "other"},
	} {
		t.Run(tc.method, func(t *testing.T) {
			var out bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug}))
			handler := WrapInboundHTTP("webhook", logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			}))
			req := httptest.NewRequest(tc.method, "http://localhost/webhook/line", nil)
			handler.ServeHTTP(httptest.NewRecorder(), req)
			if !strings.Contains(out.String(), "\"method\":\""+tc.want+"\"") {
				t.Fatalf("expected bounded method %q in %q", tc.want, out.String())
			}
			if tc.method != tc.want && strings.Contains(out.String(), tc.method) {
				t.Fatal("client-controlled method leaked to telemetry")
			}
		})
	}
}

func TestInboundEarlyHintsPreservesFinalStatus(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug}))
	instrumented := WrapInboundHTTP("worker", logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusEarlyHints)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
	}))
	// The logger runs after the inner handler returns. Synchronize before
	// reading the buffer; the real HTTP server exercises 1xx wire semantics.
	logged := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		instrumented.ServeHTTP(w, r)
		close(logged)
	}))
	defer server.Close()

	response, err := server.Client().Get(server.URL + "/tasks/v2/line-reply")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("informational status hid final authentication rejection: got %d, want %d",
			response.StatusCode, http.StatusUnauthorized)
	}
	<-logged
	if !strings.Contains(out.String(), "\"status_code\":401") {
		t.Fatalf("terminal status missing from safe telemetry: %s", out.String())
	}
}

type responseWriterWithoutFlush struct{ http.ResponseWriter }

func TestInboundFlushTracksCommittedStatus(t *testing.T) {
	for _, tc := range []struct {
		name          string
		beforeFlush   int
		wantStatus    int
		wantStatusLog string
	}{
		{"implicit OK", 0, http.StatusOK, `"status_code":200`},
		{"explicit accepted", http.StatusAccepted, http.StatusAccepted, `"status_code":202`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug}))
			handler := WrapInboundHTTP("worker", logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tc.beforeFlush != 0 {
					w.WriteHeader(tc.beforeFlush)
				}
				if err := http.NewResponseController(w).Flush(); err != nil {
					t.Errorf("Flush through inbound wrapper: %v", err)
				}
				// Once flushed, this is too late to replace the HTTP status.
				w.WriteHeader(http.StatusUnauthorized)
			}))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tasks/v2/line-reply", nil))
			if !recorder.Flushed || recorder.Code != tc.wantStatus {
				t.Fatalf("underlying response: flushed=%t status=%d, want %d",
					recorder.Flushed, recorder.Code, tc.wantStatus)
			}
			if !strings.Contains(out.String(), tc.wantStatusLog) {
				t.Fatalf("status logged after Flush differs from the committed HTTP response: %s", out.String())
			}
		})
	}
}

func TestInboundUnsupportedFlushPreservesLaterStatus(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := WrapInboundHTTP("worker", logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := http.NewResponseController(w).Flush(); !errors.Is(err, http.ErrNotSupported) {
			t.Errorf("Flush without an underlying flusher: got %v, want ErrNotSupported", err)
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(responseWriterWithoutFlush{recorder}, httptest.NewRequest(http.MethodPost, "/tasks/v2/dispatch", nil))
	if recorder.Code != http.StatusUnauthorized || !strings.Contains(out.String(), `"status_code":401`) {
		t.Fatalf("unsupported Flush changed final status or telemetry: code=%d log=%s",
			recorder.Code, out.String())
	}
}
