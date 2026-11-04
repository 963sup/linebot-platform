package line

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
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

func typedReportResult(op string, response *http.Response, data any, err error) (outbound.LINEResult, error) {
	payload, err := reportResult(op, response, data, err)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	value, err := decodeJSONValue(payload)
	if err != nil {
		return outbound.LINEResult{}, errors.New("invalid LINE report")
	}
	return outbound.LINEResult{Value: value}, nil
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

func (s *Sender) BotInfo(ctx context.Context) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetBotInfoWithHttpInfo()
	return typedReportResult("getBotInfo", response, result, err)
}

func (s *Sender) WebhookEndpoint(ctx context.Context) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetWebhookEndpointWithHttpInfo()
	return typedReportResult("getWebhookEndpoint", response, result, err)
}

func (s *Sender) MessageQuota(ctx context.Context) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetMessageQuotaWithHttpInfo()
	return typedReportResult("getMessageQuota", response, result, err)
}

func (s *Sender) MessageQuotaConsumption(ctx context.Context) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetMessageQuotaConsumptionWithHttpInfo()
	return typedReportResult("getMessageQuotaConsumption", response, result, err)
}

func (s *Sender) AggregationUnitUsage(ctx context.Context) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetAggregationUnitUsageWithHttpInfo()
	return typedReportResult("getAggregationUnitUsage", response, result, err)
}

func (s *Sender) AggregationUnitNames(ctx context.Context, limit, start string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetAggregationUnitNameListWithHttpInfo(limit, start)
	return typedReportResult("getAggregationUnitNameList", response, result, err)
}

func (s *Sender) DeliveryReply(ctx context.Context, date string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetNumberOfSentReplyMessagesWithHttpInfo(date)
	return typedReportResult("getNumberOfSentReplyMessages", response, result, err)
}

func (s *Sender) DeliveryPush(ctx context.Context, date string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetNumberOfSentPushMessagesWithHttpInfo(date)
	return typedReportResult("getNumberOfSentPushMessages", response, result, err)
}

func (s *Sender) DeliveryMulticast(ctx context.Context, date string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetNumberOfSentMulticastMessagesWithHttpInfo(date)
	return typedReportResult("getNumberOfSentMulticastMessages", response, result, err)
}

func (s *Sender) DeliveryBroadcast(ctx context.Context, date string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetNumberOfSentBroadcastMessagesWithHttpInfo(date)
	return typedReportResult("getNumberOfSentBroadcastMessages", response, result, err)
}

func (s *Sender) NarrowcastProgress(ctx context.Context, requestID string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetNarrowcastProgressWithHttpInfo(requestID)
	return typedReportResult("getNarrowcastProgress", response, result, err)
}
