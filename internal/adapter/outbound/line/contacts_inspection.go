package line

import (
	"context"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

func (s *Sender) FollowerIDs(ctx context.Context, start string, limit int32) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetFollowersWithHttpInfo(start, limit)
	return typedReportResult("getFollowers", response, result, err)
}
func (s *Sender) GroupSummary(ctx context.Context, groupID string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetGroupSummaryWithHttpInfo(groupID)
	return typedReportResult("getGroupSummary", response, result, err)
}
func (s *Sender) GroupMemberCount(ctx context.Context, groupID string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetGroupMemberCountWithHttpInfo(groupID)
	return typedReportResult("getGroupMemberCount", response, result, err)
}
func (s *Sender) RoomMemberCount(ctx context.Context, roomID string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetRoomMemberCountWithHttpInfo(roomID)
	return typedReportResult("getRoomMemberCount", response, result, err)
}
func (s *Sender) GroupMemberIDs(ctx context.Context, groupID, start string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetGroupMembersIdsWithHttpInfo(groupID, start)
	return typedReportResult("getGroupMembersIds", response, result, err)
}
func (s *Sender) RoomMemberIDs(ctx context.Context, roomID, start string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetRoomMembersIdsWithHttpInfo(roomID, start)
	return typedReportResult("getRoomMembersIds", response, result, err)
}
func (s *Sender) GroupMemberProfile(ctx context.Context, groupID, userID string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetGroupMemberProfileWithHttpInfo(groupID, userID)
	return typedReportResult("getGroupMemberProfile", response, result, err)
}
func (s *Sender) RoomMemberProfile(ctx context.Context, roomID, userID string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetRoomMemberProfileWithHttpInfo(roomID, userID)
	return typedReportResult("getRoomMemberProfile", response, result, err)
}
