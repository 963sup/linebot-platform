package line

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/line/line-bot-sdk-go/v8/linebot/manage_audience"
)

type AudienceManager struct {
	client *manage_audience.ManageAudienceAPI
	blob *manage_audience.ManageAudienceBlobAPI
}

func NewAudienceManager(client *manage_audience.ManageAudienceAPI, blob *manage_audience.ManageAudienceBlobAPI) (*AudienceManager, error) {
	if client == nil || blob == nil { return nil, errors.New("audience management requires JSON and Blob clients") }
	return &AudienceManager{client: client, blob: blob}, nil
}

func (s *AudienceManager) clientFor(ctx context.Context) (*manage_audience.ManageAudienceAPI, error) {
	if ctx == nil { return nil, errors.New("context is required") }
	if err := ctx.Err(); err != nil { return nil, err }
	c := *s.client
	return c.WithContext(ctx), nil
}

func (s *AudienceManager) blobFor(ctx context.Context) (*manage_audience.ManageAudienceBlobAPI, error) {
	if ctx == nil { return nil, errors.New("context is required") }
	if err := ctx.Err(); err != nil { return nil, err }
	c := *s.blob
	return c.WithContext(ctx), nil
}

func (s *AudienceManager) CreateAudienceGroup(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var req manage_audience.CreateAudienceGroupRequest
	if err := json.Unmarshal(payload, &req); err != nil { return nil, errors.New("invalid LINE audience request") }
	c, err := s.clientFor(ctx); if err != nil { return nil, err }
	r, v, err := c.CreateAudienceGroupWithHttpInfo(&req)
	return reportResult("createAudienceGroup", r, v, err)
}

func (s *AudienceManager) AddAudienceToGroup(ctx context.Context, payload json.RawMessage) error {
	var req manage_audience.AddAudienceToAudienceGroupRequest
	if err := json.Unmarshal(payload, &req); err != nil { return errors.New("invalid audience append request") }
	c, err := s.clientFor(ctx); if err != nil { return err }
	r, _, err := c.AddAudienceToAudienceGroupWithHttpInfo(&req)
	return lineWriteError("addAudienceToAudienceGroup", r, err)
}

func (s *AudienceManager) CreateClickAudience(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var req manage_audience.CreateClickBasedAudienceGroupRequest
	if err := json.Unmarshal(payload, &req); err != nil { return nil, errors.New("invalid click audience request") }
	c, err := s.clientFor(ctx); if err != nil { return nil, err }
	r, v, err := c.CreateClickBasedAudienceGroupWithHttpInfo(&req)
	return reportResult("createClickBasedAudienceGroup", r, v, err)
}

func (s *AudienceManager) CreateImpressionAudience(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var req manage_audience.CreateImpBasedAudienceGroupRequest
	if err := json.Unmarshal(payload, &req); err != nil { return nil, errors.New("invalid impression audience request") }
	c, err := s.clientFor(ctx); if err != nil { return nil, err }
	r, v, err := c.CreateImpBasedAudienceGroupWithHttpInfo(&req)
	return reportResult("createImpBasedAudienceGroup", r, v, err)
}

func (s *AudienceManager) UpdateAudienceDescription(ctx context.Context, id int64, payload json.RawMessage) error {
	var req manage_audience.UpdateAudienceGroupDescriptionRequest
	if err := json.Unmarshal(payload, &req); err != nil { return errors.New("invalid audience description request") }
	c, err := s.clientFor(ctx); if err != nil { return err }
	r, _, err := c.UpdateAudienceGroupDescriptionWithHttpInfo(id, &req)
	return lineWriteError("updateAudienceGroupDescription", r, err)
}

func (s *AudienceManager) DeleteAudienceGroup(ctx context.Context, id int64) error {
	c, err := s.clientFor(ctx); if err != nil { return err }
	r, _, err := c.DeleteAudienceGroupWithHttpInfo(id)
	return lineWriteError("deleteAudienceGroup", r, err)
}

func audienceTempFile(contents []byte) (*os.File, func(), error) {
	if len(contents) == 0 || len(contents) > 20<<20 {
		return nil, nil, errors.New("invalid audience upload size")
	}
	f, err := os.CreateTemp("", "line-audience-*")
	if err != nil { return nil, nil, errors.New("cannot create temporary upload") }
	cleanup := func(){ _ = f.Close(); _ = os.Remove(f.Name()) }
	if _, err := f.Write(contents); err != nil { cleanup(); return nil, nil, errors.New("cannot write temporary upload") }
	if _, err := f.Seek(0, io.SeekStart); err != nil { cleanup(); return nil, nil, errors.New("cannot rewind temporary upload") }
	return f, cleanup, nil
}

func (s *AudienceManager) CreateAudienceFromFile(ctx context.Context, contents []byte, description string, isIFA bool, uploadDescription string) (json.RawMessage, error) {
	c, err := s.blobFor(ctx); if err != nil { return nil, err }
	f, cleanup, err := audienceTempFile(contents); if err != nil { return nil, err }
	defer cleanup()
	r, v, err := c.CreateAudienceForUploadingUserIdsWithHttpInfo(f, description, isIFA, uploadDescription)
	return reportResult("createAudienceForUploadingUserIds", r, v, err)
}

func (s *AudienceManager) AddAudienceFromFile(ctx context.Context, id int64, contents []byte, uploadDescription string) error {
	c, err := s.blobFor(ctx); if err != nil { return err }
	f, cleanup, err := audienceTempFile(contents); if err != nil { return err }
	defer cleanup()
	r, _, err := c.AddUserIdsToAudienceWithHttpInfo(f, id, uploadDescription)
	return lineWriteError("addUserIdsToAudience", r, err)
}
