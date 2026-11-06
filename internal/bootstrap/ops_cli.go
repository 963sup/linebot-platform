package bootstrap

import (
	"context"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/cli"
	"golang.org/x/oauth2"
)

func newWorkspaceRuntime(ctx context.Context, source oauth2.TokenSource) (cli.WorkspaceRuntime, error) {
	operations, err := NewWorkspaceOperations(ctx, source)
	if err != nil {
		return nil, err
	}
	return cli.NewWorkspaceRuntime(operations.Execute), nil
}

func lineRuntimeAdapters(runtime *LINEOpsRuntime) cli.RuntimeAdapters {
	return cli.RuntimeAdapters{
		Messaging: runtime.Messaging, BulkMessaging: runtime.BulkMessaging,
		PhoneDelivery: runtime.PhoneDelivery, PhoneReports: runtime.PhoneReports,
		Validation: runtime.Validation, Profiles: runtime.Profiles, Contacts: runtime.Contacts,
		WebhookAdmin: runtime.WebhookAdmin, Channel: runtime.Channel, DeliveryReports: runtime.DeliveryReports,
		Audiences: runtime.Audiences, AudienceManager: runtime.AudienceManager,
		Coupons: runtime.Coupons, CouponCreator: runtime.CouponCreator, Memberships: runtime.Memberships,
		Menus: runtime.Menus, MenuCommands: runtime.MenuCommands, Insights: runtime.Insights,
		Content: runtime.Content, RichMenuImages: runtime.RichMenuImages, MenuImageReader: runtime.MenuImageReader,
		Actions: runtime.Actions, AccountLink: runtime.AccountLink, Close: runtime.Close,
	}
}

type lineRuntimeFactories struct {
	messaging func(string) (*LINEOpsRuntime, error)
	insight   func(string) (*LINEOpsRuntime, error)
	blob      func(string) (*LINEOpsRuntime, error)
	audience  func(string) (*LINEOpsRuntime, error)
}

func defaultLineRuntimeFactories() lineRuntimeFactories {
	return lineRuntimeFactories{
		messaging: newLINEMessagingOpsRuntime,
		insight:   newLINEInsightOpsRuntime,
		blob:      newLINEBlobOpsRuntime,
		audience:  newLINEAudienceOpsRuntime,
	}
}

// BuildOpsCLI assembles the operator CLI's consumer-owned runtime factories.
// Runtime creation stays lazy so commands that do not need an external
// provider (for example help) remain side-effect free.
func BuildOpsCLI() cli.Dependencies {
	return buildOpsCLI(defaultLineRuntimeFactories())
}

func buildOpsCLI(lineRuntimes lineRuntimeFactories) cli.Dependencies {
	return cli.Dependencies{
		Attendance: newAttendanceOpsRuntime,
		AudienceDefinitions: func(ctx context.Context, project string) (cli.AudienceDefinitionsRuntime, error) {
			return newOperatorDomainRuntime(ctx, project)
		},
		InteractionSurfaces: func(ctx context.Context, project string) (cli.InteractionSurfacesRuntime, error) {
			return newOperatorDomainRuntime(ctx, project)
		},
		Promotions: func(ctx context.Context, project string) (cli.PromotionsRuntime, error) {
			return newOperatorDomainRuntime(ctx, project)
		},
		AutomationRequests: func(ctx context.Context, project string) (cli.AutomationRequestsRuntime, error) {
			return newOperatorDomainRuntime(ctx, project)
		},
		PushRuntime: func(token string) (cli.PushRuntime, error) {
			runtime, err := lineRuntimes.messaging(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).Push(), nil
		},
		ProfileRuntime: func(token string) (cli.ProfileRuntime, error) {
			runtime, err := lineRuntimes.messaging(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).Profile(), nil
		},
		ValidationRuntime: func(token string) (cli.ValidationRuntime, error) {
			runtime, err := lineRuntimes.messaging(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).ValidationRuntime(), nil
		},
		StatusRuntime: func(ctx context.Context, project string) (cli.StatusRuntime, error) {
			runtime, err := NewStatusRuntime(ctx, project)
			if err != nil {
				return nil, err
			}
			return cli.NewStatusRuntime(runtime.Reader, runtime.Close), nil
		},
		WorkspaceRuntime: func(ctx context.Context, source oauth2.TokenSource) (cli.WorkspaceRuntime, error) {
			return newWorkspaceRuntime(ctx, source)
		},
		LinkTokenRuntime: func(token string) (cli.LinkTokenRuntime, error) {
			runtime, err := lineRuntimes.messaging(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).LinkToken(), nil
		},
		ActionsRuntime: func(token string) (cli.ActionsRuntime, error) {
			runtime, err := lineRuntimes.messaging(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).ActionsRuntime(), nil
		},
		EligibilityRuntime: func(token string) (cli.EligibilityRuntime, error) {
			runtime, err := lineRuntimes.messaging(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).Eligibility(), nil
		},
		AudienceRuntime: func(token string) (cli.AudienceRuntime, error) {
			runtime, err := lineRuntimes.audience(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).AudienceRuntime(), nil
		},
		ContactsRuntime: func(token string) (cli.ContactsRuntime, error) {
			runtime, err := lineRuntimes.messaging(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).ContactsRuntime(), nil
		},
		CouponsRuntime: func(token string) (cli.CouponsRuntime, error) {
			runtime, err := lineRuntimes.messaging(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).CouponsRuntime(), nil
		},
		InsightsRuntime: func(token string) (cli.InsightsRuntime, error) {
			runtime, err := lineRuntimes.insight(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).InsightsRuntime(), nil
		},
		MembershipsRuntime: func(token string) (cli.MembershipsRuntime, error) {
			runtime, err := lineRuntimes.messaging(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).MembershipsRuntime(), nil
		},
		MenusRuntime: func(token string) (cli.MenusRuntime, error) {
			runtime, err := lineRuntimes.messaging(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).MenusRuntime(), nil
		},
		ChannelInspectionRuntime: func(token string) (cli.ChannelInspectionRuntime, error) {
			runtime, err := lineRuntimes.messaging(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).ChannelInspection(), nil
		},
		ContentRuntime: func(token string) (cli.ContentRuntime, error) {
			runtime, err := lineRuntimes.blob(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).ContentRuntime(), nil
		},
		MenuImageRuntime: func(token string) (cli.MenuImageRuntime, error) {
			runtime, err := lineRuntimes.blob(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).MenuImageRuntime(), nil
		},
		MenuManagementRuntime: func(token string) (cli.MenuManagementRuntime, error) {
			runtime, err := lineRuntimes.messaging(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).MenuManagement(), nil
		},
		MenuImageUploadRuntime: func(token string) (cli.MenuImageUploadRuntime, error) {
			runtime, err := lineRuntimes.blob(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).MenuImageUpload(), nil
		},
		WebhookRuntime: func(token string) (cli.WebhookRuntime, error) {
			runtime, err := lineRuntimes.messaging(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).Webhook(), nil
		},
		TokenRuntime: func() (cli.TokenRuntime, error) {
			runtime, err := NewLINETokenOpsRuntime()
			if err != nil {
				return nil, err
			}
			return cli.NewTokenRuntime(runtime.Credentials, runtime.Close), nil
		},
		AudienceManagementRuntime: func(token string) (cli.AudienceManagementRuntime, error) {
			runtime, err := lineRuntimes.audience(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).AudienceManagement(), nil
		},
		BulkMessagingRuntime: func(token string) (cli.BulkMessagingRuntime, error) {
			runtime, err := lineRuntimes.messaging(token)
			if err != nil {
				return nil, err
			}
			return lineRuntimeAdapters(runtime).BulkMessagingRuntime(), nil
		},
	}
}

// RunOps owns the CLI composition boundary so the process entrypoint imports
// only bootstrap and standard-library packages.
func RunOps(ctx context.Context, args []string, getenv func(string) string) error {
	return cli.Run(ctx, args, getenv, BuildOpsCLI())
}
