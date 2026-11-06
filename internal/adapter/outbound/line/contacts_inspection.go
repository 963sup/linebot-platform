package line

import (
	"context"
	"errors"
	"time"

	contactdirectory "github.com/963sup/linebot-platform/internal/application/usecases/contact-directory"
)

func (s *Sender) ListFollowers(ctx context.Context, page contactdirectory.PageRequest) (contactdirectory.RecipientPage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return contactdirectory.RecipientPage{}, err
	}
	response, result, err := client.GetFollowersWithHttpInfo(page.Cursor, int32(page.Limit))
	if err != nil {
		return contactdirectory.RecipientPage{}, lineWriteError("getFollowers", response, err)
	}
	object, err := providerObject(result)
	if err != nil {
		return contactdirectory.RecipientPage{}, err
	}
	output := contactdirectory.RecipientPage{NextCursor: stringValue(object, "next")}
	for _, id := range stringSlice(object["userIds"]) {
		output.Recipients = append(output.Recipients, contactdirectory.RecipientReference{ID: id, Kind: contactdirectory.RecipientUser})
	}
	return output, nil
}

func (s *Sender) GetConversationSummary(ctx context.Context, conversation contactdirectory.RecipientReference) (contactdirectory.ConversationSummary, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return contactdirectory.ConversationSummary{}, err
	}
	var summary, count any
	if conversation.Kind == contactdirectory.RecipientGroup {
		r1, v1, e1 := client.GetGroupSummaryWithHttpInfo(conversation.ID)
		if e1 != nil {
			return contactdirectory.ConversationSummary{}, lineWriteError("getGroupSummary", r1, e1)
		}
		summary = v1
		r2, v2, e2 := client.GetGroupMemberCountWithHttpInfo(conversation.ID)
		if e2 != nil {
			return contactdirectory.ConversationSummary{}, lineWriteError("getGroupMemberCount", r2, e2)
		}
		count = v2
	} else if conversation.Kind == contactdirectory.RecipientRoom {
		r, v, e := client.GetRoomMemberCountWithHttpInfo(conversation.ID)
		if e != nil {
			return contactdirectory.ConversationSummary{}, lineWriteError("getRoomMemberCount", r, e)
		}
		count = v
	} else {
		return contactdirectory.ConversationSummary{}, errors.New("unsupported LINE conversation kind")
	}
	summaryObject := make(map[string]any)
	if summary != nil {
		summaryObject, err = providerObject(summary)
		if err != nil {
			return contactdirectory.ConversationSummary{}, err
		}
	}
	countObject, err := providerObject(count)
	if err != nil {
		return contactdirectory.ConversationSummary{}, err
	}
	return contactdirectory.ConversationSummary{Conversation: conversation, Name: stringValue(summaryObject, "groupName"), PictureURL: stringValue(summaryObject, "pictureUrl"), MemberCount: int64Value(countObject, "count"), ObservedAt: time.Now()}, nil
}

func (s *Sender) ListMembers(ctx context.Context, conversation contactdirectory.RecipientReference, page contactdirectory.PageRequest) (contactdirectory.RecipientPage, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return contactdirectory.RecipientPage{}, err
	}
	var responseErr error
	var result any
	if conversation.Kind == contactdirectory.RecipientGroup {
		r, v, e := client.GetGroupMembersIdsWithHttpInfo(conversation.ID, page.Cursor)
		responseErr = lineWriteError("getGroupMembersIds", r, e)
		result = v
	} else if conversation.Kind == contactdirectory.RecipientRoom {
		r, v, e := client.GetRoomMembersIdsWithHttpInfo(conversation.ID, page.Cursor)
		responseErr = lineWriteError("getRoomMembersIds", r, e)
		result = v
	} else {
		return contactdirectory.RecipientPage{}, errors.New("unsupported LINE conversation kind")
	}
	if responseErr != nil {
		return contactdirectory.RecipientPage{}, responseErr
	}
	object, err := providerObject(result)
	if err != nil {
		return contactdirectory.RecipientPage{}, err
	}
	output := contactdirectory.RecipientPage{NextCursor: stringValue(object, "next")}
	for _, id := range stringSlice(object["memberIds"]) {
		output.Recipients = append(output.Recipients, contactdirectory.RecipientReference{ID: id, Kind: contactdirectory.RecipientUser})
	}
	return output, nil
}

func (s *Sender) GetMemberProfile(ctx context.Context, conversation contactdirectory.RecipientReference, member contactdirectory.RecipientID) (contactdirectory.ProfileSnapshot, error) {
	client, err := s.clientFor(ctx)
	if err != nil {
		return contactdirectory.ProfileSnapshot{}, err
	}
	var responseErr error
	var result any
	if conversation.Kind == contactdirectory.RecipientGroup {
		r, v, e := client.GetGroupMemberProfileWithHttpInfo(conversation.ID, string(member))
		responseErr = lineWriteError("getGroupMemberProfile", r, e)
		result = v
	} else if conversation.Kind == contactdirectory.RecipientRoom {
		r, v, e := client.GetRoomMemberProfileWithHttpInfo(conversation.ID, string(member))
		responseErr = lineWriteError("getRoomMemberProfile", r, e)
		result = v
	} else {
		return contactdirectory.ProfileSnapshot{}, errors.New("unsupported LINE conversation kind")
	}
	if responseErr != nil {
		return contactdirectory.ProfileSnapshot{}, responseErr
	}
	object, err := providerObject(result)
	if err != nil {
		return contactdirectory.ProfileSnapshot{}, err
	}
	return contactdirectory.ProfileSnapshot{Recipient: contactdirectory.RecipientReference{ID: string(member), Kind: contactdirectory.RecipientUser}, DisplayName: stringValue(object, "displayName"), PictureURL: stringValue(object, "pictureUrl"), ObservedAt: time.Now()}, nil
}

func stringSlice(value any) []string {
	values, _ := value.([]any)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

var _ contactdirectory.ConversationReader = (*Sender)(nil)
