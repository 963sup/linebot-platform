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

type fakePresentationManager struct{ image SurfaceImage }

func (*fakePresentationManager) CreateSurface(context.Context, SurfaceDefinition) (ProviderSurface, error) {
	return ProviderSurface{}, nil
}
func (*fakePresentationManager) ValidateSurface(context.Context, SurfaceDefinition) error { return nil }
func (*fakePresentationManager) CreateAlias(context.Context, ProviderAlias) error         { return nil }
func (*fakePresentationManager) UpdateAlias(context.Context, ProviderAlias) error         { return nil }
func (*fakePresentationManager) DeleteAlias(context.Context, AliasID) error               { return nil }
func (*fakePresentationManager) SetDefault(context.Context, ProviderSurfaceID) error      { return nil }
func (*fakePresentationManager) ClearDefault(context.Context) error                       { return nil }
func (*fakePresentationManager) AssignRecipient(context.Context, RecipientAssignment) error {
	return nil
}
func (*fakePresentationManager) UnassignRecipient(context.Context, RecipientID) error   { return nil }
func (*fakePresentationManager) DeleteSurface(context.Context, ProviderSurfaceID) error { return nil }
func (*fakePresentationManager) ExecuteBatch(context.Context, BatchCommand) (BatchProgress, error) {
	return BatchProgress{}, nil
}
func (*fakePresentationManager) ValidateBatch(context.Context, BatchCommand) error { return nil }
func (f *fakePresentationManager) UploadImage(_ context.Context, _ ProviderSurfaceID, image SurfaceImage) error {
	f.image = image
	return nil
}

func TestProviderOperationsValidateImage(t *testing.T) {
	manager := &fakePresentationManager{}
	operations, err := NewProviderOperations(manager)
	if err != nil {
		t.Fatal(err)
	}
	image := SurfaceImage{ContentType: "image/png", Data: []byte("png")}
	if err := operations.UploadImage(context.Background(), "menu-1", image); err != nil || manager.image.ContentType != image.ContentType {
		t.Fatalf("UploadImage() image = %+v, error = %v", manager.image, err)
	}
	if err := operations.UploadImage(context.Background(), "menu-1", SurfaceImage{ContentType: "text/plain", Data: []byte("x")}); err == nil {
		t.Fatal("UploadImage accepted non-image")
	}
}
