package firestore

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/audience/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/audience/value-objects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewAudienceRepositoryRequiresClient(t *testing.T) {
	if _, err := NewAudienceRepository(nil); err == nil {
		t.Fatal("NewAudienceRepository(nil) error = nil")
	}
}

func TestAudienceRepositoryRejectsVersionOutsideFirestoreIntegerRange(t *testing.T) {
	id, _ := valueobjects.NewAudienceID("repeat-buyers")
	criteria, _ := valueobjects.NewSelectionCriteria("orders", "count > 1")
	value, err := aggregates.RestoreAudienceDefinition(id, criteria, valueobjects.Version(math.MaxUint64), aggregates.StatusDraft)
	if err != nil {
		t.Fatal(err)
	}
	if err := (&AudienceRepository{}).Save(context.Background(), value); err == nil {
		t.Fatal("Save(max uint64 version) error = nil")
	}
}

func TestAudienceRepositoryMappingRoundTripDoesNotRaiseEvents(t *testing.T) {
	id, _ := valueobjects.NewAudienceID("repeat-buyers")
	criteria, _ := valueobjects.NewSelectionCriteria("orders", "count > 1")
	value, err := aggregates.NewAudienceDefinition(id, criteria)
	if err != nil {
		t.Fatal(err)
	}
	stored := storeAudienceDefinition(value, time.Unix(1, 0).UTC())
	stored.CreatedAt = stored.UpdatedAt
	restored, err := restoreAudienceDefinition(stored)
	if err != nil {
		t.Fatal(err)
	}
	if restored.ID() != value.ID() || restored.Version() != value.Version() || restored.Status() != value.Status() || !restored.Criteria().Equal(value.Criteria()) {
		t.Fatalf("restored audience = %#v, want %#v", restored, value)
	}
}

func TestAudienceRepositoryRejectsCorruptAndConcurrentState(t *testing.T) {
	if _, err := restoreAudienceDefinition(storedAudienceDefinition{SchemaVersion: 2}); err == nil {
		t.Fatal("unsupported schema restored")
	}
	if _, err := restoreAudienceDefinition(storedAudienceDefinition{SchemaVersion: 1, Version: -1}); err == nil {
		t.Fatal("negative version restored")
	}
	now := time.Unix(1, 0).UTC()
	current := storedAudienceDefinition{SchemaVersion: 1, ID: "audience", CriteriaSource: "orders", CriteriaDefinition: "count > 1", Version: 3, Status: uint8(aggregates.StatusDraft), CreatedAt: now, UpdatedAt: now}
	stale := current
	stale.Version = 2
	if err := validateAudienceRevision(current, stale); !errors.Is(err, ErrConcurrentUpdate) {
		t.Fatalf("stale revision error = %v, want ErrConcurrentUpdate", err)
	}
	conflicting := current
	conflicting.CriteriaDefinition = "count > 2"
	if err := validateAudienceRevision(current, conflicting); !errors.Is(err, ErrConcurrentUpdate) {
		t.Fatalf("same-version update error = %v, want ErrConcurrentUpdate", err)
	}
	duplicate := current
	duplicate.Version = 1
	if err := validateAudienceRevision(current, duplicate); !errors.Is(err, ErrRepositoryConflict) {
		t.Fatalf("duplicate create error = %v, want ErrRepositoryConflict", err)
	}
	next := current
	next.Version++
	next.CriteriaDefinition = "count > 2"
	if err := validateAudienceRevision(current, next); err != nil {
		t.Fatalf("next revision error = %v", err)
	}
}

func TestAudienceRepositoryClassifiesProviderErrorsWithoutLeakingDetails(t *testing.T) {
	err := classifyAudienceSaveError(context.Background(), status.Error(codes.Internal, "sensitive endpoint"))
	if !errors.Is(err, ErrRepositoryUnavailable) || strings.Contains(err.Error(), "sensitive endpoint") {
		t.Fatalf("classified error = %q", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := classifyAudienceSaveError(ctx, status.Error(codes.Canceled, "provider detail")); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled error = %v, want context.Canceled", err)
	}
}
