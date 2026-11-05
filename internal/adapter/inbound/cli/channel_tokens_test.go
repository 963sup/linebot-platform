package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	credentiallifecycle "github.com/963sup/linebot-platform/internal/application/usecases/credential-lifecycle"
)

type fakeTokenRuntime struct {
	client credentiallifecycle.ClientCredential
	grant  string
	result credentiallifecycle.IssuedCredential
}

func (f *fakeTokenRuntime) IssueShortLived(_ context.Context, grant string, client credentiallifecycle.ClientCredential) (credentiallifecycle.IssuedCredential, error) {
	f.grant, f.client = grant, client
	return f.result, nil
}
func (*fakeTokenRuntime) IssueStatelessWithClient(context.Context, string, credentiallifecycle.ClientCredential) (credentiallifecycle.IssuedCredential, error) {
	return credentiallifecycle.IssuedCredential{}, nil
}
func (*fakeTokenRuntime) IssueStatelessWithAssertion(context.Context, string, credentiallifecycle.AssertionCredential) (credentiallifecycle.IssuedCredential, error) {
	return credentiallifecycle.IssuedCredential{}, nil
}
func (*fakeTokenRuntime) ValidKeyIDs(context.Context, credentiallifecycle.AssertionCredential) ([]string, error) {
	return nil, nil
}
func (*fakeTokenRuntime) IssueJWT(context.Context, string, credentiallifecycle.AssertionCredential) (credentiallifecycle.IssuedCredential, error) {
	return credentiallifecycle.IssuedCredential{}, nil
}
func (*fakeTokenRuntime) VerifyJWT(context.Context, string) (credentiallifecycle.Verification, error) {
	return credentiallifecycle.Verification{}, nil
}
func (*fakeTokenRuntime) RevokeJWT(context.Context, credentiallifecycle.ClientCredential, string) error {
	return nil
}
func (*fakeTokenRuntime) VerifyShortLived(context.Context, string) (credentiallifecycle.Verification, error) {
	return credentiallifecycle.Verification{}, nil
}
func (*fakeTokenRuntime) RevokeShortLived(context.Context, string) error { return nil }

func (*fakeTokenRuntime) Close() {}

func TestTokenOperationsFailClosed(t *testing.T) {
	for _, args := range [][]string{
		{"--action", "issue-short"},
		{"--action", "issue-short", "--request", "unknown", "--apply", "--confirm", "issue-short", "--out", "token.json"},
		{"--action", "revoke-short", "--request", "unknown", "--read"},
		{"--action", "verify-jwt", "--request", "unknown", "--read"},
	} {
		if err := runChannelTokens(args); err == nil {
			t.Fatalf("invalid token operation accepted: %v", args)
		}
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(p, []byte(`{"accessToken":"secret"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runChannelTokens([]string{"--action", "verify-jwt", "--request", p, "--read", "--out", filepath.Join(dir, "result.json")}); err == nil {
		t.Fatal("world-readable secrets allowed")
	}
	if err := os.Chmod(p, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := tokenRequestFromPrivateFile(p); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(other, []byte(`{"accessToken":"secret","unknown":"secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := tokenRequestFromPrivateFile(other); err == nil {
		t.Fatal("unknown token request field accepted")
	}
}

func TestPrivateOperationRunsOnlyWithReservedOutput(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "private.json")
	calls := 0
	fn := func() ([]byte, error) { calls++; return []byte("secret"), nil }
	if err := writePrivateOperation(dest, fn); err != nil {
		t.Fatal(err)
	}
	if err := writePrivateOperation(dest, fn); err == nil {
		t.Fatal("existing secret output overwritten")
	}
	if calls != 1 {
		t.Fatalf("external operation was called %d times", calls)
	}
}

func TestTokenRequestMapsSecretsOnlyAtCLIBoundary(t *testing.T) {
	dir := t.TempDir()
	request := filepath.Join(dir, "request.json")
	if err := os.WriteFile(request, []byte(`{"grantType":"client_credentials","clientId":"id","clientSecret":"secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	command, err := tokenRequestFromPrivateFile(request)
	if err != nil {
		t.Fatal(err)
	}
	if command.GrantType != "client_credentials" || command.ClientID != "id" || command.ClientSecret != "secret" {
		t.Fatal("private request was not mapped to the application command")
	}
	commandType := reflect.TypeOf(command)
	for i := 0; i < commandType.NumField(); i++ {
		if commandType.Field(i).Tag.Get("json") != "" {
			t.Fatalf("application field %s owns JSON metadata", commandType.Field(i).Name)
		}
	}
}

func TestTokenResultEncodingStaysPrivateAndAtCLIBoundary(t *testing.T) {
	dir := t.TempDir()
	request := filepath.Join(dir, "request.json")
	output := filepath.Join(dir, "result.json")
	if err := os.WriteFile(request, []byte(`{"grantType":"client_credentials","clientId":"id","clientSecret":"secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime := &fakeTokenRuntime{result: credentiallifecycle.IssuedCredential{AccessToken: "issued-secret"}}
	deps := Dependencies{TokenRuntime: func() (TokenRuntime, error) { return runtime, nil }}
	if err := runChannelTokens([]string{"--action", "issue-short", "--request", request, "--apply", "--confirm", "issue-short", "--out", output}, deps); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("private result mode = %o", info.Mode().Perm())
	}
	payload, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var encoded map[string]any
	if err := json.Unmarshal(payload, &encoded); err != nil || encoded["access_token"] != "issued-secret" {
		t.Fatalf("unexpected token result: %s", payload)
	}
	if runtime.client.ClientSecret != "secret" || runtime.grant != "client_credentials" {
		t.Fatal("token command was not routed intact")
	}
}
