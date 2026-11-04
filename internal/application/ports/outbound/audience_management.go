package outbound

import "context"

// AudienceManager is an operator-only write port, separate from read-only
// audience inspection. File uploads use bounded bytes, never public webhook data.
type AudienceManager interface {
	CreateAudienceGroup(context.Context, LINERequest) (LINEResult, error)
	AddAudienceToGroup(context.Context, LINERequest) error
	CreateClickAudience(context.Context, LINERequest) (LINEResult, error)
	CreateImpressionAudience(context.Context, LINERequest) (LINEResult, error)
	UpdateAudienceDescription(context.Context, int64, LINERequest) error
	DeleteAudienceGroup(context.Context, int64) error
	CreateAudienceFromFile(context.Context, []byte, string, bool, string) (LINEResult, error)
	AddAudienceFromFile(context.Context, int64, []byte, string) error
}
