// Package httpclient manages owned outbound HTTP transports shared by SDK clients.
package httpclient

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

const (
	DefaultTimeout = 8 * time.Second
	MaxTimeout     = 60 * time.Second
)

// Config is the outbound HTTP policy used when constructing one owned client.
type Config struct {
	Timeout time.Duration
}

// Managed owns its transport and idle connections. Clients are reused for the
// lifetime of an entrypoint, not recreated for each API call.
type Managed struct {
	Client    *http.Client
	transport *http.Transport
	once      sync.Once
}

// New clones Go's default transport (without modifying global defaults).
// decorate may install metrics/tracing, and must preserve the underlying
// transport's context and error behavior. New never performs network I/O.
func New(cfg Config, decorate func(http.RoundTripper) http.RoundTripper) (*Managed, error) {
	if cfg.Timeout <= 0 || cfg.Timeout > MaxTimeout {
		return nil, errors.New("outbound HTTP timeout must be > 0 and <= 60s")
	}
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("default HTTP transport must be *http.Transport")
	}
	transport := base.Clone()
	transport.MaxIdleConnsPerHost = 16

	var roundTripper http.RoundTripper = transport
	if decorate != nil {
		roundTripper = decorate(transport)
		if roundTripper == nil {
			transport.CloseIdleConnections()
			return nil, errors.New("HTTP transport decorator returned nil")
		}
	}
	return &Managed{
		Client:    &http.Client{Timeout: cfg.Timeout, Transport: roundTripper},
		transport: transport,
	}, nil
}

// Close releases idle connections owned by this client. Call only after the
// server/worker has stopped accepting and processing its requests.
func (m *Managed) Close() {
	if m == nil {
		return
	}
	m.once.Do(func() {
		if m.transport != nil {
			m.transport.CloseIdleConnections()
		}
	})
}
