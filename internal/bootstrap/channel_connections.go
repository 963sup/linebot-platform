package bootstrap

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/cli"
	storeout "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	channelconnection "github.com/963sup/linebot-platform/internal/application/usecases/channel-connection"
	storeinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/firestore"
)

func newChannelConnectionsRuntime(ctx context.Context, project string) (cli.ChannelConnectionsRuntime, error) {
	if ctx == nil || strings.TrimSpace(project) == "" {
		return nil, errors.New("channel connections runtime requires context and project")
	}
	client, err := storeinfra.NewClient(ctx, project)
	if err != nil {
		return nil, err
	}
	repository, err := storeout.NewChannelConnectionRepository(client)
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	runtime, err := cli.NewChannelConnectionsRuntime(func(ctx context.Context, command channelconnection.RegisterCommand) (channelconnection.ResolvedScope, error) {
		return channelconnection.Register(ctx, repository, command)
	}, client.Close)
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	return runtime, nil
}
