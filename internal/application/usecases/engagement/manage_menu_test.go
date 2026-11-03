package engagement

import (
	"context"
	"errors"
	"testing"
)

type fakeMenuManager struct{ calls int }
func (f *fakeMenuManager) SetDefaultRichMenu(context.Context,string) error { f.calls++; return nil }
func (f *fakeMenuManager) CancelDefaultRichMenu(context.Context) error { f.calls++; return nil }
func (f *fakeMenuManager) LinkRichMenuToUser(context.Context,string,string) error { f.calls++; return nil }
func (f *fakeMenuManager) UnlinkRichMenuFromUser(context.Context,string) error { f.calls++; return nil }
func (f *fakeMenuManager) DeleteRichMenu(context.Context,string) error { f.calls++; return nil }
func (f *fakeMenuManager) DeleteRichMenuAlias(context.Context,string) error { f.calls++; return nil }

func TestMenuManagementValidatesAndRoutes(t *testing.T) {
	f := &fakeMenuManager{}
	s, err := NewMenuManager(f)
	if err != nil { t.Fatal(err) }
	for _, cmd := range []MenuCommand{
		{Action:"set-default",RichMenuID:""},
		{Action:"link-user",UserID:"U123"},
		{Action:"delete-alias"},
		{Action:"invalid"},
	} {
		if err := s.Execute(context.Background(),cmd); err == nil {
			t.Fatalf("invalid richmenu command passed: %+v",cmd)
		}
	}
	if f.calls != 0 { t.Fatal("invalid inputs reached adapter") }
	if err := s.Execute(context.Background(),MenuCommand{Action:"cancel-default"}); err != nil {t.Fatal(err)}
	if err := s.Execute(context.Background(),MenuCommand{Action:"link-user",UserID:"U123",RichMenuID:"richmenu-1"});err != nil{t.Fatal(err)}
	if f.calls != 2 {t.Fatalf("calls = %d",f.calls)}
	ctx,cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Execute(ctx,MenuCommand{Action:"cancel-default"});!errors.Is(err,context.Canceled){t.Fatal(err)}
}
