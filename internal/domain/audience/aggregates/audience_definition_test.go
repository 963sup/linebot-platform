package aggregates_test

import (
	"errors"
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/audience/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/audience/events"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/audience/value-objects"
)

func TestAudienceDefinitionLifecycle(t *testing.T) {
	t.Parallel()

	audience := newAudience(t)
	if got, want := audience.Status(), aggregates.StatusDraft; got != want {
		t.Errorf("NewAudienceDefinition(...).Status() = %v, want %v", got, want)
	}
	if err := audience.Activate(); err != nil {
		t.Fatalf("AudienceDefinition.Activate() error = %v, want nil", err)
	}
	if got, want := audience.Status(), aggregates.StatusActive; got != want {
		t.Errorf("AudienceDefinition.Status() after Activate = %v, want %v", got, want)
	}
	if err := audience.Archive(); err != nil {
		t.Fatalf("AudienceDefinition.Archive() error = %v, want nil", err)
	}
	if got, want := audience.Status(), aggregates.StatusArchived; got != want {
		t.Errorf("AudienceDefinition.Status() after Archive = %v, want %v", got, want)
	}
	if err := audience.Activate(); !errors.Is(err, aggregates.ErrInvalidAudienceTransition) {
		t.Errorf("AudienceDefinition.Activate() after Archive error = %v, want %v", err, aggregates.ErrInvalidAudienceTransition)
	}
}

func TestActiveAudienceRevisionAdvancesVersionAndStaysActive(t *testing.T) {
	t.Parallel()

	audience := newAudience(t)
	if err := audience.Activate(); err != nil {
		t.Fatalf("AudienceDefinition.Activate() error = %v, want nil", err)
	}
	revised := mustCriteria(t, "customer-profile", "country = TW")
	if err := audience.Revise(revised); err != nil {
		t.Fatalf("AudienceDefinition.Revise(%v) error = %v, want nil", revised, err)
	}
	if got, want := audience.Version(), valueobjects.Version(2); got != want {
		t.Errorf("AudienceDefinition.Version() after active revision = %v, want %v", got, want)
	}
	if got, want := audience.Status(), aggregates.StatusActive; got != want {
		t.Errorf("AudienceDefinition.Status() after active revision = %v, want %v", got, want)
	}
	if got := audience.Criteria(); !got.Equal(revised) {
		t.Errorf("AudienceDefinition.Criteria() = %v, want %v", got, revised)
	}
}

func TestAudienceRevisionNoOpAndArchivedProtection(t *testing.T) {
	t.Parallel()

	audience := newAudience(t)
	original := audience.Criteria()
	if err := audience.Revise(original); err != nil {
		t.Fatalf("AudienceDefinition.Revise(same) error = %v, want nil", err)
	}
	if got, want := audience.Version(), valueobjects.InitialVersion; got != want {
		t.Errorf("AudienceDefinition.Version() after no-op = %v, want %v", got, want)
	}
	if err := audience.Archive(); err != nil {
		t.Fatalf("AudienceDefinition.Archive() error = %v, want nil", err)
	}
	if err := audience.Revise(mustCriteria(t, "orders", "count > 3")); !errors.Is(err, aggregates.ErrArchivedAudience) {
		t.Errorf("AudienceDefinition.Revise(...) after Archive error = %v, want %v", err, aggregates.ErrArchivedAudience)
	}
}

func TestAudienceEventsAreDetached(t *testing.T) {
	t.Parallel()

	audience := newAudience(t)
	got := audience.PendingEvents()
	if len(got) != 1 {
		t.Fatalf("AudienceDefinition.PendingEvents() length = %d, want 1", len(got))
	}
	if _, ok := got[0].(events.Defined); !ok {
		t.Errorf("AudienceDefinition.PendingEvents()[0] = %T, want events.Defined", got[0])
	}
	got[0] = nil
	if audience.PendingEvents()[0] == nil {
		t.Error("AudienceDefinition.PendingEvents() returned mutable internal storage, want detached slice")
	}
}

func TestRestoreAudienceDefinitionRejectsInvalidState(t *testing.T) {
	t.Parallel()

	criteria := mustCriteria(t, "orders", "count > 0")
	if _, err := aggregates.RestoreAudienceDefinition("", criteria, valueobjects.InitialVersion, aggregates.StatusDraft); !errors.Is(err, aggregates.ErrInvalidAudienceDefinition) {
		t.Errorf("RestoreAudienceDefinition(invalid id) error = %v, want %v", err, aggregates.ErrInvalidAudienceDefinition)
	}
}

func newAudience(t *testing.T) aggregates.AudienceDefinition {
	t.Helper()
	id, err := valueobjects.NewAudienceID("repeat-buyers")
	if err != nil {
		t.Fatalf("NewAudienceID(...) error = %v, want nil", err)
	}
	audience, err := aggregates.NewAudienceDefinition(id, mustCriteria(t, "orders", "count > 1"))
	if err != nil {
		t.Fatalf("NewAudienceDefinition(...) error = %v, want nil", err)
	}
	return audience
}

func mustCriteria(t *testing.T, source, definition string) valueobjects.SelectionCriteria {
	t.Helper()
	criteria, err := valueobjects.NewSelectionCriteria(source, definition)
	if err != nil {
		t.Fatalf("NewSelectionCriteria(%q, %q) error = %v, want nil", source, definition, err)
	}
	return criteria
}
