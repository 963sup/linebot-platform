package cli

import (
	"context"
	"time"

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
	workspaceautomation "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
	"golang.org/x/oauth2"
)

// PushRuntime is the narrow dependency required by the push command.
// Runtime construction belongs to the composition root, not this adapter.
type PushRuntime interface {
	Push(ctx context.Context, recipient string, messages []messagingdelivery.Message, retryKey string) error
	Close() error
}

type ProfileRuntime interface {
	GetProfile(context.Context, contactdirectory.ProfileQuery) (contactdirectory.ProfileSnapshot, error)
	Close() error
}

type ValidationRuntime interface {
	Validate(context.Context, requestvalidation.Request) (requestvalidation.Report, error)
	Close() error
}

type StatusRuntime interface {
	Inspect(ctx context.Context, state messagingdelivery.EventState, limit int, now time.Time) (messagingdelivery.EventStatusReport, error)
	Close() error
}

type WorkspaceRuntime interface {
	Execute(context.Context, workspaceautomation.API, workspaceautomation.OperationCommand, workspaceautomation.OperationApproval) (workspaceautomation.OperationResult, error)
}

type LinkTokenRuntime interface {
	Issue(context.Context, accountlinktoken.Subject) (accountlinktoken.Token, error)
	Close() error
}

type ActionsRuntime interface {
	Leave(context.Context, conversationoperations.Conversation) error
	ShowLoading(context.Context, conversationoperations.LoadingIndicator) error
	MarkRead(context.Context, conversationoperations.ReadTarget) error
	MarkReadByToken(context.Context, conversationoperations.ReadToken) error
	Close() error
}

type EligibilityRuntime interface {
	PushMessagesByPhone(context.Context, messagingdelivery.PhoneDeliveryRequest) error
	PushTemplatedMessagesByPhone(context.Context, messagingdelivery.PhoneDeliveryRequest) error
	PNPMessageStatistics(context.Context, insightsreporting.DailyQuery) (insightsreporting.CountReport, error)
	PNPTemplatedMessageStatistics(context.Context, insightsreporting.DailyQuery) (insightsreporting.CountReport, error)
	CreateCouponDocument(context.Context, promotion.ProviderCouponDocument) (promotion.CouponView, error)
	CloseCoupon(context.Context, promotion.CouponID) error
	Close() error
}

type AudienceRuntime interface {
	Get(context.Context, audience.ProviderAudienceID, bool) (audience.ProviderAudience, error)
	List(context.Context, audience.PageRequest, bool) (audience.ProviderAudiencePage, error)
	Close() error
}

type ContactsRuntime interface {
	ListFollowers(context.Context, contactdirectory.PageRequest) (contactdirectory.RecipientPage, error)
	GetSummary(context.Context, contactdirectory.RecipientReference) (contactdirectory.ConversationSummary, error)
	ListMembers(context.Context, contactdirectory.RecipientReference, contactdirectory.PageRequest) (contactdirectory.RecipientPage, error)
	GetMemberProfile(context.Context, contactdirectory.RecipientReference, contactdirectory.RecipientID) (contactdirectory.ProfileSnapshot, error)
	Close() error
}

type CouponsRuntime interface {
	List(context.Context, promotion.CouponPageRequest) (promotion.CouponPage, error)
	Get(context.Context, promotion.CouponID) (promotion.CouponView, error)
	Close() error
}

type InsightsRuntime interface {
	Demographics(context.Context) (insightsreporting.DemographicsReport, error)
	Deliveries(context.Context, insightsreporting.DailyQuery) (insightsreporting.CountReport, error)
	Followers(context.Context, insightsreporting.DailyQuery) (insightsreporting.CountReport, error)
	MessageEvent(context.Context, insightsreporting.MessageEventQuery) (insightsreporting.MessageEventReport, error)
	StatisticsPerUnit(context.Context, insightsreporting.UnitStatisticsQuery) (insightsreporting.StatisticsReport, error)
	PresentationSummary(context.Context, insightsreporting.PresentationStatisticsQuery) (insightsreporting.StatisticsReport, error)
	PresentationDaily(context.Context, insightsreporting.PresentationStatisticsQuery) (insightsreporting.StatisticsReport, error)
	Close() error
}

type MembershipsRuntime interface {
	ListMemberships(context.Context, membership.PageRequest) (membership.MembershipPage, error)
	GetSubscription(context.Context, membership.RecipientID) (membership.SubscriptionView, error)
	ListMembers(context.Context, membership.MembershipID, membership.PageRequest) (membership.MemberPage, error)
	Close() error
}

type MenusRuntime interface {
	List(context.Context) ([]experiencepresentation.ProviderSurface, error)
	Get(context.Context, experiencepresentation.ProviderSurfaceID) (experiencepresentation.ProviderSurface, error)
	Default(context.Context) (experiencepresentation.ProviderSurface, error)
	ListAliases(context.Context) ([]experiencepresentation.ProviderAlias, error)
	GetAlias(context.Context, experiencepresentation.AliasID) (experiencepresentation.ProviderAlias, error)
	ForRecipient(context.Context, experiencepresentation.RecipientID) (experiencepresentation.ProviderSurface, error)
	BatchProgress(context.Context, string) (experiencepresentation.BatchProgress, error)
	Close() error
}

type ChannelInspectionRuntime interface {
	providerconfiguration.ChannelReader
	insightsreporting.DeliveryReader
	Close() error
}

type ContentRuntime interface {
	MessageContent(context.Context, string, mediatransfer.ContentVariant) (mediatransfer.BinaryContent, error)
	MessageContentStatus(context.Context, string) (mediatransfer.PreparationStatus, error)
	Close() error
}

type MenuImageRuntime interface {
	PresentationImage(context.Context, string) (mediatransfer.BinaryContent, error)
	Close() error
}

type MenuManagementRuntime interface {
	experiencepresentation.ProviderCommandGateway
	Close() error
}

type MenuImageUploadRuntime interface {
	StorePresentationImage(context.Context, string, mediatransfer.BinaryContent) error
	Close() error
}

type WebhookRuntime interface {
	ConfigureWebhook(context.Context, string) error
	TestWebhook(context.Context, string) (providerconfiguration.WebhookTestResult, error)
	Close() error
}

type TokenRuntime interface {
	IssueStatelessWithClient(context.Context, string, credentiallifecycle.ClientCredential) (credentiallifecycle.IssuedCredential, error)
	IssueStatelessWithAssertion(context.Context, string, credentiallifecycle.AssertionCredential) (credentiallifecycle.IssuedCredential, error)
	ValidKeyIDs(context.Context, credentiallifecycle.AssertionCredential) ([]string, error)
	IssueJWT(context.Context, string, credentiallifecycle.AssertionCredential) (credentiallifecycle.IssuedCredential, error)
	VerifyJWT(context.Context, string) (credentiallifecycle.Verification, error)
	RevokeJWT(context.Context, credentiallifecycle.ClientCredential, string) error
	IssueShortLived(context.Context, string, credentiallifecycle.ClientCredential) (credentiallifecycle.IssuedCredential, error)
	VerifyShortLived(context.Context, string) (credentiallifecycle.Verification, error)
	RevokeShortLived(context.Context, string) error
	Close()
}

type AudienceManagementRuntime interface {
	CreateGroup(context.Context, audience.CreateGroupCommand) (audience.ProviderAudience, error)
	AppendRecipients(context.Context, audience.AppendRecipientsCommand) error
	CreateInteractionGroup(context.Context, audience.CreateInteractionGroupCommand, bool) (audience.ProviderAudience, error)
	UpdateDescription(context.Context, audience.ProviderAudienceID, string) error
	Delete(context.Context, audience.ProviderAudienceID) error
	CreateFromFile(context.Context, audience.AudienceFile) (audience.ProviderAudience, error)
	AppendFromFile(context.Context, audience.ProviderAudienceID, audience.AudienceFile) error
	Close() error
}

type BulkMessagingRuntime interface {
	Broadcast(context.Context, []messagingdelivery.Message, string) error
	Multicast(context.Context, []string, []messagingdelivery.Message, string) error
	SendNarrowcast(context.Context, messagingdelivery.NarrowcastRequest, string) error
	Close() error
}

// Dependencies groups the runtime factories owned by the composition root.
// ConfigureXRuntime remains available while command entry points migrate to
// receiving this bundle explicitly.
type Dependencies struct {
	PushRuntime               func(string) (PushRuntime, error)
	ProfileRuntime            func(string) (ProfileRuntime, error)
	ValidationRuntime         func(string) (ValidationRuntime, error)
	StatusRuntime             func(context.Context, string) (StatusRuntime, error)
	WorkspaceRuntime          func(context.Context, oauth2.TokenSource) (WorkspaceRuntime, error)
	LinkTokenRuntime          func(string) (LinkTokenRuntime, error)
	ActionsRuntime            func(string) (ActionsRuntime, error)
	EligibilityRuntime        func(string) (EligibilityRuntime, error)
	AudienceRuntime           func(string) (AudienceRuntime, error)
	ContactsRuntime           func(string) (ContactsRuntime, error)
	CouponsRuntime            func(string) (CouponsRuntime, error)
	InsightsRuntime           func(string) (InsightsRuntime, error)
	MembershipsRuntime        func(string) (MembershipsRuntime, error)
	MenusRuntime              func(string) (MenusRuntime, error)
	ChannelInspectionRuntime  func(string) (ChannelInspectionRuntime, error)
	ContentRuntime            func(string) (ContentRuntime, error)
	MenuImageRuntime          func(string) (MenuImageRuntime, error)
	MenuManagementRuntime     func(string) (MenuManagementRuntime, error)
	MenuImageUploadRuntime    func(string) (MenuImageUploadRuntime, error)
	WebhookRuntime            func(string) (WebhookRuntime, error)
	TokenRuntime              func() (TokenRuntime, error)
	AudienceManagementRuntime func(string) (AudienceManagementRuntime, error)
	BulkMessagingRuntime      func(string) (BulkMessagingRuntime, error)
}

var pushRuntimeFactory func(string) (PushRuntime, error)
var profileRuntimeFactory func(string) (ProfileRuntime, error)
var validationRuntimeFactory func(string) (ValidationRuntime, error)
var statusRuntimeFactory func(context.Context, string) (StatusRuntime, error)
var workspaceRuntimeFactory func(context.Context, oauth2.TokenSource) (WorkspaceRuntime, error)
var linkTokenRuntimeFactory func(string) (LinkTokenRuntime, error)
var actionsRuntimeFactory func(string) (ActionsRuntime, error)
var eligibilityRuntimeFactory func(string) (EligibilityRuntime, error)
var audienceRuntimeFactory func(string) (AudienceRuntime, error)
var contactsRuntimeFactory func(string) (ContactsRuntime, error)
var couponsRuntimeFactory func(string) (CouponsRuntime, error)
var insightsRuntimeFactory func(string) (InsightsRuntime, error)
var membershipsRuntimeFactory func(string) (MembershipsRuntime, error)
var menusRuntimeFactory func(string) (MenusRuntime, error)
var channelInspectionRuntimeFactory func(string) (ChannelInspectionRuntime, error)
var contentRuntimeFactory func(string) (ContentRuntime, error)
var menuImageRuntimeFactory func(string) (MenuImageRuntime, error)
var menuManagementRuntimeFactory func(string) (MenuManagementRuntime, error)
var menuImageUploadRuntimeFactory func(string) (MenuImageUploadRuntime, error)
var webhookRuntimeFactory func(string) (WebhookRuntime, error)
var tokenRuntimeFactory func() (TokenRuntime, error)
var audienceManagementRuntimeFactory func(string) (AudienceManagementRuntime, error)
var bulkMessagingRuntimeFactory func(string) (BulkMessagingRuntime, error)

// ConfigureDependencies installs the complete composition-root bundle.
// It is intentionally a value snapshot: callers finish constructing the
// bundle before publishing it to the CLI package.
func ConfigureDependencies(deps Dependencies) {
	pushRuntimeFactory = deps.PushRuntime
	profileRuntimeFactory = deps.ProfileRuntime
	validationRuntimeFactory = deps.ValidationRuntime
	statusRuntimeFactory = deps.StatusRuntime
	workspaceRuntimeFactory = deps.WorkspaceRuntime
	linkTokenRuntimeFactory = deps.LinkTokenRuntime
	actionsRuntimeFactory = deps.ActionsRuntime
	eligibilityRuntimeFactory = deps.EligibilityRuntime
	audienceRuntimeFactory = deps.AudienceRuntime
	contactsRuntimeFactory = deps.ContactsRuntime
	couponsRuntimeFactory = deps.CouponsRuntime
	insightsRuntimeFactory = deps.InsightsRuntime
	membershipsRuntimeFactory = deps.MembershipsRuntime
	menusRuntimeFactory = deps.MenusRuntime
	channelInspectionRuntimeFactory = deps.ChannelInspectionRuntime
	contentRuntimeFactory = deps.ContentRuntime
	menuImageRuntimeFactory = deps.MenuImageRuntime
	menuManagementRuntimeFactory = deps.MenuManagementRuntime
	menuImageUploadRuntimeFactory = deps.MenuImageUploadRuntime
	webhookRuntimeFactory = deps.WebhookRuntime
	tokenRuntimeFactory = deps.TokenRuntime
	audienceManagementRuntimeFactory = deps.AudienceManagementRuntime
	bulkMessagingRuntimeFactory = deps.BulkMessagingRuntime
}

// ConfigurePushRuntime supplies the composition-root factory for the push command.
// It is transitional while the remaining CLI commands migrate to injected dependencies.
func ConfigurePushRuntime(factory func(string) (PushRuntime, error)) {
	pushRuntimeFactory = factory
}

// ConfigureProfileRuntime supplies the composition-root factory for profile lookup.
func ConfigureProfileRuntime(factory func(string) (ProfileRuntime, error)) {
	profileRuntimeFactory = factory
}

func ConfigureValidationRuntime(factory func(string) (ValidationRuntime, error)) {
	validationRuntimeFactory = factory
}

func ConfigureStatusRuntime(factory func(context.Context, string) (StatusRuntime, error)) {
	statusRuntimeFactory = factory
}

func ConfigureWorkspaceRuntime(factory func(context.Context, oauth2.TokenSource) (WorkspaceRuntime, error)) {
	workspaceRuntimeFactory = factory
}

func ConfigureLinkTokenRuntime(factory func(string) (LinkTokenRuntime, error)) {
	linkTokenRuntimeFactory = factory
}

func ConfigureActionsRuntime(factory func(string) (ActionsRuntime, error)) {
	actionsRuntimeFactory = factory
}

func ConfigureEligibilityRuntime(factory func(string) (EligibilityRuntime, error)) {
	eligibilityRuntimeFactory = factory
}

func ConfigureAudienceRuntime(factory func(string) (AudienceRuntime, error)) {
	audienceRuntimeFactory = factory
}

func ConfigureContactsRuntime(factory func(string) (ContactsRuntime, error)) {
	contactsRuntimeFactory = factory
}

func ConfigureCouponsRuntime(factory func(string) (CouponsRuntime, error)) {
	couponsRuntimeFactory = factory
}

func ConfigureInsightsRuntime(factory func(string) (InsightsRuntime, error)) {
	insightsRuntimeFactory = factory
}

func ConfigureMembershipsRuntime(factory func(string) (MembershipsRuntime, error)) {
	membershipsRuntimeFactory = factory
}

func ConfigureMenusRuntime(factory func(string) (MenusRuntime, error)) {
	menusRuntimeFactory = factory
}

func ConfigureChannelInspectionRuntime(factory func(string) (ChannelInspectionRuntime, error)) {
	channelInspectionRuntimeFactory = factory
}

func ConfigureContentRuntime(factory func(string) (ContentRuntime, error)) {
	contentRuntimeFactory = factory
}

func ConfigureMenuImageRuntime(factory func(string) (MenuImageRuntime, error)) {
	menuImageRuntimeFactory = factory
}

func ConfigureMenuManagementRuntime(factory func(string) (MenuManagementRuntime, error)) {
	menuManagementRuntimeFactory = factory
}

func ConfigureMenuImageUploadRuntime(factory func(string) (MenuImageUploadRuntime, error)) {
	menuImageUploadRuntimeFactory = factory
}

func ConfigureWebhookRuntime(factory func(string) (WebhookRuntime, error)) {
	webhookRuntimeFactory = factory
}

func ConfigureTokenRuntime(factory func() (TokenRuntime, error)) {
	tokenRuntimeFactory = factory
}

func ConfigureAudienceManagementRuntime(factory func(string) (AudienceManagementRuntime, error)) {
	audienceManagementRuntimeFactory = factory
}

func ConfigureBulkMessagingRuntime(factory func(string) (BulkMessagingRuntime, error)) {
	bulkMessagingRuntimeFactory = factory
}
