package httpclient

import (
	"net/http"
	"testing"
	"time"
)

func TestConfigValidation(t *testing.T) {
	for _, d := range []time.Duration{-time.Second, 0, 61 * time.Second} {
		if _, err := New(Config{Timeout: d}, nil); err == nil {
			t.Fatalf("expected invalid timeout %v to fail", d)
		}
	}
	if _, err := New(Config{Timeout: DefaultTimeout}, func(http.RoundTripper) http.RoundTripper { return nil }); err == nil {
		t.Fatal("expected nil transport decorator to fail")
	}
}

func TestManagedClientOwnsIndependentTransport(t *testing.T) {
	a, err := New(Config{Timeout: 4 * time.Second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := New(Config{Timeout: 5 * time.Second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if a.Client.Timeout != 4*time.Second || b.Client.Timeout != 5*time.Second {
		t.Fatal("configured timeout was not preserved")
	}
	if a.transport == b.transport {
		t.Fatal("outbound transports must not share mutable configuration")
	}
	if a.transport.MaxIdleConnsPerHost != 16 {
		t.Fatalf("unexpected idle pool size %d", a.transport.MaxIdleConnsPerHost)
	}
	if a.Client.Transport != a.transport {
		t.Fatal("unwrapped client must use its owned transport")
	}
	a.Close()
	a.Close() // idempotent
}
