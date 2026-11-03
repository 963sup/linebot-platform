package outbound

import "context"

// UserProfile contains only the LINE profile fields required by the contacts use case.
// Do not include raw SDK responses or persist these fields without a business need.
type UserProfile struct {
	ID          string
	DisplayName string
	PictureURL  string
}

// ProfileReader is a consumer-owned, read-only port.
type ProfileReader interface {
	GetProfile(ctx context.Context, userID string) (UserProfile, error)
}
