package engagement

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type RichMenuImageManager struct{ manager outbound.RichMenuImageManager }

func NewRichMenuImageManager(manager outbound.RichMenuImageManager) (*RichMenuImageManager, error) {
	if manager == nil {
		return nil, errors.New("rich menu image manager is required")
	}
	return &RichMenuImageManager{manager}, nil
}
func (s *RichMenuImageManager) Upload(ctx context.Context, richMenuID, contentType string, data []byte) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(richMenuID) == "" || richMenuID != strings.TrimSpace(richMenuID) || len(data) == 0 || (contentType != "image/png" && contentType != "image/jpeg") {
		return errors.New("rich menu ID, PNG or JPEG image are required")
	}
	return s.manager.SetRichMenuImage(ctx, richMenuID, outbound.BinaryContent{Data: data, ContentType: contentType})
}
