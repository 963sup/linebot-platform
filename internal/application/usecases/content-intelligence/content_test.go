package contentintelligence

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type fakeGenerator struct {
	request GenerationRequest
	draft   ContentDraft
	err     error
}

func (f *fakeGenerator) GenerateDraft(_ context.Context, request GenerationRequest) (ContentDraft, error) {
	f.request = request
	return f.draft, f.err
}

type fakeAudit struct {
	entry AuditEntry
	err   error
}

func (f *fakeAudit) RecordGeneration(_ context.Context, entry AuditEntry) error {
	f.entry = entry
	return f.err
}

func validRequest() GenerationRequest {
	return GenerationRequest{ID: "request-1", Input: "write a greeting", Policy: GenerationPolicy{Purpose: "draft", MaxOutputBytes: 1024}}
}

func TestProcessGeneratesAndAuditsDraft(t *testing.T) {
	now := time.Now().UTC()
	draft := ContentDraft{RequestID: "request-1", Content: "Hello", Provider: "provider", Model: "model", CreatedAt: now}
	generator, audit := &fakeGenerator{draft: draft}, &fakeAudit{}
	process, err := NewProcess(generator, audit, fixedClock{now: now})
	if err != nil {
		t.Fatal(err)
	}
	request := validRequest()
	got, err := process.GenerateDraft(context.Background(), request)
	if err != nil || got != draft || generator.request != request {
		t.Fatalf("GenerateDraft() = %+v, %v", got, err)
	}
	wantAudit := AuditEntry{RequestID: request.ID, Outcome: AuditSucceeded, Provider: "provider", Model: "model", RecordedAt: now}
	if audit.entry != wantAudit {
		t.Fatalf("audit = %+v, want %+v", audit.entry, wantAudit)
	}
}

func TestProcessAuditsGenerationFailure(t *testing.T) {
	now := time.Now().UTC()
	generateErr := errors.New("generation failed")
	auditErr := errors.New("audit failed")
	audit := &fakeAudit{err: auditErr}
	process, _ := NewProcess(&fakeGenerator{err: generateErr}, audit, fixedClock{now: now})
	_, err := process.GenerateDraft(context.Background(), validRequest())
	if !errors.Is(err, generateErr) || !errors.Is(err, auditErr) {
		t.Fatalf("GenerateDraft error = %v", err)
	}
	if audit.entry.Outcome != AuditFailed || audit.entry.Failure != generateErr.Error() || audit.entry.RecordedAt != now {
		t.Fatalf("failure audit = %+v", audit.entry)
	}
}

func TestProcessRejectsInvalidDraftAndAuditsIt(t *testing.T) {
	now := time.Now().UTC()
	audit := &fakeAudit{}
	process, _ := NewProcess(&fakeGenerator{draft: ContentDraft{RequestID: "wrong"}}, audit, fixedClock{now: now})
	if _, err := process.GenerateDraft(context.Background(), validRequest()); err == nil {
		t.Fatal("invalid draft accepted")
	}
	if audit.entry.Outcome != AuditFailed || audit.entry.Failure == "" {
		t.Fatalf("invalid draft audit = %+v", audit.entry)
	}
}

func TestProcessRejectsInvalidDependenciesAndRequests(t *testing.T) {
	generator, audit, clock := &fakeGenerator{}, &fakeAudit{}, fixedClock{now: time.Now().UTC()}
	for name, create := range map[string]func() error{
		"generator": func() error { _, err := NewProcess(nil, audit, clock); return err },
		"audit":     func() error { _, err := NewProcess(generator, nil, clock); return err },
		"clock":     func() error { _, err := NewProcess(generator, audit, nil); return err },
	} {
		t.Run(name, func(t *testing.T) {
			if err := create(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
	process, _ := NewProcess(generator, audit, clock)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for name, mutate := range map[string]func(GenerationRequest) (context.Context, GenerationRequest){
		"nil context": func(r GenerationRequest) (context.Context, GenerationRequest) { return nil, r },
		"cancelled":   func(r GenerationRequest) (context.Context, GenerationRequest) { return ctx, r },
		"no ID": func(r GenerationRequest) (context.Context, GenerationRequest) {
			r.ID = ""
			return context.Background(), r
		},
		"no purpose": func(r GenerationRequest) (context.Context, GenerationRequest) {
			r.Policy.Purpose = ""
			return context.Background(), r
		},
		"no input": func(r GenerationRequest) (context.Context, GenerationRequest) {
			r.Input = ""
			return context.Background(), r
		},
		"large input": func(r GenerationRequest) (context.Context, GenerationRequest) {
			r.Input = strings.Repeat("x", MaxInputBytes+1)
			return context.Background(), r
		},
		"bad output limit": func(r GenerationRequest) (context.Context, GenerationRequest) {
			r.Policy.MaxOutputBytes = 0
			return context.Background(), r
		},
	} {
		t.Run(name, func(t *testing.T) {
			callCtx, request := mutate(validRequest())
			if _, err := process.GenerateDraft(callCtx, request); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
