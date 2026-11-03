package outbound

import (
	"context"
	"encoding/json"
)

// BulkMessaging uses stable retry identities and never retries automatically.
type BulkMessaging interface {
	Broadcast(context.Context, []Message, string) error
	Multicast(context.Context, []string, []Message, string) error
	Narrowcast(context.Context, json.RawMessage, string) error
}
