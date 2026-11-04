package line

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
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

func (s *ChannelTokenManager) IssueStateless(ctx context.Context, grantType, assertionType, assertion, clientID, clientSecret string) (outbound.LINEResult, error) {
	c, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	r, v, err := c.IssueStatelessChannelTokenWithHttpInfo(grantType, assertionType, assertion, clientID, clientSecret)
	return typedReportResult("issueStatelessChannelToken", r, v, err)
}

func (s *ChannelTokenManager) ValidKeyIDs(ctx context.Context, assertionType, assertion string) (outbound.LINEResult, error) {
	c, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	r, v, err := c.GetsAllValidChannelAccessTokenKeyIdsWithHttpInfo(assertionType, assertion)
	return typedReportResult("getsAllValidChannelAccessTokenKeyIds", r, v, err)
}

func (s *ChannelTokenManager) IssueJWT(ctx context.Context, grantType, assertionType, assertion string) (outbound.LINEResult, error) {
	c, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	r, v, err := c.IssueChannelTokenByJWTWithHttpInfo(grantType, assertionType, assertion)
	return typedReportResult("issueChannelTokenByJWT", r, v, err)
}

func (s *ChannelTokenManager) VerifyJWT(ctx context.Context, token string) (outbound.LINEResult, error) {
	c, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	r, v, err := c.VerifyChannelTokenByJWTWithHttpInfo(token)
	return typedReportResult("verifyChannelTokenByJWT", r, v, err)
}

func (s *ChannelTokenManager) RevokeJWT(ctx context.Context, clientID, clientSecret, token string) error {
	c, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := c.RevokeChannelTokenByJWTWithHttpInfo(clientID, clientSecret, token)
	return lineWriteError("revokeChannelTokenByJWT", r, err)
}

func (s *ChannelTokenManager) IssueShortLived(ctx context.Context, grantType, clientID, clientSecret string) (outbound.LINEResult, error) {
	c, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	r, v, err := c.IssueChannelTokenWithHttpInfo(grantType, clientID, clientSecret)
	return typedReportResult("issueChannelToken", r, v, err)
}

func (s *ChannelTokenManager) VerifyShortLived(ctx context.Context, token string) (outbound.LINEResult, error) {
	c, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	r, v, err := c.VerifyChannelTokenWithHttpInfo(token)
	return typedReportResult("verifyChannelToken", r, v, err)
}

func (s *ChannelTokenManager) RevokeShortLived(ctx context.Context, token string) error {
	c, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := c.RevokeChannelTokenWithHttpInfo(token)
	return lineWriteError("revokeChannelToken", r, err)
}
