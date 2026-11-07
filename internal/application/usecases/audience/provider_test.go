package audience

import (
	"context"
	"testing"
)

type fakeProviderReader struct{ page PageRequest }

func (f *fakeProviderReader) GetAudience(_ context.Context, id ProviderAudienceID, shared bool) (ProviderAudience, error) {
	return ProviderAudience{ID: id, Shared: shared}, nil
}
func (f *fakeProviderReader) ListAudiences(_ context.Context, page PageRequest, _ bool) (ProviderAudiencePage, error) {
	f.page = page
	return ProviderAudiencePage{Page: page.Number}, nil
}

func TestProviderQueriesValidateAndDispatch(t *testing.T) {
	reader := &fakeProviderReader{}
	queries, err := NewProviderQueries(reader)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := queries.Get(context.Background(), 7, true); err != nil || got.ID != 7 || !got.Shared {
		t.Fatalf("Get() = %+v, %v", got, err)
	}
	page := PageRequest{Number: 2, Size: 50}
	if _, err := queries.List(context.Background(), page, false); err != nil || reader.page != page {
		t.Fatalf("List() page = %+v, error = %v", reader.page, err)
	}
	if _, err := queries.Get(context.Background(), 0, false); err == nil {
		t.Fatal("Get accepted zero ID")
	}
	if _, err := queries.List(context.Background(), PageRequest{Number: 1, Size: 101}, false); err == nil {
		t.Fatal("List accepted oversized page")
	}
}

type fakeProviderManager struct{ created CreateGroupCommand }

func (f *fakeProviderManager) CreateGroup(_ context.Context, c CreateGroupCommand) (ProviderAudience, error) {
	f.created = c
	return ProviderAudience{ID: 1}, nil
}
func (*fakeProviderManager) AppendRecipients(context.Context, AppendRecipientsCommand) error {
	return nil
}
func (*fakeProviderManager) CreateClickGroup(context.Context, CreateInteractionGroupCommand) (ProviderAudience, error) {
	return ProviderAudience{}, nil
}
func (*fakeProviderManager) CreateImpressionGroup(context.Context, CreateInteractionGroupCommand) (ProviderAudience, error) {
	return ProviderAudience{}, nil
}
func (*fakeProviderManager) UpdateDescription(context.Context, ProviderAudienceID, string) error {
	return nil
}
func (*fakeProviderManager) DeleteGroup(context.Context, ProviderAudienceID) error { return nil }
func (*fakeProviderManager) CreateFromFile(context.Context, AudienceFile) (ProviderAudience, error) {
	return ProviderAudience{}, nil
}
func (*fakeProviderManager) AppendFromFile(context.Context, ProviderAudienceID, AudienceFile) error {
	return nil
}

func TestProviderOperationsUseTypedCommands(t *testing.T) {
	manager := &fakeProviderManager{}
	operations, err := NewProviderOperations(manager)
	if err != nil {
		t.Fatal(err)
	}
	command := CreateGroupCommand{Description: "customers", Recipients: []string{"user-1"}}
	if _, err := operations.CreateGroup(context.Background(), command); err != nil || manager.created.Description != command.Description {
		t.Fatalf("CreateGroup() command = %+v, error = %v", manager.created, err)
	}
	if _, err := operations.CreateGroup(context.Background(), CreateGroupCommand{Description: "customers"}); err == nil {
		t.Fatal("CreateGroup accepted no recipients")
	}
	if _, err := operations.CreateInteractionGroup(context.Background(), CreateInteractionGroupCommand{Description: "clicked", SourceRequestID: "request-1"}, false); err != nil {
		t.Fatalf("CreateInteractionGroup rejected valid provider request: %v", err)
	}
}

func TestProviderContextValidation(t *testing.T) {
	if err := validateContext(nil); err != ErrContextRequired {
		t.Fatalf("nil context: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := validateContext(ctx); err != context.Canceled {
		t.Fatalf("canceled context: %v", err)
	}
	if err := validateContext(context.Background()); err != nil {
		t.Fatal(err)
	}
}
