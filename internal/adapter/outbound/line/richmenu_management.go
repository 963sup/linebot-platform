package line

import (
	"context"
	"net/http"

	"github.com/963sup/linebot-platform/internal/infrastructure/resilience"
)

func lineWriteError(operation string, response *http.Response, err error) error {
	if err == nil {
		return nil
	}
	return resilience.Wrap("line", operation, httpStatus(response), err)
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
