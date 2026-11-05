package line

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	insightsreporting "github.com/963sup/linebot-platform/internal/application/usecases/insights-reporting"
	providerconfiguration "github.com/963sup/linebot-platform/internal/application/usecases/provider-configuration"
)

// reportResult strips provider errors of response bodies and never logs raw SDK responses.
func reportResult(op string, response *http.Response, data any, err error) (json.RawMessage, error) {
	if err != nil {
		return nil, lineAPIError(op, response, err)
	}
	if data == nil {
		return nil, errors.New("empty LINE report")
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, errors.New("invalid LINE report")
	}
	if string(payload) == "null" {
		return nil, errors.New("empty LINE report")
	}
	return json.RawMessage(payload), nil
}

func lineReadError(operation string, response *http.Response, err error) error {
	return lineAPIError(operation, response, err)
}

func typedReportResult(op string, response *http.Response, data any, err error) (providerconfiguration.ChannelSnapshot, error) {
	payload, err := reportResult(op, response, data, err)
	if err != nil {
		return providerconfiguration.ChannelSnapshot{}, err
	}
	value, err := decodeJSONValue(payload)
	if err != nil {
		return providerconfiguration.ChannelSnapshot{}, errors.New("invalid LINE report")
	}
	return providerconfiguration.ChannelSnapshot{Value: value}, nil
}

func decodeJSONValue(payload []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, errors.New("trailing JSON data")
	}
	return value, nil
}

func (s *Sender) BotInfo(ctx context.Context) (providerconfiguration.ChannelSnapshot, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return providerconfiguration.ChannelSnapshot{}, err
	}
	response, result, err := client.GetBotInfoWithHttpInfo()
	return typedReportResult("getBotInfo", response, result, err)
}

func (s *Sender) WebhookEndpoint(ctx context.Context) (providerconfiguration.ChannelSnapshot, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return providerconfiguration.ChannelSnapshot{}, err
	}
	response, result, err := client.GetWebhookEndpointWithHttpInfo()
	return typedReportResult("getWebhookEndpoint", response, result, err)
}

func (s *Sender) MessageQuota(ctx context.Context) (providerconfiguration.ChannelSnapshot, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return providerconfiguration.ChannelSnapshot{}, err
	}
	response, result, err := client.GetMessageQuotaWithHttpInfo()
	return typedReportResult("getMessageQuota", response, result, err)
}

func (s *Sender) MessageQuotaConsumption(ctx context.Context) (providerconfiguration.ChannelSnapshot, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return providerconfiguration.ChannelSnapshot{}, err
	}
	response, result, err := client.GetMessageQuotaConsumptionWithHttpInfo()
	return typedReportResult("getMessageQuotaConsumption", response, result, err)
}

func (s *Sender) AggregationUnitUsage(ctx context.Context) (providerconfiguration.ChannelSnapshot, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return providerconfiguration.ChannelSnapshot{}, err
	}
	response, result, err := client.GetAggregationUnitUsageWithHttpInfo()
	return typedReportResult("getAggregationUnitUsage", response, result, err)
}

func (s *Sender) AggregationUnitNames(ctx context.Context, limit, start string) (providerconfiguration.ChannelSnapshot, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return providerconfiguration.ChannelSnapshot{}, err
	}
	response, result, err := client.GetAggregationUnitNameListWithHttpInfo(limit, start)
	return typedReportResult("getAggregationUnitNameList", response, result, err)
}

func (s *Sender) DeliveryReply(ctx context.Context, query insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return insightsreporting.CountReport{}, err
	}
	response, result, err := client.GetNumberOfSentReplyMessagesWithHttpInfo(query.Day.Format("20060102"))
	return deliveryCountResult("getNumberOfSentReplyMessages", response, result, err, query.Day)
}

func (s *Sender) DeliveryPush(ctx context.Context, query insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return insightsreporting.CountReport{}, err
	}
	response, result, err := client.GetNumberOfSentPushMessagesWithHttpInfo(query.Day.Format("20060102"))
	return deliveryCountResult("getNumberOfSentPushMessages", response, result, err, query.Day)
}

func (s *Sender) DeliveryMulticast(ctx context.Context, query insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return insightsreporting.CountReport{}, err
	}
	response, result, err := client.GetNumberOfSentMulticastMessagesWithHttpInfo(query.Day.Format("20060102"))
	return deliveryCountResult("getNumberOfSentMulticastMessages", response, result, err, query.Day)
}

func (s *Sender) DeliveryBroadcast(ctx context.Context, query insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return insightsreporting.CountReport{}, err
	}
	response, result, err := client.GetNumberOfSentBroadcastMessagesWithHttpInfo(query.Day.Format("20060102"))
	return deliveryCountResult("getNumberOfSentBroadcastMessages", response, result, err, query.Day)
}

func (s *Sender) NarrowcastProgress(ctx context.Context, requestID string) (providerconfiguration.ChannelSnapshot, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return providerconfiguration.ChannelSnapshot{}, err
	}
	response, result, err := client.GetNarrowcastProgressWithHttpInfo(requestID)
	return typedReportResult("getNarrowcastProgress", response, result, err)
}

func deliveryCountResult(op string, response *http.Response, data any, err error, observedAt time.Time) (insightsreporting.CountReport, error) {
	payload, err := reportResult(op, response, data, err)
	if err != nil {
		return insightsreporting.CountReport{}, err
	}
	value, err := decodeJSONValue(payload)
	if err != nil {
		return insightsreporting.CountReport{}, errors.New("invalid LINE report")
	}
	object, _ := value.(map[string]any)
	return insightsreporting.CountReport{
		Accepted:   int64Value(object, "success", "successCount"),
		Delivered:  int64Value(object, "success", "successCount"),
		Failed:     int64Value(object, "failure", "failureCount"),
		ObservedAt: observedAt,
	}, nil
}

var _ providerconfiguration.ChannelReader = (*Sender)(nil)
var _ insightsreporting.DeliveryReader = (*Sender)(nil)
