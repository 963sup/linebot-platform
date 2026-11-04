package line

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/infrastructure/resilience"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

const maxLINEBlobBytes = 20 << 20

// BlobSender owns LINE's separate Blob API client. It is deliberately kept
// apart from Sender because Blob reads have different retention constraints.
type BlobSender struct {
	client *messaging_api.MessagingApiBlobAPI
}

func NewBlobSender(client *messaging_api.MessagingApiBlobAPI) (*BlobSender, error) {
	if client == nil {
		return nil, errors.New("LINE Blob API client is required")
	}
	return &BlobSender{client: client}, nil
}

func (s *BlobSender) clientFor(ctx context.Context) (*messaging_api.MessagingApiBlobAPI, error) {
	if ctx == nil {
		return nil, errors.New("request context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	copy := *s.client
	return copy.WithContext(ctx), nil
}

func readBlob(response *http.Response, err error) (outbound.BinaryContent, error) {
	if err != nil {
		return outbound.BinaryContent{}, resilience.Wrap("line", "blobRead", httpStatus(response), err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxLINEBlobBytes+1))
	if err != nil {
		return outbound.BinaryContent{}, err
	}
	if len(data) > maxLINEBlobBytes {
		return outbound.BinaryContent{}, errors.New("LINE blob exceeds 20 MiB limit")
	}
	return outbound.BinaryContent{Data: data, ContentType: response.Header.Get("Content-Type")}, nil
}

func (s *BlobSender) MessageContent(ctx context.Context, id string) (outbound.BinaryContent, error) {
	c, e := s.clientFor(ctx)
	if e != nil {
		return outbound.BinaryContent{}, e
	}
	r, e := c.GetMessageContent(id)
	return readBlob(r, e)
}
func (s *BlobSender) MessageContentPreview(ctx context.Context, id string) (outbound.BinaryContent, error) {
	c, e := s.clientFor(ctx)
	if e != nil {
		return outbound.BinaryContent{}, e
	}
	r, e := c.GetMessageContentPreview(id)
	return readBlob(r, e)
}
func (s *BlobSender) RichMenuImage(ctx context.Context, id string) (outbound.BinaryContent, error) {
	c, e := s.clientFor(ctx)
	if e != nil {
		return outbound.BinaryContent{}, e
	}
	r, e := c.GetRichMenuImage(id)
	return readBlob(r, e)
}
func (s *BlobSender) SetRichMenuImage(ctx context.Context, id string, blob outbound.BinaryContent) error {
	c, e := s.clientFor(ctx)
	if e != nil {
		return e
	}
	r, _, e := c.SetRichMenuImageWithHttpInfo(id, blob.ContentType, bytes.NewReader(blob.Data))
	return lineWriteError("setRichMenuImage", r, e)
}
func (s *BlobSender) MessageContentTranscoding(ctx context.Context, id string) (outbound.LINEResult, error) {
	c, e := s.clientFor(ctx)
	if e != nil {
		return outbound.LINEResult{}, e
	}
	r, v, e := c.GetMessageContentTranscodingByMessageIdWithHttpInfo(id)
	if e != nil {
		return outbound.LINEResult{}, resilience.Wrap("line", "getMessageContentTranscoding", httpStatus(r), e)
	}
	return typedReportResult("getMessageContentTranscoding", r, v, nil)
}
