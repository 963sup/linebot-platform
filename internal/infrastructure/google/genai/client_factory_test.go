package genai

import (
	"context"
	"errors"
	"net/http"
	"testing"

	sdk "google.golang.org/genai"
)

func TestNewGeminiClientValidatesAndKeepsBackendExplicit(t *testing.T) {
	var nilContext context.Context
	if _, err := NewGeminiClient(nilContext, "fake-key", &http.Client{}); err == nil {
		t.Fatal("missing context must fail")
	}
	if _, err := NewGeminiClient(context.Background(), "", &http.Client{}); err == nil {
		t.Fatal("missing API key must fail")
	}
	if _, err := NewGeminiClient(context.Background(), "fake-key", nil); err == nil {
		t.Fatal("missing HTTP client must fail")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewGeminiClient(ctx, "fake-key", &http.Client{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
	client, err := NewGeminiClient(context.Background(), "fake-key", &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	if client == nil || client.Models == nil || client.ClientConfig().Backend != sdk.BackendGeminiAPI {
		t.Fatal("native Gemini API client with explicit backend was not initialized")
	}
}

func TestNewVertexClientRequiresProjectAndLocation(t *testing.T) {
	if _, err := NewVertexClient(context.Background(), "", "us-central1"); err == nil {
		t.Fatal("Vertex AI project must be explicit")
	}
	if _, err := NewVertexClient(context.Background(), "test-project", ""); err == nil {
		t.Fatal("Vertex AI location must be explicit")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewVertexClient(ctx, "test-project", "us-central1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
}
