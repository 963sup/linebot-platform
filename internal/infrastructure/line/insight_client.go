package line

import (
	"errors"
	"net/http"
	"strings"

	"github.com/line/line-bot-sdk-go/v8/linebot/insight"
)

// NewInsightClientWithHTTPClient uses the official separate Insights SDK.
func NewInsightClientWithHTTPClient(token string, client *http.Client) (*insight.InsightAPI, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("LINE channel access token is required")
	}
	if client == nil {
		return nil, errors.New("LINE HTTP client is required")
	}
	return insight.NewInsightAPI(token, insight.WithHTTPClient(client))
}
