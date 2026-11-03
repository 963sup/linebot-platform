package bootstrap

import (
	"context"
	"errors"
	"strings"

	storeout "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	storeinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/firestore"
)

// StatusRuntime is isolated from the LINE sender and owns only Firestore.
type StatusRuntime struct {
	Reader outbound.EventStatusReader
	close  func() error
}

func NewStatusRuntime(ctx context.Context, project string) (*StatusRuntime, error) {
	if ctx == nil || strings.TrimSpace(project) == "" {
		return nil, errors.New("status requires context and staging project")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	client, err := storeinfra.NewClient(ctx, project)
	if err != nil {
		return nil, err
	}
	reader, err := storeout.NewStore(client)
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	return &StatusRuntime{Reader: reader, close: client.Close}, nil
}

func (r *StatusRuntime) Close() error {
	if r == nil || r.close == nil {
		return nil
	}
	return r.close()
}
