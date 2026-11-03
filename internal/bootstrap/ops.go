package bootstrap

import (
	"github.com/963sup/linebot-platform/internal/adapter/outbound/line"
	"github.com/963sup/linebot-platform/internal/application/usecases/contacts"
	"github.com/963sup/linebot-platform/internal/application/usecases/channel"
	"github.com/963sup/linebot-platform/internal/application/usecases/engagement"
	"github.com/963sup/linebot-platform/internal/application/usecases/messaging"
	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
	lineinfra "github.com/963sup/linebot-platform/internal/infrastructure/line"
)

// LINEOpsRuntime owns the HTTP transport used by explicit, local CLI actions.
// It is never exposed on the unauthenticated webhook process.
type LINEOpsRuntime struct {
	Messaging *messaging.Service
	Profiles  *contacts.Profiles
	Channel   *channel.Inspector
	Menus     *engagement.RichMenuInspector
	outbound  *httpclient.Managed
}

func NewLINEOpsRuntime(token string) (*LINEOpsRuntime, error) {
	transport, err := httpclient.New(httpclient.Config{Timeout: httpclient.DefaultTimeout}, nil)
	if err != nil {
		return nil, err
	}
	cleanup := func(err error) (*LINEOpsRuntime, error) {
		transport.Close()
		return nil, err
	}
	client, err := lineinfra.NewMessagingClientWithHTTPClient(token, transport.Client)
	if err != nil {
		return cleanup(err)
	}
	sender, err := line.NewSender(client)
	if err != nil {
		return cleanup(err)
	}
	messages, err := messaging.NewService(sender)
	if err != nil {
		return cleanup(err)
	}
	profiles, err := contacts.NewProfiles(sender)
	if err != nil {
		return cleanup(err)
	}
	inspector, err := channel.NewInspector(sender)
	if err != nil {
		return cleanup(err)
	}
	menus, err := engagement.NewRichMenuInspector(sender)
	if err != nil {
		return cleanup(err)
	}
	return &LINEOpsRuntime{Messaging: messages, Profiles: profiles, Channel: inspector, Menus: menus, outbound: transport}, nil
}

func (r *LINEOpsRuntime) Close() {
	if r != nil && r.outbound != nil {
		r.outbound.Close()
	}
}
