package line

import (
	"context"
	"encoding/json"
	"errors"

	accountlinktoken "github.com/963sup/linebot-platform/internal/application/usecases/account-link-token"
	conversationoperations "github.com/963sup/linebot-platform/internal/application/usecases/conversation-operations"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func (s *Sender) Leave(ctx context.Context, conversation conversationoperations.Conversation) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	switch conversation.Kind {
	case conversationoperations.Group:
		r, _, err := client.LeaveGroupWithHttpInfo(conversation.ID)
		return lineWriteError("leaveGroup", r, err)
	case conversationoperations.Room:
		r, _, err := client.LeaveRoomWithHttpInfo(conversation.ID)
		return lineWriteError("leaveRoom", r, err)
	default:
		return errors.New("unsupported LINE conversation kind")
	}
}

func (s *Sender) ShowLoading(ctx context.Context, indicator conversationoperations.LoadingIndicator) error {
	request := messaging_api.ShowLoadingAnimationRequest{ChatId: indicator.ChatID, LoadingSeconds: int32(indicator.Duration.Seconds())}
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := client.ShowLoadingAnimationWithHttpInfo(&request)
	return lineWriteError("showLoadingAnimation", r, err)
}

func (s *Sender) MarkRead(ctx context.Context, target conversationoperations.ReadTarget) error {
	request := messaging_api.MarkMessagesAsReadRequest{Chat: &messaging_api.ChatReference{UserId: target.UserID}}
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := client.MarkMessagesAsReadWithHttpInfo(&request)
	return lineWriteError("markMessagesAsRead", r, err)
}

func (s *Sender) MarkReadByToken(ctx context.Context, token conversationoperations.ReadToken) error {
	request := messaging_api.MarkMessagesAsReadByTokenRequest{MarkAsReadToken: token.Token}
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

func (s *Sender) Issue(ctx context.Context, subject accountlinktoken.Subject) (accountlinktoken.Token, error) {
	payload, err := s.IssueLinkToken(ctx, subject.ID)
	if err != nil {
		return accountlinktoken.Token{}, err
	}
	var response struct {
		LinkToken string `json:"linkToken"`
	}
	if err := json.Unmarshal(payload, &response); err != nil || response.LinkToken == "" {
		return accountlinktoken.Token{}, errors.New("invalid LINE link token response")
	}
	return accountlinktoken.Token{Value: response.LinkToken}, nil
}

var (
	_ conversationoperations.Gateway = (*Sender)(nil)
	_ accountlinktoken.Gateway       = (*Sender)(nil)
)
