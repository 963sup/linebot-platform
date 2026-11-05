package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	taskin "github.com/963sup/linebot-platform/internal/adapter/inbound/http/tasks"
)

func TestWorkerRuntimeCloseAggregatesAndIsIdempotent(t *testing.T) {
	closeErr := errors.New("close failed")
	calls := 0
	runtime := &WorkerRuntime{closers: []runtimeCloser{{
		name: "worker dependency",
		close: func() error {
			calls++
			return closeErr
		},
	}}}
	runtime.ready.Store(true)

	first := runtime.Close()
	second := runtime.Close()
	if !errors.Is(first, closeErr) || second != first {
		t.Fatalf("expected stable aggregate close error, got first=%v second=%v", first, second)
	}
	if calls != 1 {
		t.Fatalf("close calls = %d, want 1", calls)
	}
	if runtime.ready.Load() {
		t.Fatal("closed worker must fail readiness")
	}
}

func TestWorkerRuntimeCloseDrainsActiveHandlerBeforeDependencies(t *testing.T) {
	limiter, err := taskin.NewConcurrencyLimiter(1)
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	handler := limiter.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(entered)
		<-release
	}))
	dependencyClosed := make(chan struct{})
	runtime := &WorkerRuntime{
		limiter: limiter,
		closers: []runtimeCloser{{name: "worker dependency", close: func() error {
			close(dependencyClosed)
			return nil
		}}},
	}
	requestDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/tasks/v2/line-reply", nil).WithContext(context.Background()))
		close(requestDone)
	}()
	<-entered
	closeDone := make(chan error, 1)
	go func() { closeDone <- runtime.Close() }()
	select {
	case err := <-closeDone:
		t.Fatalf("Close returned before active handler drained: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	select {
	case <-dependencyClosed:
		t.Fatal("dependency closed while a worker handler was active")
	default:
	}
	close(release)
	<-requestDone
	if err := <-closeDone; err != nil {
		t.Fatal(err)
	}
	select {
	case <-dependencyClosed:
	default:
		t.Fatal("dependency remained open after worker drain")
	}
}
