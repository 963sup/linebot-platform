package engagement

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

const maxAudienceFileSize = 20 << 20

type AudienceCommand struct {
	Action            string
	ID                int64
	Request           json.RawMessage
	File              []byte
	Description       string
	UploadDescription string
	IFAAudience       bool
}

type AudienceOperations struct{ manager outbound.AudienceManager }

func NewAudienceOperations(m outbound.AudienceManager) (*AudienceOperations, error) {
	if m == nil {
		return nil, errors.New("audience manager is required")
	}
	return &AudienceOperations{manager: m}, nil
}

func ValidateAudienceCommand(c AudienceCommand) error {
	if c.Description != strings.TrimSpace(c.Description) ||
		c.UploadDescription != strings.TrimSpace(c.UploadDescription) {
		return errors.New("invalid audience description")
	}
	switch c.Action {
	case "create", "append", "create-click", "create-impression":
		if c.ID != 0 || len(c.File) != 0 || len(c.Request) == 0 ||
			!json.Valid(c.Request) || !strings.HasPrefix(strings.TrimSpace(string(c.Request)), "{") {
			return errors.New("audience action requires a JSON object with no file or group ID")
		}
	case "update-description":
		if c.ID <= 0 || len(c.File) != 0 || len(c.Request) == 0 ||
			!json.Valid(c.Request) || !strings.HasPrefix(strings.TrimSpace(string(c.Request)), "{") {
			return errors.New("update-description requires group ID and JSON object")
		}
	case "delete":
		if c.ID <= 0 || len(c.Request) != 0 || len(c.File) != 0 {
			return errors.New("delete requires only a positive group ID")
		}
	case "create-file":
		if c.ID != 0 || len(c.Request) != 0 || c.Description == "" ||
			len(c.File) == 0 || len(c.File) > maxAudienceFileSize {
			return errors.New("create-file requires bounded user ID file and description")
		}
	case "append-file":
		if c.ID <= 0 || len(c.Request) != 0 || len(c.File) == 0 ||
			len(c.File) > maxAudienceFileSize {
			return errors.New("append-file requires group ID and bounded user ID file")
		}
	default:
		return errors.New("unsupported audience action")
	}
	if c.Action != "create-file" && c.Action != "append-file" &&
		(c.Description != "" || c.UploadDescription != "" || c.IFAAudience) {
		return errors.New("additional audience file options are not permitted")
	}
	if c.Action == "append-file" && (c.Description != "" || c.IFAAudience) {
		return errors.New("append-file accepts only upload description")
	}
	return nil
}

func (s *AudienceOperations) Execute(ctx context.Context, c AudienceCommand) (json.RawMessage, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateAudienceCommand(c); err != nil {
		return nil, err
	}
	switch c.Action {
	case "create":
		return s.manager.CreateAudienceGroup(ctx, c.Request)
	case "append":
		return nil, s.manager.AddAudienceToGroup(ctx, c.Request)
	case "create-click":
		return s.manager.CreateClickAudience(ctx, c.Request)
	case "create-impression":
		return s.manager.CreateImpressionAudience(ctx, c.Request)
	case "update-description":
		return nil, s.manager.UpdateAudienceDescription(ctx, c.ID, c.Request)
	case "delete":
		return nil, s.manager.DeleteAudienceGroup(ctx, c.ID)
	case "create-file":
		return s.manager.CreateAudienceFromFile(ctx, c.File, c.Description, c.IFAAudience, c.UploadDescription)
	case "append-file":
		return nil, s.manager.AddAudienceFromFile(ctx, c.ID, c.File, c.UploadDescription)
	default:
		return nil, errors.New("unsupported audience action")
	}
}
