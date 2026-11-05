package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOperatorMutationsDryRunWithoutOpeningRuntime(t *testing.T) {
	tests := []struct {
		name string
		run  func(context.Context, []string, func(string) string, io.Writer, Dependencies) error
		args []string
	}{
		{"audience", runAudienceDefinitions, []string{"--action", "activate", "--id", "aud-1"}},
		{"surface", runInteractionSurfaces, []string{"--action", "publish", "--id", "surface-1"}},
		{"promotion", runPromotions, []string{"--action", "close", "--id", "promo-1"}},
		{"automation", runAutomationRequests, []string{"--action", "execute", "--id", "auto-1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := tt.run(testContext(), tt.args, func(string) string { return "" }, &out, Dependencies{}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), `"dry_run":true`) {
				t.Fatalf("missing dry-run summary: %s", out.String())
			}
		})
	}
}

func TestOperatorMutationRejectsWrongConfirmationBeforeRuntime(t *testing.T) {
	calls := 0
	deps := Dependencies{AudienceDefinitions: func(context.Context, string) (AudienceDefinitionsRuntime, error) {
		calls++
		return nil, errors.New("must not open")
	}}
	var out bytes.Buffer
	err := runAudienceDefinitions(testContext(), []string{
		"--action", "activate", "--id", "aud-1", "--apply", "--confirm", "archive",
	}, func(string) string { return "stage" }, &out, deps)
	if err == nil || !strings.Contains(err.Error(), `exactly equal "activate"`) {
		t.Fatalf("expected exact confirmation error, got %v", err)
	}
	if calls != 0 {
		t.Fatalf("runtime opened %d times", calls)
	}
}

func TestAudienceDefinitionDryRunRedactsDefinition(t *testing.T) {
	path := filepath.Join(t.TempDir(), "criteria.json")
	const definition = `{"segment":"private-value"}`
	if err := os.WriteFile(path, []byte(definition), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := runAudienceDefinitions(testContext(), []string{
		"--action", "define", "--id", "aud-1", "--source", "crm", "--definition-file", path,
	}, func(string) string { return "" }, &out, Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "private-value") || !strings.Contains(out.String(), "definition_sha256") {
		t.Fatalf("unsafe dry-run summary: %s", out.String())
	}
}

func TestOperatorPrivateInputRejectsUnsafeFiles(t *testing.T) {
	dir := t.TempDir()
	private := filepath.Join(dir, "private.json")
	if err := os.WriteFile(private, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(dir, "link.json")
	if err := os.Symlink(private, symlink); err != nil {
		t.Fatal(err)
	}
	groupReadable := filepath.Join(dir, "group.json")
	if err := os.WriteFile(groupReadable, []byte(`{}`), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(groupReadable, 0o640); err != nil {
		t.Fatal(err)
	}
	worldReadable := filepath.Join(dir, "world.json")
	if err := os.WriteFile(worldReadable, []byte(`{}`), 0o604); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(worldReadable, 0o604); err != nil {
		t.Fatal(err)
	}
	readOnly := filepath.Join(dir, "readonly.json")
	if err := os.WriteFile(readOnly, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(readOnly, 0o400); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{dir, symlink, groupReadable, worldReadable, readOnly} {
		if _, err := readOperatorPrivateFile(path); err == nil {
			t.Errorf("unsafe path %q was accepted", path)
		}
	}
}

func TestDeliveryIntentDryRunRedactsRecipient(t *testing.T) {
	const recipient = "sensitive-recipient-id"
	var out bytes.Buffer
	err := runPromotions(testContext(), []string{
		"--action", "delivery-intent", "--id", "promo-1",
		"--recipient", recipient, "--policy", "policy-1", "--eligible", "true",
		"--at", "2030-01-02T03:04:05Z",
	}, func(string) string { return "" }, &out, Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), recipient) || !strings.Contains(out.String(), `"recipient_sha256"`) {
		t.Fatalf("recipient was not safely summarized: %s", out.String())
	}
}

type audienceDefinitionsRuntimeStub struct {
	activated bool
	closed    bool
	closeErr  error
	entered   chan context.Context
}

func (r *audienceDefinitionsRuntimeStub) GetAudienceDefinition(ctx context.Context, _ string) (AudienceDefinitionSummary, error) {
	if r.entered != nil {
		r.entered <- ctx
		<-ctx.Done()
		return AudienceDefinitionSummary{}, ctx.Err()
	}
	return AudienceDefinitionSummary{}, nil
}
func (*audienceDefinitionsRuntimeStub) DefineAudience(context.Context, string, string, string) (AudienceDefinitionSummary, error) {
	return AudienceDefinitionSummary{}, nil
}
func (*audienceDefinitionsRuntimeStub) ReviseAudience(context.Context, string, string, string) (AudienceDefinitionSummary, error) {
	return AudienceDefinitionSummary{}, nil
}
func (r *audienceDefinitionsRuntimeStub) ActivateAudience(_ context.Context, id string) (AudienceDefinitionSummary, error) {
	r.activated = true
	return AudienceDefinitionSummary{ID: id, Status: "active", Version: 2}, nil
}
func (*audienceDefinitionsRuntimeStub) ArchiveAudience(context.Context, string) (AudienceDefinitionSummary, error) {
	return AudienceDefinitionSummary{}, nil
}
func (*audienceDefinitionsRuntimeStub) SyncAudience(context.Context, string) (AudienceSyncSummary, error) {
	return AudienceSyncSummary{}, nil
}
func (r *audienceDefinitionsRuntimeStub) Close() error { r.closed = true; return r.closeErr }

func TestOperatorRuntimePropagatesParentContextDuringExecution(t *testing.T) {
	type contextKey struct{}
	parent, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey{}, "operator"))
	runtime := &audienceDefinitionsRuntimeStub{entered: make(chan context.Context, 1)}
	deps := Dependencies{AudienceDefinitions: func(ctx context.Context, _ string) (AudienceDefinitionsRuntime, error) {
		if ctx.Value(contextKey{}) != "operator" {
			t.Fatal("operator factory context value was not propagated")
		}
		return runtime, nil
	}}
	done := make(chan error, 1)
	go func() {
		done <- runAudienceDefinitions(parent, []string{"--action", "get", "--id", "aud-1", "--project", "stage", "--read"}, func(string) string { return "stage" }, &bytes.Buffer{}, deps)
	}()
	ctx := <-runtime.entered
	deadline, ok := ctx.Deadline()
	if ctx.Value(contextKey{}) != "operator" || !ok || time.Until(deadline) > operatorCommandTimeout {
		t.Fatalf("operator context value/deadline not propagated: %v %v", ctx.Value(contextKey{}), deadline)
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("operator cancellation error = %v", err)
	}
	if !runtime.closed {
		t.Fatal("operator runtime was not closed after cancellation")
	}
}

func TestAudienceMutationAppliesAndClosesRuntime(t *testing.T) {
	runtime := &audienceDefinitionsRuntimeStub{}
	deps := Dependencies{AudienceDefinitions: func(_ context.Context, project string) (AudienceDefinitionsRuntime, error) {
		if project != "stage" {
			t.Fatalf("unexpected project %q", project)
		}
		return runtime, nil
	}}
	var out bytes.Buffer
	err := runAudienceDefinitions(testContext(), []string{
		"--action", "activate", "--id", "aud-1", "--project", "stage", "--apply", "--confirm", "activate",
	}, func(key string) string {
		if key == stagingProjectEnv {
			return "stage"
		}
		return ""
	}, &out, deps)
	if err != nil {
		t.Fatal(err)
	}
	if !runtime.activated || !runtime.closed {
		t.Fatalf("runtime lifecycle mismatch: activated=%v closed=%v", runtime.activated, runtime.closed)
	}
	if !strings.Contains(out.String(), `"status":"active"`) {
		t.Fatalf("unexpected summary: %s", out.String())
	}
}

func TestAudienceRuntimeCloseErrorIsReturned(t *testing.T) {
	sentinel := errors.New("close failed")
	runtime := &audienceDefinitionsRuntimeStub{closeErr: sentinel}
	deps := Dependencies{AudienceDefinitions: func(context.Context, string) (AudienceDefinitionsRuntime, error) {
		return runtime, nil
	}}
	var out bytes.Buffer
	err := runAudienceDefinitions(testContext(), []string{
		"--action", "get", "--id", "aud-1", "--project", "stage", "--read",
	}, func(string) string { return "stage" }, &out, deps)
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want close error", err)
	}
}
