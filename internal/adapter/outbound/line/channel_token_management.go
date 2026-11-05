package line

import (
	"context"
	"errors"
	"time"

	credentiallifecycle "github.com/963sup/linebot-platform/internal/application/usecases/credential-lifecycle"
	"github.com/line/line-bot-sdk-go/v8/linebot/channel_access_token"
)

type ChannelTokenManager struct {
	client *channel_access_token.ChannelAccessTokenAPI
}

func NewChannelTokenManager(c *channel_access_token.ChannelAccessTokenAPI) (*ChannelTokenManager, error) {
	if c == nil {
		return nil, errors.New("LINE Channel Token client is required")
	}
	return &ChannelTokenManager{client: c}, nil
}

func (s *ChannelTokenManager) clientFor(ctx context.Context) (*channel_access_token.ChannelAccessTokenAPI, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	copy := *s.client
	return copy.WithContext(ctx), nil
}

func (s *ChannelTokenManager) IssueStatelessWithClient(ctx context.Context, grantType string, credential credentiallifecycle.ClientCredential) (credentiallifecycle.IssuedCredential, error) {
	c, err := s.clientFor(ctx)
	if err != nil {
		return credentiallifecycle.IssuedCredential{}, err
	}
	r, v, err := c.IssueStatelessChannelTokenWithHttpInfo(grantType, "", "", credential.ClientID, credential.ClientSecret)
	if err != nil {
		return credentiallifecycle.IssuedCredential{}, lineWriteError("issueStatelessChannelToken", r, err)
	}
	return issuedCredential(v.AccessToken, v.TokenType, v.ExpiresIn), nil
}

func (s *ChannelTokenManager) IssueStatelessWithAssertion(ctx context.Context, grantType string, credential credentiallifecycle.AssertionCredential) (credentiallifecycle.IssuedCredential, error) {
	c, err := s.clientFor(ctx)
	if err != nil {
		return credentiallifecycle.IssuedCredential{}, err
	}
	r, v, err := c.IssueStatelessChannelTokenWithHttpInfo(grantType, credential.Type, credential.Assertion, "", "")
	if err != nil {
		return credentiallifecycle.IssuedCredential{}, lineWriteError("issueStatelessChannelToken", r, err)
	}
	return issuedCredential(v.AccessToken, v.TokenType, v.ExpiresIn), nil
}

func (s *ChannelTokenManager) ValidKeyIDs(ctx context.Context, credential credentiallifecycle.AssertionCredential) ([]string, error) {
	c, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	r, v, err := c.GetsAllValidChannelAccessTokenKeyIdsWithHttpInfo(credential.Type, credential.Assertion)
	if err != nil {
		return nil, lineWriteError("getsAllValidChannelAccessTokenKeyIds", r, err)
	}
	return append([]string(nil), v.Kids...), nil
}

func (s *ChannelTokenManager) IssueJWT(ctx context.Context, grantType string, credential credentiallifecycle.AssertionCredential) (credentiallifecycle.IssuedCredential, error) {
	c, err := s.clientFor(ctx)
	if err != nil {
		return credentiallifecycle.IssuedCredential{}, err
	}
	r, v, err := c.IssueChannelTokenByJWTWithHttpInfo(grantType, credential.Type, credential.Assertion)
	if err != nil {
		return credentiallifecycle.IssuedCredential{}, lineWriteError("issueChannelTokenByJWT", r, err)
	}
	return issuedCredential(v.AccessToken, v.TokenType, v.ExpiresIn), nil
}

func (s *ChannelTokenManager) VerifyJWT(ctx context.Context, token string) (credentiallifecycle.Verification, error) {
	c, err := s.clientFor(ctx)
	if err != nil {
		return credentiallifecycle.Verification{}, err
	}
	r, v, err := c.VerifyChannelTokenByJWTWithHttpInfo(token)
	if err != nil {
		return credentiallifecycle.Verification{}, lineWriteError("verifyChannelTokenByJWT", r, err)
	}
	return verification(v.ClientId, v.ExpiresIn), nil
}

func (s *ChannelTokenManager) RevokeJWT(ctx context.Context, credential credentiallifecycle.ClientCredential, token string) error {
	c, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := c.RevokeChannelTokenByJWTWithHttpInfo(credential.ClientID, credential.ClientSecret, token)
	return lineWriteError("revokeChannelTokenByJWT", r, err)
}

func (s *ChannelTokenManager) IssueShortLived(ctx context.Context, grantType string, credential credentiallifecycle.ClientCredential) (credentiallifecycle.IssuedCredential, error) {
	c, err := s.clientFor(ctx)
	if err != nil {
		return credentiallifecycle.IssuedCredential{}, err
	}
	r, v, err := c.IssueChannelTokenWithHttpInfo(grantType, credential.ClientID, credential.ClientSecret)
	if err != nil {
		return credentiallifecycle.IssuedCredential{}, lineWriteError("issueChannelToken", r, err)
	}
	return issuedCredential(v.AccessToken, v.TokenType, v.ExpiresIn), nil
}

func (s *ChannelTokenManager) VerifyShortLived(ctx context.Context, token string) (credentiallifecycle.Verification, error) {
	c, err := s.clientFor(ctx)
	if err != nil {
		return credentiallifecycle.Verification{}, err
	}
	r, v, err := c.VerifyChannelTokenWithHttpInfo(token)
	if err != nil {
		return credentiallifecycle.Verification{}, lineWriteError("verifyChannelToken", r, err)
	}
	return verification(v.ClientId, v.ExpiresIn), nil
}

func (s *ChannelTokenManager) RevokeShortLived(ctx context.Context, token string) error {
	c, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := c.RevokeChannelTokenWithHttpInfo(token)
	return lineWriteError("revokeChannelToken", r, err)
}

func issuedCredential(token, tokenType string, expiresIn int32) credentiallifecycle.IssuedCredential {
	return credentiallifecycle.IssuedCredential{AccessToken: token, TokenType: tokenType, ExpiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second)}
}

func verification(clientID string, expiresIn int64) credentiallifecycle.Verification {
	return credentiallifecycle.Verification{Active: expiresIn > 0, ClientID: clientID, ExpiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second)}
}

var _ credentiallifecycle.Gateway = (*ChannelTokenManager)(nil)
