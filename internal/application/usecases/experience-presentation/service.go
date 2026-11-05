// Package experiencepresentation orchestrates the Experience Presentation bounded context.
package experiencepresentation

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/domain/experience-presentation/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/experience-presentation/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/experience-presentation/value-objects"
)

var (
	ErrContextRequired   = errors.New("context is required")
	ErrPublishedRequired = errors.New("interaction surface must be published before deployment")
)

type Deployment struct {
	ProviderReference string
	Version           valueobjects.Version
}

// ProviderGateway deploys a published, provider-independent interaction surface.
type ProviderGateway interface {
	Deploy(context.Context, aggregates.InteractionSurface) (Deployment, error)
}

type Service struct {
	repository repositories.InteractionSurfaceRepository
	provider   ProviderGateway
}

func NewService(repository repositories.InteractionSurfaceRepository, provider ProviderGateway) (*Service, error) {
	if repository == nil || provider == nil {
		return nil, errors.New("interaction surface repository and provider gateway are required")
	}
	return &Service{repository: repository, provider: provider}, nil
}

func (s *Service) Define(ctx context.Context, id valueobjects.SurfaceID, presentation valueobjects.Presentation) (aggregates.InteractionSurface, error) {
	if err := validateContext(ctx); err != nil {
		return aggregates.InteractionSurface{}, err
	}
	surface, err := aggregates.NewInteractionSurface(id, presentation)
	if err != nil {
		return aggregates.InteractionSurface{}, err
	}
	if err := s.repository.Save(ctx, surface); err != nil {
		return aggregates.InteractionSurface{}, err
	}
	return surface, nil
}

func (s *Service) Revise(ctx context.Context, id valueobjects.SurfaceID, presentation valueobjects.Presentation) (aggregates.InteractionSurface, error) {
	return s.change(ctx, id, func(surface *aggregates.InteractionSurface) error {
		return surface.Revise(presentation)
	})
}

func (s *Service) Publish(ctx context.Context, id valueobjects.SurfaceID) (aggregates.InteractionSurface, error) {
	return s.change(ctx, id, func(surface *aggregates.InteractionSurface) error {
		return surface.Publish()
	})
}

func (s *Service) Retire(ctx context.Context, id valueobjects.SurfaceID) (aggregates.InteractionSurface, error) {
	return s.change(ctx, id, func(surface *aggregates.InteractionSurface) error {
		return surface.Retire()
	})
}

func (s *Service) Deploy(ctx context.Context, id valueobjects.SurfaceID) (Deployment, error) {
	if err := validateContext(ctx); err != nil {
		return Deployment{}, err
	}
	surface, err := s.repository.Find(ctx, id)
	if err != nil {
		return Deployment{}, err
	}
	if surface.Status() != aggregates.StatusPublished {
		return Deployment{}, ErrPublishedRequired
	}
	return s.provider.Deploy(ctx, surface)
}

func (s *Service) change(ctx context.Context, id valueobjects.SurfaceID, change func(*aggregates.InteractionSurface) error) (aggregates.InteractionSurface, error) {
	if err := validateContext(ctx); err != nil {
		return aggregates.InteractionSurface{}, err
	}
	surface, err := s.repository.Find(ctx, id)
	if err != nil {
		return aggregates.InteractionSurface{}, err
	}
	if err := change(&surface); err != nil {
		return aggregates.InteractionSurface{}, err
	}
	if err := s.repository.Save(ctx, surface); err != nil {
		return aggregates.InteractionSurface{}, err
	}
	return surface, nil
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return ErrContextRequired
	}
	return ctx.Err()
}
