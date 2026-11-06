package line

import (
	"context"
	"errors"
	"time"

	contactdirectory "github.com/963sup/linebot-platform/internal/application/usecases/contact-directory"
)

// GetProfile implements the contacts ProfileReader port using the existing
// typed SDK client. Sender.clientFor isolates SDK WithContext mutations.
func (s *Sender) GetProfile(ctx context.Context, query contactdirectory.ProfileQuery) (contactdirectory.ProfileSnapshot, error) {
	if query.Recipient.ID == "" || query.Recipient.Kind != contactdirectory.RecipientUser {
		return contactdirectory.ProfileSnapshot{}, errors.New("LINE user ID is required")
	}
	client, err := s.clientFor(ctx)
	if err != nil {
		return contactdirectory.ProfileSnapshot{}, err
	}
	response, profile, err := client.GetProfileWithHttpInfo(query.Recipient.ID)
	if err != nil {
		return contactdirectory.ProfileSnapshot{}, lineAPIError("get_profile", response, err)
	}
	if profile == nil {
		return contactdirectory.ProfileSnapshot{}, errors.New("LINE profile response is empty")
	}
	return contactdirectory.ProfileSnapshot{
		Recipient:     contactdirectory.RecipientReference{ID: profile.UserId, Kind: contactdirectory.RecipientUser},
		DisplayName:   profile.DisplayName,
		PictureURL:    profile.PictureUrl,
		StatusMessage: profile.StatusMessage,
		ObservedAt:    time.Now(),
	}, nil
}

var _ contactdirectory.Reader = (*Sender)(nil)
