package audience

import (
	"context"
	"errors"
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/audience/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/audience/value-objects"
)

var errAudiencePort = errors.New("audience port failure")

type audienceRepository struct {
	value     aggregates.AudienceDefinition
	findErr   error
	saveErr   error
	saveCalls int
}

func (r *audienceRepository) Find(context.Context, valueobjects.AudienceID) (aggregates.AudienceDefinition, error) {
	return r.value, r.findErr
}
func (r *audienceRepository) Save(_ context.Context, value aggregates.AudienceDefinition) error {
	r.saveCalls++
	if r.saveErr == nil {
		r.value = value
	}
	return r.saveErr
}

type audienceProvider struct {
	result SyncResult
	err    error
	calls  int
	value  aggregates.AudienceDefinition
}

func (p *audienceProvider) Synchronize(_ context.Context, value aggregates.AudienceDefinition) (SyncResult, error) {
	p.calls++
	p.value = value
	return p.result, p.err
}

func TestNewServiceRequiresDependencies(t *testing.T) {
	t.Parallel()
	provider := &audienceProvider{}
	repository := &audienceRepository{}
	if _, err := NewService(nil, provider); err == nil {
		t.Fatal("NewService(nil, provider) error = nil, want error")
	}
	if _, err := NewService(repository, nil); err == nil {
		t.Fatal("NewService(repository, nil) error = nil, want error")
	}
}

func TestAudienceCommandsPersistDomainTransitions(t *testing.T) {
	t.Parallel()
	repository := &audienceRepository{}
	service, _ := NewService(repository, &audienceProvider{})
	id, criteria := audienceValues(t, "repeat-buyers", "orders", "count > 1")

	defined, err := service.Define(context.Background(), id, criteria)
	if err != nil || defined.Status() != aggregates.StatusDraft {
		t.Fatalf("Define() = (%v, %v), want draft, nil", defined.Status(), err)
	}
	revisedCriteria := mustAudienceCriteria(t, "orders", "count > 2")
	revised, err := service.Revise(context.Background(), id, revisedCriteria)
	if err != nil || revised.Version() != 2 || !revised.Criteria().Equal(revisedCriteria) {
		t.Fatalf("Revise() = (%v, %v), want version 2 with revised criteria", revised, err)
	}
	active, err := service.Activate(context.Background(), id)
	if err != nil || active.Status() != aggregates.StatusActive {
		t.Fatalf("Activate() = (%v, %v), want active, nil", active.Status(), err)
	}
	archived, err := service.Archive(context.Background(), id)
	if err != nil || archived.Status() != aggregates.StatusArchived {
		t.Fatalf("Archive() = (%v, %v), want archived, nil", archived.Status(), err)
	}
	if repository.saveCalls != 4 {
		t.Fatalf("Save calls = %d, want 4", repository.saveCalls)
	}
}

func TestAudienceSyncRequiresActiveAndPropagatesProviderResult(t *testing.T) {
	t.Parallel()
	id, criteria := audienceValues(t, "active", "orders", "count > 0")
	draft, _ := aggregates.NewAudienceDefinition(id, criteria)
	repository := &audienceRepository{value: draft}
	provider := &audienceProvider{result: SyncResult{ProviderReference: "provider-1", SynchronizedCount: 8}}
	service, _ := NewService(repository, provider)

	if _, err := service.Sync(context.Background(), id); !errors.Is(err, ErrActiveRequired) {
		t.Fatalf("Sync(draft) error = %v, want %v", err, ErrActiveRequired)
	}
	if provider.calls != 0 {
		t.Fatal("draft audience reached provider")
	}
	_ = repository.value.Activate()
	result, err := service.Sync(context.Background(), id)
	if err != nil || result != provider.result || provider.value.Status() != aggregates.StatusActive {
		t.Fatalf("Sync(active) = (%v, %v), want (%v, nil)", result, err, provider.result)
	}
	provider.err = errAudiencePort
	if _, err := service.Sync(context.Background(), id); !errors.Is(err, errAudiencePort) {
		t.Fatalf("Sync(provider error) = %v, want %v", err, errAudiencePort)
	}
}

func TestAudienceFailuresDoNotCauseLaterEffects(t *testing.T) {
	t.Parallel()
	id, criteria := audienceValues(t, "audience", "orders", "count > 0")
	repository := &audienceRepository{findErr: errAudiencePort}
	provider := &audienceProvider{}
	service, _ := NewService(repository, provider)
	if _, err := service.Activate(context.Background(), id); !errors.Is(err, errAudiencePort) {
		t.Fatalf("Activate(find error) = %v, want %v", err, errAudiencePort)
	}
	if repository.saveCalls != 0 {
		t.Fatal("find failure was followed by Save")
	}
	repository.findErr = nil
	repository.value, _ = aggregates.NewAudienceDefinition(id, criteria)
	repository.saveErr = errAudiencePort
	if _, err := service.Activate(context.Background(), id); !errors.Is(err, errAudiencePort) {
		t.Fatalf("Activate(save error) = %v, want %v", err, errAudiencePort)
	}
	if provider.calls != 0 {
		t.Fatal("save failure reached provider")
	}
}

func TestAudienceCommandsHonorContext(t *testing.T) {
	t.Parallel()
	id, criteria := audienceValues(t, "audience", "orders", "count > 0")
	repository := &audienceRepository{}
	service, _ := NewService(repository, &audienceProvider{})
	if _, err := service.Define(nil, id, criteria); !errors.Is(err, ErrContextRequired) {
		t.Fatalf("Define(nil) error = %v, want %v", err, ErrContextRequired)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Define(ctx, id, criteria); !errors.Is(err, context.Canceled) {
		t.Fatalf("Define(canceled) error = %v, want %v", err, context.Canceled)
	}
	if repository.saveCalls != 0 {
		t.Fatal("invalid context reached repository")
	}
}

func audienceValues(t *testing.T, id, source, definition string) (valueobjects.AudienceID, valueobjects.SelectionCriteria) {
	t.Helper()
	audienceID, err := valueobjects.NewAudienceID(id)
	if err != nil {
		t.Fatal(err)
	}
	return audienceID, mustAudienceCriteria(t, source, definition)
}

func mustAudienceCriteria(t *testing.T, source, definition string) valueobjects.SelectionCriteria {
	t.Helper()
	criteria, err := valueobjects.NewSelectionCriteria(source, definition)
	if err != nil {
		t.Fatal(err)
	}
	return criteria
}
