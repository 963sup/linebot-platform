package line

import (
	"errors"
	"net/http"
	"strings"

	"github.com/line/line-bot-sdk-go/v8/linebot/manage_audience"
)

// NewAudienceClientWithHTTPClient uses the official audience SDK and an owned transport.
func NewAudienceClientWithHTTPClient(token string, client *http.Client) (*manage_audience.ManageAudienceAPI, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("LINE channel access token is required")
	}
	if client == nil {
		return nil, errors.New("LINE HTTP client is required")
	}
	return manage_audience.NewManageAudienceAPI(token, manage_audience.WithHTTPClient(client))
}
