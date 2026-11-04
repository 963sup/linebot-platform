package line

import (
	"context"
	"encoding/json"
)

func (s *Sender) FollowerIDs(ctx context.Context, start string, limit int32) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetFollowersWithHttpInfo(start, limit)
	return reportResult("getFollowers", response, result, err)
}
func (s *Sender) GroupSummary(ctx context.Context, groupID string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetGroupSummaryWithHttpInfo(groupID)
	return reportResult("getGroupSummary", response, result, err)
}
func (s *Sender) GroupMemberCount(ctx context.Context, groupID string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetGroupMemberCountWithHttpInfo(groupID)
	return reportResult("getGroupMemberCount", response, result, err)
}
func (s *Sender) RoomMemberCount(ctx context.Context, roomID string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetRoomMemberCountWithHttpInfo(roomID)
	return reportResult("getRoomMemberCount", response, result, err)
}
func (s *Sender) GroupMemberIDs(ctx context.Context, groupID, start string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetGroupMembersIdsWithHttpInfo(groupID, start)
	return reportResult("getGroupMembersIds", response, result, err)
}
func (s *Sender) RoomMemberIDs(ctx context.Context, roomID, start string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetRoomMembersIdsWithHttpInfo(roomID, start)
	return reportResult("getRoomMembersIds", response, result, err)
}
func (s *Sender) GroupMemberProfile(ctx context.Context, groupID, userID string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetGroupMemberProfileWithHttpInfo(groupID, userID)
	return reportResult("getGroupMemberProfile", response, result, err)
}
func (s *Sender) RoomMemberProfile(ctx context.Context, roomID, userID string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetRoomMemberProfileWithHttpInfo(roomID, userID)
	return reportResult("getRoomMemberProfile", response, result, err)
}
