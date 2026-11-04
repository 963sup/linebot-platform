package bootstrap

import (
	"context"
	"testing"

	"golang.org/x/oauth2"
)

func TestWorkspaceRuntimeExplicitUserToken(t *testing.T) {
	var nilContext context.Context
	if _, err := NewWorkspaceOperations(context.Background(), nil); err == nil {
		t.Fatal("nil user OAuth TokenSource accepted")
	}
	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "fake", TokenType: "Bearer"})
	if _, err := NewWorkspaceOperations(nilContext, source); err == nil {
		t.Fatal("nil context accepted")
	}
	ops, err := NewWorkspaceOperations(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	if ops == nil {
		t.Fatal("workspace operations not created")
	}
}
