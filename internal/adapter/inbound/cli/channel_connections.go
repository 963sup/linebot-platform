package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"

	channelconnection "github.com/963sup/linebot-platform/internal/application/usecases/channel-connection"
)

type channelConnectionsRuntime struct {
	register func(context.Context, channelconnection.RegisterCommand) (channelconnection.ResolvedScope, error)
	close    func() error
}

func NewChannelConnectionsRuntime(register func(context.Context, channelconnection.RegisterCommand) (channelconnection.ResolvedScope, error), close func() error) (ChannelConnectionsRuntime, error) {
	if register == nil || close == nil {
		return nil, errors.New("channel connections operations and close are required")
	}
	return &channelConnectionsRuntime{register: register, close: close}, nil
}

func (r *channelConnectionsRuntime) Register(ctx context.Context, tenant, connection, destination string) (ChannelConnectionSummary, error) {
	value, err := r.register(ctx, channelconnection.RegisterCommand{TenantID: tenant, ConnectionID: connection, MessagingDestination: destination})
	return ChannelConnectionSummary{TenantID: value.TenantID, ConnectionID: value.ConnectionID, MessagingDestination: value.MessagingDestination}, err
}

func (r *channelConnectionsRuntime) Close() error { return r.close() }

func runChannelConnections(ctx context.Context, args []string, output io.Writer, deps Dependencies) (err error) {
	if len(args) == 0 || args[0] != "register" {
		return errors.New("usage: ops channel-connections register --project PROJECT --tenant TENANT --connection CONNECTION --destination BOT_USER_ID --write")
	}
	fs := flag.NewFlagSet("channel-connections register", flag.ContinueOnError)
	project := fs.String("project", "", "Google Cloud project")
	tenant := fs.String("tenant", "", "tenant ID")
	connection := fs.String("connection", "", "channel connection ID")
	destination := fs.String("destination", "", "LINE webhook destination bot user ID")
	write := fs.Bool("write", false, "explicitly create the Firestore binding")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if !*write {
		return errors.New("dry run only: specify --write to create the channel connection")
	}
	if deps.ChannelConnections == nil {
		return errors.New("channel connections runtime is not configured")
	}
	runtime, err := deps.ChannelConnections(ctx, *project)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, runtime.Close()) }()
	summary, err := runtime.Register(ctx, *tenant, *connection, *destination)
	if err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(summary)
}
