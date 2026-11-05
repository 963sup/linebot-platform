package audience

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/963sup/linebot-platform/internal/domain/audience/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/audience/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/audience/value-objects"
)

var ErrOperatorProviderUnavailable = errors.New("audience sync provider is not configured for operator CLI")

type OperatorSummary struct {
	ID               string
	Source           string
	DefinitionSHA256 string
	DefinitionBytes  int
	Version          uint64
	Status           string
}

type Operator struct {
	repository         repositories.AudienceRepository
	service            *Service
	providerConfigured bool
}

type unavailableOperatorProvider struct{}

func (unavailableOperatorProvider) Synchronize(context.Context, aggregates.AudienceDefinition) (SyncResult, error) {
	return SyncResult{}, ErrOperatorProviderUnavailable
}

func NewOperator(repository repositories.AudienceRepository, provider ProviderGateway) (*Operator, error) {
	if repository == nil {
		return nil, errors.New("audience repository is required")
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
	id, err := valueobjects.NewAudienceID(rawID)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.repository.Find(ctx, id)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizeAudience(value), nil
}

func (o *Operator) Define(ctx context.Context, rawID, source, definition string) (OperatorSummary, error) {
	id, criteria, err := audienceInput(rawID, source, definition)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.service.Define(ctx, id, criteria)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizeAudience(value), nil
}

func (o *Operator) Revise(ctx context.Context, rawID, source, definition string) (OperatorSummary, error) {
	id, criteria, err := audienceInput(rawID, source, definition)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.service.Revise(ctx, id, criteria)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizeAudience(value), nil
}

func (o *Operator) Activate(ctx context.Context, rawID string) (OperatorSummary, error) {
	id, err := valueobjects.NewAudienceID(rawID)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.service.Activate(ctx, id)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizeAudience(value), nil
}

func (o *Operator) Archive(ctx context.Context, rawID string) (OperatorSummary, error) {
	id, err := valueobjects.NewAudienceID(rawID)
	if err != nil {
		return OperatorSummary{}, err
	}
	value, err := o.service.Archive(ctx, id)
	if err != nil {
		return OperatorSummary{}, err
	}
	return summarizeAudience(value), nil
}

func (o *Operator) Sync(ctx context.Context, rawID string) (SyncResult, error) {
	if !o.providerConfigured {
		return SyncResult{}, ErrOperatorProviderUnavailable
	}
	id, err := valueobjects.NewAudienceID(rawID)
	if err != nil {
		return SyncResult{}, err
	}
	return o.service.Sync(ctx, id)
}

func audienceInput(rawID, source, definition string) (valueobjects.AudienceID, valueobjects.SelectionCriteria, error) {
	id, err := valueobjects.NewAudienceID(rawID)
	if err != nil {
		return valueobjects.AudienceID(""), valueobjects.SelectionCriteria{}, err
	}
	criteria, err := valueobjects.NewSelectionCriteria(source, definition)
	return id, criteria, err
}

func summarizeAudience(value aggregates.AudienceDefinition) OperatorSummary {
	definition := value.Criteria().Definition()
	return OperatorSummary{
		ID: value.ID().String(), Source: value.Criteria().Source(),
		DefinitionSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(definition))),
		DefinitionBytes:  len([]byte(definition)), Version: uint64(value.Version()),
		Status: audienceStatus(value.Status()),
	}
}

func audienceStatus(status aggregates.Status) string {
	switch status {
	case aggregates.StatusDraft:
		return "draft"
	case aggregates.StatusActive:
		return "active"
	case aggregates.StatusArchived:
		return "archived"
	default:
		return "unknown"
	}
}
