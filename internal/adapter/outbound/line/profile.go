package line

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/infrastructure/resilience"
)

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
