package experiencepresentation

import (
	"context"
	"errors"
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/experience-presentation/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/experience-presentation/value-objects"
)

var errPresentationPort = errors.New("presentation port failure")

type surfaceRepository struct {
	value     aggregates.InteractionSurface
	attempted aggregates.InteractionSurface
	findErr   error
	saveErr   error
	saveCalls int
}

func (r *surfaceRepository) Find(context.Context, valueobjects.SurfaceID) (aggregates.InteractionSurface, error) {
	return r.value, r.findErr
}
func (r *surfaceRepository) Save(_ context.Context, value aggregates.InteractionSurface) error {
	r.saveCalls++
	r.attempted = value
	if r.saveErr == nil {
		r.value = value
	}
	return r.saveErr
}

func TestPresentationNoopRevisionSkipsPersistence(t *testing.T) {
	t.Parallel()
	id, presentation := surfaceValues(t, "same", "primary")
	current, _ := aggregates.RestoreInteractionSurface(id, presentation, 4, aggregates.StatusDraft)
	repository := &surfaceRepository{value: current}
	service, _ := NewService(repository, &presentationProvider{})

	result, err := service.Revise(context.Background(), id, presentation)
	if err != nil || result.Version() != 4 {
		t.Fatalf("Revise(no-op) = (%v, %v), want version 4, nil", result.Version(), err)
	}
	if repository.saveCalls != 0 {
		t.Fatalf("Save calls = %d, want 0", repository.saveCalls)
	}
}

func TestPresentationReturnsDefinitionOnlyAfterSuccessfulSave(t *testing.T) {
	t.Parallel()
	id, presentation := surfaceValues(t, "events", "primary")
	repository := &surfaceRepository{saveErr: errPresentationPort}
	service, _ := NewService(repository, &presentationProvider{})
	if _, err := service.Define(context.Background(), id, presentation); !errors.Is(err, errPresentationPort) {
		t.Fatalf("Define(save error) = %v, want %v", err, errPresentationPort)
	}
	repository.saveErr = nil
	defined, err := service.Define(context.Background(), id, presentation)
	if err != nil {
		t.Fatal(err)
	}
	if defined.ID() != id || repository.value.ID() != id || repository.saveCalls != 2 {
		t.Fatal("successful save did not preserve the definition")
	}
}

type presentationProvider struct {
	result Deployment
	err    error
	calls  int
	value  aggregates.InteractionSurface
}

func (p *presentationProvider) Deploy(_ context.Context, value aggregates.InteractionSurface) (Deployment, error) {
	p.calls++
	p.value = value
	return p.result, p.err
}

func TestPresentationCommandsPersistDomainTransitions(t *testing.T) {
	t.Parallel()
	repository := &surfaceRepository{}
	service, _ := NewService(repository, &presentationProvider{})
	id, first := surfaceValues(t, "main", "primary")

	defined, err := service.Define(context.Background(), id, first)
	if err != nil || defined.Status() != aggregates.StatusDraft {
		t.Fatalf("Define() = (%v, %v), want draft, nil", defined.Status(), err)
	}
	second := mustPresentation(t, "secondary")
	revised, err := service.Revise(context.Background(), id, second)
	if err != nil || revised.Version() != 2 || !revised.Presentation().Equal(second) {
		t.Fatalf("Revise() = (%v, %v), want version 2", revised, err)
	}
	published, err := service.Publish(context.Background(), id)
	if err != nil || published.Status() != aggregates.StatusPublished {
		t.Fatalf("Publish() = (%v, %v), want published, nil", published.Status(), err)
	}
	retired, err := service.Retire(context.Background(), id)
	if err != nil || retired.Status() != aggregates.StatusRetired {
		t.Fatalf("Retire() = (%v, %v), want retired, nil", retired.Status(), err)
	}
	if repository.saveCalls != 4 {
		t.Fatalf("Save calls = %d, want 4", repository.saveCalls)
	}
}

func TestDeployRequiresPublishedSurface(t *testing.T) {
	t.Parallel()
	id, presentation := surfaceValues(t, "main", "primary")
	draft, _ := aggregates.NewInteractionSurface(id, presentation)
	repository := &surfaceRepository{value: draft}
	provider := &presentationProvider{result: Deployment{ProviderReference: "rich-menu-1", Version: 1}}
	service, _ := NewService(repository, provider)
	if _, err := service.Deploy(context.Background(), id); !errors.Is(err, ErrPublishedRequired) {
		t.Fatalf("Deploy(draft) error = %v, want %v", err, ErrPublishedRequired)
	}
	if provider.calls != 0 {
		t.Fatal("draft surface reached provider")
	}
	_ = repository.value.Publish()
	result, err := service.Deploy(context.Background(), id)
	if err != nil || result != provider.result || provider.value.Status() != aggregates.StatusPublished {
		t.Fatalf("Deploy(published) = (%v, %v), want (%v, nil)", result, err, provider.result)
	}
	provider.err = errPresentationPort
	if _, err := service.Deploy(context.Background(), id); !errors.Is(err, errPresentationPort) {
		t.Fatalf("Deploy(provider error) = %v, want %v", err, errPresentationPort)
	}
}

func TestPresentationErrorsStopTheWorkflow(t *testing.T) {
	t.Parallel()
	id, presentation := surfaceValues(t, "main", "primary")
	repository := &surfaceRepository{findErr: errPresentationPort}
	provider := &presentationProvider{}
	service, _ := NewService(repository, provider)
	if _, err := service.Publish(context.Background(), id); !errors.Is(err, errPresentationPort) {
		t.Fatalf("Publish(find error) = %v, want %v", err, errPresentationPort)
	}
	repository.findErr = nil
	repository.value, _ = aggregates.NewInteractionSurface(id, presentation)
	repository.saveErr = errPresentationPort
	if _, err := service.Publish(context.Background(), id); !errors.Is(err, errPresentationPort) {
		t.Fatalf("Publish(save error) = %v, want %v", err, errPresentationPort)
	}
	if provider.calls != 0 {
		t.Fatal("failed workflow reached provider")
	}
}

func TestPresentationServiceRequiresDependenciesAndContext(t *testing.T) {
	t.Parallel()
	provider, repository := &presentationProvider{}, &surfaceRepository{}
	if _, err := NewService(nil, provider); err == nil {
		t.Fatal("NewService(nil, provider) error = nil")
	}
	if _, err := NewService(repository, nil); err == nil {
		t.Fatal("NewService(repository, nil) error = nil")
	}
	service, _ := NewService(repository, provider)
	id, presentation := surfaceValues(t, "main", "primary")
	if _, err := service.Define(nil, id, presentation); !errors.Is(err, ErrContextRequired) {
		t.Fatalf("Define(nil) error = %v, want %v", err, ErrContextRequired)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Define(ctx, id, presentation); !errors.Is(err, context.Canceled) {
		t.Fatalf("Define(canceled) error = %v, want %v", err, context.Canceled)
	}
}

func surfaceValues(t *testing.T, id, region string) (valueobjects.SurfaceID, valueobjects.Presentation) {
	t.Helper()
	surfaceID, err := valueobjects.NewSurfaceID(id)
	if err != nil {
		t.Fatal(err)
	}
	return surfaceID, mustPresentation(t, region)
}

func mustPresentation(t *testing.T, regionID string) valueobjects.Presentation {
	t.Helper()
	region, err := valueobjects.NewRegion(regionID, "message:hello", "")
	if err != nil {
		t.Fatal(err)
	}
	presentation, err := valueobjects.NewPresentation([]valueobjects.Region{region})
	if err != nil {
		t.Fatal(err)
	}
	return presentation
}
