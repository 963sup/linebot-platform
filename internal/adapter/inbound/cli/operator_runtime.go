package cli

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/audience"
	experiencepresentation "github.com/963sup/linebot-platform/internal/application/usecases/experience-presentation"
	"github.com/963sup/linebot-platform/internal/application/usecases/promotion"
	workspaceautomation "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
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

type AutomationOperator interface {
	Get(context.Context, string) (workspaceautomation.OperatorSummary, error)
	Request(context.Context, string, string, []string) (workspaceautomation.OperatorSummary, error)
	Approve(context.Context, string, string) (workspaceautomation.OperatorSummary, error)
	Execute(context.Context, string) (workspaceautomation.OperatorSummary, error)
}

type OperatorRuntime interface {
	AudienceDefinitionsRuntime
	InteractionSurfacesRuntime
	PromotionsRuntime
	AutomationRequestsRuntime
}

// OperatorDomainRuntime is the inbound presentation facade for provider-
// independent operator commands. Bootstrap owns only its concrete wiring.
type OperatorDomainRuntime struct {
	audiences   AudienceOperator
	surfaces    SurfaceOperator
	promotions  PromotionOperator
	automations AutomationOperator
	close       func() error
	closeOnce   sync.Once
	closeErr    error
}

func NewOperatorDomainRuntime(a AudienceOperator, s SurfaceOperator, p PromotionOperator, w AutomationOperator, close func() error) (*OperatorDomainRuntime, error) {
	if a == nil || s == nil || p == nil || w == nil || close == nil {
		return nil, errors.New("operator runtime dependencies are required")
	}
	return &OperatorDomainRuntime{audiences: a, surfaces: s, promotions: p, automations: w, close: close}, nil
}

func (r *OperatorDomainRuntime) Close() error {
	if r == nil {
		return nil
	}
	r.closeOnce.Do(func() { r.closeErr = r.close() })
	return r.closeErr
}

func (r *OperatorDomainRuntime) GetAudienceDefinition(ctx context.Context, id string) (AudienceDefinitionSummary, error) {
	v, e := r.audiences.Get(ctx, id)
	return audienceSummary(v), e
}
func (r *OperatorDomainRuntime) DefineAudience(ctx context.Context, id, source, definition string) (AudienceDefinitionSummary, error) {
	v, e := r.audiences.Define(ctx, id, source, definition)
	return audienceSummary(v), e
}
func (r *OperatorDomainRuntime) ReviseAudience(ctx context.Context, id, source, definition string) (AudienceDefinitionSummary, error) {
	v, e := r.audiences.Revise(ctx, id, source, definition)
	return audienceSummary(v), e
}
func (r *OperatorDomainRuntime) ActivateAudience(ctx context.Context, id string) (AudienceDefinitionSummary, error) {
	v, e := r.audiences.Activate(ctx, id)
	return audienceSummary(v), e
}
func (r *OperatorDomainRuntime) ArchiveAudience(ctx context.Context, id string) (AudienceDefinitionSummary, error) {
	v, e := r.audiences.Archive(ctx, id)
	return audienceSummary(v), e
}
func (r *OperatorDomainRuntime) SyncAudience(ctx context.Context, id string) (AudienceSyncSummary, error) {
	v, e := r.audiences.Sync(ctx, id)
	return AudienceSyncSummary{ProviderReference: v.ProviderReference, SynchronizedCount: v.SynchronizedCount}, e
}

func audienceSummary(v audience.OperatorSummary) AudienceDefinitionSummary {
	return AudienceDefinitionSummary{ID: v.ID, Source: v.Source, DefinitionSHA256: v.DefinitionSHA256, DefinitionBytes: v.DefinitionBytes, Version: v.Version, Status: v.Status}
}

func (r *OperatorDomainRuntime) GetSurface(ctx context.Context, id string) (InteractionSurfaceSummary, error) {
	v, e := r.surfaces.Get(ctx, id)
	return surfaceSummary(v), e
}
func (r *OperatorDomainRuntime) DefineSurface(ctx context.Context, id string, in []InteractionRegion) (InteractionSurfaceSummary, error) {
	v, e := r.surfaces.Define(ctx, id, operatorRegions(in))
	return surfaceSummary(v), e
}
func (r *OperatorDomainRuntime) ReviseSurface(ctx context.Context, id string, in []InteractionRegion) (InteractionSurfaceSummary, error) {
	v, e := r.surfaces.Revise(ctx, id, operatorRegions(in))
	return surfaceSummary(v), e
}
func (r *OperatorDomainRuntime) PublishSurface(ctx context.Context, id string) (InteractionSurfaceSummary, error) {
	v, e := r.surfaces.Publish(ctx, id)
	return surfaceSummary(v), e
}
func (r *OperatorDomainRuntime) RetireSurface(ctx context.Context, id string) (InteractionSurfaceSummary, error) {
	v, e := r.surfaces.Retire(ctx, id)
	return surfaceSummary(v), e
}
func (r *OperatorDomainRuntime) DeploySurface(ctx context.Context, id string) (SurfaceDeploymentSummary, error) {
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

func (r *OperatorDomainRuntime) GetPromotion(ctx context.Context, id string) (PromotionSummary, error) {
	v, e := r.promotions.Get(ctx, id)
	return promotionSummary(v), e
}
func (r *OperatorDomainRuntime) DefinePromotion(ctx context.Context, id, offer, policy string, starts, ends time.Time) (PromotionSummary, error) {
	v, e := r.promotions.Define(ctx, id, offer, policy, starts, ends)
	return promotionSummary(v), e
}
func (r *OperatorDomainRuntime) ActivatePromotion(ctx context.Context, id string, at time.Time) (PromotionSummary, error) {
	v, e := r.promotions.Activate(ctx, id, at)
	return promotionSummary(v), e
}
func (r *OperatorDomainRuntime) ClosePromotion(ctx context.Context, id string) (PromotionSummary, error) {
	v, e := r.promotions.Close(ctx, id)
	return promotionSummary(v), e
}
func (r *OperatorDomainRuntime) CreateDeliveryIntent(ctx context.Context, id, recipient, policy string, eligible bool, at time.Time) (DeliveryIntentSummary, error) {
	v, e := r.promotions.CreateDeliveryIntent(ctx, id, recipient, policy, eligible, at)
	return DeliveryIntentSummary{PromotionID: v.PromotionID, RecipientSHA256: v.RecipientSHA256, OfferSHA256: v.OfferSHA256}, e
}
func promotionSummary(v promotion.OperatorSummary) PromotionSummary {
	return PromotionSummary{ID: v.ID, OfferSHA256: v.OfferSHA256, EligibilityPolicySHA256: v.EligibilityPolicySHA256, StartsAt: v.StartsAt, EndsAt: v.EndsAt, Revision: v.Revision, Status: v.Status}
}

func (r *OperatorDomainRuntime) GetAutomationRequest(ctx context.Context, id string) (AutomationRequestSummary, error) {
	v, e := r.automations.Get(ctx, id)
	return automationSummary(v), e
}
func (r *OperatorDomainRuntime) RequestAutomation(ctx context.Context, id, action string, scopes []string) (AutomationRequestSummary, error) {
	v, e := r.automations.Request(ctx, id, action, scopes)
	return automationSummary(v), e
}
func (r *OperatorDomainRuntime) ApproveAutomation(ctx context.Context, id, by string) (AutomationRequestSummary, error) {
	v, e := r.automations.Approve(ctx, id, by)
	return automationSummary(v), e
}
func (r *OperatorDomainRuntime) ExecuteAutomation(ctx context.Context, id string) (AutomationRequestSummary, error) {
	v, e := r.automations.Execute(ctx, id)
	return automationSummary(v), e
}
func automationSummary(v workspaceautomation.OperatorSummary) AutomationRequestSummary {
	return AutomationRequestSummary{ID: v.ID, Action: v.Action, ScopeCount: v.ScopeCount, Approved: v.Approved, HasOutput: v.HasOutput, Revision: v.Revision, Status: v.Status}
}
