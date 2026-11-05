package bootstrap

import (
	"github.com/963sup/linebot-platform/internal/adapter/outbound/line"
	credentiallifecycle "github.com/963sup/linebot-platform/internal/application/usecases/credential-lifecycle"
	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
	lineinfra "github.com/963sup/linebot-platform/internal/infrastructure/line"
)

// LINETokenOpsRuntime is intentionally separate from Messaging runtime; no
// Channel Token API client is ever constructed inside a public webhook.
type LINETokenOpsRuntime struct {
	Credentials *credentiallifecycle.Service
	transport   *httpclient.Managed
}

func NewLINETokenOpsRuntime() (*LINETokenOpsRuntime, error) {
	transport, err := httpclient.New(httpclient.Config{Timeout: httpclient.DefaultTimeout}, nil)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*LINETokenOpsRuntime, error) { transport.Close(); return nil, err }
	client, err := lineinfra.NewChannelTokenClientWithHTTPClient(transport.Client)
	if err != nil {
		return fail(err)
	}
	adapter, err := line.NewChannelTokenManager(client)
	if err != nil {
		return fail(err)
	}
	service, err := credentiallifecycle.New(adapter)
	if err != nil {
		return fail(err)
	}
	return &LINETokenOpsRuntime{Credentials: service, transport: transport}, nil
}

func (r *LINETokenOpsRuntime) Close() {
	if r != nil && r.transport != nil {
		r.transport.Close()
	}
}
