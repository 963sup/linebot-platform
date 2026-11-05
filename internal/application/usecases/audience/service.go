// Package audience orchestrates the Audience bounded context.
package audience

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/domain/audience/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/audience/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/audience/value-objects"
)

var (
	ErrContextRequired = errors.New("context is required")
	ErrActiveRequired  = errors.New("audience must be active before synchronization")
)

// SyncResult is the provider-independent result of synchronizing an audience.
type SyncResult struct {
	ProviderReference string
	SynchronizedCount int
}

// ProviderGateway synchronizes the current, active audience definition with an
// external audience provider. Implementations must not mutate the aggregate.
type ProviderGateway interface {
	Synchronize(context.Context, aggregates.AudienceDefinition) (SyncResult, error)
}

type Service struct {
	repository repositories.AudienceRepository
	provider   ProviderGateway
}

func NewService(repository repositories.AudienceRepository, provider ProviderGateway) (*Service, error) {
	if repository == nil || provider == nil {
		return nil, errors.New("audience repository and provider gateway are required")
	}
	return &Service{repository: repository, provider: provider}, nil
}

func (s *Service) Define(ctx context.Context, id valueobjects.AudienceID, criteria valueobjects.SelectionCriteria) (aggregates.AudienceDefinition, error) {
	if err := validateContext(ctx); err != nil {
		return aggregates.AudienceDefinition{}, err
	}
	audience, err := aggregates.NewAudienceDefinition(id, criteria)
	if err != nil {
		return aggregates.AudienceDefinition{}, err
	}
	if err := s.repository.Save(ctx, audience); err != nil {
		return aggregates.AudienceDefinition{}, err
	}
	return audience, nil
}

func (s *Service) Revise(ctx context.Context, id valueobjects.AudienceID, criteria valueobjects.SelectionCriteria) (aggregates.AudienceDefinition, error) {
	return s.change(ctx, id, func(audience *aggregates.AudienceDefinition) error {
		return audience.Revise(criteria)
	})
}

func (s *Service) Activate(ctx context.Context, id valueobjects.AudienceID) (aggregates.AudienceDefinition, error) {
	return s.change(ctx, id, func(audience *aggregates.AudienceDefinition) error {
		return audience.Activate()
	})
}

func (s *Service) Archive(ctx context.Context, id valueobjects.AudienceID) (aggregates.AudienceDefinition, error) {
	return s.change(ctx, id, func(audience *aggregates.AudienceDefinition) error {
		return audience.Archive()
	})
}

func (s *Service) Sync(ctx context.Context, id valueobjects.AudienceID) (SyncResult, error) {
	if err := validateContext(ctx); err != nil {
		return SyncResult{}, err
	}
	audience, err := s.repository.Find(ctx, id)
	if err != nil {
		return SyncResult{}, err
	}
	if audience.Status() != aggregates.StatusActive {
		return SyncResult{}, ErrActiveRequired
	}
	return s.provider.Synchronize(ctx, audience)
}

func (s *Service) change(ctx context.Context, id valueobjects.AudienceID, change func(*aggregates.AudienceDefinition) error) (aggregates.AudienceDefinition, error) {
	if err := validateContext(ctx); err != nil {
		return aggregates.AudienceDefinition{}, err
	}
	audience, err := s.repository.Find(ctx, id)
	if err != nil {
		return aggregates.AudienceDefinition{}, err
	}
	if err := change(&audience); err != nil {
		return aggregates.AudienceDefinition{}, err
	}
	if err := s.repository.Save(ctx, audience); err != nil {
		return aggregates.AudienceDefinition{}, err
	}
	return audience, nil
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return ErrContextRequired
	}
	return ctx.Err()
}
