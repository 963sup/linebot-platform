package contacts

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type ContentQuery struct{ Kind, MessageID string }
type ContentInspector struct{ reader outbound.MessageContentReader }

func NewContentInspector(reader outbound.MessageContentReader) (*ContentInspector, error) {
	if reader == nil {
		return nil, errors.New("message content reader is required")
	}
	return &ContentInspector{reader}, nil
}
func (s *ContentInspector) Inspect(ctx context.Context, q ContentQuery) (outbound.BinaryContent, outbound.LINEResult, error) {
	if ctx == nil {
		return outbound.BinaryContent{}, outbound.LINEResult{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return outbound.BinaryContent{}, outbound.LINEResult{}, err
	}
	if strings.TrimSpace(q.MessageID) == "" || q.MessageID != strings.TrimSpace(q.MessageID) {
		return outbound.BinaryContent{}, outbound.LINEResult{}, errors.New("message ID is required")
	}
	switch q.Kind {
	case "content":
		b, e := s.reader.MessageContent(ctx, q.MessageID)
		return b, outbound.LINEResult{}, e
	case "preview":
		b, e := s.reader.MessageContentPreview(ctx, q.MessageID)
		return b, outbound.LINEResult{}, e
	case "transcoding":
		j, e := s.reader.MessageContentTranscoding(ctx, q.MessageID)
		return outbound.BinaryContent{}, j, e
	default:
		return outbound.BinaryContent{}, outbound.LINEResult{}, errors.New("unsupported message content kind")
	}
}
