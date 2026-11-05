package accountlinktoken

import (
	"context"
	"errors"
	"testing"
)

type tokenStub struct {
	calls int
	token Token
	err   error
}

func (s *tokenStub) Issue(context.Context, Subject) (Token, error) { s.calls++; return s.token, s.err }

func TestIssueReturnsTypedToken(t *testing.T) {
	stub := &tokenStub{token: Token{Value: "opaque-token"}}
	service, err := New(stub)
	if err != nil {
		t.Fatal(err)
	}
	token, err := service.Issue(context.Background(), Subject{ID: "user-1"})
	if err != nil || token.Value != "opaque-token" || stub.calls != 1 {
		t.Fatalf("token=%+v err=%v calls=%d", token, err, stub.calls)
	}
}

func TestIssueRejectsInvalidInputAndProviderResult(t *testing.T) {
	stub := &tokenStub{token: Token{Value: "opaque-token"}}
	service, _ := New(stub)
	for _, subject := range []Subject{{}, {ID: " user-1"}} {
		if _, err := service.Issue(context.Background(), subject); err == nil {
			t.Fatalf("accepted %+v", subject)
		}
	}
	if stub.calls != 0 {
		t.Fatal("invalid subject reached gateway")
	}
	stub.token = Token{}
	if _, err := service.Issue(context.Background(), Subject{ID: "user-1"}); err == nil {
		t.Fatal("empty provider token accepted")
	}
	stub.err = errors.New("provider unavailable")
	if _, err := service.Issue(context.Background(), Subject{ID: "user-1"}); err == nil {
		t.Fatal("provider error hidden")
	}
}
