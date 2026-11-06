package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterWithoutCredentials(t *testing.T) {
	runtime, err := NewWebhookRuntime(Config{Port: "8080"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := runtime.Close(); err != nil {
			t.Error(err)
		}
	})
	for _, tc := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/healthz", http.StatusOK},
		{http.MethodGet, "/readyz", http.StatusOK},
		{http.MethodPost, "/healthz", http.StatusMethodNotAllowed},
		{http.MethodPost, "/readyz", http.StatusMethodNotAllowed},
		{http.MethodPost, "/webhook/line", http.StatusServiceUnavailable},
		{http.MethodGet, "/missing", http.StatusNotFound},
	} {
		rec := httptest.NewRecorder()
		runtime.Handler.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != tc.status {
			t.Fatalf("%s %s status = %d, expected %d", tc.method, tc.path, rec.Code, tc.status)
		}
	}
}
