package engagement

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeAudienceManager struct { called string }
func (f *fakeAudienceManager) CreateAudienceGroup(context.Context,json.RawMessage)(json.RawMessage,error){f.called="create";return json.RawMessage("{}"),nil}
func (f *fakeAudienceManager) AddAudienceToGroup(context.Context,json.RawMessage)error{f.called="append";return nil}
func (f *fakeAudienceManager) CreateClickAudience(context.Context,json.RawMessage)(json.RawMessage,error){f.called="create-click";return json.RawMessage("{}"),nil}
func (f *fakeAudienceManager) CreateImpressionAudience(context.Context,json.RawMessage)(json.RawMessage,error){f.called="create-impression";return json.RawMessage("{}"),nil}
func (f *fakeAudienceManager) UpdateAudienceDescription(context.Context,int64,json.RawMessage)error{f.called="update-description";return nil}
func (f *fakeAudienceManager) DeleteAudienceGroup(context.Context,int64)error{f.called="delete";return nil}
func (f *fakeAudienceManager) CreateAudienceFromFile(context.Context,[]byte,string,bool,string)(json.RawMessage,error){f.called="create-file";return json.RawMessage("{}"),nil}
func (f *fakeAudienceManager) AddAudienceFromFile(context.Context,int64,[]byte,string)error{f.called="append-file";return nil}
func TestAudienceManagementValidatesAndRoutesAllEight(t *testing.T) {
	if _,err:=NewAudienceOperations(nil);err==nil{t.Fatal("nil manager accepted")}
	f:=&fakeAudienceManager{}
	m,err:=NewAudienceOperations(f);if err!=nil{t.Fatal(err)}
	for _,cmd:=range []AudienceCommand{
		{Action:"create"},
		{Action:"append-file",ID:1},
		{Action:"delete",ID:0},
		{Action:"create-file",Description:"test",File:make([]byte,(20<<20)+1)},
		{Action:"update-description",ID:1,Request:json.RawMessage("[]")},
		{Action:"create",Request:json.RawMessage("{}"),IFAAudience:true},
	} {
		if _,err:=m.Execute(context.Background(),cmd);err==nil{t.Fatalf("invalid command accepted: %+v",cmd.Action)}
		if f.called!=""{t.Fatal("invalid command caused side effect")}
	}
	payload:=json.RawMessage("{}")
	for _,cmd:=range []AudienceCommand{
		{Action:"create",Request:payload},
		{Action:"append",Request:payload},
		{Action:"create-click",Request:payload},
		{Action:"create-impression",Request:payload},
		{Action:"update-description",ID:123,Request:payload},
		{Action:"delete",ID:123},
		{Action:"create-file",Description:"test",File:[]byte("U123")},
		{Action:"append-file",ID:123,File:[]byte("U123")},
	} {
		if _,err:=m.Execute(context.Background(),cmd);err!=nil{t.Fatalf("%s: %v",cmd.Action,err)}
		if f.called!=cmd.Action{t.Fatalf("wanted %s, got %s",cmd.Action,f.called)}
	}
}
