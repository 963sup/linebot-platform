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
	storeinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/firestore"
)

func newAudienceDomainRuntime(ctx context.Context, project string) (cli.AudienceDefinitionsRuntime, error) {
	if ctx == nil || strings.TrimSpace(project) == "" {
		return nil, errors.New("operator runtime requires context and project")
	}
	client, err := storeinfra.NewClient(ctx, project)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (cli.AudienceDefinitionsRuntime, error) { return nil, errors.Join(err, client.Close()) }
	repo, err := storeout.NewAudienceRepository(client)
	if err != nil {
		return fail(err)
	}
	op, err := audience.NewOperator(repo, nil)
	if err != nil {
		return fail(err)
	}
	runtime, err := cli.NewAudienceDomainRuntime(op, client.Close)
	if err != nil {
		return fail(err)
	}
	return runtime, nil
}
func newSurfaceDomainRuntime(ctx context.Context, project string) (cli.InteractionSurfacesRuntime, error) {
	if ctx == nil || strings.TrimSpace(project) == "" {
		return nil, errors.New("operator runtime requires context and project")
	}
	client, err := storeinfra.NewClient(ctx, project)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (cli.InteractionSurfacesRuntime, error) { return nil, errors.Join(err, client.Close()) }
	repo, err := storeout.NewInteractionSurfaceRepository(client)
	if err != nil {
		return fail(err)
	}
	op, err := experiencepresentation.NewOperator(repo, nil)
	if err != nil {
		return fail(err)
	}
	runtime, err := cli.NewSurfaceDomainRuntime(op, client.Close)
	if err != nil {
		return fail(err)
	}
	return runtime, nil
}
func newPromotionDomainRuntime(ctx context.Context, project string) (cli.PromotionsRuntime, error) {
	if ctx == nil || strings.TrimSpace(project) == "" {
		return nil, errors.New("operator runtime requires context and project")
	}
	client, err := storeinfra.NewClient(ctx, project)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (cli.PromotionsRuntime, error) { return nil, errors.Join(err, client.Close()) }
	repo, err := storeout.NewPromotionRepository(client)
	if err != nil {
		return fail(err)
	}
	op, err := promotion.NewOperator(repo)
	if err != nil {
		return fail(err)
	}
	runtime, err := cli.NewPromotionDomainRuntime(op, client.Close)
	if err != nil {
		return fail(err)
	}
	return runtime, nil
}
