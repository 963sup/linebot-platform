package bootstrap

import (
	"io"
	"testing"

	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
)

var _ io.Closer = (*LINEOpsRuntime)(nil)

func TestLINEOpsRuntimeCloseContract(t *testing.T) {
	transport, err := httpclient.New(httpclient.Config{Timeout: httpclient.DefaultTimeout}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(transport.Close)
	for name, runtime := range map[string]*LINEOpsRuntime{
		"nil":     nil,
		"zero":    {},
		"managed": {outbound: transport},
	} {
		t.Run(name, func(t *testing.T) {
			for i := 0; i < 2; i++ {
				if err := runtime.Close(); err != nil {
					t.Fatalf("close %d: %v", i, err)
				}
			}
		})
	}
}
