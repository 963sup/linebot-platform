package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	channelconnection "github.com/963sup/linebot-platform/internal/application/usecases/channel-connection"
)

func TestChannelConnectionsRequireExplicitWrite(t *testing.T) {
	deps := Dependencies{ChannelConnections: func(context.Context, string) (ChannelConnectionsRuntime, error) {
		return NewChannelConnectionsRuntime(func(_ context.Context, command channelconnection.RegisterCommand) (channelconnection.ResolvedScope, error) {
			return channelconnection.ResolvedScope{TenantID: command.TenantID, ConnectionID: command.ConnectionID, MessagingDestination: command.MessagingDestination}, nil
		}, func() error { return nil })
	}}
	args := []string{"register", "--project", "project", "--tenant", "tenant-a", "--connection", "line-main", "--destination", "U0123456789abcdef0123456789abcdef"}
	if err := runChannelConnections(context.Background(), args, &bytes.Buffer{}, deps); err == nil {
		t.Fatal("registration without --write succeeded")
	}
	var output bytes.Buffer
	if err := runChannelConnections(context.Background(), append(args, "--write"), &output, deps); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"tenant_id":"tenant-a"`) {
		t.Fatalf("unexpected output %s", output.String())
	}
}
