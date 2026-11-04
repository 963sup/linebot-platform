package line

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func (s *Sender) PushMessagesByPhone(ctx context.Context, payload json.RawMessage) error {
	var req messaging_api.PnpMessagesRequest
	if err := json.Unmarshal(payload, &req); err != nil { return errors.New("invalid LINE PNP request") }
	client, err := s.clientFor(ctx)
	if err != nil { return err }
	r, _, err := client.PushMessagesByPhoneWithHttpInfo(&req, "")
	return lineWriteError("pushMessagesByPhone", r, err)
}

func (s *Sender) PushTemplatedMessagesByPhone(ctx context.Context, payload json.RawMessage) error {
	var req messaging_api.PnpTemplatedMessageRequest
	if err := json.Unmarshal(payload, &req); err != nil { return errors.New("invalid LINE PNP template request") }
	client, err := s.clientFor(ctx)
	if err != nil { return err }
	r, _, err := client.PushTemplatedMessagesByPhoneWithHttpInfo(&req, "")
	return lineWriteError("pushTemplatedMessagesByPhone", r, err)
}

func (s *Sender) PNPMessageStatistics(ctx context.Context, date string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil { return nil, err }
	r, v, err := client.GetPNPMessageStatisticsWithHttpInfo(date)
	return reportResult("getPNPMessageStatistics", r, v, err)
}

func (s *Sender) PNPTemplatedMessageStatistics(ctx context.Context, date string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil { return nil, err }
	r, v, err := client.GetPNPTemplatedMessageStatisticsWithHttpInfo(date)
	return reportResult("getPNPTemplatedMessageStatistics", r, v, err)
}

func (s *Sender) CreateCoupon(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var request messaging_api.CouponCreateRequest
	if err := json.Unmarshal(payload, &request); err != nil { return nil, errors.New("invalid LINE coupon creation request") }
	client, err := s.clientFor(ctx)
	if err != nil { return nil, err }
	r, v, err := client.CreateCouponWithHttpInfo(&request)
	return reportResult("createCoupon", r, v, err)
}

func (s *Sender) CloseCoupon(ctx context.Context, couponID string) error {
	client, err := s.clientFor(ctx)
	if err != nil { return err }
	r, _, err := client.CloseCouponWithHttpInfo(couponID)
	return lineWriteError("closeCoupon", r, err)
}
