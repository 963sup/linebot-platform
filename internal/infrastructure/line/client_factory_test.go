package line

import (
	"net/http"
	"testing"
	"time"
)

func TestNewMessagingClientWithHTTPClient(t *testing.T) {
	if _, err := NewMessagingClientWithHTTPClient("", &http.Client{}); err == nil {
		t.Fatal("missing LINE token must be rejected")
	}
	if _, err := NewMessagingClientWithHTTPClient("token", nil); err == nil {
		t.Fatal("nil HTTP client must be rejected")
	}
	if _, err := NewMessagingClientWithHTTPClient("token", &http.Client{Timeout: time.Second}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewMessagingClient("token"); err != nil {
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
