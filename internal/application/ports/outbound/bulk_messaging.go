package outbound

import "context"

// BulkMessaging uses stable retry identities and never retries automatically.
type BulkMessaging interface {
	Broadcast(context.Context, []Message, string) error
	Multicast(context.Context, []string, []Message, string) error
}
