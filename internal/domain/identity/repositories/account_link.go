// Package repositories defines domain persistence contracts independent of Firestore.
package repositories

import (
 "context"
 "github.com/963sup/linebot-platform/internal/domain/identity/aggregates"
 "github.com/963sup/linebot-platform/internal/domain/identity/valueobjects"
)
// AccountLinks implementations must enforce (owner,subject) uniqueness
// and prevent concurrent lost updates atomically.
type AccountLinks interface {
 Find(ctx context.Context,owner valueobjects.LineUserID,subject valueobjects.GoogleAccountID) (aggregates.AccountLink,error)
 Save(ctx context.Context,link aggregates.AccountLink) error
}
