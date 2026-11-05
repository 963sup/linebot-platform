package bootstrap

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/cli"
	storeout "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	"github.com/963sup/linebot-platform/internal/application/usecases/audience"
	experiencepresentation "github.com/963sup/linebot-platform/internal/application/usecases/experience-presentation"
	"github.com/963sup/linebot-platform/internal/application/usecases/promotion"
	workspaceautomation "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
	storeinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/firestore"
)

// OperatorDomainRuntime owns the one Firestore client opened for a single
// operator command. Provider-dependent actions remain deliberately
// unconfigured until an explicit credentialed composition is supplied.
type OperatorDomainRuntime struct {
	audiences   *audience.Operator
	surfaces    *experiencepresentation.Operator
	promotions  *promotion.Operator
	automations *workspaceautomation.Operator
	close       func() error
}

func NewOperatorDomainRuntime(ctx context.Context, project string) (*OperatorDomainRuntime, error) {
	if ctx == nil || strings.TrimSpace(project) == "" {
		return nil, errors.New("operator domain runtime requires context and staging project")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	client, err := storeinfra.NewClient(ctx, project)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*OperatorDomainRuntime, error) {
		_ = client.Close()
		return nil, err
	}

	audienceRepository, err := storeout.NewAudienceRepository(client)
	if err != nil {
		return fail(err)
	}
	surfaceRepository, err := storeout.NewInteractionSurfaceRepository(client)
	if err != nil {
		return fail(err)
	}
	promotionRepository, err := storeout.NewPromotionRepository(client)
	if err != nil {
		return fail(err)
	}
	automationRepository, err := storeout.NewAutomationRequestRepository(client)
	if err != nil {
		return fail(err)
	}

	audiences, err := audience.NewOperator(audienceRepository, nil)
	if err != nil {
		return fail(err)
	}
	surfaces, err := experiencepresentation.NewOperator(surfaceRepository, nil)
	if err != nil {
		return fail(err)
	}
	promotions, err := promotion.NewOperator(promotionRepository)
	if err != nil {
		return fail(err)
	}
	automations, err := workspaceautomation.NewOperator(automationRepository, nil)
	if err != nil {
		return fail(err)
	}
	return &OperatorDomainRuntime{
		audiences: audiences, surfaces: surfaces, promotions: promotions,
		automations: automations, close: client.Close,
	}, nil
}

func (r *OperatorDomainRuntime) Close() error {
	if r == nil || r.close == nil {
		return nil
	}
	return r.close()
}

func (r *OperatorDomainRuntime) GetAudienceDefinition(ctx context.Context, id string) (cli.AudienceDefinitionSummary, error) {
	value, err := r.audiences.Get(ctx, id)
	return audienceSummary(value), err
}

func (r *OperatorDomainRuntime) DefineAudience(ctx context.Context, id, source, definition string) (cli.AudienceDefinitionSummary, error) {
	value, err := r.audiences.Define(ctx, id, source, definition)
	return audienceSummary(value), err
}

func (r *OperatorDomainRuntime) ReviseAudience(ctx context.Context, id, source, definition string) (cli.AudienceDefinitionSummary, error) {
	value, err := r.audiences.Revise(ctx, id, source, definition)
	return audienceSummary(value), err
}

func (r *OperatorDomainRuntime) ActivateAudience(ctx context.Context, id string) (cli.AudienceDefinitionSummary, error) {
	value, err := r.audiences.Activate(ctx, id)
	return audienceSummary(value), err
}

func (r *OperatorDomainRuntime) ArchiveAudience(ctx context.Context, id string) (cli.AudienceDefinitionSummary, error) {
	value, err := r.audiences.Archive(ctx, id)
	return audienceSummary(value), err
}

func (r *OperatorDomainRuntime) SyncAudience(ctx context.Context, id string) (cli.AudienceSyncSummary, error) {
	value, err := r.audiences.Sync(ctx, id)
	return cli.AudienceSyncSummary{ProviderReference: value.ProviderReference, SynchronizedCount: value.SynchronizedCount}, err
}

func audienceSummary(value audience.OperatorSummary) cli.AudienceDefinitionSummary {
	return cli.AudienceDefinitionSummary{
		ID: value.ID, Source: value.Source, DefinitionSHA256: value.DefinitionSHA256,
		DefinitionBytes: value.DefinitionBytes, Version: value.Version, Status: value.Status,
	}
}

func (r *OperatorDomainRuntime) GetSurface(ctx context.Context, id string) (cli.InteractionSurfaceSummary, error) {
	value, err := r.surfaces.Get(ctx, id)
	return surfaceSummary(value), err
}

func (r *OperatorDomainRuntime) DefineSurface(ctx context.Context, id string, input []cli.InteractionRegion) (cli.InteractionSurfaceSummary, error) {
	value, err := r.surfaces.Define(ctx, id, operatorRegions(input))
	return surfaceSummary(value), err
}

func (r *OperatorDomainRuntime) ReviseSurface(ctx context.Context, id string, input []cli.InteractionRegion) (cli.InteractionSurfaceSummary, error) {
	value, err := r.surfaces.Revise(ctx, id, operatorRegions(input))
	return surfaceSummary(value), err
}

func (r *OperatorDomainRuntime) PublishSurface(ctx context.Context, id string) (cli.InteractionSurfaceSummary, error) {
	value, err := r.surfaces.Publish(ctx, id)
	return surfaceSummary(value), err
}

func (r *OperatorDomainRuntime) RetireSurface(ctx context.Context, id string) (cli.InteractionSurfaceSummary, error) {
	value, err := r.surfaces.Retire(ctx, id)
	return surfaceSummary(value), err
}

func (r *OperatorDomainRuntime) DeploySurface(ctx context.Context, id string) (cli.SurfaceDeploymentSummary, error) {
	value, err := r.surfaces.Deploy(ctx, id)
	return cli.SurfaceDeploymentSummary{ProviderReference: value.ProviderReference, Version: uint64(value.Version)}, err
}

func operatorRegions(input []cli.InteractionRegion) []experiencepresentation.OperatorRegion {
	regions := make([]experiencepresentation.OperatorRegion, 0, len(input))
	for _, item := range input {
		regions = append(regions, experiencepresentation.OperatorRegion{
			ID: item.ID, ActionReference: item.ActionReference, AssetReference: item.AssetReference,
		})
	}
	return regions
}

func surfaceSummary(value experiencepresentation.OperatorSummary) cli.InteractionSurfaceSummary {
	return cli.InteractionSurfaceSummary{ID: value.ID, RegionCount: value.RegionCount, Version: value.Version, Status: value.Status}
}

func (r *OperatorDomainRuntime) GetPromotion(ctx context.Context, id string) (cli.PromotionSummary, error) {
	value, err := r.promotions.Get(ctx, id)
	return promotionSummary(value), err
}

func (r *OperatorDomainRuntime) DefinePromotion(ctx context.Context, id, offer, policy string, startsAt, endsAt time.Time) (cli.PromotionSummary, error) {
	value, err := r.promotions.Define(ctx, id, offer, policy, startsAt, endsAt)
	return promotionSummary(value), err
}

func (r *OperatorDomainRuntime) ActivatePromotion(ctx context.Context, id string, at time.Time) (cli.PromotionSummary, error) {
	value, err := r.promotions.Activate(ctx, id, at)
	return promotionSummary(value), err
}

func (r *OperatorDomainRuntime) ClosePromotion(ctx context.Context, id string) (cli.PromotionSummary, error) {
	value, err := r.promotions.Close(ctx, id)
	return promotionSummary(value), err
}

func (r *OperatorDomainRuntime) CreateDeliveryIntent(ctx context.Context, id, recipient, policy string, eligible bool, at time.Time) (cli.DeliveryIntentSummary, error) {
	value, err := r.promotions.CreateDeliveryIntent(ctx, id, recipient, policy, eligible, at)
	return cli.DeliveryIntentSummary{
		PromotionID: value.PromotionID, RecipientSHA256: value.RecipientSHA256, OfferSHA256: value.OfferSHA256,
	}, err
}

func promotionSummary(value promotion.OperatorSummary) cli.PromotionSummary {
	return cli.PromotionSummary{
		ID: value.ID, OfferSHA256: value.OfferSHA256, EligibilityPolicySHA256: value.EligibilityPolicySHA256,
		StartsAt: value.StartsAt, EndsAt: value.EndsAt, Revision: value.Revision, Status: value.Status,
	}
}

func (r *OperatorDomainRuntime) GetAutomationRequest(ctx context.Context, id string) (cli.AutomationRequestSummary, error) {
	value, err := r.automations.Get(ctx, id)
	return automationSummary(value), err
}

func (r *OperatorDomainRuntime) RequestAutomation(ctx context.Context, id, action string, scopes []string) (cli.AutomationRequestSummary, error) {
	value, err := r.automations.Request(ctx, id, action, scopes)
	return automationSummary(value), err
}

func (r *OperatorDomainRuntime) ApproveAutomation(ctx context.Context, id, approvedBy string) (cli.AutomationRequestSummary, error) {
	value, err := r.automations.Approve(ctx, id, approvedBy)
	return automationSummary(value), err
}

func (r *OperatorDomainRuntime) ExecuteAutomation(ctx context.Context, id string) (cli.AutomationRequestSummary, error) {
	value, err := r.automations.Execute(ctx, id)
	return automationSummary(value), err
}

func automationSummary(value workspaceautomation.OperatorSummary) cli.AutomationRequestSummary {
	return cli.AutomationRequestSummary{
		ID: value.ID, Action: value.Action, ScopeCount: value.ScopeCount,
		Approved: value.Approved, HasOutput: value.HasOutput, Revision: value.Revision, Status: value.Status,
	}
}
