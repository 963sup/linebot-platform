package engagement

import (
	"context"
	"errors"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type fakeMenuManager struct{ calls int }

func (f *fakeMenuManager) SetDefaultRichMenu(context.Context, string) error { f.calls++; return nil }
func (f *fakeMenuManager) CancelDefaultRichMenu(context.Context) error      { f.calls++; return nil }
func (f *fakeMenuManager) LinkRichMenuToUser(context.Context, string, string) error {
	f.calls++
	return nil
}
func (f *fakeMenuManager) UnlinkRichMenuFromUser(context.Context, string) error {
	f.calls++
	return nil
}
func (f *fakeMenuManager) DeleteRichMenu(context.Context, string) error      { f.calls++; return nil }
func (f *fakeMenuManager) DeleteRichMenuAlias(context.Context, string) error { f.calls++; return nil }
func (f *fakeMenuManager) CreateRichMenu(context.Context, outbound.LINERequest) error {
	f.calls++
	return nil
}
func (f *fakeMenuManager) ValidateRichMenu(context.Context, outbound.LINERequest) error {
	f.calls++
	return nil
}
func (f *fakeMenuManager) CreateRichMenuAlias(context.Context, outbound.LINERequest) error {
	f.calls++
	return nil
}
func (f *fakeMenuManager) UpdateRichMenuAlias(context.Context, string, outbound.LINERequest) error {
	f.calls++
	return nil
}
func (f *fakeMenuManager) BulkLinkRichMenu(context.Context, outbound.LINERequest) error {
	f.calls++
	return nil
}
func (f *fakeMenuManager) BulkUnlinkRichMenu(context.Context, outbound.LINERequest) error {
	f.calls++
	return nil
}
func (f *fakeMenuManager) RichMenuBatch(context.Context, outbound.LINERequest) error {
	f.calls++
	return nil
}
func (f *fakeMenuManager) ValidateRichMenuBatch(context.Context, outbound.LINERequest) error {
	f.calls++
	return nil
}

func TestMenuManagementValidatesAndRoutes(t *testing.T) {
	f := &fakeMenuManager{}
	s, err := NewMenuManager(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []MenuCommand{
		{Action: "set-default", RichMenuID: ""},
		{Action: "link-user", UserID: "U123"},
		{Action: "delete-alias"},
		{Action: "invalid"},
	} {
		if err := s.Execute(context.Background(), cmd); err == nil {
			t.Fatalf("invalid richmenu command passed: %+v", cmd)
		}
	}
	if f.calls != 0 {
		t.Fatal("invalid inputs reached adapter")
	}
	if err := s.Execute(context.Background(), MenuCommand{Action: "cancel-default"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Execute(context.Background(), MenuCommand{Action: "link-user", UserID: "U123", RichMenuID: "richmenu-1"}); err != nil {
		t.Fatal(err)
	}
	if f.calls != 2 {
		t.Fatalf("calls = %d", f.calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Execute(ctx, MenuCommand{Action: "cancel-default"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestMenuCommandAcceptsRichMenuInfrastructureActions(t *testing.T) {
	for _, cmd := range []MenuCommand{
		{Action: "create", Payload: outbound.LINERequest{Value: map[string]any{"areas": []any{}}}},
		{Action: "validate", Payload: outbound.LINERequest{Value: map[string]any{"areas": []any{}}}},
		{Action: "upload-image", RichMenuID: "richmenu-1", Image: []byte("png"), ContentType: "image/png"},
		{Action: "create-alias", Payload: outbound.LINERequest{Value: map[string]any{"richMenuAliasId": "main", "richMenuId": "richmenu-1"}}},
		{Action: "update-alias", AliasID: "main", Payload: outbound.LINERequest{Value: map[string]any{"richMenuId": "richmenu-2"}}},
		{Action: "bulk-link", Payload: outbound.LINERequest{Value: map[string]any{"richMenuId": "richmenu-1", "userIds": []any{"U1"}}}},
		{Action: "bulk-unlink", Payload: outbound.LINERequest{Value: map[string]any{"userIds": []any{"U1"}}}},
		{Action: "batch", Payload: outbound.LINERequest{Value: map[string]any{"operations": []any{}}}},
		{Action: "validate-batch", Payload: outbound.LINERequest{Value: map[string]any{"operations": []any{}}}},
	} {
		if err := ValidateMenuCommand(cmd); err != nil {
			t.Fatalf("valid command rejected: %+v: %v", cmd, err)
		}
	}
}

func TestMenuManagerRoutesRichMenuPayloadActions(t *testing.T) {
	f := &fakeMenuManager{}
	s, err := NewMenuManager(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []MenuCommand{
		{Action: "create", Payload: outbound.LINERequest{Value: map[string]any{"areas": []any{}}}},
		{Action: "validate", Payload: outbound.LINERequest{Value: map[string]any{"areas": []any{}}}},
		{Action: "create-alias", Payload: outbound.LINERequest{Value: map[string]any{"richMenuAliasId": "main", "richMenuId": "richmenu-1"}}},
		{Action: "update-alias", AliasID: "main", Payload: outbound.LINERequest{Value: map[string]any{"richMenuId": "richmenu-2"}}},
		{Action: "bulk-link", Payload: outbound.LINERequest{Value: map[string]any{"richMenuId": "richmenu-1", "userIds": []any{"U1"}}}},
		{Action: "bulk-unlink", Payload: outbound.LINERequest{Value: map[string]any{"userIds": []any{"U1"}}}},
		{Action: "batch", Payload: outbound.LINERequest{Value: map[string]any{"operations": []any{}}}},
		{Action: "validate-batch", Payload: outbound.LINERequest{Value: map[string]any{"operations": []any{}}}},
	} {
		if err := s.Execute(context.Background(), cmd); err != nil {
			t.Fatal(err)
		}
	}
	if f.calls != 8 {
		t.Fatalf("calls = %d", f.calls)
	}
}
