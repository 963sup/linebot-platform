package bootstrap

import (
	"context"
	"testing"
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
			if runtime, err := newOperatorDomainRuntime(test.ctx, test.project); err == nil || runtime != nil {
				t.Fatalf("newOperatorDomainRuntime() = (%v, %v), want nil, error", runtime, err)
			}
		})
	}
}
