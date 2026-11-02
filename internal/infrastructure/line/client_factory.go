package line

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func NewMessagingClient(token string) (*messaging_api.MessagingApiAPI, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("LINE channel access token is required")
	}
	client := &http.Client{Timeout: 8 * time.Second}
	return messaging_api.NewMessagingApiAPI(token, messaging_api.WithHTTPClient(client))
}
