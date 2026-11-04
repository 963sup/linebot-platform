// Package tasksauth validates Google-signed OIDC ID tokens for a private
// Cloud Run worker. It does not trust Cloud Tasks headers as authentication.
package tasksauth

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"google.golang.org/api/idtoken"
)

type Verifier struct {
	audience string
	email    string
}

func NewVerifier(audience, serviceAccountEmail string) (*Verifier, error) {
	u, err := url.Parse(audience)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil ||
		u.Fragment != "" || u.RawQuery != "" || u.Path != "" && u.Path != "/" {
		return nil, errors.New("OIDC audience must be an HTTPS Cloud Run service origin")
	}
	if strings.TrimSpace(serviceAccountEmail) == "" {
		return nil, errors.New("OIDC invoker email is required")
	}
	return &Verifier{audience: audience, email: serviceAccountEmail}, nil
}

func (v *Verifier) Verify(ctx context.Context, token string) error {
	if v == nil || strings.TrimSpace(token) == "" {
		return errors.New("missing OIDC ID token")
	}
	payload, err := idtoken.Validate(ctx, token, v.audience)
	if err != nil {
		return errors.New("invalid Google-signed OIDC token")
	}
	email, _ := payload.Claims["email"].(string)
	if !strings.EqualFold(email, v.email) {
		return errors.New("OIDC invoker identity mismatch")
	}
	return nil
}
