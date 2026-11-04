package contacts

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type Inspection struct {
	Kind           string
	ConversationID string
	UserID         string
	Start          string
	Limit          int
}

type Inspector struct {
	reader outbound.ContactsReader
}

func NewInspector(reader outbound.ContactsReader) (*Inspector, error) {
	if reader == nil {
		return nil, errors.New("contacts reader is required")
	}
	return &Inspector{reader: reader}, nil
}

// ValidateInspection does not initiate a network call or store personal data.
func ValidateInspection(q Inspection) error {
	if q.Limit < 0 || q.Limit > 1000 || strings.TrimSpace(q.Start) != q.Start {
		return errors.New("invalid contacts pagination")
	}
	switch q.Kind {
	case "followers":
		if q.ConversationID != "" || q.UserID != "" {
			return errors.New("follower query does not accept conversation ID or user ID")
		}
		return nil
	case "group-summary", "group-member-count", "room-member-count", "group-members", "room-members", "group-member-profile", "room-member-profile":
		if q.ConversationID == "" || q.ConversationID != strings.TrimSpace(q.ConversationID) {
			return errors.New("group/room ID is required")
		}
		if q.Kind == "group-member-profile" || q.Kind == "room-member-profile" {
			if q.UserID == "" || q.UserID != strings.TrimSpace(q.UserID) {
				return errors.New("member user ID is required")
			}
		} else if q.UserID != "" {
			return errors.New("member user ID only allowed for profile queries")
		}
		return nil
	default:
		return errors.New("unsupported contacts inspection kind")
	}
}

func (s *Inspector) Inspect(ctx context.Context, q Inspection) (json.RawMessage, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateInspection(q); err != nil {
		return nil, err
	}
	switch q.Kind {
	case "followers":
		return s.reader.FollowerIDs(ctx, q.Start, int32(q.Limit))
	case "group-summary":
		return s.reader.GroupSummary(ctx, q.ConversationID)
	case "group-member-count":
		return s.reader.GroupMemberCount(ctx, q.ConversationID)
	case "room-member-count":
		return s.reader.RoomMemberCount(ctx, q.ConversationID)
	case "group-members":
		return s.reader.GroupMemberIDs(ctx, q.ConversationID, q.Start)
	case "room-members":
		return s.reader.RoomMemberIDs(ctx, q.ConversationID, q.Start)
	case "group-member-profile":
		return s.reader.GroupMemberProfile(ctx, q.ConversationID, q.UserID)
	default:
		return s.reader.RoomMemberProfile(ctx, q.ConversationID, q.UserID)
	}
}
