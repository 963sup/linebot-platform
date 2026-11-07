package line

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	linesession "github.com/963sup/linebot-platform/internal/application/usecases/line-session"
)

const (
	lineIDTokenIssuer       = "https://access.line.me"
	lineIDTokenVerifyURL    = "https://api.line.me/oauth2/v2.1/verify"
	maxIDTokenResponseBytes = 64 << 10
)

type IDTokenVerifier struct {
	client             *http.Client
	endpoint, clientID string
	now                func() time.Time
}

func NewIDTokenVerifier(client *http.Client, clientID string) (*IDTokenVerifier, error) {
	return newIDTokenVerifier(client, lineIDTokenVerifyURL, clientID)
}
func newIDTokenVerifier(client *http.Client, endpoint, clientID string) (*IDTokenVerifier, error) {
	parsed, err := url.Parse(endpoint)
	if client == nil || client.Timeout <= 0 || err != nil || parsed == nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || strings.TrimSpace(clientID) == "" {
		return nil, errors.New("LINE ID token verifier configuration is required")
	}
	return &IDTokenVerifier{client: client, endpoint: endpoint, clientID: clientID, now: time.Now}, nil
}

type lineIDTokenClaims struct {
	Issuer   string `json:"iss"`
	Audience string `json:"aud"`
	Expires  int64  `json:"exp"`
	Subject  string `json:"sub"`
}

func (v *IDTokenVerifier) Verify(ctx context.Context, token string) (linesession.Identity, error) {
	if ctx == nil || strings.TrimSpace(token) == "" {
		return linesession.Identity{}, linesession.ErrUnauthenticated
	}
	if err := ctx.Err(); err != nil {
		return linesession.Identity{}, err
	}
	form := url.Values{"id_token": {token}, "client_id": {v.clientID}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return linesession.Identity{}, linesession.ErrUnavailable
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := v.client.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return linesession.Identity{}, ctxErr
		}
		return linesession.Identity{}, linesession.ErrUnavailable
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxIDTokenResponseBytes+1))
	if err != nil || len(body) > maxIDTokenResponseBytes {
		return linesession.Identity{}, linesession.ErrUnavailable
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if response.StatusCode >= 500 || response.StatusCode == http.StatusTooManyRequests {
			return linesession.Identity{}, linesession.ErrUnavailable
		}
		return linesession.Identity{}, linesession.ErrUnauthenticated
	}
	var claims lineIDTokenClaims
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	if err := decoder.Decode(&claims); err != nil || ensureJSONEOF(decoder) != nil {
		return linesession.Identity{}, linesession.ErrUnavailable
	}
	expiresAt := time.Unix(claims.Expires, 0)
	if claims.Issuer != lineIDTokenIssuer || claims.Audience != v.clientID || claims.Expires <= 0 || !expiresAt.After(v.now()) || strings.TrimSpace(claims.Subject) == "" {
		return linesession.Identity{}, linesession.ErrUnauthenticated
	}
	return linesession.Identity{Subject: claims.Subject, ValidatedAt: v.now().UTC()}, nil
}
func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("trailing LINE verification response")
	}
	return nil
}

var _ linesession.Verifier = (*IDTokenVerifier)(nil)
