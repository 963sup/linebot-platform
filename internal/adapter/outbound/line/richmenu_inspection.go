package line

import (
	"context"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

func (s *Sender) RichMenuList(ctx context.Context) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetRichMenuListWithHttpInfo()
	return typedReportResult("getRichMenuList", response, result, err)
}

func (s *Sender) RichMenu(ctx context.Context, id string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetRichMenuWithHttpInfo(id)
	return typedReportResult("getRichMenu", response, result, err)
}

func (s *Sender) DefaultRichMenu(ctx context.Context) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetDefaultRichMenuIdWithHttpInfo()
	return typedReportResult("getDefaultRichMenuId", response, result, err)
}

func (s *Sender) RichMenuAliasList(ctx context.Context) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetRichMenuAliasListWithHttpInfo()
	return typedReportResult("getRichMenuAliasList", response, result, err)
}

func (s *Sender) RichMenuAlias(ctx context.Context, id string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetRichMenuAliasWithHttpInfo(id)
	return typedReportResult("getRichMenuAlias", response, result, err)
}

func (s *Sender) UserRichMenu(ctx context.Context, userID string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetRichMenuIdOfUserWithHttpInfo(userID)
	return typedReportResult("getRichMenuIdOfUser", response, result, err)
}

func (s *Sender) RichMenuBatchProgress(ctx context.Context, id string) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetRichMenuBatchProgressWithHttpInfo(id)
	return typedReportResult("getRichMenuBatchProgress", response, result, err)
}
