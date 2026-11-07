// Package aggregates owns tenant-scoped provider connection invariants.
package aggregates

import (
	"errors"
	"regexp"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/tenant-administration/value-objects"
)

var messagingDestinationPattern = regexp.MustCompile(`^U[0-9a-f]{32}$`)

type ChannelConnection struct {
	tenantID             valueobjects.TenantID
	id                   valueobjects.ChannelConnectionID
	messagingDestination string
}

func NewChannelConnection(tenantID valueobjects.TenantID, id valueobjects.ChannelConnectionID, messagingDestination string) (ChannelConnection, error) {
	if !tenantID.Valid() || !id.Valid() || !messagingDestinationPattern.MatchString(messagingDestination) {
		return ChannelConnection{}, errors.New("valid tenant-scoped LINE channel connection is required")
	}
	return ChannelConnection{tenantID: tenantID, id: id, messagingDestination: messagingDestination}, nil
}

func (c ChannelConnection) TenantID() valueobjects.TenantID      { return c.tenantID }
func (c ChannelConnection) ID() valueobjects.ChannelConnectionID { return c.id }
func (c ChannelConnection) MessagingDestination() string         { return c.messagingDestination }
