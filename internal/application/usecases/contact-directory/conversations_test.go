package contactdirectory

import (
	"context"
	"testing"
)

type fakeConversationReader struct {
	conversation RecipientReference
	page         PageRequest
}

func (f *fakeConversationReader) ListFollowers(_ context.Context, page PageRequest) (RecipientPage, error) {
	f.page = page
	return RecipientPage{}, nil
}
func (f *fakeConversationReader) GetConversationSummary(_ context.Context, conversation RecipientReference) (ConversationSummary, error) {
	f.conversation = conversation
	return ConversationSummary{Conversation: conversation}, nil
}
func (f *fakeConversationReader) ListMembers(_ context.Context, conversation RecipientReference, page PageRequest) (RecipientPage, error) {
	f.conversation, f.page = conversation, page
	return RecipientPage{}, nil
}
func (f *fakeConversationReader) GetMemberProfile(_ context.Context, conversation RecipientReference, member RecipientID) (ProfileSnapshot, error) {
	f.conversation = conversation
	return ProfileSnapshot{Recipient: RecipientReference{ID: string(member), Kind: RecipientUser}}, nil
}

func TestConversationQueriesNormalizeAndDispatch(t *testing.T) {
	reader := &fakeConversationReader{}
	queries, err := NewConversationQueries(reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queries.ListFollowers(context.Background(), PageRequest{}); err != nil || reader.page.Limit != DefaultPageSize {
		t.Fatalf("ListFollowers() page = %+v, error = %v", reader.page, err)
	}
	group := RecipientReference{ID: "group-1", Kind: RecipientGroup}
	if _, err := queries.ListMembers(context.Background(), group, PageRequest{Limit: 10}); err != nil || reader.conversation != group {
		t.Fatalf("ListMembers() conversation = %+v, error = %v", reader.conversation, err)
	}
}

func TestConversationQueriesRejectUserAsConversation(t *testing.T) {
	queries, _ := NewConversationQueries(&fakeConversationReader{})
	if _, err := queries.GetSummary(context.Background(), RecipientReference{ID: "user-1", Kind: RecipientUser}); err == nil {
		t.Fatal("GetSummary accepted user recipient")
	}
	if _, err := queries.GetMemberProfile(context.Background(), RecipientReference{ID: "group-1", Kind: RecipientGroup}, " "); err == nil {
		t.Fatal("GetMemberProfile accepted invalid member")
	}
}
