package line

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func (s *Sender) PushMessagesByPhone(ctx context.Context, payload outbound.LINERequest) error {
	var req messaging_api.PnpMessagesRequest
	if err := decodeLINERequest(payload, &req); err != nil {
		return errors.New("invalid LINE PNP request")
	}
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := client.PushMessagesByPhoneWithHttpInfo(&req, "")
	return lineWriteError("pushMessagesByPhone", r, err)
}

func (s *Sender) PushTemplatedMessagesByPhone(ctx context.Context, payload outbound.LINERequest) error {
	var req messaging_api.PnpTemplatedMessageRequest
	if err := decodeLINERequest(payload, &req); err != nil {
		return errors.New("invalid LINE PNP template request")
	}
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := client.PushTemplatedMessagesByPhoneWithHttpInfo(&req, "")
	return lineWriteError("pushTemplatedMessagesByPhone", r, err)
}

func (s *Sender) PNPMessageStatistics(ctx context.Context, date string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	r, v, err := client.GetPNPMessageStatisticsWithHttpInfo(date)
	return typedReportResult("getPNPMessageStatistics", r, v, err)
}

func (s *Sender) PNPTemplatedMessageStatistics(ctx context.Context, date string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	r, v, err := client.GetPNPTemplatedMessageStatisticsWithHttpInfo(date)
	return typedReportResult("getPNPTemplatedMessageStatistics", r, v, err)
}

func (s *Sender) CreateCoupon(ctx context.Context, payload outbound.LINERequest) (outbound.LINEResult, error) {
	var request messaging_api.CouponCreateRequest
	if err := decodeLINERequest(payload, &request); err != nil {
		return outbound.LINEResult{}, errors.New("invalid LINE coupon creation request")
	}
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	r, v, err := client.CreateCouponWithHttpInfo(&request)
	return typedReportResult("createCoupon", r, v, err)
}

func (s *Sender) CloseCoupon(ctx context.Context, couponID string) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := client.CloseCouponWithHttpInfo(couponID)
	return lineWriteError("closeCoupon", r, err)
}
