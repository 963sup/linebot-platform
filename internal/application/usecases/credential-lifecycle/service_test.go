package credentiallifecycle

import (
	"context"
	"testing"
)

type credentialStub struct{ called string }

func (s *credentialStub) IssueStatelessWithClient(context.Context, string, ClientCredential) (IssuedCredential, error) {
	s.called = "stateless-client"
	return IssuedCredential{AccessToken: "secret"}, nil
}
func (s *credentialStub) IssueStatelessWithAssertion(context.Context, string, AssertionCredential) (IssuedCredential, error) {
	s.called = "stateless-assertion"
	return IssuedCredential{AccessToken: "secret"}, nil
}
func (s *credentialStub) ValidKeyIDs(context.Context, AssertionCredential) ([]string, error) {
	s.called = "keys"
	return []string{"key-1"}, nil
}
func (s *credentialStub) IssueJWT(context.Context, string, AssertionCredential) (IssuedCredential, error) {
	s.called = "issue-jwt"
	return IssuedCredential{}, nil
}
func (s *credentialStub) VerifyJWT(context.Context, string) (Verification, error) {
	s.called = "verify-jwt"
	return Verification{Active: true}, nil
}
func (s *credentialStub) RevokeJWT(context.Context, ClientCredential, string) error {
	s.called = "revoke-jwt"
	return nil
}
func (s *credentialStub) IssueShortLived(context.Context, string, ClientCredential) (IssuedCredential, error) {
	s.called = "issue-short"
	return IssuedCredential{}, nil
}
func (s *credentialStub) VerifyShortLived(context.Context, string) (Verification, error) {
	s.called = "verify-short"
	return Verification{Active: true}, nil
}
func (s *credentialStub) RevokeShortLived(context.Context, string) error {
	s.called = "revoke-short"
	return nil
}

func TestServiceUsesTypedCredentialOperations(t *testing.T) {
	stub := &credentialStub{}
	service, err := New(stub)
	if err != nil {
		t.Fatal(err)
	}
	client := ClientCredential{ClientID: "client", ClientSecret: "secret"}
	assertion := AssertionCredential{Type: "jwt", Assertion: "assertion"}
	cases := []struct {
		want string
		run  func() error
	}{
		{"stateless-client", func() error {
			_, err := service.IssueStatelessWithClient(context.Background(), "client_credentials", client)
			return err
		}},
		{"stateless-assertion", func() error {
			_, err := service.IssueStatelessWithAssertion(context.Background(), "client_credentials", assertion)
			return err
		}},
		{"keys", func() error { _, err := service.ValidKeyIDs(context.Background(), assertion); return err }},
		{"issue-jwt", func() error {
			_, err := service.IssueJWT(context.Background(), "client_credentials", assertion)
			return err
		}},
		{"verify-jwt", func() error { _, err := service.VerifyJWT(context.Background(), "token"); return err }},
		{"revoke-jwt", func() error { return service.RevokeJWT(context.Background(), client, "token") }},
		{"issue-short", func() error {
			_, err := service.IssueShortLived(context.Background(), "client_credentials", client)
			return err
		}},
		{"verify-short", func() error { _, err := service.VerifyShortLived(context.Background(), "token"); return err }},
		{"revoke-short", func() error { return service.RevokeShortLived(context.Background(), "token") }},
	}
	for _, tc := range cases {
		if err := tc.run(); err != nil {
			t.Fatalf("%s: %v", tc.want, err)
		}
		if stub.called != tc.want {
			t.Fatalf("called %q, want %q", stub.called, tc.want)
		}
	}
}

func TestInvalidCredentialNeverReachesGateway(t *testing.T) {
	stub := &credentialStub{}
	service, _ := New(stub)
	if _, err := service.IssueStatelessWithClient(context.Background(), "grant", ClientCredential{ClientID: "client", ClientSecret: " "}); err == nil {
		t.Fatal("invalid credential accepted")
	}
	if stub.called != "" {
		t.Fatal("invalid credential reached gateway")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.VerifyJWT(ctx, "token"); err == nil {
		t.Fatal("canceled context accepted")
	}
}
