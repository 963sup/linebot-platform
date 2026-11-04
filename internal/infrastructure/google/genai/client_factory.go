// Package genai constructs the official Google Gen AI SDK client. Gemini
// Developer API and Vertex AI require different authentication strategies.
package genai

import (
	"context"
	"errors"
	"net/http"
	"strings"

	sdk "google.golang.org/genai"
)

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("GenAI context is required")
	}
	return ctx.Err()
}

// NewGeminiClient requires an explicit Gemini Developer API key. The caller
// owns its HTTP client/transport; Google Gen AI handles its own API requests.
func NewGeminiClient(ctx context.Context, apiKey string, httpClient *http.Client) (*sdk.Client, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("Gemini API key is required")
	}
	if httpClient == nil {
		return nil, errors.New("Gemini HTTP client is required")
	}
	return sdk.NewClient(ctx, &sdk.ClientConfig{
		Backend:    sdk.BackendGeminiAPI,
		APIKey:     apiKey,
		HTTPClient: httpClient,
	})
}

// NewVertexClient deliberately leaves HTTPClient nil. The SDK uses ADC/
// workload identity when enabled; passing an unauthenticated custom HTTP
// client would bypass correct Vertex AI authentication. Project and location
// must be configured explicitly. No service is enabled automatically.
func NewVertexClient(ctx context.Context, project, location string) (*sdk.Client, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(project) == "" || strings.TrimSpace(location) == "" {
		return nil, errors.New("Vertex AI project and location are required")
	}
	return sdk.NewClient(ctx, &sdk.ClientConfig{
		Backend:  sdk.BackendVertexAI,
		Project:  project,
		Location: location,
	})
}
