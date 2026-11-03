// Package openai constructs the official OpenAI Go SDK client without
// enabling or making any remote requests during application bootstrap.
package openai

import (
	"errors"
	"net/http"
	"strings"

	sdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// NewClient uses explicit credentials rather than silently importing an
// environment's unrelated OPENAI_API_KEY. The caller owns the HTTP client.
// API calls are still governed by the SDK's native retry and timeout policy;
// do not apply a second generic automatic retry layer.
func NewClient(apiKey string, httpClient *http.Client) (sdk.Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return sdk.Client{}, errors.New("OpenAI API key is required")
	}
	if httpClient == nil {
		return sdk.Client{}, errors.New("OpenAI HTTP client is required")
	}
	return sdk.NewClient(
		option.WithAPIKey(apiKey),
		option.WithHTTPClient(httpClient),
	), nil
}
