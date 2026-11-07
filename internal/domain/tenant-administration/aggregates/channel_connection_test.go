package aggregates

import (
	"testing"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/tenant-administration/value-objects"
)

func TestChannelConnectionRequiresTenantAndMessagingDestination(t *testing.T) {
	tenant, _ := valueobjects.NewTenantID("tenant-a")
	connection, _ := valueobjects.NewChannelConnectionID("line-main")
	value, err := NewChannelConnection(tenant, connection, "U0123456789abcdef0123456789abcdef")
	if err != nil || value.TenantID() != tenant || value.ID() != connection {
		t.Fatalf("valid channel connection: value=%#v err=%v", value, err)
	}
	if _, err := NewChannelConnection(tenant, connection, "bad"); err == nil {
		t.Fatal("invalid messaging destination accepted")
	}
}
