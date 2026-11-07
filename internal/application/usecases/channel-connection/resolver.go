// Package channelconnection resolves a verified provider channel to tenant scope.
package channelconnection

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/domain/tenant-administration/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/tenant-administration/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/tenant-administration/value-objects"
)

type ResolvedScope struct {
	TenantID, ConnectionID, MessagingDestination string
}

type Resolver struct {
	reader repositories.ChannelConnectionReader
}

type RegisterCommand struct {
	TenantID, ConnectionID, MessagingDestination string
}

func Register(ctx context.Context, writer repositories.ChannelConnectionWriter, command RegisterCommand) (ResolvedScope, error) {
	if ctx == nil || writer == nil {
		return ResolvedScope{}, errors.New("context and channel connection writer are required")
	}
	tenantID, err := valueobjects.NewTenantID(command.TenantID)
	if err != nil {
		return ResolvedScope{}, err
	}
	connectionID, err := valueobjects.NewChannelConnectionID(command.ConnectionID)
	if err != nil {
		return ResolvedScope{}, err
	}
	connection, err := aggregates.NewChannelConnection(tenantID, connectionID, command.MessagingDestination)
	if err != nil {
		return ResolvedScope{}, err
	}
	if err := writer.Create(ctx, connection); err != nil {
		return ResolvedScope{}, err
	}
	return ResolvedScope{TenantID: tenantID.String(), ConnectionID: connectionID.String(), MessagingDestination: connection.MessagingDestination()}, nil
}

func NewResolver(reader repositories.ChannelConnectionReader) (*Resolver, error) {
	if reader == nil {
		return nil, errors.New("channel connection reader is required")
	}
	return &Resolver{reader: reader}, nil
}

func (r *Resolver) ResolveMessagingDestination(ctx context.Context, destination string) (ResolvedScope, error) {
	if ctx == nil {
		return ResolvedScope{}, errors.New("context is required")
	}
	connection, err := r.reader.FindByMessagingDestination(ctx, destination)
	if err != nil {
		return ResolvedScope{}, err
	}
	return ResolvedScope{
		TenantID: connection.TenantID().String(), ConnectionID: connection.ID().String(),
		MessagingDestination: connection.MessagingDestination(),
	}, nil
}
