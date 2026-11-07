package cli

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/audience"
	experiencepresentation "github.com/963sup/linebot-platform/internal/application/usecases/experience-presentation"
	"github.com/963sup/linebot-platform/internal/application/usecases/promotion"
)

type AudienceOperator interface {
	Get(context.Context, string) (audience.OperatorSummary, error)
	Define(context.Context, string, string, string) (audience.OperatorSummary, error)
	Revise(context.Context, string, string, string) (audience.OperatorSummary, error)
	Activate(context.Context, string) (audience.OperatorSummary, error)
	Archive(context.Context, string) (audience.OperatorSummary, error)
	Sync(context.Context, string) (audience.SyncResult, error)
}

type SurfaceOperator interface {
	Get(context.Context, string) (experiencepresentation.OperatorSummary, error)
	Define(context.Context, string, []experiencepresentation.OperatorRegion) (experiencepresentation.OperatorSummary, error)
	Revise(context.Context, string, []experiencepresentation.OperatorRegion) (experiencepresentation.OperatorSummary, error)
	Publish(context.Context, string) (experiencepresentation.OperatorSummary, error)
	Retire(context.Context, string) (experiencepresentation.OperatorSummary, error)
	Deploy(context.Context, string) (experiencepresentation.Deployment, error)
}

type PromotionOperator interface {
	Get(context.Context, string) (promotion.OperatorSummary, error)
	Define(context.Context, string, string, string, time.Time, time.Time) (promotion.OperatorSummary, error)
	Activate(context.Context, string, time.Time) (promotion.OperatorSummary, error)
	Close(context.Context, string) (promotion.OperatorSummary, error)
	CreateDeliveryIntent(context.Context, string, string, string, bool, time.Time) (promotion.DeliveryIntentSummary, error)
}

type operatorCloser struct {
	close func() error
	once  sync.Once
	err   error
}

func (r *operatorCloser) Close() error { r.once.Do(func() { r.err = r.close() }); return r.err }

type AudienceDomainRuntime struct {
	operatorCloser
	audiences AudienceOperator
}

func NewAudienceDomainRuntime(op AudienceOperator, close func() error) (*AudienceDomainRuntime, error) {
	if op == nil || close == nil {
		return nil, errors.New("operator and close are required")
	}
	return &AudienceDomainRuntime{operatorCloser: operatorCloser{close: close}, audiences: op}, nil
}

type SurfaceDomainRuntime struct {
	operatorCloser
	surfaces SurfaceOperator
}

func NewSurfaceDomainRuntime(op SurfaceOperator, close func() error) (*SurfaceDomainRuntime, error) {
	if op == nil || close == nil {
		return nil, errors.New("operator and close are required")
	}
	return &SurfaceDomainRuntime{operatorCloser: operatorCloser{close: close}, surfaces: op}, nil
}

type PromotionDomainRuntime struct {
	operatorCloser
	promotions PromotionOperator
}

func NewPromotionDomainRuntime(op PromotionOperator, close func() error) (*PromotionDomainRuntime, error) {
	if op == nil || close == nil {
		return nil, errors.New("operator and close are required")
	}
	return &PromotionDomainRuntime{operatorCloser: operatorCloser{close: close}, promotions: op}, nil
}
func (r *AudienceDomainRuntime) GetAudienceDefinition(ctx context.Context, id string) (AudienceDefinitionSummary, error) {
	v, e := r.audiences.Get(ctx, id)
	return audienceSummary(v), e
}
func (r *AudienceDomainRuntime) DefineAudience(ctx context.Context, id, source, definition string) (AudienceDefinitionSummary, error) {
	v, e := r.audiences.Define(ctx, id, source, definition)
	return audienceSummary(v), e
}
func (r *AudienceDomainRuntime) ReviseAudience(ctx context.Context, id, source, definition string) (AudienceDefinitionSummary, error) {
	v, e := r.audiences.Revise(ctx, id, source, definition)
	return audienceSummary(v), e
}
func (r *AudienceDomainRuntime) ActivateAudience(ctx context.Context, id string) (AudienceDefinitionSummary, error) {
	v, e := r.audiences.Activate(ctx, id)
	return audienceSummary(v), e
}
func (r *AudienceDomainRuntime) ArchiveAudience(ctx context.Context, id string) (AudienceDefinitionSummary, error) {
	v, e := r.audiences.Archive(ctx, id)
	return audienceSummary(v), e
}
func (r *AudienceDomainRuntime) SyncAudience(ctx context.Context, id string) (AudienceSyncSummary, error) {
	v, e := r.audiences.Sync(ctx, id)
	return AudienceSyncSummary{ProviderReference: v.ProviderReference, SynchronizedCount: v.SynchronizedCount}, e
}

func audienceSummary(v audience.OperatorSummary) AudienceDefinitionSummary {
	return AudienceDefinitionSummary{ID: v.ID, Source: v.Source, DefinitionSHA256: v.DefinitionSHA256, DefinitionBytes: v.DefinitionBytes, Version: v.Version, Status: v.Status}
}

func (r *SurfaceDomainRuntime) GetSurface(ctx context.Context, id string) (InteractionSurfaceSummary, error) {
	v, e := r.surfaces.Get(ctx, id)
	return surfaceSummary(v), e
}
func (r *SurfaceDomainRuntime) DefineSurface(ctx context.Context, id string, in []InteractionRegion) (InteractionSurfaceSummary, error) {
	v, e := r.surfaces.Define(ctx, id, operatorRegions(in))
	return surfaceSummary(v), e
}
func (r *SurfaceDomainRuntime) ReviseSurface(ctx context.Context, id string, in []InteractionRegion) (InteractionSurfaceSummary, error) {
	v, e := r.surfaces.Revise(ctx, id, operatorRegions(in))
	return surfaceSummary(v), e
}
func (r *SurfaceDomainRuntime) PublishSurface(ctx context.Context, id string) (InteractionSurfaceSummary, error) {
	v, e := r.surfaces.Publish(ctx, id)
	return surfaceSummary(v), e
}
func (r *SurfaceDomainRuntime) RetireSurface(ctx context.Context, id string) (InteractionSurfaceSummary, error) {
	v, e := r.surfaces.Retire(ctx, id)
	return surfaceSummary(v), e
}
func (r *SurfaceDomainRuntime) DeploySurface(ctx context.Context, id string) (SurfaceDeploymentSummary, error) {
	v, e := r.surfaces.Deploy(ctx, id)
	return SurfaceDeploymentSummary{ProviderReference: v.ProviderReference, Version: uint64(v.Version)}, e
}

func operatorRegions(in []InteractionRegion) []experiencepresentation.OperatorRegion {
	out := make([]experiencepresentation.OperatorRegion, 0, len(in))
	for _, v := range in {
		out = append(out, experiencepresentation.OperatorRegion{ID: v.ID, ActionReference: v.ActionReference, AssetReference: v.AssetReference})
	}
	return out
}
func surfaceSummary(v experiencepresentation.OperatorSummary) InteractionSurfaceSummary {
	return InteractionSurfaceSummary{ID: v.ID, RegionCount: v.RegionCount, Version: v.Version, Status: v.Status}
}

func (r *PromotionDomainRuntime) GetPromotion(ctx context.Context, id string) (PromotionSummary, error) {
	v, e := r.promotions.Get(ctx, id)
	return promotionSummary(v), e
}
func (r *PromotionDomainRuntime) DefinePromotion(ctx context.Context, id, offer, policy string, starts, ends time.Time) (PromotionSummary, error) {
	v, e := r.promotions.Define(ctx, id, offer, policy, starts, ends)
	return promotionSummary(v), e
}
func (r *PromotionDomainRuntime) ActivatePromotion(ctx context.Context, id string, at time.Time) (PromotionSummary, error) {
	v, e := r.promotions.Activate(ctx, id, at)
	return promotionSummary(v), e
}
func (r *PromotionDomainRuntime) ClosePromotion(ctx context.Context, id string) (PromotionSummary, error) {
	v, e := r.promotions.Close(ctx, id)
	return promotionSummary(v), e
}
func (r *PromotionDomainRuntime) CreateDeliveryIntent(ctx context.Context, id, recipient, policy string, eligible bool, at time.Time) (DeliveryIntentSummary, error) {
	v, e := r.promotions.CreateDeliveryIntent(ctx, id, recipient, policy, eligible, at)
	return DeliveryIntentSummary{PromotionID: v.PromotionID, RecipientSHA256: v.RecipientSHA256, OfferSHA256: v.OfferSHA256}, e
}
func promotionSummary(v promotion.OperatorSummary) PromotionSummary {
	return PromotionSummary{ID: v.ID, OfferSHA256: v.OfferSHA256, EligibilityPolicySHA256: v.EligibilityPolicySHA256, StartsAt: v.StartsAt, EndsAt: v.EndsAt, Revision: v.Revision, Status: v.Status}
}
