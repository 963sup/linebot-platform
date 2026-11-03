package line

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/963sup/linebot-platform/internal/infrastructure/resilience"
)

// reportResult strips provider errors of response bodies and never logs raw SDK responses.
func reportResult(op string, response *http.Response, data any, err error) (json.RawMessage, error) {
	if err != nil {
		return nil, resilience.Wrap("line", op, httpStatus(response), err)
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

func (s *Sender) BotInfo(ctx context.Context) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetBotInfoWithHttpInfo()
	return reportResult("getBotInfo", response, result, err)
}

func (s *Sender) WebhookEndpoint(ctx context.Context) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetWebhookEndpointWithHttpInfo()
	return reportResult("getWebhookEndpoint", response, result, err)
}

func (s *Sender) MessageQuota(ctx context.Context) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetMessageQuotaWithHttpInfo()
	return reportResult("getMessageQuota", response, result, err)
}

func (s *Sender) MessageQuotaConsumption(ctx context.Context) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetMessageQuotaConsumptionWithHttpInfo()
	return reportResult("getMessageQuotaConsumption", response, result, err)
}

func (s *Sender) AggregationUnitUsage(ctx context.Context) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetAggregationUnitUsageWithHttpInfo()
	return reportResult("getAggregationUnitUsage", response, result, err)
}

func (s *Sender) AggregationUnitNames(ctx context.Context, limit, start string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetAggregationUnitNameListWithHttpInfo(limit, start)
	return reportResult("getAggregationUnitNameList", response, result, err)
}

func (s *Sender) DeliveryReply(ctx context.Context, date string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetNumberOfSentReplyMessagesWithHttpInfo(date)
	return reportResult("getNumberOfSentReplyMessages", response, result, err)
}

func (s *Sender) DeliveryPush(ctx context.Context, date string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetNumberOfSentPushMessagesWithHttpInfo(date)
	return reportResult("getNumberOfSentPushMessages", response, result, err)
}

func (s *Sender) DeliveryMulticast(ctx context.Context, date string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetNumberOfSentMulticastMessagesWithHttpInfo(date)
	return reportResult("getNumberOfSentMulticastMessages", response, result, err)
}

func (s *Sender) DeliveryBroadcast(ctx context.Context, date string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetNumberOfSentBroadcastMessagesWithHttpInfo(date)
	return reportResult("getNumberOfSentBroadcastMessages", response, result, err)
}

func (s *Sender) NarrowcastProgress(ctx context.Context, requestID string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetNarrowcastProgressWithHttpInfo(requestID)
	return reportResult("getNarrowcastProgress", response, result, err)
}
