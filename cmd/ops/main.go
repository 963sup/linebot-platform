package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/cli"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/application/usecases/channel"
	"github.com/963sup/linebot-platform/internal/application/usecases/contacts"
	"github.com/963sup/linebot-platform/internal/application/usecases/engagement"
	"github.com/963sup/linebot-platform/internal/application/usecases/insights"
	"github.com/963sup/linebot-platform/internal/application/usecases/messaging"
	"github.com/963sup/linebot-platform/internal/bootstrap"
	"golang.org/x/oauth2"
)

func main() {
	deps := cli.Dependencies{
		PushRuntime: func(token string) (cli.PushRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return pushRuntime{runtime}, nil
		},
		ProfileRuntime: func(token string) (cli.ProfileRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return profileRuntime{runtime}, nil
		},
		ValidationRuntime: func(token string) (cli.ValidationRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return validationRuntime{runtime}, nil
		},
		StatusRuntime: func(ctx context.Context, project string) (cli.StatusRuntime, error) {
			runtime, err := bootstrap.NewStatusRuntime(ctx, project)
			if err != nil {
				return nil, err
			}
			return statusRuntime{runtime}, nil
		},
		WorkspaceRuntime: func(ctx context.Context, source oauth2.TokenSource) (cli.WorkspaceRuntime, error) {
			return newWorkspaceRuntime(ctx, source)
		},
		LinkTokenRuntime: func(token string) (cli.LinkTokenRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return linkTokenRuntime{runtime}, nil
		},
		ActionsRuntime: func(token string) (cli.ActionsRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return actionsRuntime{runtime}, nil
		},
		EligibilityRuntime: func(token string) (cli.EligibilityRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return eligibilityRuntime{runtime}, nil
		},
		AudienceRuntime: func(token string) (cli.AudienceRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return audienceRuntime{runtime}, nil
		},
		ContactsRuntime: func(token string) (cli.ContactsRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return contactsRuntime{runtime}, nil
		},
		CouponsRuntime: func(token string) (cli.CouponsRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return couponsRuntime{runtime}, nil
		},
		InsightsRuntime: func(token string) (cli.InsightsRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return insightsRuntime{runtime}, nil
		},
		MembershipsRuntime: func(token string) (cli.MembershipsRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return membershipsRuntime{runtime}, nil
		},
		MenusRuntime: func(token string) (cli.MenusRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return menusRuntime{runtime}, nil
		},
		ChannelInspectionRuntime: func(token string) (cli.ChannelInspectionRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return channelInspectionRuntime{runtime}, nil
		},
		ContentRuntime: func(token string) (cli.ContentRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return contentRuntime{runtime}, nil
		},
		MenuImageRuntime: func(token string) (cli.MenuImageRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return menuImageRuntime{runtime}, nil
		},
		MenuManagementRuntime: func(token string) (cli.MenuManagementRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return menuManagementRuntime{runtime}, nil
		},
		MenuImageUploadRuntime: func(token string) (cli.MenuImageUploadRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return menuImageUploadRuntime{runtime}, nil
		},
		WebhookRuntime: func(token string) (cli.WebhookRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return webhookRuntime{runtime}, nil
		},
		TokenRuntime: func() (cli.TokenRuntime, error) {
			runtime, err := bootstrap.NewLINETokenOpsRuntime()
			if err != nil {
				return nil, err
			}
			return tokenRuntime{runtime}, nil
		},
		AudienceManagementRuntime: func(token string) (cli.AudienceManagementRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return audienceManagementRuntime{runtime}, nil
		},
		BulkMessagingRuntime: func(token string) (cli.BulkMessagingRuntime, error) {
			runtime, err := bootstrap.NewLINEOpsRuntime(token)
			if err != nil {
				return nil, err
			}
			return bulkMessagingRuntime{runtime}, nil
		},
	}
	cli.ConfigureDependencies(deps)
	if err := cli.Run(os.Args[1:], os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, "ops:", err)
		os.Exit(1)
	}
}

type pushRuntime struct {
	*bootstrap.LINEOpsRuntime
}

type webhookRuntime struct{ *bootstrap.LINEOpsRuntime }

func (r webhookRuntime) Set(ctx context.Context, endpoint string) error {
	return r.WebhookAdmin.Set(ctx, endpoint)
}

func (r webhookRuntime) Test(ctx context.Context, endpoint string) (outbound.WebhookEndpointTest, error) {
	return r.WebhookAdmin.Test(ctx, endpoint)
}

type tokenRuntime struct{ *bootstrap.LINETokenOpsRuntime }

func (r tokenRuntime) Execute(ctx context.Context, cmd channel.TokenCommand) (outbound.LINEResult, error) {
	return r.Tokens.Execute(ctx, cmd)
}

type audienceManagementRuntime struct{ *bootstrap.LINEOpsRuntime }

func (r audienceManagementRuntime) Execute(ctx context.Context, cmd engagement.AudienceCommand) (json.RawMessage, error) {
	result, err := r.AudienceManager.Execute(ctx, cmd)
	return marshalLINEResult(result, err)
}

type bulkMessagingRuntime struct{ *bootstrap.LINEOpsRuntime }

func (r bulkMessagingRuntime) Broadcast(ctx context.Context, messages []outbound.Message, retryKey string) error {
	return r.BulkMessaging.Broadcast(ctx, messages, retryKey)
}

func (r bulkMessagingRuntime) Multicast(ctx context.Context, users []string, messages []outbound.Message, retryKey string) error {
	return r.BulkMessaging.Multicast(ctx, users, messages, retryKey)
}

func (r bulkMessagingRuntime) Narrowcast(ctx context.Context, payload outbound.LINERequest, retryKey string) error {
	return r.BulkMessaging.Narrowcast(ctx, payload, retryKey)
}

func (r pushRuntime) Push(ctx context.Context, recipient string, messages []outbound.Message, retryKey string) error {
	return r.Messaging.Push(ctx, recipient, messages, retryKey)
}

type profileRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r profileRuntime) Get(ctx context.Context, userID string) (outbound.UserProfile, error) {
	return r.Profiles.Get(ctx, userID)
}

type validationRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r validationRuntime) Validate(ctx context.Context, kind string, messages []outbound.Message) error {
	return r.Validation.Validate(ctx, kind, messages)
}

type statusRuntime struct {
	*bootstrap.StatusRuntime
}

type linkTokenRuntime struct {
	*bootstrap.LINEOpsRuntime
}

type actionsRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r actionsRuntime) Execute(ctx context.Context, cmd messaging.ConversationActionCommand) error {
	return r.Actions.Execute(ctx, cmd)
}

type eligibilityRuntime struct {
	*bootstrap.LINEOpsRuntime
}

type audienceRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r audienceRuntime) Inspect(ctx context.Context, query engagement.AudienceQuery) (json.RawMessage, error) {
	result, err := r.Audiences.Inspect(ctx, query)
	return marshalLINEResult(result, err)
}

type contactsRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r contactsRuntime) Inspect(ctx context.Context, query contacts.Inspection) (outbound.LINEResult, error) {
	return r.Contacts.Inspect(ctx, query)
}

type couponsRuntime struct {
	*bootstrap.LINEOpsRuntime
}

type insightsRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r insightsRuntime) Inspect(ctx context.Context, query insights.Query) (outbound.LINEResult, error) {
	return r.Insights.Inspect(ctx, query)
}

type membershipsRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r membershipsRuntime) Inspect(ctx context.Context, query engagement.MembershipQuery) (json.RawMessage, error) {
	result, err := r.Memberships.Inspect(ctx, query)
	return marshalLINEResult(result, err)
}

type menusRuntime struct {
	*bootstrap.LINEOpsRuntime
}

type channelInspectionRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r channelInspectionRuntime) Inspect(ctx context.Context, query channel.Inspection) (outbound.LINEResult, error) {
	return r.Channel.Inspect(ctx, query)
}

type contentRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r contentRuntime) Inspect(ctx context.Context, query contacts.ContentQuery) (outbound.BinaryContent, outbound.LINEResult, error) {
	return r.Content.Inspect(ctx, query)
}

type menuImageRuntime struct {
	*bootstrap.LINEOpsRuntime
}

type menuManagementRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r menuManagementRuntime) Execute(ctx context.Context, cmd engagement.MenuCommand) error {
	return r.MenuManager.Execute(ctx, cmd)
}

type menuImageUploadRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r menuImageUploadRuntime) Upload(ctx context.Context, id, contentType string, data []byte) error {
	return r.RichMenuImages.Upload(ctx, id, contentType, data)
}

func (r menuImageRuntime) RichMenuImage(ctx context.Context, id string) (outbound.BinaryContent, error) {
	return r.MenuImageReader.RichMenuImage(ctx, id)
}

func (r menusRuntime) Inspect(ctx context.Context, query engagement.RichMenuQuery) (json.RawMessage, error) {
	result, err := r.Menus.Inspect(ctx, query)
	return marshalLINEResult(result, err)
}

func (r couponsRuntime) Inspect(ctx context.Context, query engagement.CouponQuery) (json.RawMessage, error) {
	result, err := r.Coupons.Inspect(ctx, query)
	return marshalLINEResult(result, err)
}

func (r eligibilityRuntime) Execute(ctx context.Context, cmd engagement.EligibilityCommand) (json.RawMessage, error) {
	result, err := r.Eligibility.Execute(ctx, cmd)
	return marshalLINEResult(result, err)
}

func marshalLINEResult(result outbound.LINEResult, err error) (json.RawMessage, error) {
	if err != nil {
		return nil, err
	}
	if result.Value == nil {
		return nil, nil
	}
	payload, err := json.Marshal(result.Value)
	if err != nil {
		return nil, fmt.Errorf("encode LINE result: %w", err)
	}
	return payload, nil
}

func (r linkTokenRuntime) IssueLinkToken(ctx context.Context, userID string) (outbound.LINEResult, error) {
	return r.AccountLink.IssueLinkToken(ctx, userID)
}

func (r statusRuntime) Inspect(ctx context.Context, state outbound.EventState, limit int, now time.Time) (outbound.EventStatusReport, error) {
	return r.Reader.Inspect(ctx, state, limit, now)
}
