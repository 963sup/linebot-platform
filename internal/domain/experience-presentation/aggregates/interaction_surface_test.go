package aggregates_test

import (
	"errors"
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/experience-presentation/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/experience-presentation/events"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/experience-presentation/value-objects"
)

func TestInteractionSurfaceLifecycle(t *testing.T) {
	t.Parallel()

	surface := newSurface(t)
	if got, want := surface.Status(), aggregates.StatusDraft; got != want {
		t.Errorf("NewInteractionSurface(...).Status() = %v, want %v", got, want)
	}
	if err := surface.Publish(); err != nil {
		t.Fatalf("InteractionSurface.Publish() error = %v, want nil", err)
	}
	if got, want := surface.Status(), aggregates.StatusPublished; got != want {
		t.Errorf("InteractionSurface.Status() after Publish = %v, want %v", got, want)
	}
	if got, want := surface.Version(), valueobjects.Version(2); got != want {
		t.Errorf("InteractionSurface.Version() after Publish = %v, want %v", got, want)
	}
	if err := surface.Retire(); err != nil {
		t.Fatalf("InteractionSurface.Retire() error = %v, want nil", err)
	}
	if got, want := surface.Status(), aggregates.StatusRetired; got != want {
		t.Errorf("InteractionSurface.Status() after Retire = %v, want %v", got, want)
	}
	if got, want := surface.Version(), valueobjects.Version(3); got != want {
		t.Errorf("InteractionSurface.Version() after Retire = %v, want %v", got, want)
	}
}

func TestPublishedInteractionSurfaceIsImmutable(t *testing.T) {
	t.Parallel()

	surface := newSurface(t)
	if err := surface.Publish(); err != nil {
		t.Fatalf("InteractionSurface.Publish() error = %v, want nil", err)
	}
	revised := mustPresentation(t, "secondary")
	if err := surface.Revise(revised); !errors.Is(err, aggregates.ErrPublishedSurfaceImmutable) {
		t.Errorf("InteractionSurface.Revise(...) after Publish error = %v, want %v", err, aggregates.ErrPublishedSurfaceImmutable)
	}
	if got, want := surface.Version(), valueobjects.Version(2); got != want {
		t.Errorf("InteractionSurface.Version() after rejected revision = %v, want %v", got, want)
	}
}

func TestDraftRevisionAdvancesVersion(t *testing.T) {
	t.Parallel()

	surface := newSurface(t)
	if err := surface.Revise(mustPresentation(t, "secondary")); err != nil {
		t.Fatalf("InteractionSurface.Revise(...) error = %v, want nil", err)
	}
	if got, want := surface.Version(), valueobjects.Version(2); got != want {
		t.Errorf("InteractionSurface.Version() after revision = %v, want %v", got, want)
	}
}

func TestInteractionSurfaceRejectsInvalidTransitions(t *testing.T) {
	t.Parallel()

	surface := newSurface(t)
	if err := surface.Retire(); !errors.Is(err, aggregates.ErrInvalidSurfaceTransition) {
		t.Errorf("InteractionSurface.Retire() from draft error = %v, want %v", err, aggregates.ErrInvalidSurfaceTransition)
	}
	if err := surface.Publish(); err != nil {
		t.Fatalf("InteractionSurface.Publish() error = %v, want nil", err)
	}
	if err := surface.Publish(); !errors.Is(err, aggregates.ErrInvalidSurfaceTransition) {
		t.Errorf("InteractionSurface.Publish() twice error = %v, want %v", err, aggregates.ErrInvalidSurfaceTransition)
	}
}

func TestInteractionSurfaceEventsAreDetached(t *testing.T) {
	t.Parallel()

	surface := newSurface(t)
	got := surface.PendingEvents()
	if len(got) != 1 {
		t.Fatalf("InteractionSurface.PendingEvents() length = %d, want 1", len(got))
	}
	if _, ok := got[0].(events.Defined); !ok {
		t.Errorf("InteractionSurface.PendingEvents()[0] = %T, want events.Defined", got[0])
	}
	got[0] = nil
	if surface.PendingEvents()[0] == nil {
		t.Error("InteractionSurface.PendingEvents() returned mutable internal storage, want detached slice")
	}
	surface.AcknowledgeEvents()
	if len(surface.PendingEvents()) != 0 {
		t.Error("InteractionSurface.AcknowledgeEvents() did not clear committed events")
	}
}

func newSurface(t *testing.T) aggregates.InteractionSurface {
	t.Helper()
	id, err := valueobjects.NewSurfaceID("main-menu")
	if err != nil {
		t.Fatalf("NewSurfaceID(...) error = %v, want nil", err)
	}
	surface, err := aggregates.NewInteractionSurface(id, mustPresentation(t, "primary"))
	if err != nil {
		t.Fatalf("NewInteractionSurface(...) error = %v, want nil", err)
	}
	return surface
}

func mustPresentation(t *testing.T, regionID string) valueobjects.Presentation {
	t.Helper()
	region, err := valueobjects.NewRegion(regionID, "show-products", "hero-image")
	if err != nil {
		t.Fatalf("NewRegion(...) error = %v, want nil", err)
	}
	presentation, err := valueobjects.NewPresentation([]valueobjects.Region{region})
	if err != nil {
		t.Fatalf("NewPresentation(...) error = %v, want nil", err)
	}
	return presentation
}
