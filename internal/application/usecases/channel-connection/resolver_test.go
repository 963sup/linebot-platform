package channelconnection

import (
	"context"
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/tenant-administration/aggregates"
)

type connectionRepositoryStub struct{ value aggregates.ChannelConnection }

func (s *connectionRepositoryStub) Create(_ context.Context, value aggregates.ChannelConnection) error {
	s.value = value
	return nil
}

func (s *connectionRepositoryStub) FindByMessagingDestination(context.Context, string) (aggregates.ChannelConnection, error) {
	return s.value, nil
}

func TestRegisterThenResolveMessagingDestination(t *testing.T) {
	repository := &connectionRepositoryStub{}
	registered, err := Register(context.Background(), repository, RegisterCommand{
		TenantID: "tenant-a", ConnectionID: "line-main", MessagingDestination: "U0123456789abcdef0123456789abcdef",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver, _ := NewResolver(repository)
	resolved, err := resolver.ResolveMessagingDestination(context.Background(), registered.MessagingDestination)
	if err != nil || resolved != registered {
		t.Fatalf("resolved=%#v registered=%#v err=%v", resolved, registered, err)
	}
}
