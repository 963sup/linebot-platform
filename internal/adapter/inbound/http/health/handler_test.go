package health

import (
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
