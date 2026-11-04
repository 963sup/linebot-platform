package oauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
)

type fakeLinks struct {
	state string
	calls int
	owner string
}
func (f *fakeLinks) Begin(_ context.Context,owner string, _ []string)(string,error){
	f.owner=owner
	f.state=strings.Repeat("A",43)
	return "https://accounts.google.com/o/oauth2/auth?state="+f.state,nil
}
func (f *fakeLinks) Complete(_ context.Context,state,code string)(workspace.LinkedAccount,error){
	f.calls++
	if state!=f.state||code!="code" {return workspace.LinkedAccount{},errors.New("bad state")}
	return workspace.LinkedAccount{OwnerID:f.owner,GoogleSubject:"google-sub"},nil
}
func TestHTTPStartRequiresVerifiedLINEIdentity(t *testing.T){
	links:=&fakeLinks{}
	_,err:=NewHandler(links,nil,[]string{"openid"})
	if err==nil{t.Fatal("missing identity resolver allowed")}
	h,err:=NewHandler(links,func(*http.Request)(string,error){return "",errors.New("unauthorized")},[]string{"openid"})
	if err!=nil{t.Fatal(err)}
	w:=httptest.NewRecorder()
	h.Start(w,httptest.NewRequest(http.MethodGet,"https://app.example.test/google/oauth/start?userId=forged",nil))
	if w.Code!=http.StatusUnauthorized||links.owner!=""{t.Fatal("untrusted LINE identity accepted")}
}

func TestHTTPCallbackRequiresMatchingSecureStateCookie(t *testing.T){
	links:=&fakeLinks{}
	h,err:=NewHandler(links,func(*http.Request)(string,error){return "verified-line",nil},[]string{"openid"})
	if err!=nil{t.Fatal(err)}
	start:=httptest.NewRecorder()
	h.Start(start,httptest.NewRequest(http.MethodGet,"https://app.example.test/google/oauth/start",nil))
	if start.Code!=http.StatusFound{t.Fatalf("start status %d",start.Code)}
	if links.owner!="verified-line"{t.Fatal("principal not attached")}
	cookies:=start.Result().Cookies()
	if len(cookies)!=1||!cookies[0].Secure||!cookies[0].HttpOnly||cookies[0].SameSite!=http.SameSiteLaxMode {
		t.Fatal("OAuth cookie not secure")
	}
	r:=httptest.NewRequest(http.MethodGet,"https://app.example.test/google/oauth/callback?state="+links.state+"&code=code",nil)
	bad:=httptest.NewRecorder()
	h.Callback(bad,r)
	if bad.Code!=http.StatusBadRequest||links.calls!=0{t.Fatal("missing cookie accepted")}
	r2:=httptest.NewRequest(http.MethodGet,"https://app.example.test/google/oauth/callback?state="+url.QueryEscape(links.state)+"&code=code",nil)
	r2.AddCookie(cookies[0])
	good:=httptest.NewRecorder()
	h.Callback(good,r2)
	if good.Code!=http.StatusOK||links.calls!=1{t.Fatalf("valid callback rejected %d",good.Code)}
	if strings.Contains(good.Body.String(),"google-sub"){t.Fatal("Google subject disclosed")}
}
