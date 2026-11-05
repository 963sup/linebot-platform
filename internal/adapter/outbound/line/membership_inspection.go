package line

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/membership"
)

func (s *Sender) ListMemberships(ctx context.Context, page membership.PageRequest) (membership.MembershipPage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return membership.MembershipPage{}, err
	}
	response, result, err := client.GetMembershipListWithHttpInfo()
	if err != nil {
		return membership.MembershipPage{}, lineReadError("getMembershipList", response, err)
	}
	if result == nil {
		return membership.MembershipPage{}, errors.New("empty LINE membership list")
	}
	out := membership.MembershipPage{Memberships: make([]membership.MembershipView, 0, len(result.Memberships))}
	for _, item := range result.Memberships {
		out.Memberships = append(out.Memberships, membership.MembershipView{ID: membership.MembershipID(strconv.FormatInt(int64(item.MembershipId), 10)), Name: item.Title, ObservedAt: time.Now().UTC()})
	}
	return out, nil
}
func (s *Sender) GetSubscription(ctx context.Context, userID membership.RecipientID) (membership.SubscriptionView, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return membership.SubscriptionView{}, err
	}
	response, result, err := client.GetMembershipSubscriptionWithHttpInfo(string(userID))
	if err != nil {
		return membership.SubscriptionView{}, lineReadError("getMembershipSubscription", response, err)
	}
	if result == nil {
		return membership.SubscriptionView{}, errors.New("empty LINE membership subscription")
	}
	return membership.SubscriptionView{RecipientID: userID, ObservedAt: time.Now().UTC()}, nil
}
func (s *Sender) ListMembershipMembers(ctx context.Context, membershipID membership.MembershipID, page membership.PageRequest) (membership.MemberPage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return membership.MemberPage{}, err
	}
	id, err := strconv.ParseInt(string(membershipID), 10, 32)
	if err != nil {
		return membership.MemberPage{}, errors.New("invalid membership ID")
	}
	response, result, err := client.GetJoinedMembershipUsersWithHttpInfo(int32(id), page.Token, int32(page.Limit))
	if err != nil {
		return membership.MemberPage{}, lineReadError("getJoinedMembershipUsers", response, err)
	}
	if result == nil {
		return membership.MemberPage{}, errors.New("empty LINE membership members")
	}
	out := membership.MemberPage{NextToken: result.Next, Members: make([]membership.MemberView, 0, len(result.UserIds))}
	for _, id := range result.UserIds {
		out.Members = append(out.Members, membership.MemberView{RecipientID: membership.RecipientID(id), ObservedAt: time.Now().UTC()})
	}
	return out, nil
}

var _ membership.Reader = (*Sender)(nil)
