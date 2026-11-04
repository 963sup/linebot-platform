package line

import (
	"context"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

func (s *Sender) CouponList(ctx context.Context, start string, limit int32) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	statuses := []string{} // SDK v8.23.0 dereferences this pointer even for an unfiltered request.
	response, result, err := client.ListCouponWithHttpInfo(&statuses, start, limit)
	return typedReportResult("listCoupon", response, result, err)
}
func (s *Sender) CouponDetail(ctx context.Context, couponID string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetCouponDetailWithHttpInfo(couponID)
	return typedReportResult("getCouponDetail", response, result, err)
}
