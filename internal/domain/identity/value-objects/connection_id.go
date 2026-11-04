package valueobjects

import "errors"

// ConnectionID is the pair identifying one LINE-to-Google relationship.
// Persistence adapters are responsible for hashing it into document IDs.
type ConnectionID struct {
	owner   LineUserID
	subject GoogleAccountID
}

func NewConnectionID(owner LineUserID, subject GoogleAccountID) (ConnectionID, error) {
	if !owner.Valid() || !subject.Valid() {
		return ConnectionID{}, errors.New("invalid connection ID")
	}
	return ConnectionID{owner: owner, subject: subject}, nil
}

func (id ConnectionID) Owner() LineUserID              { return id.owner }
func (id ConnectionID) GoogleAccount() GoogleAccountID { return id.subject }
func (id ConnectionID) Valid() bool {
	return id.owner.Valid() && id.subject.Valid()
}
