package observability

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
