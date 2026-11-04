// Package oauth contains a Google-only Authorization Code + PKCE helper.
// It does not expose an HTTP endpoint or persist any user credential.
package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
)

// allowedScopes is a hard boundary: a callback or chat message must not be
// able to request arbitrary Google account permissions.
var allowedScopes = map[string]bool{
	"openid": true,
	"email":  true,
	"https://www.googleapis.com/auth/calendar.events.readonly":       true,
	"https://www.googleapis.com/auth/calendar.events":                true,
	"https://www.googleapis.com/auth/calendar.calendarlist.readonly": true,
	"https://www.googleapis.com/auth/calendar.acls":                  true,
	"https://www.googleapis.com/auth/tasks.readonly":                 true,
	"https://www.googleapis.com/auth/tasks":                          true,
	"https://www.googleapis.com/auth/forms.body.readonly":            true,
	"https://www.googleapis.com/auth/forms.body":                     true,
	"https://www.googleapis.com/auth/forms.responses.readonly":       true,
}

type Settings struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// VerifiedUser uses Google's stable subject as identity, never an email address.
type VerifiedUser struct {
	Subject       string
	VerifiedEmail string
}

type ExchangeResult struct {
	User  VerifiedUser
	Token *oauth2.Token
}

type idVerifier func(context.Context, string, string) (*idtoken.Payload, error)

type Flow struct {
	config oauth2.Config
	verify idVerifier
}

// NewFlow uses the official Google OAuth endpoint; no caller-supplied token URL.
func NewFlow(s Settings) (*Flow, error) {
	if strings.TrimSpace(s.ClientID) == "" || strings.TrimSpace(s.ClientSecret) == "" {
		return nil, errors.New("Google OAuth client ID and secret are required")
	}
	u, err := url.Parse(s.RedirectURL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil ||
		u.Fragment != "" || u.RawQuery != "" {
		return nil, errors.New("Google OAuth callback must be a fixed HTTPS URL")
	}
	return &Flow{config: oauth2.Config{
		ClientID:     s.ClientID,
		ClientSecret: s.ClientSecret,
		RedirectURL:  s.RedirectURL,
		Endpoint:     google.Endpoint,
	}, verify: idtoken.Validate}, nil
}

// RandomState creates an opaque nonce for state or OpenID Connect nonce.
// Persist only its digest as an indexing key, and consume the state exactly once.
func RandomState() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", errors.New("unable to generate OAuth state")
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// AuthorizationURL must receive state and PKCE verifier from a trusted,
// single-use durable state store. The caller decides consented scopes.
func (f *Flow) AuthorizationURL(state, verifier, nonce string, scopes []string) (string, error) {
	if f == nil || len(state) < 32 || len(nonce) < 32 || len(verifier) < 43 || len(verifier) > 128 {
		return "", errors.New("OAuth state, nonce or PKCE verifier is invalid")
	}
	if len(scopes) == 0 {
		return "", errors.New("OAuth scopes required")
	}
	seen := map[string]bool{"openid": true}
	granted := []string{"openid"}
	for _, s := range scopes {
		if !allowedScopes[s] {
			return "", errors.New("unsupported Google OAuth scope")
		}
		if !seen[s] {
			seen[s] = true
			granted = append(granted, s)
		}
	}
	config := f.config
	config.Scopes = granted
	return config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("nonce", nonce),
	), nil
}

// ExchangeAndVerify verifies a Google-signed ID Token and the original nonce.
// The returned token is sensitive; a caller must securely persist it before
// declaring any account link active. Raw provider errors are not returned.
func (f *Flow) ExchangeAndVerify(ctx context.Context, code, verifier, nonce string) (*ExchangeResult, error) {
	if f == nil || ctx == nil {
		return nil, errors.New("OAuth flow and context required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(code) == "" || len(verifier) < 43 || len(verifier) > 128 || len(nonce) < 32 {
		return nil, errors.New("OAuth code, PKCE or nonce invalid")
	}
	token, err := f.config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("Google OAuth code exchange failed")
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok || raw == "" {
		return nil, errors.New("Google ID token missing")
	}
	user, err := verifyUser(ctx, raw, f.config.ClientID, nonce, f.verify)
	if err != nil {
		return nil, errors.New("Google ID token verification failed")
	}
	return &ExchangeResult{User: user, Token: token}, nil
}

func verifyUser(ctx context.Context, raw, audience, nonce string, verify idVerifier) (VerifiedUser, error) {
	if ctx == nil || verify == nil || raw == "" || audience == "" || nonce == "" {
		return VerifiedUser{}, errors.New("invalid ID token verifier inputs")
	}
	if err := ctx.Err(); err != nil {
		return VerifiedUser{}, err
	}
	p, err := verify(ctx, raw, audience)
	if err != nil || p == nil {
		return VerifiedUser{}, errors.New("ID token invalid")
	}
	if p.Issuer != "accounts.google.com" && p.Issuer != "https://accounts.google.com" {
		return VerifiedUser{}, errors.New("ID token issuer invalid")
	}
	if p.Audience != audience || p.Subject == "" || p.Expires <= time.Now().Unix() {
		return VerifiedUser{}, errors.New("ID token audience, subject or expiry invalid")
	}
	gotNonce, _ := p.Claims["nonce"].(string)
	if gotNonce != nonce {
		return VerifiedUser{}, errors.New("ID token nonce invalid")
	}
	user := VerifiedUser{Subject: p.Subject}
	if verified, _ := p.Claims["email_verified"].(bool); verified {
		user.VerifiedEmail, _ = p.Claims["email"].(string)
	}
	return user, nil
}
