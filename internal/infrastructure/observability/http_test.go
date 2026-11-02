package observability

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

type fakeTransport func(*http.Request) (*http.Response, error)

func (f fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLoggingDoesNotExposeSecretsOrURLs(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	base := fakeTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 202, Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})
	decorated := WrapHTTP("line", logger)(base)
	req, err := http.NewRequest(http.MethodPost, "https://example.test/secret-path?token=private-query", strings.NewReader("private-message"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer private-token")
	resp, err := decorated.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	logged := logs.String()
	if !strings.Contains(logged, "line") || !strings.Contains(logged, "202") {
		t.Fatalf("missing safe telemetry fields: %q", logged)
	}
	for _, secret := range []string{"private-path", "private-query", "private-message", "private-token"} {
		if strings.Contains(logged, secret) {
			t.Fatalf("sensitive value %q leaked to log", secret)
		}
	}
	if _, ok := WrapHTTP("line", nil)(base).(fakeTransport); !ok {
		t.Fatal("nil logger should preserve the original RoundTripper")
	}
}
