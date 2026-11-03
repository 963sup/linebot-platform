package engagement

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type RichMenuQuery struct {
	Kind string
	ID   string
}

type RichMenuInspector struct {
	reader outbound.RichMenuReader
}

func NewRichMenuInspector(reader outbound.RichMenuReader) (*RichMenuInspector, error) {
	if reader == nil {
		return nil, errors.New("rich menu reader is required")
	}
	return &RichMenuInspector{reader: reader}, nil
}

func ValidateRichMenuQuery(query RichMenuQuery) error {
	switch query.Kind {
	case "list", "default", "alias-list":
		if query.ID != "" {
			return errors.New("rich menu ID not accepted for this query")
		}
		return nil
	case "menu", "alias", "user", "batch-progress":
		if query.ID == "" || query.ID != strings.TrimSpace(query.ID) {
			return errors.New("rich menu query ID is required")
		}
		return nil
	default:
		return errors.New("unsupported rich menu query kind")
	}
}

func (s *RichMenuInspector) Inspect(ctx context.Context, query RichMenuQuery) (json.RawMessage, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateRichMenuQuery(query); err != nil {
		return nil, err
	}
	switch query.Kind {
	case "list":
		return s.reader.RichMenuList(ctx)
	case "menu":
		return s.reader.RichMenu(ctx, query.ID)
	case "default":
		return s.reader.DefaultRichMenu(ctx)
	case "alias-list":
		return s.reader.RichMenuAliasList(ctx)
	case "alias":
		return s.reader.RichMenuAlias(ctx, query.ID)
	case "user":
		return s.reader.UserRichMenu(ctx, query.ID)
	default:
		return s.reader.RichMenuBatchProgress(ctx, query.ID)
	}
}
