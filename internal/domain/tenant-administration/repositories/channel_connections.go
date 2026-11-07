// Package repositories defines persistence semantics owned by Tenant Administration.
package repositories

import (
	"context"

	"github.com/963sup/linebot-platform/internal/domain/tenant-administration/aggregates"
)

type ChannelConnectionReader interface {
	FindByMessagingDestination(context.Context, string) (aggregates.ChannelConnection, error)
}

type ChannelConnectionWriter interface {
	Create(context.Context, aggregates.ChannelConnection) error
}

type ChannelConnectionRepository interface {
	ChannelConnectionReader
	ChannelConnectionWriter
}
