package bootstrap

import (
	"errors"
	"testing"
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
