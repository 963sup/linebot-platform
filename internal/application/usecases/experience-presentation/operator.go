package experiencepresentation

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/domain/experience-presentation/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/experience-presentation/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/experience-presentation/value-objects"
)

var ErrOperatorProviderUnavailable = errors.New("interaction surface deploy provider is not configured for operator CLI")

type OperatorRegion struct {
	ID              string
	ActionReference string
	AssetReference  string
}

type OperatorSummary struct {
	ID          string
	RegionCount int
	Version     uint64
	Status      string
}

type Operator struct {
	repository         repositories.InteractionSurfaceRepository
	service            *Service
	providerConfigured bool
}

type unavailableOperatorProvider struct{}

func (unavailableOperatorProvider) Deploy(context.Context, aggregates.InteractionSurface) (Deployment, error) {
	return Deployment{}, ErrOperatorProviderUnavailable
}

func NewOperator(repository repositories.InteractionSurfaceRepository, provider ProviderGateway) (*Operator, error) {
	if repository == nil {
		return nil, errors.New("interaction surface repository is required")
	}
	configured := provider != nil
	if provider == nil {
		provider = unavailableOperatorProvider{}
	}
	service, err := NewService(repository, provider)
	if err != nil {
		return nil, err
	}
	return &Operator{repository: repository, service: service, providerConfigured: configured}, nil
}

func (o *Operator) Get(ctx context.Context, rawID string) (OperatorSummary, error) {
	id, err := valueobjects.NewSurfaceID(rawID)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.repository.Find(ctx, id)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizeSurface(value), nil
}

func (o *Operator) Define(ctx context.Context, rawID string, regions []OperatorRegion) (OperatorSummary, error) {
	id, presentation, err := surfaceInput(rawID, regions)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.service.Define(ctx, id, presentation)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizeSurface(value), nil
}

func (o *Operator) Revise(ctx context.Context, rawID string, regions []OperatorRegion) (OperatorSummary, error) {
	id, presentation, err := surfaceInput(rawID, regions)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.service.Revise(ctx, id, presentation)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizeSurface(value), nil
}

func (o *Operator) Publish(ctx context.Context, rawID string) (OperatorSummary, error) {
	id, err := valueobjects.NewSurfaceID(rawID)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.service.Publish(ctx, id)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizeSurface(value), nil
}

func (o *Operator) Retire(ctx context.Context, rawID string) (OperatorSummary, error) {
	id, err := valueobjects.NewSurfaceID(rawID)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.service.Retire(ctx, id)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizeSurface(value), nil
}

func (o *Operator) Deploy(ctx context.Context, rawID string) (Deployment, error) {
	if !o.providerConfigured {
		return Deployment{}, ErrOperatorProviderUnavailable
	}
	id, err := valueobjects.NewSurfaceID(rawID)
	if err != nil {
		return Deployment{}, err
	}
	return o.service.Deploy(ctx, id)
}

func surfaceInput(rawID string, input []OperatorRegion) (valueobjects.SurfaceID, valueobjects.Presentation, error) {
	id, err := valueobjects.NewSurfaceID(rawID)
	if err != nil {
		return valueobjects.SurfaceID(""), valueobjects.Presentation{}, err
	}
	regions := make([]valueobjects.Region, 0, len(input))
	for _, item := range input {
		region, err := valueobjects.NewRegion(item.ID, item.ActionReference, item.AssetReference)
		if err != nil {
			return valueobjects.SurfaceID(""), valueobjects.Presentation{}, err
		}
		regions = append(regions, region)
	}
	presentation, err := valueobjects.NewPresentation(regions)
	return id, presentation, err
}

func summarizeSurface(value aggregates.InteractionSurface) OperatorSummary {
	return OperatorSummary{ID: value.ID().String(), RegionCount: len(value.Presentation().Regions()), Version: uint64(value.Version()), Status: surfaceStatus(value.Status())}
}

func surfaceStatus(status aggregates.Status) string {
	switch status {
	case aggregates.StatusDraft:
		return "draft"
	case aggregates.StatusPublished:
		return "published"
	case aggregates.StatusRetired:
		return "retired"
	default:
		return "unknown"
	}
}
