package outbound

import "context"

// ContactsReader only performs read operations. Caller authorization,
// privacy and retention are enforced at the inbound boundary.
type ContactsReader interface {
	FollowerIDs(ctx context.Context, start string, limit int32) (LINEResult, error)
	GroupSummary(ctx context.Context, groupID string) (LINEResult, error)
	GroupMemberCount(ctx context.Context, groupID string) (LINEResult, error)
	RoomMemberCount(ctx context.Context, roomID string) (LINEResult, error)
	GroupMemberIDs(ctx context.Context, groupID, start string) (LINEResult, error)
	RoomMemberIDs(ctx context.Context, roomID, start string) (LINEResult, error)
	GroupMemberProfile(ctx context.Context, groupID, userID string) (LINEResult, error)
	RoomMemberProfile(ctx context.Context, roomID, userID string) (LINEResult, error)
}
