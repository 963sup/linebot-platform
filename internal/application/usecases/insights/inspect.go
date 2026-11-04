package insights

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type Query struct {
	Kind      string
	Date      string
	RequestID string
	Unit      string
	RichMenu  string
	From      string
	To        string
}

type Inspector struct {
	reader outbound.InsightsReader
}

func NewInspector(reader outbound.InsightsReader) (*Inspector, error) {
	if reader == nil {
		return nil, errors.New("insights reader is required")
	}
	return &Inspector{reader: reader}, nil
}

func validDay(value string) bool {
	if len(value) != 8 {
		return false
	}
	day, err := time.Parse("20060102", value)
	return err == nil && day.Format("20060102") == value
}

func ValidateQuery(q Query) error {
	switch q.Kind {
	case "demographics":
		return nil
	case "deliveries", "followers":
		if !validDay(q.Date) {
			return errors.New("insights date must be valid YYYYMMDD")
		}
		return nil
	case "message-event":
		if q.RequestID == "" || q.RequestID != strings.TrimSpace(q.RequestID) {
			return errors.New("message event request ID is required")
		}
		return nil
	case "statistics-per-unit", "menu-summary", "menu-daily":
		if !validDay(q.From) || !validDay(q.To) || q.From > q.To {
			return errors.New("insights date range must be valid YYYYMMDD and ordered")
		}
		switch q.Kind {
		case "statistics-per-unit":
			if q.Unit == "" || q.Unit != strings.TrimSpace(q.Unit) {
				return errors.New("aggregation unit name is required")
			}
		default:
			if q.RichMenu == "" || q.RichMenu != strings.TrimSpace(q.RichMenu) {
				return errors.New("rich menu ID is required")
			}
		}
		return nil
	default:
		return errors.New("unsupported insights inspection kind")
	}
}

func (s *Inspector) Inspect(ctx context.Context, q Query) (json.RawMessage, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateQuery(q); err != nil {
		return nil, err
	}
	switch q.Kind {
	case "demographics":
		return s.reader.FriendsDemographics(ctx)
	case "deliveries":
		return s.reader.MessageDeliveries(ctx, q.Date)
	case "followers":
		return s.reader.FollowerMetrics(ctx, q.Date)
	case "message-event":
		return s.reader.MessageEvent(ctx, q.RequestID)
	case "statistics-per-unit":
		return s.reader.StatisticsPerUnit(ctx, q.Unit, q.From, q.To)
	case "menu-summary":
		return s.reader.RichMenuSummary(ctx, q.RichMenu, q.From, q.To)
	default:
		return s.reader.RichMenuDaily(ctx, q.RichMenu, q.From, q.To)
	}
}
