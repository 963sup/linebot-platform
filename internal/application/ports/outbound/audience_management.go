package outbound

import (
	"context"
	"encoding/json"
)

// AudienceManager is an operator-only write port, separate from read-only
// audience inspection. File uploads use bounded bytes, never public webhook data.
type AudienceManager interface {
	CreateAudienceGroup(context.Context, json.RawMessage) (json.RawMessage, error)
	AddAudienceToGroup(context.Context, json.RawMessage) error
	CreateClickAudience(context.Context, json.RawMessage) (json.RawMessage, error)
	CreateImpressionAudience(context.Context, json.RawMessage) (json.RawMessage, error)
	UpdateAudienceDescription(context.Context, int64, json.RawMessage) error
	DeleteAudienceGroup(context.Context, int64) error
	CreateAudienceFromFile(context.Context, []byte, string, bool, string) (json.RawMessage, error)
	AddAudienceFromFile(context.Context, int64, []byte, string) error
}
