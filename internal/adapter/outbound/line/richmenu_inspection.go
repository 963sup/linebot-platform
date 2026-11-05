package line

import (
	"context"
	"errors"
	"fmt"

	experiencepresentation "github.com/963sup/linebot-platform/internal/application/usecases/experience-presentation"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func (s *Sender) ListSurfaces(ctx context.Context) ([]experiencepresentation.ProviderSurface, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetRichMenuListWithHttpInfo()
	if err != nil {
		return nil, lineReadError("getRichMenuList", response, err)
	}
	if result == nil {
		return nil, errors.New("empty LINE rich menu list")
	}
	out := make([]experiencepresentation.ProviderSurface, 0, len(result.Richmenus))
	for _, item := range result.Richmenus {
		out = append(out, surfaceView(item.RichMenuId, item.Name, item.Size, item.Selected))
	}
	return out, nil
}

func (s *Sender) GetSurface(ctx context.Context, id experiencepresentation.ProviderSurfaceID) (experiencepresentation.ProviderSurface, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return experiencepresentation.ProviderSurface{}, err
	}
	response, result, err := client.GetRichMenuWithHttpInfo(string(id))
	if err != nil {
		return experiencepresentation.ProviderSurface{}, lineReadError("getRichMenu", response, err)
	}
	if result == nil {
		return experiencepresentation.ProviderSurface{}, errors.New("empty LINE rich menu")
	}
	return surfaceView(result.RichMenuId, result.Name, result.Size, result.Selected), nil
}

func (s *Sender) GetDefaultSurface(ctx context.Context) (experiencepresentation.ProviderSurface, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return experiencepresentation.ProviderSurface{}, err
	}
	response, result, err := client.GetDefaultRichMenuIdWithHttpInfo()
	if err != nil {
		return experiencepresentation.ProviderSurface{}, lineReadError("getDefaultRichMenuId", response, err)
	}
	if result == nil {
		return experiencepresentation.ProviderSurface{}, errors.New("empty LINE default rich menu")
	}
	return experiencepresentation.ProviderSurface{ID: experiencepresentation.ProviderSurfaceID(result.RichMenuId), Selected: true}, nil
}

func (s *Sender) ListAliases(ctx context.Context) ([]experiencepresentation.ProviderAlias, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return nil, err
	}
	response, result, err := client.GetRichMenuAliasListWithHttpInfo()
	if err != nil {
		return nil, lineReadError("getRichMenuAliasList", response, err)
	}
	if result == nil {
		return nil, errors.New("empty LINE rich menu alias list")
	}
	out := make([]experiencepresentation.ProviderAlias, 0, len(result.Aliases))
	for _, item := range result.Aliases {
		out = append(out, experiencepresentation.ProviderAlias{ID: experiencepresentation.AliasID(item.RichMenuAliasId), SurfaceID: experiencepresentation.ProviderSurfaceID(item.RichMenuId)})
	}
	return out, nil
}

func (s *Sender) GetAlias(ctx context.Context, id experiencepresentation.AliasID) (experiencepresentation.ProviderAlias, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return experiencepresentation.ProviderAlias{}, err
	}
	response, result, err := client.GetRichMenuAliasWithHttpInfo(string(id))
	if err != nil {
		return experiencepresentation.ProviderAlias{}, lineReadError("getRichMenuAlias", response, err)
	}
	if result == nil {
		return experiencepresentation.ProviderAlias{}, errors.New("empty LINE rich menu alias")
	}
	return experiencepresentation.ProviderAlias{ID: experiencepresentation.AliasID(result.RichMenuAliasId), SurfaceID: experiencepresentation.ProviderSurfaceID(result.RichMenuId)}, nil
}

func (s *Sender) GetRecipientSurface(ctx context.Context, userID experiencepresentation.RecipientID) (experiencepresentation.ProviderSurface, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return experiencepresentation.ProviderSurface{}, err
	}
	response, result, err := client.GetRichMenuIdOfUserWithHttpInfo(string(userID))
	if err != nil {
		return experiencepresentation.ProviderSurface{}, lineReadError("getRichMenuIdOfUser", response, err)
	}
	if result == nil {
		return experiencepresentation.ProviderSurface{}, errors.New("empty LINE recipient rich menu")
	}
	return experiencepresentation.ProviderSurface{ID: experiencepresentation.ProviderSurfaceID(result.RichMenuId)}, nil
}

func (s *Sender) GetBatchProgress(ctx context.Context, id string) (experiencepresentation.BatchProgress, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return experiencepresentation.BatchProgress{}, err
	}
	response, result, err := client.GetRichMenuBatchProgressWithHttpInfo(id)
	if err != nil {
		return experiencepresentation.BatchProgress{}, lineReadError("getRichMenuBatchProgress", response, err)
	}
	if result == nil {
		return experiencepresentation.BatchProgress{}, errors.New("empty LINE rich menu batch progress")
	}
	return experiencepresentation.BatchProgress{RequestID: id, Status: string(result.Phase)}, nil
}

func surfaceView(id, name string, size *messaging_api.RichMenuSize, selected bool) experiencepresentation.ProviderSurface {
	value := ""
	if size != nil {
		value = fmt.Sprintf("%dx%d", size.Width, size.Height)
	}
	return experiencepresentation.ProviderSurface{ID: experiencepresentation.ProviderSurfaceID(id), Name: name, Size: value, Selected: selected}
}

var _ experiencepresentation.ProviderReader = (*Sender)(nil)
