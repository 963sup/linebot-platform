package line

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/application/usecases/audience"
	"github.com/line/line-bot-sdk-go/v8/linebot/manage_audience"
)

type AudienceReader struct {
	client *manage_audience.ManageAudienceAPI
}

func NewAudienceReader(client *manage_audience.ManageAudienceAPI) (*AudienceReader, error) {
	if client == nil {
		return nil, errors.New("LINE audience client is required")
	}
	return &AudienceReader{client: client}, nil
}

func (s *AudienceReader) clientFor(ctx context.Context) (*manage_audience.ManageAudienceAPI, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	call := *s.client
	return call.WithContext(ctx), nil
}

func (s *AudienceReader) GetAudience(ctx context.Context, id audience.ProviderAudienceID, shared bool) (audience.ProviderAudience, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return audience.ProviderAudience{}, err
	}
	var result any
	var responseErr error
	if shared {
		response, value, callErr := client.GetSharedAudienceDataWithHttpInfo(int64(id))
		responseErr = lineWriteError("getSharedAudienceData", response, callErr)
		result = value
	} else {
		response, value, callErr := client.GetAudienceDataWithHttpInfo(int64(id))
		responseErr = lineWriteError("getAudienceData", response, callErr)
		result = value
	}
	if responseErr != nil {
		return audience.ProviderAudience{}, responseErr
	}
	return mapProviderAudience(result, shared)
}
func (s *AudienceReader) ListAudiences(ctx context.Context, page audience.PageRequest, shared bool) (audience.ProviderAudiencePage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return audience.ProviderAudiencePage{}, err
	}
	var result any
	var responseErr error
	if shared {
		response, value, callErr := client.GetSharedAudienceGroupsWithHttpInfo(page.Number, "", "", page.Size, "", false)
		responseErr = lineWriteError("getSharedAudienceGroups", response, callErr)
		result = value
	} else {
		response, value, callErr := client.GetAudienceGroupsWithHttpInfo(page.Number, "", "", page.Size, false, "")
		responseErr = lineWriteError("getAudienceGroups", response, callErr)
		result = value
	}
	if responseErr != nil {
		return audience.ProviderAudiencePage{}, responseErr
	}
	var groups []manage_audience.AudienceGroup
	var hasNext bool
	switch value := result.(type) {
	case *manage_audience.GetAudienceGroupsResponse:
		if value == nil {
			return audience.ProviderAudiencePage{}, errors.New("invalid LINE audience page")
		}
		groups, hasNext = value.AudienceGroups, value.HasNextPage
	case *manage_audience.GetSharedAudienceGroupsResponse:
		if value == nil {
			return audience.ProviderAudiencePage{}, errors.New("invalid LINE audience page")
		}
		groups, hasNext = value.AudienceGroups, value.HasNextPage
	default:
		return audience.ProviderAudiencePage{}, errors.New("invalid LINE audience page")
	}
	if groups == nil {
		return audience.ProviderAudiencePage{}, errors.New("invalid LINE audience page")
	}
	output := audience.ProviderAudiencePage{Page: page.Number}
	if hasNext {
		output.NextPage = page.Number + 1
	}
	for _, value := range groups {
		mapped, mapErr := mapProviderAudience(value, shared)
		if mapErr != nil {
			return audience.ProviderAudiencePage{}, mapErr
		}
		output.Audiences = append(output.Audiences, mapped)
	}
	return output, nil
}

func mapProviderAudience(value any, shared bool) (audience.ProviderAudience, error) {
	object, err := providerObject(value)
	if err != nil {
		return audience.ProviderAudience{}, err
	}
	created := timeValue(object, "created")
	return audience.ProviderAudience{ID: audience.ProviderAudienceID(int64Value(object, "audienceGroupId")), Description: stringValue(object, "description"), Status: stringValue(object, "status"), Shared: shared, CreatedAt: created}, nil
}

var _ audience.ProviderReader = (*AudienceReader)(nil)
