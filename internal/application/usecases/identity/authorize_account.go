package identity

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/domain/identity/repositories"
	"github.com/963sup/linebot-platform/internal/domain/identity/specifications"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

var ErrAccountNotAuthorized = errors.New("Google account not linked or authorized")

// AuthorizeLinkedGoogleAccount verifies ownership, lifecycle and scopes before
// an adapter constructs a user-scoped Google client.
func AuthorizeLinkedGoogleAccount(
	ctx context.Context,
	reader repositories.ConnectionReader,
	ownerID, googleSubject string,
	requiredScopes []string,
) (valueobjects.ConnectionID, error) {
	if ctx == nil || reader == nil {
		return valueobjects.ConnectionID{}, errors.New("authorized account context and repository required")
	}
	if err := ctx.Err(); err != nil {
		return valueobjects.ConnectionID{}, err
	}
	owner, err := valueobjects.NewLineUserID(ownerID)
	if err != nil {
		return valueobjects.ConnectionID{}, errors.New("invalid LINE account identity")
	}
	subject, err := valueobjects.NewGoogleAccountID(googleSubject)
	if err != nil {
		return valueobjects.ConnectionID{}, errors.New("invalid Google account identity")
	}
	id, err := valueobjects.NewConnectionID(owner, subject)
	if err != nil {
		return valueobjects.ConnectionID{}, err
	}
	link, err := reader.FindConnection(ctx, id)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return valueobjects.ConnectionID{}, ctxErr
		}
		return valueobjects.ConnectionID{}, ErrAccountNotAuthorized
	}
	if link.Owner() != owner || link.GoogleAccount() != subject ||
		!specifications.CanUseGoogleAPI(link, requiredScopes) {
		return valueobjects.ConnectionID{}, ErrAccountNotAuthorized
	}
	return id, nil
}
