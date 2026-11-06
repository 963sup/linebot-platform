package line

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type factoryTransport func(*http.Request) (*http.Response, error)

func (f factoryTransport) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

type trackedBody struct {
	io.Reader
	closed bool
}

func (b *trackedBody) Close() error { b.closed = true; return nil }

func TestNewMessagingClientWithHTTPClient(t *testing.T) {
	if _, err := NewMessagingClientWithHTTPClient("", &http.Client{}); err == nil {
		t.Fatal("missing LINE token must be rejected")
	}
	if _, err := NewMessagingClientWithHTTPClient("token", nil); err == nil {
		t.Fatal("nil HTTP client must be rejected")
	}
	if _, err := NewMessagingClientWithHTTPClient("token", &http.Client{}); err == nil {
		t.Fatal("LINE client without a timeout must be rejected")
	}
	if _, err := NewMessagingClientWithHTTPClient("token", &http.Client{Timeout: time.Second}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewMessagingBlobClientWithHTTPClient("", &http.Client{}); err == nil {
		t.Fatal("missing LINE token must be rejected for blob client")
	}
	if _, err := NewMessagingBlobClientWithHTTPClient("token", nil); err == nil {
		t.Fatal("nil HTTP client must be rejected for blob client")
	}
	if _, err := NewMessagingBlobClientWithHTTPClient("token", &http.Client{Timeout: time.Second}); err != nil {
		t.Fatal(err)
	}
}

func TestLINETransportClosesProviderBodyAtEOF(t *testing.T) {
	body := &trackedBody{Reader: strings.NewReader("provider response")}
	client, err := managedLINEHTTPClient(&http.Client{
		Timeout: time.Second,
		Transport: factoryTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusBadRequest, Body: body}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(mustRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(response.Body); err != nil {
		t.Fatal(err)
	}
	if !body.closed {
		t.Fatal("provider response body was not closed after EOF")
	}
}

func mustRequest(t *testing.T) *http.Request {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, "https://example.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	return request
}
