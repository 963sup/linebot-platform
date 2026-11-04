package line

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
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

func (s *AudienceReader) AudienceGroup(ctx context.Context, id int64) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetAudienceDataWithHttpInfo(id)
	return typedReportResult("getAudienceData", response, result, err)
}
func (s *AudienceReader) SharedAudienceGroup(ctx context.Context, id int64) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetSharedAudienceDataWithHttpInfo(id)
	return typedReportResult("getSharedAudienceData", response, result, err)
}
func (s *AudienceReader) AudienceGroups(ctx context.Context, page, size int64) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetAudienceGroupsWithHttpInfo(page, "", "", size, false, "")
	return typedReportResult("getAudienceGroups", response, result, err)
}
func (s *AudienceReader) SharedAudienceGroups(ctx context.Context, page, size int64) (outbound.LINEResult, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return outbound.LINEResult{}, err
	}
	response, result, err := client.GetSharedAudienceGroupsWithHttpInfo(page, "", "", size, "", false)
	return typedReportResult("getSharedAudienceGroups", response, result, err)
}
