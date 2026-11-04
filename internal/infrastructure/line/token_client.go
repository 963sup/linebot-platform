package line

import (
	"errors"
	"net/http"

	"github.com/line/line-bot-sdk-go/v8/linebot/channel_access_token"
)

// Token API requests use OAuth client credentials, not a LINE Messaging
// Channel access token. Do not leak this client into the webhook runtime.
func NewChannelTokenClientWithHTTPClient(httpClient *http.Client) (*channel_access_token.ChannelAccessTokenAPI, error) {
	if httpClient == nil { return nil, errors.New("LINE token HTTP client is required") }
	return channel_access_token.NewChannelAccessTokenAPI(channel_access_token.WithHTTPClient(httpClient))
}
