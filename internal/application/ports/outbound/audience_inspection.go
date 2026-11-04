package outbound

import "context"

// AudienceReader is the minimum query-only port needed by audience inspection.
type AudienceReader interface {
	AudienceGroup(ctx context.Context, id int64) (LINEResult, error)
	SharedAudienceGroup(ctx context.Context, id int64) (LINEResult, error)
	AudienceGroups(ctx context.Context, page, size int64) (LINEResult, error)
	SharedAudienceGroups(ctx context.Context, page, size int64) (LINEResult, error)
}
