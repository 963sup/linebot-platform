package line

import (
	"context"
	"encoding/json"
)

func (s *Sender) RichMenuList(ctx context.Context) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetRichMenuListWithHttpInfo()
	return reportResult("getRichMenuList", response, result, err)
}

func (s *Sender) RichMenu(ctx context.Context, id string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetRichMenuWithHttpInfo(id)
	return reportResult("getRichMenu", response, result, err)
}

func (s *Sender) DefaultRichMenu(ctx context.Context) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetDefaultRichMenuIdWithHttpInfo()
	return reportResult("getDefaultRichMenuId", response, result, err)
}

func (s *Sender) RichMenuAliasList(ctx context.Context) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetRichMenuAliasListWithHttpInfo()
	return reportResult("getRichMenuAliasList", response, result, err)
}

func (s *Sender) RichMenuAlias(ctx context.Context, id string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetRichMenuAliasWithHttpInfo(id)
	return reportResult("getRichMenuAlias", response, result, err)
}

func (s *Sender) UserRichMenu(ctx context.Context, userID string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetRichMenuIdOfUserWithHttpInfo(userID)
	return reportResult("getRichMenuIdOfUser", response, result, err)
}

func (s *Sender) RichMenuBatchProgress(ctx context.Context, id string) (json.RawMessage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetRichMenuBatchProgressWithHttpInfo(id)
	return reportResult("getRichMenuBatchProgress", response, result, err)
}
