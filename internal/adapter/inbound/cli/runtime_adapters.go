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
	workspaceautomation "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
)

// RuntimeAdapters owns the presentation-facing method adaptation required by
// CLI commands. Bootstrap supplies already-composed Application services and
// provider gateways; it does not implement inbound command contracts.
type RuntimeAdapters struct {
	Messaging       *messagingdelivery.DeliveryService
	BulkMessaging   *messagingdelivery.DeliveryService
	PhoneDelivery   messagingdelivery.PhoneDeliveryGateway
	PhoneReports    insightsreporting.PhoneDeliveryReader
	Validation      *messagingdelivery.ValidationService
	Profiles        *contactdirectory.Queries
	Contacts        *contactdirectory.ConversationQueries
	WebhookAdmin    *providerconfiguration.Service
	Channel         providerconfiguration.ChannelReader
	DeliveryReports insightsreporting.DeliveryReader
	Audiences       *audience.ProviderQueries
	AudienceManager *audience.ProviderOperations
	Coupons         *promotion.CouponQueries
	CouponCreator   promotion.ProviderCouponGateway
	Memberships     *membership.Queries
	Menus           *experiencepresentation.ProviderQueries
	MenuCommands    experiencepresentation.ProviderCommandGateway
	Insights        *insightsreporting.Reports
	Content         *mediatransfer.Service
	RichMenuImages  *mediatransfer.Service
	MenuImageReader *mediatransfer.Service
	Actions         *conversationoperations.Service
	AccountLink     *accountlinktoken.Service
	Close           func() error
}

type runtimeCloser struct{ close func() error }

func (r runtimeCloser) Close() error { return r.close() }

type workspaceExecutor func(context.Context, workspaceautomation.API, workspaceautomation.OperationCommand, workspaceautomation.OperationApproval) (workspaceautomation.OperationResult, error)

func (execute workspaceExecutor) Execute(ctx context.Context, api workspaceautomation.API, request workspaceautomation.OperationCommand, approval workspaceautomation.OperationApproval) (workspaceautomation.OperationResult, error) {
	result, err := execute(ctx, api, request, approval)
	if err != nil {
		return workspaceautomation.OperationResult{}, err
	}
	return result, nil
}

func NewWorkspaceRuntime(execute func(context.Context, workspaceautomation.API, workspaceautomation.OperationCommand, workspaceautomation.OperationApproval) (workspaceautomation.OperationResult, error)) WorkspaceRuntime {
	return workspaceExecutor(execute)
}

type pushRuntime struct {
	runtimeCloser
	service *messagingdelivery.DeliveryService
}

func (r pushRuntime) Push(ctx context.Context, recipient string, messages []messagingdelivery.Message, retryKey string) error {
	return r.service.Push(ctx, recipient, messages, retryKey)
}

type webhookRuntime struct {
	runtimeCloser
	service *providerconfiguration.Service
}

func (r webhookRuntime) ConfigureWebhook(ctx context.Context, endpoint string) error {
	return r.service.ConfigureWebhook(ctx, endpoint)
}
func (r webhookRuntime) TestWebhook(ctx context.Context, endpoint string) (providerconfiguration.WebhookTestResult, error) {
	return r.service.TestWebhook(ctx, endpoint)
}

type audienceManagementRuntime struct {
	runtimeCloser
	service *audience.ProviderOperations
}

func (r audienceManagementRuntime) CreateGroup(ctx context.Context, cmd audience.CreateGroupCommand) (audience.ProviderAudience, error) {
	return r.service.CreateGroup(ctx, cmd)
}
func (r audienceManagementRuntime) AppendRecipients(ctx context.Context, cmd audience.AppendRecipientsCommand) error {
	return r.service.AppendRecipients(ctx, cmd)
}
func (r audienceManagementRuntime) CreateInteractionGroup(ctx context.Context, cmd audience.CreateInteractionGroupCommand, impression bool) (audience.ProviderAudience, error) {
	return r.service.CreateInteractionGroup(ctx, cmd, impression)
}
func (r audienceManagementRuntime) UpdateDescription(ctx context.Context, id audience.ProviderAudienceID, description string) error {
	return r.service.UpdateDescription(ctx, id, description)
}
func (r audienceManagementRuntime) Delete(ctx context.Context, id audience.ProviderAudienceID) error {
	return r.service.Delete(ctx, id)
}
func (r audienceManagementRuntime) CreateFromFile(ctx context.Context, file audience.AudienceFile) (audience.ProviderAudience, error) {
	return r.service.CreateFromFile(ctx, file)
}
func (r audienceManagementRuntime) AppendFromFile(ctx context.Context, id audience.ProviderAudienceID, file audience.AudienceFile) error {
	return r.service.AppendFromFile(ctx, id, file)
}

type bulkMessagingRuntime struct {
	runtimeCloser
	service *messagingdelivery.DeliveryService
}

func (r bulkMessagingRuntime) Broadcast(ctx context.Context, messages []messagingdelivery.Message, retryKey string) error {
	return r.service.Broadcast(ctx, messages, retryKey)
}
func (r bulkMessagingRuntime) Multicast(ctx context.Context, users []string, messages []messagingdelivery.Message, retryKey string) error {
	return r.service.Multicast(ctx, users, messages, retryKey)
}
func (r bulkMessagingRuntime) SendNarrowcast(ctx context.Context, payload messagingdelivery.NarrowcastRequest, retryKey string) error {
	return r.service.SendNarrowcast(ctx, payload, retryKey)
}

type profileRuntime struct {
	runtimeCloser
	service *contactdirectory.Queries
}

func (r profileRuntime) GetProfile(ctx context.Context, query contactdirectory.ProfileQuery) (contactdirectory.ProfileSnapshot, error) {
	return r.service.GetProfile(ctx, query)
}

type validationRuntime struct {
	runtimeCloser
	service *messagingdelivery.ValidationService
}

func (r validationRuntime) Validate(ctx context.Context, request messagingdelivery.ValidationRequest) (messagingdelivery.ValidationReport, error) {
	return r.service.Validate(ctx, request)
}

type actionsRuntime struct {
	runtimeCloser
	service *conversationoperations.Service
}

func (r actionsRuntime) Leave(ctx context.Context, value conversationoperations.Conversation) error {
	return r.service.Leave(ctx, value)
}
func (r actionsRuntime) ShowLoading(ctx context.Context, value conversationoperations.LoadingIndicator) error {
	return r.service.ShowLoading(ctx, value)
}
func (r actionsRuntime) MarkRead(ctx context.Context, value conversationoperations.ReadTarget) error {
	return r.service.MarkRead(ctx, value)
}
func (r actionsRuntime) MarkReadByToken(ctx context.Context, value conversationoperations.ReadToken) error {
	return r.service.MarkReadByToken(ctx, value)
}

type audienceRuntime struct {
	runtimeCloser
	service *audience.ProviderQueries
}

func (r audienceRuntime) Get(ctx context.Context, id audience.ProviderAudienceID, shared bool) (audience.ProviderAudience, error) {
	return r.service.Get(ctx, id, shared)
}
func (r audienceRuntime) List(ctx context.Context, page audience.PageRequest, shared bool) (audience.ProviderAudiencePage, error) {
	return r.service.List(ctx, page, shared)
}

type contactsRuntime struct {
	runtimeCloser
	service *contactdirectory.ConversationQueries
}

func (r contactsRuntime) ListFollowers(ctx context.Context, page contactdirectory.PageRequest) (contactdirectory.RecipientPage, error) {
	return r.service.ListFollowers(ctx, page)
}
func (r contactsRuntime) GetSummary(ctx context.Context, ref contactdirectory.RecipientReference) (contactdirectory.ConversationSummary, error) {
	return r.service.GetSummary(ctx, ref)
}
func (r contactsRuntime) ListMembers(ctx context.Context, ref contactdirectory.RecipientReference, page contactdirectory.PageRequest) (contactdirectory.RecipientPage, error) {
	return r.service.ListMembers(ctx, ref, page)
}
func (r contactsRuntime) GetMemberProfile(ctx context.Context, ref contactdirectory.RecipientReference, id contactdirectory.RecipientID) (contactdirectory.ProfileSnapshot, error) {
	return r.service.GetMemberProfile(ctx, ref, id)
}

type couponsRuntime struct {
	runtimeCloser
	service *promotion.CouponQueries
}

func (r couponsRuntime) List(ctx context.Context, page promotion.CouponPageRequest) (promotion.CouponPage, error) {
	return r.service.List(ctx, page)
}
func (r couponsRuntime) Get(ctx context.Context, id promotion.CouponID) (promotion.CouponView, error) {
	return r.service.Get(ctx, id)
}

type insightsRuntime struct {
	runtimeCloser
	service *insightsreporting.Reports
}

func (r insightsRuntime) Demographics(ctx context.Context) (insightsreporting.DemographicsReport, error) {
	return r.service.Demographics(ctx)
}
func (r insightsRuntime) Deliveries(ctx context.Context, q insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	return r.service.Deliveries(ctx, q)
}
func (r insightsRuntime) Followers(ctx context.Context, q insightsreporting.DailyQuery) (insightsreporting.CountReport, error) {
	return r.service.Followers(ctx, q)
}
func (r insightsRuntime) MessageEvent(ctx context.Context, q insightsreporting.MessageEventQuery) (insightsreporting.MessageEventReport, error) {
	return r.service.MessageEvent(ctx, q)
}
func (r insightsRuntime) StatisticsPerUnit(ctx context.Context, q insightsreporting.UnitStatisticsQuery) (insightsreporting.StatisticsReport, error) {
	return r.service.StatisticsPerUnit(ctx, q)
}
func (r insightsRuntime) PresentationSummary(ctx context.Context, q insightsreporting.PresentationStatisticsQuery) (insightsreporting.StatisticsReport, error) {
	return r.service.PresentationSummary(ctx, q)
}
func (r insightsRuntime) PresentationDaily(ctx context.Context, q insightsreporting.PresentationStatisticsQuery) (insightsreporting.StatisticsReport, error) {
	return r.service.PresentationDaily(ctx, q)
}

type membershipsRuntime struct {
	runtimeCloser
	service *membership.Queries
}

func (r membershipsRuntime) ListMemberships(ctx context.Context, page membership.PageRequest) (membership.MembershipPage, error) {
	return r.service.ListMemberships(ctx, page)
}
func (r membershipsRuntime) GetSubscription(ctx context.Context, id membership.RecipientID) (membership.SubscriptionView, error) {
	return r.service.GetSubscription(ctx, id)
}
func (r membershipsRuntime) ListMembers(ctx context.Context, id membership.MembershipID, page membership.PageRequest) (membership.MemberPage, error) {
	return r.service.ListMembers(ctx, id, page)
}

type menusRuntime struct {
	runtimeCloser
	service *experiencepresentation.ProviderQueries
}

func (r menusRuntime) List(ctx context.Context) ([]experiencepresentation.ProviderSurface, error) {
	return r.service.List(ctx)
}
func (r menusRuntime) Get(ctx context.Context, id experiencepresentation.ProviderSurfaceID) (experiencepresentation.ProviderSurface, error) {
	return r.service.Get(ctx, id)
}
func (r menusRuntime) Default(ctx context.Context) (experiencepresentation.ProviderSurface, error) {
	return r.service.Default(ctx)
}
func (r menusRuntime) ListAliases(ctx context.Context) ([]experiencepresentation.ProviderAlias, error) {
	return r.service.ListAliases(ctx)
}
func (r menusRuntime) GetAlias(ctx context.Context, id experiencepresentation.AliasID) (experiencepresentation.ProviderAlias, error) {
	return r.service.GetAlias(ctx, id)
}
func (r menusRuntime) ForRecipient(ctx context.Context, id experiencepresentation.RecipientID) (experiencepresentation.ProviderSurface, error) {
	return r.service.ForRecipient(ctx, id)
}
func (r menusRuntime) BatchProgress(ctx context.Context, id string) (experiencepresentation.BatchProgress, error) {
	return r.service.BatchProgress(ctx, id)
}

type channelInspectionRuntime struct {
	runtimeCloser
	providerconfiguration.ChannelReader
	insightsreporting.DeliveryReader
}

type contentRuntime struct {
	runtimeCloser
	service *mediatransfer.Service
}

func (r contentRuntime) MessageContent(ctx context.Context, id string, variant mediatransfer.ContentVariant) (mediatransfer.BinaryContent, error) {
	return r.service.MessageContent(ctx, id, variant)
}
func (r contentRuntime) MessageContentStatus(ctx context.Context, id string) (mediatransfer.PreparationStatus, error) {
	return r.service.MessageContentStatus(ctx, id)
}

type menuImageRuntime struct {
	runtimeCloser
	service *mediatransfer.Service
}

func (r menuImageRuntime) PresentationImage(ctx context.Context, id string) (mediatransfer.BinaryContent, error) {
	return r.service.PresentationImage(ctx, id)
}

type menuManagementRuntime struct {
	runtimeCloser
	experiencepresentation.ProviderCommandGateway
}

type menuImageUploadRuntime struct {
	runtimeCloser
	service *mediatransfer.Service
}

func (r menuImageUploadRuntime) StorePresentationImage(ctx context.Context, id string, content mediatransfer.BinaryContent) error {
	return r.service.StorePresentationImage(ctx, id, content)
}

type eligibilityRuntime struct {
	runtimeCloser
	messagingdelivery.PhoneDeliveryGateway
	insightsreporting.PhoneDeliveryReader
	promotion.ProviderCouponGateway
}

type linkTokenRuntime struct {
	runtimeCloser
	service *accountlinktoken.Service
}

func (r linkTokenRuntime) Issue(ctx context.Context, subject accountlinktoken.Subject) (accountlinktoken.Token, error) {
	return r.service.Issue(ctx, subject)
}

type statusRuntime struct {
	runtimeCloser
	reader messagingdelivery.EventStatusReader
}

func (r statusRuntime) Inspect(ctx context.Context, state messagingdelivery.EventState, limit int, now time.Time) (messagingdelivery.EventStatusReport, error) {
	return r.reader.Inspect(ctx, state, limit, now)
}

func (r RuntimeAdapters) Push() PushRuntime {
	return pushRuntime{runtimeCloser{r.Close}, r.Messaging}
}
func (r RuntimeAdapters) Profile() ProfileRuntime {
	return profileRuntime{runtimeCloser{r.Close}, r.Profiles}
}
func (r RuntimeAdapters) ValidationRuntime() ValidationRuntime {
	return validationRuntime{runtimeCloser{r.Close}, r.Validation}
}
func (r RuntimeAdapters) LinkToken() LinkTokenRuntime {
	return linkTokenRuntime{runtimeCloser{r.Close}, r.AccountLink}
}
func (r RuntimeAdapters) ActionsRuntime() ActionsRuntime {
	return actionsRuntime{runtimeCloser{r.Close}, r.Actions}
}
func (r RuntimeAdapters) Eligibility() EligibilityRuntime {
	return eligibilityRuntime{runtimeCloser{r.Close}, r.PhoneDelivery, r.PhoneReports, r.CouponCreator}
}
func (r RuntimeAdapters) AudienceRuntime() AudienceRuntime {
	return audienceRuntime{runtimeCloser{r.Close}, r.Audiences}
}
func (r RuntimeAdapters) ContactsRuntime() ContactsRuntime {
	return contactsRuntime{runtimeCloser{r.Close}, r.Contacts}
}
func (r RuntimeAdapters) CouponsRuntime() CouponsRuntime {
	return couponsRuntime{runtimeCloser{r.Close}, r.Coupons}
}
func (r RuntimeAdapters) InsightsRuntime() InsightsRuntime {
	return insightsRuntime{runtimeCloser{r.Close}, r.Insights}
}
func (r RuntimeAdapters) MembershipsRuntime() MembershipsRuntime {
	return membershipsRuntime{runtimeCloser{r.Close}, r.Memberships}
}
func (r RuntimeAdapters) MenusRuntime() MenusRuntime {
	return menusRuntime{runtimeCloser{r.Close}, r.Menus}
}
func (r RuntimeAdapters) ChannelInspection() ChannelInspectionRuntime {
	return channelInspectionRuntime{runtimeCloser{r.Close}, r.Channel, r.DeliveryReports}
}
func (r RuntimeAdapters) ContentRuntime() ContentRuntime {
	return contentRuntime{runtimeCloser{r.Close}, r.Content}
}
func (r RuntimeAdapters) MenuImageRuntime() MenuImageRuntime {
	return menuImageRuntime{runtimeCloser{r.Close}, r.MenuImageReader}
}
func (r RuntimeAdapters) MenuManagement() MenuManagementRuntime {
	return menuManagementRuntime{runtimeCloser{r.Close}, r.MenuCommands}
}
func (r RuntimeAdapters) MenuImageUpload() MenuImageUploadRuntime {
	return menuImageUploadRuntime{runtimeCloser{r.Close}, r.RichMenuImages}
}
func (r RuntimeAdapters) Webhook() WebhookRuntime {
	return webhookRuntime{runtimeCloser{r.Close}, r.WebhookAdmin}
}
func (r RuntimeAdapters) AudienceManagement() AudienceManagementRuntime {
	return audienceManagementRuntime{runtimeCloser{r.Close}, r.AudienceManager}
}
func (r RuntimeAdapters) BulkMessagingRuntime() BulkMessagingRuntime {
	return bulkMessagingRuntime{runtimeCloser{r.Close}, r.BulkMessaging}
}

func NewStatusRuntime(reader messagingdelivery.EventStatusReader, close func() error) StatusRuntime {
	return statusRuntime{runtimeCloser{close}, reader}
}

type tokenRuntime struct {
	service *credentiallifecycle.Service
	close   func()
}

func (r tokenRuntime) Close() { r.close() }
func (r tokenRuntime) IssueStatelessWithClient(ctx context.Context, grant string, credential credentiallifecycle.ClientCredential) (credentiallifecycle.IssuedCredential, error) {
	return r.service.IssueStatelessWithClient(ctx, grant, credential)
}
func (r tokenRuntime) IssueStatelessWithAssertion(ctx context.Context, grant string, credential credentiallifecycle.AssertionCredential) (credentiallifecycle.IssuedCredential, error) {
	return r.service.IssueStatelessWithAssertion(ctx, grant, credential)
}
func (r tokenRuntime) ValidKeyIDs(ctx context.Context, credential credentiallifecycle.AssertionCredential) ([]string, error) {
	return r.service.ValidKeyIDs(ctx, credential)
}
func (r tokenRuntime) IssueJWT(ctx context.Context, grant string, credential credentiallifecycle.AssertionCredential) (credentiallifecycle.IssuedCredential, error) {
	return r.service.IssueJWT(ctx, grant, credential)
}
func (r tokenRuntime) VerifyJWT(ctx context.Context, token string) (credentiallifecycle.Verification, error) {
	return r.service.VerifyJWT(ctx, token)
}
func (r tokenRuntime) RevokeJWT(ctx context.Context, credential credentiallifecycle.ClientCredential, token string) error {
	return r.service.RevokeJWT(ctx, credential, token)
}
func (r tokenRuntime) IssueShortLived(ctx context.Context, grant string, credential credentiallifecycle.ClientCredential) (credentiallifecycle.IssuedCredential, error) {
	return r.service.IssueShortLived(ctx, grant, credential)
}
func (r tokenRuntime) VerifyShortLived(ctx context.Context, token string) (credentiallifecycle.Verification, error) {
	return r.service.VerifyShortLived(ctx, token)
}
func (r tokenRuntime) RevokeShortLived(ctx context.Context, token string) error {
	return r.service.RevokeShortLived(ctx, token)
}

func NewTokenRuntime(service *credentiallifecycle.Service, close func()) TokenRuntime {
	return tokenRuntime{service: service, close: close}
}
