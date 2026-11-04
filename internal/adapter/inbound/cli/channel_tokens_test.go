package cli

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/application/usecases/channel"
)

type fakeTokenRuntime struct {
	command channel.TokenCommand
	result  outbound.LINEResult
}

func (f *fakeTokenRuntime) Execute(_ context.Context, command channel.TokenCommand) (outbound.LINEResult, error) {
	f.command = command
	return f.result, nil
}

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
	runtime := &fakeTokenRuntime{result: outbound.LINEResult{Value: outbound.LINEObject{"access_token": "issued-secret"}}}
	previous := tokenRuntimeFactory
	t.Cleanup(func() { tokenRuntimeFactory = previous })
	tokenRuntimeFactory = func() (TokenRuntime, error) { return runtime, nil }
	if err := runChannelTokens([]string{"--action", "issue-short", "--request", request, "--apply", "--confirm", "issue-short", "--out", output}); err != nil {
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
	if string(payload) != `{"access_token":"issued-secret"}` {
		t.Fatalf("unexpected token result: %s", payload)
	}
	if runtime.command.ClientSecret != "secret" || runtime.command.Action != "issue-short" {
		t.Fatal("token command was not routed intact")
	}
}
