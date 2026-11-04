package channel

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeTokenManager struct{ called string }
func(f *fakeTokenManager) IssueStateless(context.Context,string,string,string,string,string)(json.RawMessage,error){f.called="issue-stateless";return json.RawMessage("{}"),nil}
func(f *fakeTokenManager) ValidKeyIDs(context.Context,string,string)(json.RawMessage,error){f.called="key-ids";return json.RawMessage("{}"),nil}
func(f *fakeTokenManager) IssueJWT(context.Context,string,string,string)(json.RawMessage,error){f.called="issue-jwt";return json.RawMessage("{}"),nil}
func(f *fakeTokenManager) VerifyJWT(context.Context,string)(json.RawMessage,error){f.called="verify-jwt";return json.RawMessage("{}"),nil}
func(f *fakeTokenManager) RevokeJWT(context.Context,string,string,string)error{f.called="revoke-jwt";return nil}
func(f *fakeTokenManager) IssueShortLived(context.Context,string,string,string)(json.RawMessage,error){f.called="issue-short";return json.RawMessage("{}"),nil}
func(f *fakeTokenManager) VerifyShortLived(context.Context,string)(json.RawMessage,error){f.called="verify-short";return json.RawMessage("{}"),nil}
func(f *fakeTokenManager) RevokeShortLived(context.Context,string)error{f.called="revoke-short";return nil}

func TestTokenCommandsValidateAndRouteEightAPIs(t *testing.T) {
	if _,err:=NewTokenOperations(nil);err==nil{t.Fatal("nil manager allowed")}
	f:=&fakeTokenManager{};s,err:=NewTokenOperations(f);if err!=nil{t.Fatal(err)}
	for _,c:=range []TokenCommand{
		{Action:"issue-stateless"},
		{Action:"issue-short",GrantType:"client_credentials"},
		{Action:"revoke-short"},
		{Action:"verify-jwt",AccessToken:" token "},
		{Action:"issue-jwt",GrantType:"client_credentials",ClientAssertion:"only-one"},
		{Action:"bad"},
	} {
		if _,err:=s.Execute(context.Background(),c);err==nil{t.Fatalf("invalid token command accepted %s",c.Action)}
		if f.called!=""{t.Fatal("invalid command caused side effect")}
	}
	for _,c:=range []TokenCommand{
		{Action:"issue-stateless",GrantType:"client_credentials",ClientID:"client-id",ClientSecret:"secret"},
		{Action:"key-ids",ClientAssertionType:"jwt-bearer",ClientAssertion:"assertion"},
		{Action:"issue-jwt",GrantType:"client_credentials",ClientAssertionType:"jwt-bearer",ClientAssertion:"assertion"},
		{Action:"verify-jwt",AccessToken:"token"},
		{Action:"revoke-jwt",ClientID:"client-id",ClientSecret:"secret",AccessToken:"token"},
		{Action:"issue-short",GrantType:"client_credentials",ClientID:"client-id",ClientSecret:"secret"},
		{Action:"verify-short",AccessToken:"token"},
		{Action:"revoke-short",AccessToken:"token"},
	} {
		if _,err:=s.Execute(context.Background(),c);err!=nil{t.Fatalf("%s: %v",c.Action,err)}
		if f.called!=c.Action{t.Fatalf("called %s instead of %s",f.called,c.Action)}
	}
	ctx,cancel:=context.WithCancel(context.Background());cancel()
	if _,err:=s.Execute(ctx,TokenCommand{Action:"verify-short",AccessToken:"token"});err==nil{t.Fatal("cancelled context accepted")}
}
