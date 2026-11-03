package line

import (
	"context"
	"encoding/json"
)

func (s *Sender) MembershipList(ctx context.Context) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetMembershipListWithHttpInfo()
	return reportResult("getMembershipList", response, result, err)
}
func (s *Sender) MembershipSubscription(ctx context.Context, userID string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetMembershipSubscriptionWithHttpInfo(userID)
	return reportResult("getMembershipSubscription", response, result, err)
}
func (s *Sender) MembershipUsers(ctx context.Context, membershipID int32, start string, limit int32) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetJoinedMembershipUsersWithHttpInfo(membershipID, start, limit)
	return reportResult("getJoinedMembershipUsers", response, result, err)
}
