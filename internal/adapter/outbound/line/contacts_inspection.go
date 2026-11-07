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
	if conversation.Kind != contactdirectory.RecipientGroup {
		return contactdirectory.ConversationSummary{}, errors.New("unsupported LINE conversation kind")
	}
	r1, summary, err := client.GetGroupSummaryWithHttpInfo(conversation.ID)
	if err != nil {
		return contactdirectory.ConversationSummary{}, lineWriteError("getGroupSummary", r1, err)
	}
	r2, count, err := client.GetGroupMemberCountWithHttpInfo(conversation.ID)
	if err != nil {
		return contactdirectory.ConversationSummary{}, lineWriteError("getGroupMemberCount", r2, err)
	}
	summaryObject, err := providerObject(summary)
	if err != nil {
		return contactdirectory.ConversationSummary{}, err
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
	if conversation.Kind != contactdirectory.RecipientGroup {
		return contactdirectory.RecipientPage{}, errors.New("unsupported LINE conversation kind")
	}
	r, result, err := client.GetGroupMembersIdsWithHttpInfo(conversation.ID, page.Cursor)
	if err != nil {
		return contactdirectory.RecipientPage{}, lineWriteError("getGroupMembersIds", r, err)
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
	if conversation.Kind != contactdirectory.RecipientGroup {
		return contactdirectory.ProfileSnapshot{}, errors.New("unsupported LINE conversation kind")
	}
	r, result, err := client.GetGroupMemberProfileWithHttpInfo(conversation.ID, string(member))
	if err != nil {
		return contactdirectory.ProfileSnapshot{}, lineWriteError("getGroupMemberProfile", r, err)
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
