package workspace

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type fakeFormsOperator struct{ count int }
func (f *fakeFormsOperator) ExecuteForms(context.Context, outbound.WorkspaceRequest) (json.RawMessage, error) {
	f.count++
	return json.RawMessage(`{"formId":"ok"}`), nil
}

func TestFormsAuthorizationPolicy(t *testing.T) {
	cal, tasks, forms := &fakeCalendar{}, &fakeTasks{}, &fakeFormsOperator{}
	if _, err := NewOperationsWithForms(cal, tasks, nil); err == nil { t.Fatal("nil forms operator accepted") }
	s, err := NewOperationsWithForms(cal, tasks, forms)
	if err != nil { t.Fatal(err) }
	for _, tc := range []struct{ op string; approval Approval; allowed bool }{
		{"forms.get", Approval{Read:true}, true},
		{"forms.responses.list", Approval{Read:true}, true},
		{"forms.watches.list", Approval{Read:true}, true},
		{"forms.create", Approval{Apply:true,Confirm:"forms.create"}, true},
		{"forms.watches.create", Approval{Read:true}, false},
		{"forms.batchUpdate", Approval{Apply:true,Confirm:"forms.batchUpdate"}, false},
		{"forms.batchUpdate", Approval{Apply:true,Confirm:"forms.batchUpdate",ConfirmDangerous:"forms.batchUpdate"}, true},
		{"forms.setPublishSettings", Approval{Apply:true,Confirm:"forms.setPublishSettings"}, false},
		{"forms.setPublishSettings", Approval{Apply:true,Confirm:"forms.setPublishSettings",ConfirmDangerous:"forms.setPublishSettings"}, true},
		{"forms.watches.delete", Approval{Apply:true,Confirm:"forms.watches.delete"}, false},
		{"forms.watches.delete", Approval{Apply:true,Confirm:"forms.watches.delete",ConfirmDangerous:"forms.watches.delete"}, true},
		{"forms.watches.renew", Approval{Apply:true,Confirm:"forms.watches.renew"}, true},
	} {
		before := forms.count
		_, err := s.Execute(context.Background(), "forms", outbound.WorkspaceRequest{Operation:tc.op}, tc.approval)
		if (err==nil)!=tc.allowed { t.Fatalf("%s approved=%t err=%v",tc.op,tc.allowed,err) }
		if !tc.allowed && forms.count!=before { t.Fatal("rejected command caused side effect") }
	}
	legacy, err := NewOperations(cal, tasks)
	if err != nil { t.Fatal(err) }
	if _, err = legacy.Execute(context.Background(),"forms",outbound.WorkspaceRequest{Operation:"forms.get"},Approval{Read:true});err==nil {t.Fatal("Forms not configured in legacy constructor")}
}
