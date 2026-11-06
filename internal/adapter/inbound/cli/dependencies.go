package cli

import (
	"context"
	"time"

	accountlinktoken "github.com/963sup/linebot-platform/internal/application/usecases/account-link-token"
	"github.com/963sup/linebot-platform/internal/application/usecases/attendance"
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
	workspaceautomation "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
	"golang.org/x/oauth2"
)

// The Runtime interfaces below are narrow effect dependencies consumed by
// this driving CLI adapter. They are not Hexagonal inbound ports: the
// Application service methods are the inbound use-case API. Runtime
// construction belongs to the composition root, while method adaptation stays
// in this adapter.
//
// PushRuntime is the narrow dependency required by the push command.
type PushRuntime interface {
	Push(ctx context.Context, recipient string, messages []messagingdelivery.Message, retryKey string) error
	Close() error
}

type ProfileRuntime interface {
	GetProfile(context.Context, contactdirectory.ProfileQuery) (contactdirectory.ProfileSnapshot, error)
	Close() error
}

type ValidationRuntime interface {
	Validate(context.Context, messagingdelivery.ValidationRequest) (messagingdelivery.ValidationReport, error)
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

type AudienceDefinitionSummary struct {
	ID               string `json:"id"`
	Source           string `json:"source"`
	DefinitionSHA256 string `json:"definition_sha256"`
	DefinitionBytes  int    `json:"definition_bytes"`
	Version          uint64 `json:"version"`
	Status           string `json:"status"`
}

type AudienceSyncSummary struct {
	ProviderReference string `json:"provider_reference"`
	SynchronizedCount int    `json:"synchronized_count"`
}

type AudienceDefinitionsRuntime interface {
	GetAudienceDefinition(context.Context, string) (AudienceDefinitionSummary, error)
	DefineAudience(context.Context, string, string, string) (AudienceDefinitionSummary, error)
	ReviseAudience(context.Context, string, string, string) (AudienceDefinitionSummary, error)
	ActivateAudience(context.Context, string) (AudienceDefinitionSummary, error)
	ArchiveAudience(context.Context, string) (AudienceDefinitionSummary, error)
	SyncAudience(context.Context, string) (AudienceSyncSummary, error)
	Close() error
}

type InteractionRegion struct {
	ID              string `json:"id"`
	ActionReference string `json:"action_reference"`
	AssetReference  string `json:"asset_reference,omitempty"`
}

type InteractionSurfaceSummary struct {
	ID          string `json:"id"`
	RegionCount int    `json:"region_count"`
	Version     uint64 `json:"version"`
	Status      string `json:"status"`
}

type SurfaceDeploymentSummary struct {
	ProviderReference string `json:"provider_reference"`
	Version           uint64 `json:"version"`
}

type InteractionSurfacesRuntime interface {
	GetSurface(context.Context, string) (InteractionSurfaceSummary, error)
	DefineSurface(context.Context, string, []InteractionRegion) (InteractionSurfaceSummary, error)
	ReviseSurface(context.Context, string, []InteractionRegion) (InteractionSurfaceSummary, error)
	PublishSurface(context.Context, string) (InteractionSurfaceSummary, error)
	RetireSurface(context.Context, string) (InteractionSurfaceSummary, error)
	DeploySurface(context.Context, string) (SurfaceDeploymentSummary, error)
	Close() error
}

type PromotionSummary struct {
	ID                      string    `json:"id"`
	OfferSHA256             string    `json:"offer_sha256"`
	EligibilityPolicySHA256 string    `json:"eligibility_policy_sha256"`
	StartsAt                time.Time `json:"starts_at"`
	EndsAt                  time.Time `json:"ends_at"`
	Revision                uint64    `json:"revision"`
	Status                  string    `json:"status"`
}

type DeliveryIntentSummary struct {
	PromotionID     string `json:"promotion_id"`
	RecipientSHA256 string `json:"recipient_sha256"`
	OfferSHA256     string `json:"offer_sha256"`
}

type PromotionsRuntime interface {
	GetPromotion(context.Context, string) (PromotionSummary, error)
	DefinePromotion(context.Context, string, string, string, time.Time, time.Time) (PromotionSummary, error)
	ActivatePromotion(context.Context, string, time.Time) (PromotionSummary, error)
	ClosePromotion(context.Context, string) (PromotionSummary, error)
	CreateDeliveryIntent(context.Context, string, string, string, bool, time.Time) (DeliveryIntentSummary, error)
	Close() error
}

type AutomationRequestSummary struct {
	ID         string `json:"id"`
	Action     string `json:"action"`
	ScopeCount int    `json:"scope_count"`
	Approved   bool   `json:"approved"`
	HasOutput  bool   `json:"has_output"`
	Revision   uint64 `json:"revision"`
	Status     string `json:"status"`
}

type AutomationRequestsRuntime interface {
	GetAutomationRequest(context.Context, string) (AutomationRequestSummary, error)
	RequestAutomation(context.Context, string, string, []string) (AutomationRequestSummary, error)
	ApproveAutomation(context.Context, string, string) (AutomationRequestSummary, error)
	ExecuteAutomation(context.Context, string) (AutomationRequestSummary, error)
	Close() error
}

// AttendanceRuntime is the narrow operator dependency used by Attendance
// administration commands. The Application service owns all lifecycle and
// review policy; this driving adapter only validates operator intent and I/O.
type AttendanceRuntime interface {
	InviteEmployee(context.Context, attendance.InviteEmployeeCommand) (attendance.InviteEmployeeResult, error)
	DeactivateEmployee(context.Context, attendance.DeactivateEmployeeCommand) error
	UpsertWorkplace(context.Context, attendance.UpsertWorkplaceCommand) error
	CreateShift(context.Context, attendance.CreateShiftCommand) error
	AssignEmployee(context.Context, attendance.AssignEmployeeCommand) error
	ListReviewRequired(context.Context, int) ([]attendance.ReviewItem, error)
	ShowReview(context.Context, attendance.ShowReviewQuery) (attendance.ReviewDetail, error)
	Review(context.Context, attendance.ReviewCommand) error
	Close() error
}

// AttendanceRuntimeConfig carries explicit operator-selected environment and
// cloud resource settings into the composition root. Project and environment
// are never inferred from ambient credentials.
type AttendanceRuntimeConfig struct {
	ProjectID         string
	Environment       string
	FirestoreDatabase string
	ActiveKeyID       string
	KeyVersions       string
}

// Dependencies is the immutable set of runtime factories supplied by the
// composition root for one CLI invocation.
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
	AudienceDefinitions       func(context.Context, string) (AudienceDefinitionsRuntime, error)
	InteractionSurfaces       func(context.Context, string) (InteractionSurfacesRuntime, error)
	Promotions                func(context.Context, string) (PromotionsRuntime, error)
	AutomationRequests        func(context.Context, string) (AutomationRequestsRuntime, error)
	Attendance                func(context.Context, AttendanceRuntimeConfig) (AttendanceRuntime, error)
}

func dependency(values []Dependencies) Dependencies {
	if len(values) == 0 {
		return Dependencies{}
	}
	return values[0]
}
