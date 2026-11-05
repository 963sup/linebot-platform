package line

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
	"github.com/line/line-bot-sdk-go/v8/linebot/channel_access_token"
	"github.com/line/line-bot-sdk-go/v8/linebot/insight"
	"github.com/line/line-bot-sdk-go/v8/linebot/manage_audience"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

type closeOnEOFTransport struct{ next http.RoundTripper }

func (t closeOnEOFTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.next.RoundTrip(request)
	if response != nil && response.Body != nil {
		response.Body = &closeOnEOFBody{ReadCloser: response.Body}
	}
	return response, err
}

type closeOnEOFBody struct{ io.ReadCloser }

func (b *closeOnEOFBody) Read(data []byte) (int, error) {
	n, err := b.ReadCloser.Read(data)
	if err == io.EOF {
		_ = b.ReadCloser.Close()
	}
	return n, err
}

func managedLINEHTTPClient(client *http.Client) (*http.Client, error) {
	if client == nil {
		return nil, errors.New("LINE HTTP client is required")
	}
	if client.Timeout <= 0 {
		return nil, errors.New("LINE HTTP client must have a positive timeout")
	}
	copy := *client
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	copy.Transport = closeOnEOFTransport{next: transport}
	return &copy, nil
}

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
	managed, err := managedLINEHTTPClient(httpClient)
	if err != nil {
		return nil, err
	}
	return messaging_api.NewMessagingApiAPI(token, messaging_api.WithHTTPClient(managed))
}

// NewMessagingBlobClientWithHTTPClient constructs the separate official LINE
// Blob API client used for message content and Rich Menu image endpoints.
// It shares the caller-owned managed transport with the JSON Messaging client.
func NewMessagingBlobClientWithHTTPClient(token string, httpClient *http.Client) (*messaging_api.MessagingApiBlobAPI, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("LINE channel access token is required")
	}
	managed, err := managedLINEHTTPClient(httpClient)
	if err != nil {
		return nil, err
	}
	return messaging_api.NewMessagingApiBlobAPI(token, messaging_api.WithBlobHTTPClient(managed))
}

// NewAudienceClientWithHTTPClient constructs the official LINE audience API client.
func NewAudienceClientWithHTTPClient(token string, httpClient *http.Client) (*manage_audience.ManageAudienceAPI, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("LINE channel access token is required")
	}
	managed, err := managedLINEHTTPClient(httpClient)
	if err != nil {
		return nil, err
	}
	return manage_audience.NewManageAudienceAPI(token, manage_audience.WithHTTPClient(managed))
}

// NewAudienceBlobClientWithHTTPClient constructs the official LINE audience blob API client.
func NewAudienceBlobClientWithHTTPClient(token string, httpClient *http.Client) (*manage_audience.ManageAudienceBlobAPI, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("LINE channel access token is required")
	}
	managed, err := managedLINEHTTPClient(httpClient)
	if err != nil {
		return nil, err
	}
	return manage_audience.NewManageAudienceBlobAPI(token, manage_audience.WithBlobHTTPClient(managed))
}

// NewInsightClientWithHTTPClient constructs the separate official LINE Insights API client.
func NewInsightClientWithHTTPClient(token string, httpClient *http.Client) (*insight.InsightAPI, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("LINE channel access token is required")
	}
	managed, err := managedLINEHTTPClient(httpClient)
	if err != nil {
		return nil, err
	}
	return insight.NewInsightAPI(token, insight.WithHTTPClient(managed))
}

// NewChannelTokenClientWithHTTPClient constructs the LINE channel token API client.
// Token API requests use OAuth client credentials, not a channel access token.
func NewChannelTokenClientWithHTTPClient(httpClient *http.Client) (*channel_access_token.ChannelAccessTokenAPI, error) {
	managed, err := managedLINEHTTPClient(httpClient)
	if err != nil {
		return nil, err
	}
	return channel_access_token.NewChannelAccessTokenAPI(channel_access_token.WithHTTPClient(managed))
}
