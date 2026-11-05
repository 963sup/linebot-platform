package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler(t *testing.T) {
	for _, tc := range []struct {
		method string
		status int
	}{
		{http.MethodGet, http.StatusOK},
		{http.MethodPost, http.StatusMethodNotAllowed},
	} {
		w := httptest.NewRecorder()
		Handler(w, httptest.NewRequest(tc.method, "/healthz", nil))
		if w.Code != tc.status {
			t.Fatalf("%s status = %d, want %d", tc.method, w.Code, tc.status)
		}
	}
}

func TestReadyHandler(t *testing.T) {
	ready := true
	handler := ReadyHandler(func(context.Context) error {
		if !ready {
			return errors.New("runtime closing")
		}
		return nil
	})

	for _, tc := range []struct {
		name   string
		method string
		ready  bool
		status int
	}{
		{name: "ready", method: http.MethodGet, ready: true, status: http.StatusOK},
		{name: "not ready", method: http.MethodGet, ready: false, status: http.StatusServiceUnavailable},
		{name: "wrong method", method: http.MethodPost, ready: true, status: http.StatusMethodNotAllowed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ready = tc.ready
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(tc.method, "/readyz", nil))
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d", w.Code, tc.status)
			}
		})
	}
}

func TestReadyHandlerWithoutCheckFailsClosed(t *testing.T) {
	w := httptest.NewRecorder()
	ReadyHandler(nil).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}
