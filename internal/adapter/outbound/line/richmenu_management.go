package line

import (
	"context"
	"net/http"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/infrastructure/resilience"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func lineWriteError(operation string, response *http.Response, err error) error {
	if err == nil {
		return nil
	}
	return resilience.Wrap("line", operation, httpStatus(response), err)
}

func (s *Sender) CreateRichMenu(ctx context.Context, payload outbound.LINERequest) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	var request messaging_api.RichMenuRequest
	if err := decodeLINERequest(payload, &request); err != nil {
		return err
	}
	response, _, err := client.CreateRichMenuWithHttpInfo(&request)
	return lineWriteError("createRichMenu", response, err)
}
func (s *Sender) ValidateRichMenu(ctx context.Context, payload outbound.LINERequest) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	var request messaging_api.RichMenuRequest
	if err := decodeLINERequest(payload, &request); err != nil {
		return err
	}
	response, _, err := client.ValidateRichMenuObjectWithHttpInfo(&request)
	return lineWriteError("validateRichMenuObject", response, err)
}
func (s *Sender) CreateRichMenuAlias(ctx context.Context, payload outbound.LINERequest) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	var request messaging_api.CreateRichMenuAliasRequest
	if err := decodeLINERequest(payload, &request); err != nil {
		return err
	}
	response, _, err := client.CreateRichMenuAliasWithHttpInfo(&request)
	return lineWriteError("createRichMenuAlias", response, err)
}
func (s *Sender) UpdateRichMenuAlias(ctx context.Context, aliasID string, payload outbound.LINERequest) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	var request messaging_api.UpdateRichMenuAliasRequest
	if err := decodeLINERequest(payload, &request); err != nil {
		return err
	}
	response, _, err := client.UpdateRichMenuAliasWithHttpInfo(aliasID, &request)
	return lineWriteError("updateRichMenuAlias", response, err)
}
func (s *Sender) BulkLinkRichMenu(ctx context.Context, payload outbound.LINERequest) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	var request messaging_api.RichMenuBulkLinkRequest
	if err := decodeLINERequest(payload, &request); err != nil {
		return err
	}
	response, _, err := client.LinkRichMenuIdToUsersWithHttpInfo(&request)
	return lineWriteError("linkRichMenuIdToUsers", response, err)
}
func (s *Sender) BulkUnlinkRichMenu(ctx context.Context, payload outbound.LINERequest) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	var request messaging_api.RichMenuBulkUnlinkRequest
	if err := decodeLINERequest(payload, &request); err != nil {
		return err
	}
	response, _, err := client.UnlinkRichMenuIdFromUsersWithHttpInfo(&request)
	return lineWriteError("unlinkRichMenuIdFromUsers", response, err)
}
func (s *Sender) RichMenuBatch(ctx context.Context, payload outbound.LINERequest) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	var request messaging_api.RichMenuBatchRequest
	if err := decodeLINERequest(payload, &request); err != nil {
		return err
	}
	response, _, err := client.RichMenuBatchWithHttpInfo(&request)
	return lineWriteError("richMenuBatch", response, err)
}
func (s *Sender) ValidateRichMenuBatch(ctx context.Context, payload outbound.LINERequest) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	var request messaging_api.RichMenuBatchRequest
	if err := decodeLINERequest(payload, &request); err != nil {
		return err
	}
	response, _, err := client.ValidateRichMenuBatchRequestWithHttpInfo(&request)
	return lineWriteError("validateRichMenuBatchRequest", response, err)
}

func (s *Sender) SetDefaultRichMenu(ctx context.Context, richMenuID string) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	response, _, err := client.SetDefaultRichMenuWithHttpInfo(richMenuID)
	return lineWriteError("setDefaultRichMenu", response, err)
}
func (s *Sender) CancelDefaultRichMenu(ctx context.Context) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	response, _, err := client.CancelDefaultRichMenuWithHttpInfo()
	return lineWriteError("cancelDefaultRichMenu", response, err)
}
func (s *Sender) LinkRichMenuToUser(ctx context.Context, userID, richMenuID string) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	response, _, err := client.LinkRichMenuIdToUserWithHttpInfo(userID, richMenuID)
	return lineWriteError("linkRichMenuIdToUser", response, err)
}
func (s *Sender) UnlinkRichMenuFromUser(ctx context.Context, userID string) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	response, _, err := client.UnlinkRichMenuIdFromUserWithHttpInfo(userID)
	return lineWriteError("unlinkRichMenuIdFromUser", response, err)
}
func (s *Sender) DeleteRichMenu(ctx context.Context, richMenuID string) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	response, _, err := client.DeleteRichMenuWithHttpInfo(richMenuID)
	return lineWriteError("deleteRichMenu", response, err)
}
func (s *Sender) DeleteRichMenuAlias(ctx context.Context, aliasID string) error {
	client, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	response, _, err := client.DeleteRichMenuAliasWithHttpInfo(aliasID)
	return lineWriteError("deleteRichMenuAlias", response, err)
}
