package line

import (
	"context"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

func (s *Sender) MembershipList(ctx context.Context) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetMembershipListWithHttpInfo()
	return typedReportResult("getMembershipList", response, result, err)
}
func (s *Sender) MembershipSubscription(ctx context.Context, userID string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetMembershipSubscriptionWithHttpInfo(userID)
	return typedReportResult("getMembershipSubscription", response, result, err)
}
func (s *Sender) MembershipUsers(ctx context.Context, membershipID int32, start string, limit int32) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetJoinedMembershipUsersWithHttpInfo(membershipID, start, limit)
	return typedReportResult("getJoinedMembershipUsers", response, result, err)
}
