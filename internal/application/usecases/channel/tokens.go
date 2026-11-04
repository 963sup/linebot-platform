package channel

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

// TokenCommand must be populated through a protected operator-only boundary.
// Never log this struct, stringify it, or include its values in errors.
type TokenCommand struct {
	Action              string
	GrantType           string
	ClientID            string
	ClientSecret        string
	ClientAssertionType string
	ClientAssertion     string
	AccessToken         string
}

type TokenOperations struct{ manager outbound.ChannelTokenManager }

func NewTokenOperations(m outbound.ChannelTokenManager) (*TokenOperations, error) {
	if m == nil {
		return nil, errors.New("channel token manager is required")
	}
	return &TokenOperations{manager: m}, nil
}

func ValidateTokenCommand(c TokenCommand) error {
	fields := []string{c.GrantType, c.ClientID, c.ClientSecret, c.ClientAssertionType, c.ClientAssertion, c.AccessToken}
	for _, value := range fields {
		if value != strings.TrimSpace(value) {
			return errors.New("invalid channel token request field")
		}
	}
	bothAssertion := c.ClientAssertionType != "" && c.ClientAssertion != ""
	bothSecret := c.ClientID != "" && c.ClientSecret != ""
	switch c.Action {
	case "issue-stateless":
		if c.GrantType == "" || c.AccessToken != "" || (bothAssertion == bothSecret) ||
			(c.ClientAssertionType != "" || c.ClientAssertion != "") != bothAssertion ||
			(c.ClientID != "" || c.ClientSecret != "") != bothSecret {
			return errors.New("stateless issuance requires exactly one complete credential mode")
		}
	case "key-ids":
		if !bothAssertion || c.GrantType != "" || c.ClientID != "" || c.ClientSecret != "" || c.AccessToken != "" {
			return errors.New("key IDs require only client assertion fields")
		}
	case "issue-jwt":
		if c.GrantType == "" || !bothAssertion || c.ClientID != "" || c.ClientSecret != "" || c.AccessToken != "" {
			return errors.New("JWT issuance requires grant type and client assertion")
		}
	case "verify-jwt", "verify-short", "revoke-short":
		if c.AccessToken == "" || c.GrantType != "" || c.ClientID != "" || c.ClientSecret != "" || c.ClientAssertionType != "" || c.ClientAssertion != "" {
			return errors.New("token operation requires only an access token")
		}
	case "revoke-jwt":
		if !bothSecret || c.AccessToken == "" || c.GrantType != "" || c.ClientAssertionType != "" || c.ClientAssertion != "" {
			return errors.New("JWT revocation requires client credentials and access token")
		}
	case "issue-short":
		if c.GrantType == "" || !bothSecret || c.AccessToken != "" || c.ClientAssertionType != "" || c.ClientAssertion != "" {
			return errors.New("short-lived issuance requires grant type and client credentials")
		}
	default:
		return errors.New("unsupported channel token action")
	}
	return nil
}

// Execute performs exactly one SDK operation. No retry is permitted for an
// unknown token issuance/revocation outcome.
func (s *TokenOperations) Execute(ctx context.Context, c TokenCommand) (outbound.LINEResult, error) {
	if ctx == nil {
		return outbound.LINEResult{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return outbound.LINEResult{}, err
	}
	if err := ValidateTokenCommand(c); err != nil {
		return outbound.LINEResult{}, err
	}
	switch c.Action {
	case "issue-stateless":
		return s.manager.IssueStateless(ctx, c.GrantType, c.ClientAssertionType, c.ClientAssertion, c.ClientID, c.ClientSecret)
	case "key-ids":
		return s.manager.ValidKeyIDs(ctx, c.ClientAssertionType, c.ClientAssertion)
	case "issue-jwt":
		return s.manager.IssueJWT(ctx, c.GrantType, c.ClientAssertionType, c.ClientAssertion)
	case "verify-jwt":
		return s.manager.VerifyJWT(ctx, c.AccessToken)
	case "revoke-jwt":
		return outbound.LINEResult{}, s.manager.RevokeJWT(ctx, c.ClientID, c.ClientSecret, c.AccessToken)
	case "issue-short":
		return s.manager.IssueShortLived(ctx, c.GrantType, c.ClientID, c.ClientSecret)
	case "verify-short":
		return s.manager.VerifyShortLived(ctx, c.AccessToken)
	case "revoke-short":
		return outbound.LINEResult{}, s.manager.RevokeShortLived(ctx, c.AccessToken)
	default:
		return outbound.LINEResult{}, errors.New("unsupported channel token action")
	}
}
