package firestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	sdk "cloud.google.com/go/firestore"
	"github.com/963sup/linebot-platform/internal/domain/tenant-administration/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/tenant-administration/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/tenant-administration/value-objects"
)

const channelConnectionsCollection = "line_channel_connections_v1"

type storedChannelConnection struct {
	TenantID             string `firestore:"tenant_id"`
	ConnectionID         string `firestore:"connection_id"`
	MessagingDestination string `firestore:"messaging_destination"`
}

type ChannelConnectionRepository struct{ client *sdk.Client }

var _ repositories.ChannelConnectionRepository = (*ChannelConnectionRepository)(nil)

func NewChannelConnectionRepository(client *sdk.Client) (*ChannelConnectionRepository, error) {
	if client == nil {
		return nil, errors.New("Firestore client is required")
	}
	return &ChannelConnectionRepository{client: client}, nil
}

func (r *ChannelConnectionRepository) Create(ctx context.Context, connection aggregates.ChannelConnection) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	_, err := r.document(connection.MessagingDestination()).Create(ctx, storedChannelConnection{
		TenantID: connection.TenantID().String(), ConnectionID: connection.ID().String(),
		MessagingDestination: connection.MessagingDestination(),
	})
	return err
}

func (r *ChannelConnectionRepository) FindByMessagingDestination(ctx context.Context, destination string) (aggregates.ChannelConnection, error) {
	if ctx == nil {
		return aggregates.ChannelConnection{}, errors.New("context is required")
	}
	snapshot, err := r.document(destination).Get(ctx)
	if err != nil {
		return aggregates.ChannelConnection{}, err
	}
	var stored storedChannelConnection
	if err := snapshot.DataTo(&stored); err != nil {
		return aggregates.ChannelConnection{}, err
	}
	if stored.MessagingDestination != destination {
		return aggregates.ChannelConnection{}, errors.New("channel connection lookup mismatch")
	}
	tenantID, err := valueobjects.NewTenantID(stored.TenantID)
	if err != nil {
		return aggregates.ChannelConnection{}, err
	}
	connectionID, err := valueobjects.NewChannelConnectionID(stored.ConnectionID)
	if err != nil {
		return aggregates.ChannelConnection{}, err
	}
	return aggregates.NewChannelConnection(tenantID, connectionID, stored.MessagingDestination)
}

func (r *ChannelConnectionRepository) document(destination string) *sdk.DocumentRef {
	hash := sha256.Sum256([]byte(destination))
	return r.client.Collection(channelConnectionsCollection).Doc(hex.EncodeToString(hash[:]))
}
