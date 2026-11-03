package outbound

import "context"

// CampaignMessaging uses stable retry identities and never retries automatically.
type CampaignMessaging interface {
	Broadcast(context.Context, []Message, string) error
	Multicast(context.Context, []string, []Message, string) error
}
