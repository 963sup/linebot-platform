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

var benchmarkHTTPResponse *http.Response

func TestLoggingDoesNotExposeSecretsOrURLs(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	base := fakeTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 202, Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})
	decorated := WrapHTTP("line", logger)(base)
	req, err := http.NewRequest(http.MethodPost, "https://example.test/private-path?token=private-query", strings.NewReader("private-message"))
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

func BenchmarkLoggingTransportRoundTrip(b *testing.B) {
	for _, benchmark := range []struct {
		name   string
		logger *slog.Logger
	}{
		{name: "without-logger"},
		{name: "with-logger", logger: slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))},
	} {
		b.Run(benchmark.name, func(b *testing.B) {
			runtime := newLoggingRuntime(benchmark.logger)
			response := &http.Response{StatusCode: http.StatusAccepted, Body: http.NoBody}
			transport := runtime.WrapHTTP("line")(fakeTransport(func(*http.Request) (*http.Response, error) {
				return response, nil
			}))
			request, err := http.NewRequest(http.MethodPost, "https://example.test/path", strings.NewReader("body"))
			if err != nil {
				b.Fatal(err)
			}
			request.Header.Set("Authorization", "Bearer private-token")
			request.Header.Set("X-Test", "value")
			b.ReportAllocs()
			for b.Loop() {
				benchmarkHTTPResponse, err = transport.RoundTrip(request)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
