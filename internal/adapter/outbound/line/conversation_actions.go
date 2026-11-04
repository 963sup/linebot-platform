package line

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func (s *Sender) LeaveGroup(ctx context.Context, groupID string) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := client.LeaveGroupWithHttpInfo(groupID)
	return lineWriteError("leaveGroup", r, err)
}

func (s *Sender) LeaveRoom(ctx context.Context, roomID string) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := client.LeaveRoomWithHttpInfo(roomID)
	return lineWriteError("leaveRoom", r, err)
}

func (s *Sender) ShowLoading(ctx context.Context, payload outbound.LINERequest) error {
	var request messaging_api.ShowLoadingAnimationRequest
	if err := decodeLINERequest(payload, &request); err != nil {
		return errors.New("invalid LINE loading request")
	}
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := client.ShowLoadingAnimationWithHttpInfo(&request)
	return lineWriteError("showLoadingAnimation", r, err)
}

func (s *Sender) MarkMessagesRead(ctx context.Context, payload outbound.LINERequest) error {
	var request messaging_api.MarkMessagesAsReadRequest
	if err := decodeLINERequest(payload, &request); err != nil {
		return errors.New("invalid LINE mark-read request")
	}
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := client.MarkMessagesAsReadWithHttpInfo(&request)
	return lineWriteError("markMessagesAsRead", r, err)
}

func (s *Sender) MarkMessagesReadByToken(ctx context.Context, payload outbound.LINERequest) error {
	var request messaging_api.MarkMessagesAsReadByTokenRequest
	if err := decodeLINERequest(payload, &request); err != nil {
		return errors.New("invalid LINE mark-read-token request")
	}
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := client.MarkMessagesAsReadByTokenWithHttpInfo(&request)
	return lineWriteError("markMessagesAsReadByToken", r, err)
}

func (s *Sender) IssueLinkToken(ctx context.Context, userID string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	r, v, err := client.IssueLinkTokenWithHttpInfo(userID)
	return reportResult("issueLinkToken", r, v, err)
}
