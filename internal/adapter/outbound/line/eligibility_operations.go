package line

import (
	"context"
	"errors"

	insightsreporting "github.com/963sup/linebot-platform/internal/application/usecases/insights-reporting"
	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/963sup/linebot-platform/internal/application/usecases/promotion"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func (s *Sender) PushMessagesByPhone(ctx context.Context, payload messagingdelivery.PhoneDeliveryRequest) error {
	var req messaging_api.PnpMessagesRequest
	if err := decodeLINERequest(payload.Payload, &req); err != nil {
		return errors.New("invalid LINE PNP request")
	}
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := client.PushMessagesByPhoneWithHttpInfo(&req, "")
	return lineWriteError("pushMessagesByPhone", r, err)
}

func (s *Sender) PushTemplatedMessagesByPhone(ctx context.Context, payload messagingdelivery.PhoneDeliveryRequest) error {
	var req messaging_api.PnpTemplatedMessageRequest
	if err := decodeLINERequest(payload.Payload, &req); err != nil {
		return errors.New("invalid LINE PNP template request")
	}
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := client.PushTemplatedMessagesByPhoneWithHttpInfo(&req, "")
	return lineWriteError("pushTemplatedMessagesByPhone", r, err)
}

func (s *Sender) PNPMessageStatistics(ctx context.Context, query insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return insightsreporting.CountReport{}, err
	}
	r, v, err := client.GetPNPMessageStatisticsWithHttpInfo(query.Day.Format("20060102"))
	return deliveryCountResult("getPNPMessageStatistics", r, v, err, query.Day)
}

func (s *Sender) PNPTemplatedMessageStatistics(ctx context.Context, query insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return insightsreporting.CountReport{}, err
	}
	r, v, err := client.GetPNPTemplatedMessageStatisticsWithHttpInfo(query.Day.Format("20060102"))
	return deliveryCountResult("getPNPTemplatedMessageStatistics", r, v, err, query.Day)
}

func (s *Sender) CreateCouponDocument(ctx context.Context, payload promotion.ProviderCouponDocument) (promotion.CouponView, error) {
	var request messaging_api.CouponCreateRequest
	if err := decodeLINERequest(payload.Value, &request); err != nil {
		return promotion.CouponView{}, errors.New("invalid LINE coupon creation request")
	}
	client, err := s.clientFor(ctx)
	if err != nil {
		return promotion.CouponView{}, err
	}
	r, v, err := client.CreateCouponWithHttpInfo(&request)
	if err != nil {
		return promotion.CouponView{}, lineReadError("createCoupon", r, err)
	}
	if v == nil {
		return promotion.CouponView{}, errors.New("empty LINE coupon creation response")
	}
	return promotion.CouponView{ID: promotion.CouponID(v.CouponId)}, nil
}

func (s *Sender) CloseCoupon(ctx context.Context, couponID promotion.CouponID) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := client.CloseCouponWithHttpInfo(string(couponID))
	return lineWriteError("closeCoupon", r, err)
}

var _ messagingdelivery.PhoneDeliveryGateway = (*Sender)(nil)
var _ promotion.ProviderCouponCreator = (*Sender)(nil)
var _ promotion.ProviderCouponGateway = (*Sender)(nil)
var _ insightsreporting.PhoneDeliveryReader = (*Sender)(nil)
