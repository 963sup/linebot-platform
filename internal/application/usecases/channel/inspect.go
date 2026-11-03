package channel

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type Inspection struct {
	Kind      string
	Date      string
	RequestID string
	Limit     int
	Start     string
}

type Inspector struct {
	reader outbound.ChannelReader
}

func NewInspector(reader outbound.ChannelReader) (*Inspector, error) {
	if reader == nil {
		return nil, errors.New("channel reader is required")
	}
	return &Inspector{reader: reader}, nil
}

// ValidateInspection rejects unknown kinds and malformed parameters before SDK construction.
func ValidateInspection(q Inspection) error {
	switch q.Kind {
	case "bot-info", "webhook-endpoint", "message-quota", "message-quota-consumption", "aggregation-usage":
		return nil
	case "aggregation-names":
		if q.Limit < 0 || q.Limit > 100 || strings.TrimSpace(q.Start) != q.Start {
			return errors.New("invalid aggregation pagination")
		}
		return nil
	case "delivery-reply", "delivery-push", "delivery-multicast", "delivery-broadcast":
		if !validDate(q.Date) {
			return errors.New("delivery date must be valid YYYYMMDD")
		}
		return nil
	case "narrowcast-progress":
		if q.RequestID == "" || strings.TrimSpace(q.RequestID) != q.RequestID {
			return errors.New("narrowcast request ID is required")
		}
		return nil
	default:
		return errors.New("unsupported channel inspection kind")
	}
}

// Inspect only dispatches an explicitly whitelisted set of read-only queries.
// No production LINE request is made without the caller's opt-in.
func (s *Inspector) Inspect(ctx context.Context, q Inspection) (json.RawMessage, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateInspection(q); err != nil { return nil, err }
	switch q.Kind {
	case "bot-info":
		return s.reader.BotInfo(ctx)
	case "webhook-endpoint":
		return s.reader.WebhookEndpoint(ctx)
	case "message-quota":
		return s.reader.MessageQuota(ctx)
	case "message-quota-consumption":
		return s.reader.MessageQuotaConsumption(ctx)
	case "aggregation-usage":
		return s.reader.AggregationUnitUsage(ctx)
	case "aggregation-names":
		if q.Limit < 0 || q.Limit > 100 || strings.TrimSpace(q.Start) != q.Start {
			return nil, errors.New("invalid aggregation pagination")
		}
		limit := ""
		if q.Limit > 0 {
			limit = strconv.Itoa(q.Limit)
		}
		return s.reader.AggregationUnitNames(ctx, limit, q.Start)
	case "delivery-reply", "delivery-push", "delivery-multicast", "delivery-broadcast":
		if !validDate(q.Date) {
			return nil, errors.New("delivery date must be valid YYYYMMDD")
		}
		switch q.Kind {
		case "delivery-reply":
			return s.reader.DeliveryReply(ctx, q.Date)
		case "delivery-push":
			return s.reader.DeliveryPush(ctx, q.Date)
		case "delivery-multicast":
			return s.reader.DeliveryMulticast(ctx, q.Date)
		default:
			return s.reader.DeliveryBroadcast(ctx, q.Date)
		}
	case "narrowcast-progress":
		if q.RequestID == "" || strings.TrimSpace(q.RequestID) != q.RequestID {
			return nil, errors.New("narrowcast request ID is required")
		}
		return s.reader.NarrowcastProgress(ctx, q.RequestID)
	default:
		return nil, errors.New("unsupported channel inspection kind")
	}
}

func validDate(value string) bool {
	if len(value) != 8 {
		return false
	}
	d, err := time.Parse("20060102", value)
	return err == nil && d.Format("20060102") == value
}
