package contacts

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

// Profiles coordinates authorized callers' explicit profile lookups.
// Authentication and caller authorization belong at the inbound boundary.
type Profiles struct {
	reader outbound.ProfileReader
}

func NewProfiles(reader outbound.ProfileReader) (*Profiles, error) {
	if reader == nil {
		return nil, errors.New("profile reader is required")
	}
	return &Profiles{reader: reader}, nil
}

func (s *Profiles) Get(ctx context.Context, userID string) (outbound.UserProfile, error) {
	if ctx == nil {
		return outbound.UserProfile{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return outbound.UserProfile{}, err
	}
	if userID == "" || userID != strings.TrimSpace(userID) {
		return outbound.UserProfile{}, errors.New("valid user ID is required")
	}
	return s.reader.GetProfile(ctx, userID)
}
