package engagement

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type AudienceQuery struct {
	Kind string
	ID   int64
	Page int64
	Size int64
}

type AudienceInspector struct {
	reader outbound.AudienceReader
}

func NewAudienceInspector(reader outbound.AudienceReader) (*AudienceInspector, error) {
	if reader == nil {
		return nil, errors.New("audience reader is required")
	}
	return &AudienceInspector{reader: reader}, nil
}

func ValidateAudienceQuery(q AudienceQuery) error {
	switch q.Kind {
	case "group", "shared-group":
		if q.ID <= 0 {
			return errors.New("audience ID must be positive")
		}
		return nil
	case "list", "shared-list":
		if q.Page < 1 || q.Size < 1 || q.Size > 100 {
			return errors.New("audience page must be positive and size must be 1-100")
		}
		if q.ID != 0 {
			return errors.New("audience ID not allowed for list requests")
		}
		return nil
	default:
		return errors.New("unsupported audience query kind")
	}
}

func (s *AudienceInspector) Inspect(ctx context.Context, q AudienceQuery) (json.RawMessage, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateAudienceQuery(q); err != nil {
		return nil, err
	}
	switch q.Kind {
	case "group":
		return s.reader.AudienceGroup(ctx, q.ID)
	case "shared-group":
		return s.reader.SharedAudienceGroup(ctx, q.ID)
	case "list":
		return s.reader.AudienceGroups(ctx, q.Page, q.Size)
	default:
		return s.reader.SharedAudienceGroups(ctx, q.Page, q.Size)
	}
}
