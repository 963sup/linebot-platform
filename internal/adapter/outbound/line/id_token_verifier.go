package line

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"encoding/hex"

	"github.com/963sup/linebot-platform/internal/application/usecases/attendance"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

const (
	lineIDTokenIssuer       = "https://access.line.me"
	lineIDTokenVerifyURL    = "https://api.line.me/oauth2/v2.1/verify"
	maxIDTokenResponseBytes = 64 << 10
)

type IDTokenVerifier struct {
	client   *http.Client
	endpoint string
	clientID string
	now      func() time.Time
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

func (v *IDTokenVerifier) Verify(ctx context.Context, token string) (attendance.VerifiedPrincipal, error) {
	if ctx == nil {
		return attendance.VerifiedPrincipal{}, attendance.ErrUnauthenticated
	}
	if err := ctx.Err(); err != nil {
		return attendance.VerifiedPrincipal{}, err
	}
	if strings.TrimSpace(token) == "" {
		return attendance.VerifiedPrincipal{}, attendance.ErrUnauthenticated
	}
	form := url.Values{"id_token": {token}, "client_id": {v.clientID}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return attendance.VerifiedPrincipal{}, attendance.ErrIdentityUnavailable
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := v.client.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return attendance.VerifiedPrincipal{}, ctxErr
		}
		return attendance.VerifiedPrincipal{}, attendance.ErrIdentityUnavailable
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxIDTokenResponseBytes+1))
	if err != nil {
		return attendance.VerifiedPrincipal{}, attendance.ErrIdentityUnavailable
	}
	if len(body) > maxIDTokenResponseBytes {
		return attendance.VerifiedPrincipal{}, attendance.ErrIdentityUnavailable
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if response.StatusCode >= 500 || response.StatusCode == http.StatusTooManyRequests {
			return attendance.VerifiedPrincipal{}, attendance.ErrIdentityUnavailable
		}
		return attendance.VerifiedPrincipal{}, attendance.ErrUnauthenticated
	}
	var claims lineIDTokenClaims
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	if err := decoder.Decode(&claims); err != nil {
		return attendance.VerifiedPrincipal{}, attendance.ErrIdentityUnavailable
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return attendance.VerifiedPrincipal{}, attendance.ErrIdentityUnavailable
	}
	expiresAt := time.Unix(claims.Expires, 0)
	if claims.Issuer != lineIDTokenIssuer || claims.Audience != v.clientID || claims.Expires <= 0 || !expiresAt.After(v.now()) || strings.TrimSpace(claims.Subject) == "" {
		return attendance.VerifiedPrincipal{}, attendance.ErrUnauthenticated
	}
	return attendance.VerifiedPrincipal{ParticipantID: participantID(v.clientID, claims.Subject), ValidatedAt: v.now().UTC()}, nil
}

func participantID(clientID, subject string) valueobjects.ParticipantID {
	hash := sha256.New()
	_, _ = hash.Write([]byte("line-participant-v1"))
	for _, value := range []string{lineIDTokenIssuer, clientID, subject} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		_, _ = hash.Write(size[:])
		_, _ = hash.Write([]byte(value))
	}
	return valueobjects.ParticipantID("line:v1:" + hex.EncodeToString(hash.Sum(nil)))
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("trailing LINE verification response")
	}
	return nil
}

var _ attendance.PrincipalVerifier = (*IDTokenVerifier)(nil)
