package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	genaisdk "google.golang.org/genai"
)

func TestAIConfigRedactsCredentials(t *testing.T) {
	cfg := AIConfig{OpenAIAPIKey: "openai-secret", GeminiAPIKey: "gemini-secret"}
	for _, rendered := range []string{fmt.Sprint(cfg), fmt.Sprintf("%#v", cfg)} {
		if rendered == "" || rendered == "openai-secret" || rendered == "gemini-secret" {
			t.Fatalf("unexpected configuration rendering %q", rendered)
		}
		if rendered != "[redacted AI configuration]" && rendered != "bootstrap.AIConfig{redacted}" {
			t.Fatalf("AI configuration was not redacted: %q", rendered)
		}
	}
}

func TestNewAIClientsBuildsOnlyExplicitProviders(t *testing.T) {
	credentialsPath := filepath.Join(t.TempDir(), "adc.json")
	credentials := `{"type":"authorized_user","client_id":"test-client","client_secret":"test-secret","refresh_token":"test-refresh"}`
	if err := os.WriteFile(credentialsPath, []byte(credentials), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credentialsPath)

	tests := []struct {
		name   string
		config AIConfig
		openAI bool
		gemini bool
		vertex bool
	}{
		{name: "none"},
		{name: "openai", config: AIConfig{OpenAIAPIKey: "test-openai"}, openAI: true},
		{name: "gemini", config: AIConfig{GeminiAPIKey: "test-gemini"}, gemini: true},
		{name: "vertex", config: AIConfig{VertexProject: "test-project", VertexLocation: "asia-east1"}, vertex: true},
		{
			name: "all",
			config: AIConfig{
				OpenAIAPIKey: "test-openai", GeminiAPIKey: "test-gemini",
				VertexProject: "test-project", VertexLocation: "asia-east1",
			},
			openAI: true, gemini: true, vertex: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clients, err := NewAIClients(context.Background(), tt.config)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(clients.Close)
			if (clients.OpenAI != nil) != tt.openAI ||
				(clients.Gemini != nil) != tt.gemini ||
				(clients.Vertex != nil) != tt.vertex {
				t.Fatalf("unexpected clients: OpenAI=%t Gemini=%t Vertex=%t", clients.OpenAI != nil, clients.Gemini != nil, clients.Vertex != nil)
			}
			if clients.Gemini != nil && clients.Gemini.ClientConfig().Backend != genaisdk.BackendGeminiAPI {
				t.Fatal("Gemini client uses the wrong backend")
			}
			if clients.Vertex != nil && clients.Vertex.ClientConfig().Backend != genaisdk.BackendVertexAI {
				t.Fatal("Vertex client uses the wrong backend")
			}
		})
	}
}

func TestNewAIClientsRejectsInvalidComposition(t *testing.T) {
	var nilContext context.Context
	if _, err := NewAIClients(nilContext, AIConfig{}); err == nil {
		t.Fatal("nil context must fail")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewAIClients(canceled, AIConfig{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
	for _, cfg := range []AIConfig{
		{VertexProject: "test-project"},
		{VertexLocation: "asia-east1"},
	} {
		if _, err := NewAIClients(context.Background(), cfg); err == nil {
			t.Fatal("partial Vertex configuration must fail")
		}
	}
	if _, err := NewAIClients(context.Background(), AIConfig{OpenAIAPIKey: "test", Timeout: time.Minute + time.Second}); err == nil {
		t.Fatal("invalid outbound timeout must fail")
	}
}

func TestAIClientsCloseIsIdempotent(t *testing.T) {
	clients, err := NewAIClients(context.Background(), AIConfig{OpenAIAPIKey: "test-openai", GeminiAPIKey: "test-gemini"})
	if err != nil {
		t.Fatal(err)
	}
	clients.Close()
	clients.Close()
	var nilClients *AIClients
	nilClients.Close()
}
