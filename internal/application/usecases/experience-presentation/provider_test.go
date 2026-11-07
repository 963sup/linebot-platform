package experiencepresentation

import (
	"context"
	"testing"
)

type fakePresentationReader struct{ recipient RecipientID }

func (*fakePresentationReader) ListSurfaces(context.Context) ([]ProviderSurface, error) {
	return nil, nil
}
func (*fakePresentationReader) GetSurface(_ context.Context, id ProviderSurfaceID) (ProviderSurface, error) {
	return ProviderSurface{ID: id}, nil
}
func (*fakePresentationReader) GetDefaultSurface(context.Context) (ProviderSurface, error) {
	return ProviderSurface{}, nil
}
func (*fakePresentationReader) ListAliases(context.Context) ([]ProviderAlias, error) { return nil, nil }
func (*fakePresentationReader) GetAlias(_ context.Context, id AliasID) (ProviderAlias, error) {
	return ProviderAlias{ID: id}, nil
}
func (f *fakePresentationReader) GetRecipientSurface(_ context.Context, id RecipientID) (ProviderSurface, error) {
	f.recipient = id
	return ProviderSurface{ID: "menu"}, nil
}
func (*fakePresentationReader) GetBatchProgress(_ context.Context, id string) (BatchProgress, error) {
	return BatchProgress{RequestID: id}, nil
}

func TestProviderQueriesUseTypedIdentifiers(t *testing.T) {
	reader := &fakePresentationReader{}
	queries, err := NewProviderQueries(reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queries.ForRecipient(context.Background(), "user-1"); err != nil || reader.recipient != "user-1" {
		t.Fatalf("ForRecipient() recipient = %q, error = %v", reader.recipient, err)
	}
	if _, err := queries.GetAlias(context.Background(), " "); err == nil {
		t.Fatal("GetAlias accepted invalid ID")
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
