package bootstrap

import (
	"context"
	"errors"
	"strings"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/cli"
	storeout "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	"github.com/963sup/linebot-platform/internal/application/usecases/audience"
	experiencepresentation "github.com/963sup/linebot-platform/internal/application/usecases/experience-presentation"
	"github.com/963sup/linebot-platform/internal/application/usecases/promotion"
	workspaceautomation "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
	storeinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/firestore"
)

func newOperatorDomainRuntime(ctx context.Context, project string) (cli.OperatorRuntime, error) {
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
	fail := func(err error) (cli.OperatorRuntime, error) { _ = client.Close(); return nil, err }
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
	runtime, err := cli.NewOperatorDomainRuntime(audiences, surfaces, promotions, automations, client.Close)
	if err != nil {
		return fail(err)
	}
	return runtime, nil
}
