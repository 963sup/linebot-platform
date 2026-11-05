package bootstrap

import (
	"context"
	"testing"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/cli"
)

var (
	_ cli.AudienceDefinitionsRuntime = (*OperatorDomainRuntime)(nil)
	_ cli.InteractionSurfacesRuntime = (*OperatorDomainRuntime)(nil)
	_ cli.PromotionsRuntime          = (*OperatorDomainRuntime)(nil)
	_ cli.AutomationRequestsRuntime  = (*OperatorDomainRuntime)(nil)
)

func TestOperatorDomainRuntimeRejectsInvalidConfigurationBeforeClientCreation(t *testing.T) {
	for _, test := range []struct {
		name    string
		ctx     context.Context
		project string
	}{
		{"nil context", nil, "stage"},
		{"empty project", context.Background(), ""},
		{"blank project", context.Background(), "  "},
	} {
		t.Run(test.name, func(t *testing.T) {
			if runtime, err := NewOperatorDomainRuntime(test.ctx, test.project); err == nil || runtime != nil {
				t.Fatalf("NewOperatorDomainRuntime() = (%v, %v), want nil, error", runtime, err)
			}
		})
	}
}
