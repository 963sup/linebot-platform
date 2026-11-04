package contacts

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type AccountLinker struct {
	manager outbound.AccountLinkManager
}

func NewAccountLinker(m outbound.AccountLinkManager) (*AccountLinker, error) {
	if m == nil {
		return nil, errors.New("account link manager is required")
	}
	return &AccountLinker{manager: m}, nil
}

// IssueLinkToken contains sensitive credentials; callers must persist it to
// an explicitly requested 0600 file and must never print it to the terminal.
func (s *AccountLinker) IssueLinkToken(ctx context.Context, userID string) (json.RawMessage, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if userID == "" || userID != strings.TrimSpace(userID) {
		return nil, errors.New("user ID is required")
	}
	return s.manager.IssueLinkToken(ctx, userID)
}
