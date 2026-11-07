package bootstrap

import (
	"context"
	"io"
	"testing"
)

func TestOperatorDomainRuntimeRejectsInvalidConfigurationBeforeClientCreation(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, test := range []struct {
		name    string
		ctx     context.Context
		project string
	}{
		{"nil context", nil, "stage"},
		{"empty project", context.Background(), ""},
		{"blank project", context.Background(), "  "},
		{"canceled context", canceled, "stage"},
	} {
		t.Run(test.name, func(t *testing.T) {
			constructors := []func(context.Context, string) (io.Closer, error){
				func(ctx context.Context, p string) (io.Closer, error) { return newAudienceDomainRuntime(ctx, p) },
				func(ctx context.Context, p string) (io.Closer, error) { return newSurfaceDomainRuntime(ctx, p) },
				func(ctx context.Context, p string) (io.Closer, error) { return newPromotionDomainRuntime(ctx, p) },
			}
			for _, build := range constructors {
				if runtime, err := build(test.ctx, test.project); err == nil || runtime != nil {
					t.Fatalf("runtime = (%v, %v), want nil, error", runtime, err)
				}
			}
		})
	}
}
