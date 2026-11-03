package outbound

import (
	"context"
	"encoding/json"
)

// ContactsReader only performs read operations. Caller authorization,
// privacy and retention are enforced at the inbound boundary.
type ContactsReader interface {
	FollowerIDs(ctx context.Context, start string, limit int32) (json.RawMessage, error)
	GroupSummary(ctx context.Context, groupID string) (json.RawMessage, error)
	GroupMemberCount(ctx context.Context, groupID string) (json.RawMessage, error)
	RoomMemberCount(ctx context.Context, roomID string) (json.RawMessage, error)
	GroupMemberIDs(ctx context.Context, groupID, start string) (json.RawMessage, error)
	RoomMemberIDs(ctx context.Context, roomID, start string) (json.RawMessage, error)
	GroupMemberProfile(ctx context.Context, groupID, userID string) (json.RawMessage, error)
	RoomMemberProfile(ctx context.Context, roomID, userID string) (json.RawMessage, error)
}
