package contacts

import (
	"context"
	"encoding/json"
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
func (s *ContentInspector) Inspect(ctx context.Context, q ContentQuery) (outbound.BinaryContent, json.RawMessage, error) {
	if ctx == nil {
		return outbound.BinaryContent{}, nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return outbound.BinaryContent{}, nil, err
	}
	if strings.TrimSpace(q.MessageID) == "" || q.MessageID != strings.TrimSpace(q.MessageID) {
		return outbound.BinaryContent{}, nil, errors.New("message ID is required")
	}
	switch q.Kind {
	case "content":
		b, e := s.reader.MessageContent(ctx, q.MessageID)
		return b, nil, e
	case "preview":
		b, e := s.reader.MessageContentPreview(ctx, q.MessageID)
		return b, nil, e
	case "transcoding":
		j, e := s.reader.MessageContentTranscoding(ctx, q.MessageID)
		return outbound.BinaryContent{}, j, e
	default:
		return outbound.BinaryContent{}, nil, errors.New("unsupported message content kind")
	}
}
