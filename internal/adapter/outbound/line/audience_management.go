package line

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/963sup/linebot-platform/internal/application/usecases/audience"
	"github.com/line/line-bot-sdk-go/v8/linebot/manage_audience"
)

type AudienceManager struct {
	client *manage_audience.ManageAudienceAPI
	blob   *manage_audience.ManageAudienceBlobAPI
}

func NewAudienceManager(client *manage_audience.ManageAudienceAPI, blob *manage_audience.ManageAudienceBlobAPI) (*AudienceManager, error) {
	if client == nil || blob == nil {
		return nil, errors.New("audience management requires JSON and Blob clients")
	}
	return &AudienceManager{client: client, blob: blob}, nil
}

func (s *AudienceManager) clientFor(ctx context.Context) (*manage_audience.ManageAudienceAPI, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c := *s.client
	return c.WithContext(ctx), nil
}

func (s *AudienceManager) blobFor(ctx context.Context) (*manage_audience.ManageAudienceBlobAPI, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c := *s.blob
	return c.WithContext(ctx), nil
}

func (s *AudienceManager) CreateGroup(ctx context.Context, command audience.CreateGroupCommand) (audience.ProviderAudience, error) {
	req := manage_audience.CreateAudienceGroupRequest{Description: command.Description, Audiences: mapAudienceRecipients(command.Recipients)}
	c, err := s.clientFor(ctx)
	if err != nil {
		return audience.ProviderAudience{}, err
	}
	r, v, err := c.CreateAudienceGroupWithHttpInfo(&req)
	if err != nil {
		return audience.ProviderAudience{}, lineWriteError("createAudienceGroup", r, err)
	}
	return mapProviderAudience(v, false)
}

func (s *AudienceManager) AppendRecipients(ctx context.Context, command audience.AppendRecipientsCommand) error {
	req := manage_audience.AddAudienceToAudienceGroupRequest{AudienceGroupId: int64(command.ID), Audiences: mapAudienceRecipients(command.Recipients)}
	c, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := c.AddAudienceToAudienceGroupWithHttpInfo(&req)
	return lineWriteError("addAudienceToAudienceGroup", r, err)
}

func (s *AudienceManager) CreateClickGroup(ctx context.Context, command audience.CreateInteractionGroupCommand) (audience.ProviderAudience, error) {
	req := manage_audience.CreateClickBasedAudienceGroupRequest{Description: command.Description, RequestId: command.SourceRequestID}
	c, err := s.clientFor(ctx)
	if err != nil {
		return audience.ProviderAudience{}, err
	}
	r, v, err := c.CreateClickBasedAudienceGroupWithHttpInfo(&req)
	if err != nil {
		return audience.ProviderAudience{}, lineWriteError("createClickBasedAudienceGroup", r, err)
	}
	return mapProviderAudience(v, false)
}

func (s *AudienceManager) CreateImpressionGroup(ctx context.Context, command audience.CreateInteractionGroupCommand) (audience.ProviderAudience, error) {
	req := manage_audience.CreateImpBasedAudienceGroupRequest{Description: command.Description, RequestId: command.SourceRequestID}
	c, err := s.clientFor(ctx)
	if err != nil {
		return audience.ProviderAudience{}, err
	}
	r, v, err := c.CreateImpBasedAudienceGroupWithHttpInfo(&req)
	if err != nil {
		return audience.ProviderAudience{}, lineWriteError("createImpBasedAudienceGroup", r, err)
	}
	return mapProviderAudience(v, false)
}

func (s *AudienceManager) UpdateDescription(ctx context.Context, id audience.ProviderAudienceID, description string) error {
	req := manage_audience.UpdateAudienceGroupDescriptionRequest{Description: description}
	c, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := c.UpdateAudienceGroupDescriptionWithHttpInfo(int64(id), &req)
	return lineWriteError("updateAudienceGroupDescription", r, err)
}

func (s *AudienceManager) DeleteGroup(ctx context.Context, id audience.ProviderAudienceID) error {
	c, err := s.clientFor(ctx)
	if err != nil {
		return err
	}
	r, _, err := c.DeleteAudienceGroupWithHttpInfo(int64(id))
	return lineWriteError("deleteAudienceGroup", r, err)
}

func audienceTempFile(contents []byte) (*os.File, func(), error) {
	if len(contents) == 0 || len(contents) > 20<<20 {
		return nil, nil, errors.New("invalid audience upload size")
	}
	f, err := os.CreateTemp("", "line-audience-*")
	if err != nil {
		return nil, nil, errors.New("cannot create temporary upload")
	}
	cleanup := func() { _ = f.Close(); _ = os.Remove(f.Name()) }
	if _, err := f.Write(contents); err != nil {
		cleanup()
		return nil, nil, errors.New("cannot write temporary upload")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		cleanup()
		return nil, nil, errors.New("cannot rewind temporary upload")
	}
	return f, cleanup, nil
}

func (s *AudienceManager) CreateFromFile(ctx context.Context, file audience.AudienceFile) (audience.ProviderAudience, error) {
	c, err := s.blobFor(ctx)
	if err != nil {
		return audience.ProviderAudience{}, err
	}
	f, cleanup, err := audienceTempFile(file.Data)
	if err != nil {
		return audience.ProviderAudience{}, err
	}
	defer cleanup()
	r, v, err := c.CreateAudienceForUploadingUserIdsWithHttpInfo(f, file.Description, file.IFAAudience, file.UploadDescription)
	if err != nil {
		return audience.ProviderAudience{}, lineWriteError("createAudienceForUploadingUserIds", r, err)
	}
	return mapProviderAudience(v, false)
}

func (s *AudienceManager) AppendFromFile(ctx context.Context, id audience.ProviderAudienceID, file audience.AudienceFile) error {
	c, err := s.blobFor(ctx)
	if err != nil {
		return err
	}
	f, cleanup, err := audienceTempFile(file.Data)
	if err != nil {
		return err
	}
	defer cleanup()
	r, _, err := c.AddUserIdsToAudienceWithHttpInfo(f, int64(id), file.UploadDescription)
	return lineWriteError("addUserIdsToAudience", r, err)
}

func mapAudienceRecipients(values []string) []manage_audience.Audience {
	result := make([]manage_audience.Audience, 0, len(values))
	for _, value := range values {
		result = append(result, manage_audience.Audience{Id: value})
	}
	return result
}

var _ audience.ProviderManager = (*AudienceManager)(nil)
