package engagement

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

const maxAudienceFileSize = 20 << 20

type AudienceCommand struct {
	Action            string
	ID                int64
	Request           outbound.LINERequest
	File              []byte
	Description       string
	UploadDescription string
	IFAAudience       bool
}

type AudienceOperations struct{ manager outbound.AudienceManager }

func isLINEObject(request outbound.LINERequest) bool {
	_, ok := request.Value.(map[string]any)
	return ok
}

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
		if c.ID != 0 || len(c.File) != 0 || !isLINEObject(c.Request) {
			return errors.New("audience action requires a JSON object with no file or group ID")
		}
	case "update-description":
		if c.ID <= 0 || len(c.File) != 0 || !isLINEObject(c.Request) {
			return errors.New("update-description requires group ID and JSON object")
		}
	case "delete":
		if c.ID <= 0 || c.Request.Value != nil || len(c.File) != 0 {
			return errors.New("delete requires only a positive group ID")
		}
	case "create-file":
		if c.ID != 0 || c.Request.Value != nil || c.Description == "" ||
			len(c.File) == 0 || len(c.File) > maxAudienceFileSize {
			return errors.New("create-file requires bounded user ID file and description")
		}
	case "append-file":
		if c.ID <= 0 || c.Request.Value != nil || len(c.File) == 0 ||
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

func (s *AudienceOperations) Execute(ctx context.Context, c AudienceCommand) (outbound.LINEResult, error) {
	if ctx == nil {
		return outbound.LINEResult{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return outbound.LINEResult{}, err
	}
	if err := ValidateAudienceCommand(c); err != nil {
		return outbound.LINEResult{}, err
	}
	switch c.Action {
	case "create":
		return s.manager.CreateAudienceGroup(ctx, c.Request)
	case "append":
		return outbound.LINEResult{}, s.manager.AddAudienceToGroup(ctx, c.Request)
	case "create-click":
		return s.manager.CreateClickAudience(ctx, c.Request)
	case "create-impression":
		return s.manager.CreateImpressionAudience(ctx, c.Request)
	case "update-description":
		return outbound.LINEResult{}, s.manager.UpdateAudienceDescription(ctx, c.ID, c.Request)
	case "delete":
		return outbound.LINEResult{}, s.manager.DeleteAudienceGroup(ctx, c.ID)
	case "create-file":
		return s.manager.CreateAudienceFromFile(ctx, c.File, c.Description, c.IFAAudience, c.UploadDescription)
	case "append-file":
		return outbound.LINEResult{}, s.manager.AddAudienceFromFile(ctx, c.ID, c.File, c.UploadDescription)
	default:
		return outbound.LINEResult{}, errors.New("unsupported audience action")
	}
}
