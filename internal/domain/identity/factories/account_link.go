// Package factories builds valid identity aggregates from verified claims.
package factories

import (
 "github.com/963sup/linebot-platform/internal/domain/identity/aggregates"
 "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)
// NewVerifiedLink requires an authenticated LINE owner and a verified Google subject.
// OAuth verification occurs outside the domain.
func NewVerifiedLink(ownerID,googleSubject string,grantedScopes []string) (aggregates.AccountLink,error) {
 owner,err:=valueobjects.NewLineUserID(ownerID)
 if err!=nil { return aggregates.AccountLink{},err }
 subject,err:=valueobjects.NewGoogleAccountID(googleSubject)
 if err!=nil { return aggregates.AccountLink{},err }
 scopes,err:=valueobjects.NewOAuthScopes(grantedScopes)
 if err!=nil { return aggregates.AccountLink{},err }
 return aggregates.NewAccountLink(owner,subject,scopes)
}
