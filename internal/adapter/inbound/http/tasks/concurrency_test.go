package tasks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestConcurrencyLimiterRejectsSaturationAndClosedState(t *testing.T) {
	limiter, err := NewConcurrencyLimiter(1)
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	handler := limiter.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(entered)
		<-release
	}))

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/tasks/v2/line-reply", nil))
		close(done)
	}()
	<-entered

	saturated := httptest.NewRecorder()
	handler.ServeHTTP(saturated, httptest.NewRequest(http.MethodPost, "/tasks/v2/line-reply", nil))
	if saturated.Code != http.StatusServiceUnavailable || saturated.Header().Get("Retry-After") != "1" {
		t.Fatalf("saturated response = %d retry-after=%q", saturated.Code, saturated.Header().Get("Retry-After"))
	}
	drainCtx, cancelDrain := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancelDrain()
	if err := limiter.Shutdown(drainCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("active drain error = %v, want deadline", err)
	}
	close(release)
	<-done
	if err := limiter.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	closed := httptest.NewRecorder()
	handler.ServeHTTP(closed, httptest.NewRequest(http.MethodPost, "/tasks/v2/line-reply", nil))
	if closed.Code != http.StatusServiceUnavailable {
		t.Fatalf("closed response = %d", closed.Code)
	}
}

func TestConcurrencyLimiterPropagatesCancellationAndReleasesSlot(t *testing.T) {
	limiter, err := NewConcurrencyLimiter(1)
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	handler := limiter.Wrap(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
	}))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/tasks/v2/line-reply", nil).WithContext(ctx))
		close(done)
	}()
	<-entered
	cancel()
	<-done
	if err := limiter.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	limiter.Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/tasks/v2/dispatch", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("closed limiter response = %d", response.Code)
	}
}

func TestConcurrencyLimiterRejectsInvalidLimit(t *testing.T) {
	if _, err := NewConcurrencyLimiter(0); err == nil {
		t.Fatal("zero concurrency limit accepted")
	}
}
