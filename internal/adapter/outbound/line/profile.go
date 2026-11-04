package line

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/infrastructure/resilience"
)

// IssueLinkTokenResult adapts the legacy wire-oriented sender method to the
// typed application boundary without exposing JSON to the use case.
func (s *Sender) IssueLinkTokenResult(ctx context.Context, userID string) (outbound.LINEResult, error) {
	payload, err := s.IssueLinkToken(ctx, userID)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	value, err := decodeJSONValue(payload)
	if err != nil {
		return outbound.LINEResult{}, errors.New("invalid LINE link token response")
	}
	return outbound.LINEResult{Value: value}, nil
}

// GetProfile implements the contacts ProfileReader port using the existing
// typed SDK client. Sender.clientFor isolates SDK WithContext mutations.
func (s *Sender) GetProfile(ctx context.Context, userID string) (outbound.UserProfile, error) {
	if userID == "" {
		return outbound.UserProfile{}, errors.New("LINE user ID is required")
	}
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.UserProfile{}, err
	}
	response, profile, err := client.GetProfileWithHttpInfo(userID)
	if err != nil {
		return outbound.UserProfile{}, resilience.Wrap("line", "get_profile", httpStatus(response), err)
	}
	if profile == nil {
		return outbound.UserProfile{}, errors.New("LINE profile response is empty")
	}
	return outbound.UserProfile{
		ID:          profile.UserId,
		DisplayName: profile.DisplayName,
		PictureURL:  profile.PictureUrl,
	}, nil
}
