package firestore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/workspace-automation/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/workspace-automation/value-objects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewAutomationRequestRepositoryRequiresClient(t *testing.T) {
	if _, err := NewAutomationRequestRepository(nil); err == nil {
		t.Fatal("NewAutomationRequestRepository(nil) error = nil")
	}
}

func TestAutomationRequestRepositoryMappingRoundTripDoesNotRaiseEvents(t *testing.T) {
	value := newTestAutomationRequest(t)
	approval, _ := valueobjects.NewApproval("operator@example.test")
	if err := value.Approve(approval); err != nil {
		t.Fatal(err)
	}
	if err := value.StartExecution(); err != nil {
		t.Fatal(err)
	}
	output, _ := valueobjects.NewOutput("calendar event created")
	if err := value.Succeed(output); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1_700_000_000, 0).UTC()
	stored := storeAutomationRequest(value, now)
	stored.CreatedAt = now
	restored, err := restoreAutomationRequest(stored)
	if err != nil {
		t.Fatal(err)
	}
	if restored.ID() != value.ID() || restored.Action() != value.Action() || restored.Approval() != value.Approval() ||
		restored.Output() != value.Output() || restored.Status() != value.Status() || restored.Revision() != value.Revision() ||
		!slicesEqual(restored.RequiredScopes(), value.RequiredScopes()) {
		t.Fatalf("restored automation request = %#v, want %#v", restored, value)
	}
	if got := len(restored.PendingEvents()); got != 0 {
		t.Fatalf("restored pending events = %d, want 0", got)
	}
}

func TestAutomationRequestRepositoryRevisionGate(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	value := newTestAutomationRequest(t)
	current := storeAutomationRequest(value, now)
	current.CreatedAt = now

	approval, _ := valueobjects.NewApproval("operator@example.test")
	if err := value.Approve(approval); err != nil {
		t.Fatal(err)
	}
	next := storeAutomationRequest(value, now.Add(time.Minute))
	next.CreatedAt = current.CreatedAt
	if err := validateAutomationRevision(current, next); err != nil {
		t.Fatalf("next revision error = %v", err)
	}

	if err := validateAutomationRevision(next, next); !errors.Is(err, ErrConcurrentUpdate) {
		t.Fatalf("stale revision error = %v, want ErrConcurrentUpdate", err)
	}
	duplicate := current
	if err := validateAutomationRevision(current, duplicate); !errors.Is(err, ErrRepositoryConflict) {
		t.Fatalf("duplicate create error = %v, want ErrRepositoryConflict", err)
	}
	changedIdentity := next
	changedIdentity.Action = "drive.delete"
	if err := validateAutomationRevision(current, changedIdentity); !errors.Is(err, ErrRepositoryConflict) {
		t.Fatalf("immutable action error = %v, want ErrRepositoryConflict", err)
	}
}

func TestAutomationRequestRepositoryRejectsInvalidDocuments(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	stored := storeAutomationRequest(newTestAutomationRequest(t), now)
	stored.CreatedAt = now

	tests := map[string]func(*storedAutomationRequest){
		"schema":         func(value *storedAutomationRequest) { value.SchemaVersion = 2 },
		"revision":       func(value *storedAutomationRequest) { value.Revision = 0 },
		"created at":     func(value *storedAutomationRequest) { value.CreatedAt = time.Time{} },
		"updated before": func(value *storedAutomationRequest) { value.UpdatedAt = value.CreatedAt.Add(-time.Second) },
		"status":         func(value *storedAutomationRequest) { value.Status = "invented" },
		"approval":       func(value *storedAutomationRequest) { value.ApprovedBy = "unexpected" },
	}
	for name, corrupt := range tests {
		t.Run(name, func(t *testing.T) {
			value := stored
			value.RequiredScopes = append([]string(nil), stored.RequiredScopes...)
			corrupt(&value)
			if _, err := restoreAutomationRequest(value); err == nil {
				t.Fatal("restoreAutomationRequest(corrupt) error = nil")
			}
		})
	}
}

func TestAutomationRequestRepositoryErrorClassification(t *testing.T) {
	if err := classifyAutomationSaveError(context.Background(), status.Error(codes.FailedPrecondition, "provider detail")); !errors.Is(err, ErrConcurrentUpdate) {
		t.Fatalf("precondition error = %v, want ErrConcurrentUpdate", err)
	}
	if err := classifyAutomationSaveError(context.Background(), status.Error(codes.PermissionDenied, "secret project")); !errors.Is(err, ErrRepositoryUnavailable) {
		t.Fatalf("permission error = %v, want ErrRepositoryUnavailable", err)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if err := classifyAutomationSaveError(ctx, status.Error(codes.DeadlineExceeded, "provider detail")); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline error = %v, want context deadline", err)
	}
}

func TestAutomationRequestDocumentIDIsOpaqueAndStable(t *testing.T) {
	first := automationRequestDocumentID("workspace/request/42")
	if first != automationRequestDocumentID("workspace/request/42") || first == "workspace/request/42" || len(first) != 64 {
		t.Fatalf("automationRequestDocumentID returned %q", first)
	}
	if first == automationRequestDocumentID("workspace/request/43") {
		t.Fatal("different aggregate IDs produced the same document ID")
	}
}

func newTestAutomationRequest(t *testing.T) aggregates.AutomationRequest {
	t.Helper()
	id, err := valueobjects.NewAutomationRequestID("request-42")
	if err != nil {
		t.Fatal(err)
	}
	action, err := valueobjects.NewAction("calendar.create")
	if err != nil {
		t.Fatal(err)
	}
	scopes, err := valueobjects.NewRequiredScopes([]string{"calendar.write", "calendar.read"})
	if err != nil {
		t.Fatal(err)
	}
	value, err := aggregates.NewAutomationRequest(id, action, scopes)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func slicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
