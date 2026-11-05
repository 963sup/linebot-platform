// Package credentiallifecycle owns protected operator workflows for provider credentials.
package credentiallifecycle

import (
	"context"
	"errors"
	"strings"
	"time"
)

type ClientCredential struct{ ClientID, ClientSecret string }
type AssertionCredential struct{ Type, Assertion string }

// IssuedCredential contains secret material. It must not be logged or included in errors.
type IssuedCredential struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
}

type Verification struct {
	Active    bool
	ClientID  string
	ExpiresAt time.Time
}

type Gateway interface {
	IssueStatelessWithClient(context.Context, string, ClientCredential) (IssuedCredential, error)
	IssueStatelessWithAssertion(context.Context, string, AssertionCredential) (IssuedCredential, error)
	ValidKeyIDs(context.Context, AssertionCredential) ([]string, error)
	IssueJWT(context.Context, string, AssertionCredential) (IssuedCredential, error)
	VerifyJWT(context.Context, string) (Verification, error)
	RevokeJWT(context.Context, ClientCredential, string) error
	IssueShortLived(context.Context, string, ClientCredential) (IssuedCredential, error)
	VerifyShortLived(context.Context, string) (Verification, error)
	RevokeShortLived(context.Context, string) error
}

type Service struct{ gateway Gateway }

func New(gateway Gateway) (*Service, error) {
	if gateway == nil {
		return nil, errors.New("credential gateway is required")
	}
	return &Service{gateway: gateway}, nil
}

func (s *Service) IssueStatelessWithClient(ctx context.Context, grant string, credential ClientCredential) (IssuedCredential, error) {
	if err := validate(ctx, grant, credential.ClientID, credential.ClientSecret); err != nil {
		return IssuedCredential{}, err
	}
	return s.gateway.IssueStatelessWithClient(ctx, grant, credential)
}

func (s *Service) IssueStatelessWithAssertion(ctx context.Context, grant string, credential AssertionCredential) (IssuedCredential, error) {
	if err := validate(ctx, grant, credential.Type, credential.Assertion); err != nil {
		return IssuedCredential{}, err
	}
	return s.gateway.IssueStatelessWithAssertion(ctx, grant, credential)
}

func (s *Service) ValidKeyIDs(ctx context.Context, credential AssertionCredential) ([]string, error) {
	if err := validate(ctx, credential.Type, credential.Assertion); err != nil {
		return nil, err
	}
	return s.gateway.ValidKeyIDs(ctx, credential)
}

func (s *Service) IssueJWT(ctx context.Context, grant string, credential AssertionCredential) (IssuedCredential, error) {
	if err := validate(ctx, grant, credential.Type, credential.Assertion); err != nil {
		return IssuedCredential{}, err
	}
	return s.gateway.IssueJWT(ctx, grant, credential)
}

func (s *Service) VerifyJWT(ctx context.Context, token string) (Verification, error) {
	if err := validate(ctx, token); err != nil {
		return Verification{}, err
	}
	return s.gateway.VerifyJWT(ctx, token)
}

func (s *Service) RevokeJWT(ctx context.Context, credential ClientCredential, token string) error {
	if err := validate(ctx, credential.ClientID, credential.ClientSecret, token); err != nil {
		return err
	}
	return s.gateway.RevokeJWT(ctx, credential, token)
}

func (s *Service) IssueShortLived(ctx context.Context, grant string, credential ClientCredential) (IssuedCredential, error) {
	if err := validate(ctx, grant, credential.ClientID, credential.ClientSecret); err != nil {
		return IssuedCredential{}, err
	}
	return s.gateway.IssueShortLived(ctx, grant, credential)
}

func (s *Service) VerifyShortLived(ctx context.Context, token string) (Verification, error) {
	if err := validate(ctx, token); err != nil {
		return Verification{}, err
	}
	return s.gateway.VerifyShortLived(ctx, token)
}

func (s *Service) RevokeShortLived(ctx context.Context, token string) error {
	if err := validate(ctx, token); err != nil {
		return err
	}
	return s.gateway.RevokeShortLived(ctx, token)
}

func validate(ctx context.Context, fields ...string) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, field := range fields {
		if field == "" || field != strings.TrimSpace(field) {
			return errors.New("credential request contains a missing or invalid field")
		}
	}
	return nil
}
