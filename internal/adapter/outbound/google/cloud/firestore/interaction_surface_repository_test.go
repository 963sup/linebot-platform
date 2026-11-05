package firestore

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/experience-presentation/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/experience-presentation/value-objects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewInteractionSurfaceRepositoryRequiresClient(t *testing.T) {
	if _, err := NewInteractionSurfaceRepository(nil); err == nil {
		t.Fatal("NewInteractionSurfaceRepository(nil) error = nil")
	}
}

func TestInteractionSurfaceRepositoryRejectsVersionOutsideFirestoreIntegerRange(t *testing.T) {
	id, _ := valueobjects.NewSurfaceID("main-menu")
	region, _ := valueobjects.NewRegion("hero", "open-promotion", "asset-1")
	presentation, _ := valueobjects.NewPresentation([]valueobjects.Region{region})
	value, err := aggregates.RestoreInteractionSurface(id, presentation, valueobjects.Version(math.MaxUint64), aggregates.StatusDraft)
	if err != nil {
		t.Fatal(err)
	}
	if err := (&InteractionSurfaceRepository{}).Save(context.Background(), value); err == nil {
		t.Fatal("Save(max uint64 version) error = nil")
	}
}

func TestInteractionSurfaceRepositoryMappingRoundTripDoesNotRaiseEvents(t *testing.T) {
	id, _ := valueobjects.NewSurfaceID("main-menu")
	region, _ := valueobjects.NewRegion("hero", "open-promotion", "asset-1")
	presentation, _ := valueobjects.NewPresentation([]valueobjects.Region{region})
	value, err := aggregates.NewInteractionSurface(id, presentation)
	if err != nil {
		t.Fatal(err)
	}
	stored := storeInteractionSurface(value, time.Unix(1, 0).UTC())
	stored.CreatedAt = stored.UpdatedAt
	restored, err := restoreInteractionSurface(stored)
	if err != nil {
		t.Fatal(err)
	}
	if restored.ID() != value.ID() || restored.Version() != value.Version() || restored.Status() != value.Status() || !restored.Presentation().Equal(value.Presentation()) {
		t.Fatalf("restored surface = %#v, want %#v", restored, value)
	}
	if len(restored.PendingEvents()) != 0 {
		t.Fatalf("restored pending events = %d, want 0", len(restored.PendingEvents()))
	}
}

func TestInteractionSurfaceRepositoryRejectsCorruptAndConcurrentState(t *testing.T) {
	if _, err := restoreInteractionSurface(storedInteractionSurface{SchemaVersion: 2}); err == nil {
		t.Fatal("unsupported schema restored")
	}
	if _, err := restoreInteractionSurface(storedInteractionSurface{SchemaVersion: 1, Version: -1}); err == nil {
		t.Fatal("negative version restored")
	}
	regions := []storedRegion{{ID: "hero", ActionReference: "open", AssetReference: "asset"}}
	now := time.Unix(1, 0).UTC()
	current := storedInteractionSurface{SchemaVersion: 1, ID: "surface", Regions: regions, Version: 3, Status: uint8(aggregates.StatusDraft), CreatedAt: now, UpdatedAt: now}
	stale := current
	stale.Version = 2
	if err := validateSurfaceRevision(current, stale); !errors.Is(err, ErrConcurrentUpdate) {
		t.Fatalf("stale revision error = %v, want ErrConcurrentUpdate", err)
	}
	conflicting := current
	conflicting.Regions = []storedRegion{{ID: "other", ActionReference: "open"}}
	if err := validateSurfaceRevision(current, conflicting); !errors.Is(err, ErrConcurrentUpdate) {
		t.Fatalf("same-version update error = %v, want ErrConcurrentUpdate", err)
	}
	duplicate := current
	duplicate.Version = 1
	if err := validateSurfaceRevision(current, duplicate); !errors.Is(err, ErrRepositoryConflict) {
		t.Fatalf("duplicate create error = %v, want ErrRepositoryConflict", err)
	}
	next := current
	next.Version++
	next.Regions = []storedRegion{{ID: "other", ActionReference: "open"}}
	if err := validateSurfaceRevision(current, next); err != nil {
		t.Fatalf("next revision error = %v", err)
	}
}

func TestInteractionSurfaceRepositoryClassifiesProviderErrorsWithoutLeakingDetails(t *testing.T) {
	err := classifySurfaceSaveError(context.Background(), status.Error(codes.Internal, "sensitive endpoint"))
	if !errors.Is(err, ErrRepositoryUnavailable) || strings.Contains(err.Error(), "sensitive endpoint") {
		t.Fatalf("classified error = %q", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := classifySurfaceSaveError(ctx, status.Error(codes.Canceled, "provider detail")); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled error = %v, want context.Canceled", err)
	}
}
