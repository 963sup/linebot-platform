package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/cli"
	accountlinktoken "github.com/963sup/linebot-platform/internal/application/usecases/account-link-token"
	"github.com/963sup/linebot-platform/internal/application/usecases/audience"
	contactdirectory "github.com/963sup/linebot-platform/internal/application/usecases/contact-directory"
	conversationoperations "github.com/963sup/linebot-platform/internal/application/usecases/conversation-operations"
	credentiallifecycle "github.com/963sup/linebot-platform/internal/application/usecases/credential-lifecycle"
	experiencepresentation "github.com/963sup/linebot-platform/internal/application/usecases/experience-presentation"
	insightsreporting "github.com/963sup/linebot-platform/internal/application/usecases/insights-reporting"
	mediatransfer "github.com/963sup/linebot-platform/internal/application/usecases/media-transfer"
	"github.com/963sup/linebot-platform/internal/application/usecases/membership"
	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/963sup/linebot-platform/internal/application/usecases/promotion"
	providerconfiguration "github.com/963sup/linebot-platform/internal/application/usecases/provider-configuration"
	requestvalidation "github.com/963sup/linebot-platform/internal/application/usecases/request-validation"
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
			return menuManagementRuntime{ProviderCommandGateway: runtime.MenuCommands, runtime: runtime}, nil
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

func (r webhookRuntime) ConfigureWebhook(ctx context.Context, endpoint string) error {
	return r.WebhookAdmin.ConfigureWebhook(ctx, endpoint)
}

func (r webhookRuntime) TestWebhook(ctx context.Context, endpoint string) (providerconfiguration.WebhookTestResult, error) {
	return r.WebhookAdmin.TestWebhook(ctx, endpoint)
}

type tokenRuntime struct{ *bootstrap.LINETokenOpsRuntime }

func (r tokenRuntime) IssueStatelessWithClient(ctx context.Context, grant string, credential credentiallifecycle.ClientCredential) (credentiallifecycle.IssuedCredential, error) {
	return r.Credentials.IssueStatelessWithClient(ctx, grant, credential)
}
func (r tokenRuntime) IssueStatelessWithAssertion(ctx context.Context, grant string, credential credentiallifecycle.AssertionCredential) (credentiallifecycle.IssuedCredential, error) {
	return r.Credentials.IssueStatelessWithAssertion(ctx, grant, credential)
}
func (r tokenRuntime) ValidKeyIDs(ctx context.Context, credential credentiallifecycle.AssertionCredential) ([]string, error) {
	return r.Credentials.ValidKeyIDs(ctx, credential)
}
func (r tokenRuntime) IssueJWT(ctx context.Context, grant string, credential credentiallifecycle.AssertionCredential) (credentiallifecycle.IssuedCredential, error) {
	return r.Credentials.IssueJWT(ctx, grant, credential)
}
func (r tokenRuntime) VerifyJWT(ctx context.Context, token string) (credentiallifecycle.Verification, error) {
	return r.Credentials.VerifyJWT(ctx, token)
}
func (r tokenRuntime) RevokeJWT(ctx context.Context, credential credentiallifecycle.ClientCredential, token string) error {
	return r.Credentials.RevokeJWT(ctx, credential, token)
}
func (r tokenRuntime) IssueShortLived(ctx context.Context, grant string, credential credentiallifecycle.ClientCredential) (credentiallifecycle.IssuedCredential, error) {
	return r.Credentials.IssueShortLived(ctx, grant, credential)
}
func (r tokenRuntime) VerifyShortLived(ctx context.Context, token string) (credentiallifecycle.Verification, error) {
	return r.Credentials.VerifyShortLived(ctx, token)
}
func (r tokenRuntime) RevokeShortLived(ctx context.Context, token string) error {
	return r.Credentials.RevokeShortLived(ctx, token)
}

type audienceManagementRuntime struct{ *bootstrap.LINEOpsRuntime }

func (r audienceManagementRuntime) CreateGroup(ctx context.Context, cmd audience.CreateGroupCommand) (audience.ProviderAudience, error) {
	return r.AudienceManager.CreateGroup(ctx, cmd)
}
func (r audienceManagementRuntime) AppendRecipients(ctx context.Context, cmd audience.AppendRecipientsCommand) error {
	return r.AudienceManager.AppendRecipients(ctx, cmd)
}
func (r audienceManagementRuntime) CreateInteractionGroup(ctx context.Context, cmd audience.CreateInteractionGroupCommand, impression bool) (audience.ProviderAudience, error) {
	return r.AudienceManager.CreateInteractionGroup(ctx, cmd, impression)
}
func (r audienceManagementRuntime) UpdateDescription(ctx context.Context, id audience.ProviderAudienceID, description string) error {
	return r.AudienceManager.UpdateDescription(ctx, id, description)
}
func (r audienceManagementRuntime) Delete(ctx context.Context, id audience.ProviderAudienceID) error {
	return r.AudienceManager.Delete(ctx, id)
}
func (r audienceManagementRuntime) CreateFromFile(ctx context.Context, file audience.AudienceFile) (audience.ProviderAudience, error) {
	return r.AudienceManager.CreateFromFile(ctx, file)
}
func (r audienceManagementRuntime) AppendFromFile(ctx context.Context, id audience.ProviderAudienceID, file audience.AudienceFile) error {
	return r.AudienceManager.AppendFromFile(ctx, id, file)
}

type bulkMessagingRuntime struct{ *bootstrap.LINEOpsRuntime }

func (r bulkMessagingRuntime) Broadcast(ctx context.Context, messages []messagingdelivery.Message, retryKey string) error {
	return r.BulkMessaging.Broadcast(ctx, messages, retryKey)
}

func (r bulkMessagingRuntime) Multicast(ctx context.Context, users []string, messages []messagingdelivery.Message, retryKey string) error {
	return r.BulkMessaging.Multicast(ctx, users, messages, retryKey)
}

func (r bulkMessagingRuntime) SendNarrowcast(ctx context.Context, payload messagingdelivery.NarrowcastRequest, retryKey string) error {
	return r.BulkMessaging.SendNarrowcast(ctx, payload, retryKey)
}

func (r pushRuntime) Push(ctx context.Context, recipient string, messages []messagingdelivery.Message, retryKey string) error {
	return r.Messaging.Push(ctx, recipient, messages, retryKey)
}

type profileRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r profileRuntime) GetProfile(ctx context.Context, query contactdirectory.ProfileQuery) (contactdirectory.ProfileSnapshot, error) {
	return r.Profiles.GetProfile(ctx, query)
}

type validationRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r validationRuntime) Validate(ctx context.Context, request requestvalidation.Request) (requestvalidation.Report, error) {
	return r.Validation.Validate(ctx, request)
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

func (r actionsRuntime) Leave(ctx context.Context, value conversationoperations.Conversation) error {
	return r.Actions.Leave(ctx, value)
}

func (r actionsRuntime) ShowLoading(ctx context.Context, value conversationoperations.LoadingIndicator) error {
	return r.Actions.ShowLoading(ctx, value)
}

func (r actionsRuntime) MarkRead(ctx context.Context, value conversationoperations.ReadTarget) error {
	return r.Actions.MarkRead(ctx, value)
}

func (r actionsRuntime) MarkReadByToken(ctx context.Context, value conversationoperations.ReadToken) error {
	return r.Actions.MarkReadByToken(ctx, value)
}

type eligibilityRuntime struct {
	*bootstrap.LINEOpsRuntime
}

type audienceRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r audienceRuntime) Get(ctx context.Context, id audience.ProviderAudienceID, shared bool) (audience.ProviderAudience, error) {
	return r.Audiences.Get(ctx, id, shared)
}

func (r audienceRuntime) List(ctx context.Context, page audience.PageRequest, shared bool) (audience.ProviderAudiencePage, error) {
	return r.Audiences.List(ctx, page, shared)
}

type contactsRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r contactsRuntime) ListFollowers(ctx context.Context, page contactdirectory.PageRequest) (contactdirectory.RecipientPage, error) {
	return r.Contacts.ListFollowers(ctx, page)
}
func (r contactsRuntime) GetSummary(ctx context.Context, ref contactdirectory.RecipientReference) (contactdirectory.ConversationSummary, error) {
	return r.Contacts.GetSummary(ctx, ref)
}
func (r contactsRuntime) ListMembers(ctx context.Context, ref contactdirectory.RecipientReference, page contactdirectory.PageRequest) (contactdirectory.RecipientPage, error) {
	return r.Contacts.ListMembers(ctx, ref, page)
}
func (r contactsRuntime) GetMemberProfile(ctx context.Context, ref contactdirectory.RecipientReference, id contactdirectory.RecipientID) (contactdirectory.ProfileSnapshot, error) {
	return r.Contacts.GetMemberProfile(ctx, ref, id)
}

type couponsRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r couponsRuntime) List(ctx context.Context, page promotion.CouponPageRequest) (promotion.CouponPage, error) {
	return r.Coupons.List(ctx, page)
}

func (r couponsRuntime) Get(ctx context.Context, id promotion.CouponID) (promotion.CouponView, error) {
	return r.Coupons.Get(ctx, id)
}

type insightsRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r insightsRuntime) Demographics(ctx context.Context) (insightsreporting.DemographicsReport, error) {
	return r.Insights.Demographics(ctx)
}
func (r insightsRuntime) Deliveries(ctx context.Context, q insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	return r.Insights.Deliveries(ctx, q)
}
func (r insightsRuntime) Followers(ctx context.Context, q insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	return r.Insights.Followers(ctx, q)
}
func (r insightsRuntime) MessageEvent(ctx context.Context, q insightsreporting.MessageEventQuery) (insightsreporting.MessageEventReport, error) {
	return r.Insights.MessageEvent(ctx, q)
}
func (r insightsRuntime) StatisticsPerUnit(ctx context.Context, q insightsreporting.UnitStatisticsQuery) (insightsreporting.StatisticsReport, error) {
	return r.Insights.StatisticsPerUnit(ctx, q)
}
func (r insightsRuntime) PresentationSummary(ctx context.Context, q insightsreporting.PresentationStatisticsQuery) (insightsreporting.StatisticsReport, error) {
	return r.Insights.PresentationSummary(ctx, q)
}
func (r insightsRuntime) PresentationDaily(ctx context.Context, q insightsreporting.PresentationStatisticsQuery) (insightsreporting.StatisticsReport, error) {
	return r.Insights.PresentationDaily(ctx, q)
}

type membershipsRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r membershipsRuntime) ListMemberships(ctx context.Context, page membership.PageRequest) (membership.MembershipPage, error) {
	return r.Memberships.ListMemberships(ctx, page)
}

func (r membershipsRuntime) GetSubscription(ctx context.Context, id membership.RecipientID) (membership.SubscriptionView, error) {
	return r.Memberships.GetSubscription(ctx, id)
}

func (r membershipsRuntime) ListMembers(ctx context.Context, id membership.MembershipID, page membership.PageRequest) (membership.MemberPage, error) {
	return r.Memberships.ListMembers(ctx, id, page)
}

type menusRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r menusRuntime) List(ctx context.Context) ([]experiencepresentation.ProviderSurface, error) {
	return r.Menus.List(ctx)
}
func (r menusRuntime) Get(ctx context.Context, id experiencepresentation.ProviderSurfaceID) (experiencepresentation.ProviderSurface, error) {
	return r.Menus.Get(ctx, id)
}
func (r menusRuntime) Default(ctx context.Context) (experiencepresentation.ProviderSurface, error) {
	return r.Menus.Default(ctx)
}
func (r menusRuntime) ListAliases(ctx context.Context) ([]experiencepresentation.ProviderAlias, error) {
	return r.Menus.ListAliases(ctx)
}
func (r menusRuntime) GetAlias(ctx context.Context, id experiencepresentation.AliasID) (experiencepresentation.ProviderAlias, error) {
	return r.Menus.GetAlias(ctx, id)
}
func (r menusRuntime) ForRecipient(ctx context.Context, id experiencepresentation.RecipientID) (experiencepresentation.ProviderSurface, error) {
	return r.Menus.ForRecipient(ctx, id)
}
func (r menusRuntime) BatchProgress(ctx context.Context, id string) (experiencepresentation.BatchProgress, error) {
	return r.Menus.BatchProgress(ctx, id)
}

type channelInspectionRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r channelInspectionRuntime) BotInfo(ctx context.Context) (providerconfiguration.ChannelSnapshot, error) {
	return r.Channel.BotInfo(ctx)
}
func (r channelInspectionRuntime) WebhookEndpoint(ctx context.Context) (providerconfiguration.ChannelSnapshot, error) {
	return r.Channel.WebhookEndpoint(ctx)
}
func (r channelInspectionRuntime) MessageQuota(ctx context.Context) (providerconfiguration.ChannelSnapshot, error) {
	return r.Channel.MessageQuota(ctx)
}
func (r channelInspectionRuntime) MessageQuotaConsumption(ctx context.Context) (providerconfiguration.ChannelSnapshot, error) {
	return r.Channel.MessageQuotaConsumption(ctx)
}
func (r channelInspectionRuntime) AggregationUnitUsage(ctx context.Context) (providerconfiguration.ChannelSnapshot, error) {
	return r.Channel.AggregationUnitUsage(ctx)
}
func (r channelInspectionRuntime) AggregationUnitNames(ctx context.Context, limit, start string) (providerconfiguration.ChannelSnapshot, error) {
	return r.Channel.AggregationUnitNames(ctx, limit, start)
}
func (r channelInspectionRuntime) NarrowcastProgress(ctx context.Context, id string) (providerconfiguration.ChannelSnapshot, error) {
	return r.Channel.NarrowcastProgress(ctx, id)
}
func (r channelInspectionRuntime) DeliveryReply(ctx context.Context, q insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	return r.DeliveryReports.DeliveryReply(ctx, q)
}
func (r channelInspectionRuntime) DeliveryPush(ctx context.Context, q insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	return r.DeliveryReports.DeliveryPush(ctx, q)
}
func (r channelInspectionRuntime) DeliveryMulticast(ctx context.Context, q insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	return r.DeliveryReports.DeliveryMulticast(ctx, q)
}
func (r channelInspectionRuntime) DeliveryBroadcast(ctx context.Context, q insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	return r.DeliveryReports.DeliveryBroadcast(ctx, q)
}

type contentRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r contentRuntime) MessageContent(ctx context.Context, id string, variant mediatransfer.ContentVariant) (mediatransfer.BinaryContent, error) {
	return r.Content.MessageContent(ctx, id, variant)
}
func (r contentRuntime) MessageContentStatus(ctx context.Context, id string) (mediatransfer.PreparationStatus, error) {
	return r.Content.MessageContentStatus(ctx, id)
}

type menuImageRuntime struct {
	*bootstrap.LINEOpsRuntime
}

type menuManagementRuntime struct {
	experiencepresentation.ProviderCommandGateway
	runtime *bootstrap.LINEOpsRuntime
}

func (r menuManagementRuntime) Close() error { return r.runtime.Close() }

type menuImageUploadRuntime struct {
	*bootstrap.LINEOpsRuntime
}

func (r menuImageUploadRuntime) StorePresentationImage(ctx context.Context, id string, content mediatransfer.BinaryContent) error {
	return r.RichMenuImages.StorePresentationImage(ctx, id, content)
}

func (r menuImageRuntime) PresentationImage(ctx context.Context, id string) (mediatransfer.BinaryContent, error) {
	return r.MenuImageReader.PresentationImage(ctx, id)
}

func (r eligibilityRuntime) PushMessagesByPhone(ctx context.Context, request messagingdelivery.PhoneDeliveryRequest) error {
	return r.PhoneDelivery.PushMessagesByPhone(ctx, request)
}
func (r eligibilityRuntime) PushTemplatedMessagesByPhone(ctx context.Context, request messagingdelivery.PhoneDeliveryRequest) error {
	return r.PhoneDelivery.PushTemplatedMessagesByPhone(ctx, request)
}
func (r eligibilityRuntime) PNPMessageStatistics(ctx context.Context, q insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	return r.PhoneReports.PNPMessageStatistics(ctx, q)
}
func (r eligibilityRuntime) PNPTemplatedMessageStatistics(ctx context.Context, q insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	return r.PhoneReports.PNPTemplatedMessageStatistics(ctx, q)
}
func (r eligibilityRuntime) CreateCouponDocument(ctx context.Context, doc promotion.ProviderCouponDocument) (promotion.CouponView, error) {
	return r.CouponCreator.CreateCouponDocument(ctx, doc)
}
func (r eligibilityRuntime) CloseCoupon(ctx context.Context, id promotion.CouponID) error {
	return r.CouponCreator.CloseCoupon(ctx, id)
}

func (r linkTokenRuntime) Issue(ctx context.Context, subject accountlinktoken.Subject) (accountlinktoken.Token, error) {
	return r.AccountLink.Issue(ctx, subject)
}

func (r statusRuntime) Inspect(ctx context.Context, state messagingdelivery.EventState, limit int, now time.Time) (messagingdelivery.EventStatusReport, error) {
	return r.Reader.Inspect(ctx, state, limit, now)
}
