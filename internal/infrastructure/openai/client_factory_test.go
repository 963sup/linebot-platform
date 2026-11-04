package openai

import (
	"net/http"
	"testing"
	"time"
)

func TestNewClientValidatesConfigurationWithoutNetwork(t *testing.T) {
	if _, err := NewClient("", &http.Client{}); err == nil {
		t.Fatal("empty API key must fail")
	}
	if _, err := NewClient("test-key", nil); err == nil {
		t.Fatal("nil HTTP client must fail")
	}
	client, err := NewClient("test-key", &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	// Compile-time proof that the pinned native Responses API is accessible.
	_ = client.Responses.New
}
