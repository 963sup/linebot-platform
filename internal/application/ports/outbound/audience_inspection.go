package outbound

import (
	"context"
	"encoding/json"
)

// AudienceReader is the minimum query-only port needed by audience inspection.
type AudienceReader interface {
	AudienceGroup(ctx context.Context, id int64) (json.RawMessage, error)
	SharedAudienceGroup(ctx context.Context, id int64) (json.RawMessage, error)
	AudienceGroups(ctx context.Context, page, size int64) (json.RawMessage, error)
	SharedAudienceGroups(ctx context.Context, page, size int64) (json.RawMessage, error)
}
