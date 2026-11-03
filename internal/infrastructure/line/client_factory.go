package line

import (
	"errors"
	"net/http"
	"strings"

	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

// NewMessagingClient preserves the existing convenience API.
// Long-running processes should inject a managed HTTP client and close its
// owned transport after shutdown.
func NewMessagingClient(token string) (*messaging_api.MessagingApiAPI, error) {
	return NewMessagingClientWithHTTPClient(token, &http.Client{Timeout: httpclient.DefaultTimeout})
}

// NewMessagingClientWithHTTPClient returns the native LINE SDK API client.
// The caller owns the supplied HTTP client and must not mutate it after setup.
func NewMessagingClientWithHTTPClient(token string, httpClient *http.Client) (*messaging_api.MessagingApiAPI, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("LINE channel access token is required")
	}
	if httpClient == nil {
		return nil, errors.New("LINE HTTP client is required")
	}
	return messaging_api.NewMessagingApiAPI(token, messaging_api.WithHTTPClient(httpClient))
}

// NewMessagingBlobClientWithHTTPClient constructs the separate official LINE
// Blob API client used for message content and Rich Menu image endpoints.
// It shares the caller-owned managed transport with the JSON Messaging client.
func NewMessagingBlobClientWithHTTPClient(token string, httpClient *http.Client) (*messaging_api.MessagingApiBlobAPI, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("LINE channel access token is required")
	}
	if httpClient == nil {
		return nil, errors.New("LINE HTTP client is required")
	}
	return messaging_api.NewMessagingApiBlobAPI(token, messaging_api.WithBlobHTTPClient(httpClient))
}
