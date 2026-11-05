package tasks

import (
	"context"
	"errors"
	"net/http"
	"sync"
)

// ConcurrencyLimiter bounds private worker requests before they enter an
// Application workflow. Saturation is retryable; no request is queued in an
// unbounded in-process backlog.
type ConcurrencyLimiter struct {
	slots   chan struct{}
	mu      sync.Mutex
	active  int
	closed  bool
	drained chan struct{}
	done    bool
}

func NewConcurrencyLimiter(limit int) (*ConcurrencyLimiter, error) {
	if limit < 1 {
		return nil, errors.New("worker concurrency limit must be positive")
	}
	return &ConcurrencyLimiter{slots: make(chan struct{}, limit), drained: make(chan struct{})}, nil
}

func (l *ConcurrencyLimiter) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l == nil || next == nil || !l.acquire() {
			workerUnavailable(w)
			return
		}
		defer l.release()
		if err := r.Context().Err(); err != nil {
			workerUnavailable(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *ConcurrencyLimiter) acquire() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return false
	}
	select {
	case l.slots <- struct{}{}:
		l.active++
		return true
	default:
		return false
	}
}

func (l *ConcurrencyLimiter) release() {
	<-l.slots
	l.mu.Lock()
	l.active--
	if l.closed && l.active == 0 && !l.done {
		close(l.drained)
		l.done = true
	}
	l.mu.Unlock()
}

// Shutdown stops admission and waits for active handlers to drain. The caller
// owns the deadline; a timeout leaves dependencies open and Shutdown may be
// called again after the handlers finish.
func (l *ConcurrencyLimiter) Shutdown(ctx context.Context) error {
	if l == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("worker drain context is required")
	}
	l.mu.Lock()
	l.closed = true
	if l.active == 0 && !l.done {
		close(l.drained)
		l.done = true
	}
	drained := l.drained
	l.mu.Unlock()
	select {
	case <-drained:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func workerUnavailable(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "1")
	http.Error(w, "worker temporarily unavailable", http.StatusServiceUnavailable)
}
