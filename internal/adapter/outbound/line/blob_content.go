package line

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"

	mediatransfer "github.com/963sup/linebot-platform/internal/application/usecases/media-transfer"
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

func readBlob(response *http.Response, err error) (mediatransfer.BinaryContent, error) {
	if err != nil {
		return mediatransfer.BinaryContent{}, lineAPIError("blobRead", response, err)
	}
	if response == nil || response.Body == nil {
		return mediatransfer.BinaryContent{}, errors.New("LINE blob response is empty")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxLINEBlobBytes+1))
	if err != nil {
		return mediatransfer.BinaryContent{}, lineAPIError("blobReadBody", response, err)
	}
	if len(data) > maxLINEBlobBytes {
		return mediatransfer.BinaryContent{}, errors.New("LINE blob exceeds 20 MiB limit")
	}
	return mediatransfer.BinaryContent{Data: data, ContentType: response.Header.Get("Content-Type")}, nil
}

func (s *BlobSender) FetchMessageContent(ctx context.Context, id string, variant mediatransfer.ContentVariant) (mediatransfer.BinaryContent, error) {
	c, e := s.clientFor(ctx)
	if e != nil {
		return mediatransfer.BinaryContent{}, e
	}
	var r *http.Response
	if variant == mediatransfer.Preview {
		r, e = c.GetMessageContentPreview(id)
	} else {
		r, e = c.GetMessageContent(id)
	}
	return readBlob(r, e)
}
func (s *BlobSender) FetchPresentationImage(ctx context.Context, id string) (mediatransfer.BinaryContent, error) {
	c, e := s.clientFor(ctx)
	if e != nil {
		return mediatransfer.BinaryContent{}, e
	}
	r, e := c.GetRichMenuImage(id)
	return readBlob(r, e)
}
func (s *BlobSender) StorePresentationImage(ctx context.Context, id string, blob mediatransfer.BinaryContent) error {
	c, e := s.clientFor(ctx)
	if e != nil {
		return e
	}
	r, _, e := c.SetRichMenuImageWithHttpInfo(id, blob.ContentType, bytes.NewReader(blob.Data))
	return lineWriteError("setRichMenuImage", r, e)
}
func (s *BlobSender) MessageContentStatus(ctx context.Context, id string) (mediatransfer.PreparationStatus, error) {
	c, e := s.clientFor(ctx)
	if e != nil {
		return "", e
	}
	r, v, e := c.GetMessageContentTranscodingByMessageIdWithHttpInfo(id)
	if e != nil {
		return "", lineAPIError("getMessageContentTranscoding", r, e)
	}
	if v == nil {
		return "", errors.New("LINE transcoding status is empty")
	}
	switch v.Status {
	case "succeeded":
		return mediatransfer.PreparationReady, nil
	case "failed":
		return mediatransfer.PreparationFailed, nil
	case "processing":
		return mediatransfer.PreparationProcessing, nil
	default:
		return "", errors.New("LINE transcoding status is invalid")
	}
}

var _ mediatransfer.Gateway = (*BlobSender)(nil)
