package line

import (
	"errors"
	"net/http"
	"strings"

	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
	"github.com/line/line-bot-sdk-go/v8/linebot/channel_access_token"
	"github.com/line/line-bot-sdk-go/v8/linebot/insight"
	"github.com/line/line-bot-sdk-go/v8/linebot/manage_audience"
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

// NewAudienceClientWithHTTPClient constructs the official LINE audience API client.
func NewAudienceClientWithHTTPClient(token string, httpClient *http.Client) (*manage_audience.ManageAudienceAPI, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("LINE channel access token is required")
	}
	if httpClient == nil {
		return nil, errors.New("LINE HTTP client is required")
	}
	return manage_audience.NewManageAudienceAPI(token, manage_audience.WithHTTPClient(httpClient))
}

// NewAudienceBlobClientWithHTTPClient constructs the official LINE audience blob API client.
func NewAudienceBlobClientWithHTTPClient(token string, httpClient *http.Client) (*manage_audience.ManageAudienceBlobAPI, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("LINE channel access token is required")
	}
	if httpClient == nil {
		return nil, errors.New("LINE HTTP client is required")
	}
	return manage_audience.NewManageAudienceBlobAPI(token, manage_audience.WithBlobHTTPClient(httpClient))
}

// NewInsightClientWithHTTPClient constructs the separate official LINE Insights API client.
func NewInsightClientWithHTTPClient(token string, httpClient *http.Client) (*insight.InsightAPI, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("LINE channel access token is required")
	}
	if httpClient == nil {
		return nil, errors.New("LINE HTTP client is required")
	}
	return insight.NewInsightAPI(token, insight.WithHTTPClient(httpClient))
}

// NewChannelTokenClientWithHTTPClient constructs the LINE channel token API client.
// Token API requests use OAuth client credentials, not a channel access token.
func NewChannelTokenClientWithHTTPClient(httpClient *http.Client) (*channel_access_token.ChannelAccessTokenAPI, error) {
	if httpClient == nil {
		return nil, errors.New("LINE token HTTP client is required")
	}
	return channel_access_token.NewChannelAccessTokenAPI(channel_access_token.WithHTTPClient(httpClient))
}
