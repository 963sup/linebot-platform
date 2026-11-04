package bootstrap

import (
	"github.com/963sup/linebot-platform/internal/adapter/outbound/line"
	"github.com/963sup/linebot-platform/internal/application/usecases/channel"
	"github.com/963sup/linebot-platform/internal/application/usecases/contacts"
	"github.com/963sup/linebot-platform/internal/application/usecases/engagement"
	"github.com/963sup/linebot-platform/internal/application/usecases/insights"
	"github.com/963sup/linebot-platform/internal/application/usecases/messaging"
	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
	lineinfra "github.com/963sup/linebot-platform/internal/infrastructure/line"
)

// LINEOpsRuntime owns the HTTP transport used by explicit, local CLI actions.
// It is never exposed on the unauthenticated webhook process.
type LINEOpsRuntime struct {
	Messaging       *messaging.Service
	Validation      *messaging.ValidationService
	Profiles        *contacts.Profiles
	Channel         *channel.Inspector
	WebhookAdmin    *channel.WebhookConfigurator
	Menus           *engagement.RichMenuInspector
	MenuManager     *engagement.MenuManager
	Memberships     *engagement.MembershipInspector
	Coupons         *engagement.CouponInspector
	Contacts        *contacts.Inspector
	Insights        *insights.Inspector
	Audiences       *engagement.AudienceInspector
	AudienceManager *engagement.AudienceOperations
	BulkMessaging   *messaging.BulkSender
	Content         *contacts.ContentInspector
	RichMenuImages  *engagement.RichMenuImageManager
	Actions         *messaging.ConversationActions
	AccountLink     *contacts.AccountLinker
	Eligibility     *engagement.EligibilityOperations
	MenuImageReader *line.BlobSender
	outbound        *httpclient.Managed
}

func NewLINEOpsRuntime(token string) (*LINEOpsRuntime, error) {
	transport, err := httpclient.New(httpclient.Config{Timeout: httpclient.DefaultTimeout}, nil)
	if err != nil {
		return nil, err
	}
	cleanup := func(err error) (*LINEOpsRuntime, error) {
		transport.Close()
		return nil, err
	}
	client, err := lineinfra.NewMessagingClientWithHTTPClient(token, transport.Client)
	if err != nil {
		return cleanup(err)
	}
	sender, err := line.NewSender(client)
	if err != nil {
		return cleanup(err)
	}
	blobClient, err := lineinfra.NewMessagingBlobClientWithHTTPClient(token, transport.Client)
	if err != nil {
		return cleanup(err)
	}
	blobSender, err := line.NewBlobSender(blobClient)
	if err != nil {
		return cleanup(err)
	}
	content, err := contacts.NewContentInspector(blobSender)
	if err != nil {
		return cleanup(err)
	}
	richMenuImages, err := engagement.NewRichMenuImageManager(blobSender)
	if err != nil {
		return cleanup(err)
	}
	messages, err := messaging.NewService(sender)
	if err != nil {
		return cleanup(err)
	}
	profiles, err := contacts.NewProfiles(sender)
	if err != nil {
		return cleanup(err)
	}
	inspector, err := channel.NewInspector(sender)
	if err != nil {
		return cleanup(err)
	}
	menus, err := engagement.NewRichMenuInspector(sender)
	if err != nil {
		return cleanup(err)
	}
	contactsInspector, err := contacts.NewInspector(sender)
	if err != nil {
		return cleanup(err)
	}
	insightClient, err := lineinfra.NewInsightClientWithHTTPClient(token, transport.Client)
	if err != nil {
		return cleanup(err)
	}
	insightReader, err := line.NewInsightReader(insightClient)
	if err != nil {
		return cleanup(err)
	}
	inspectorInsights, err := insights.NewInspector(insightReader)
	if err != nil {
		return cleanup(err)
	}
	membership, err := engagement.NewMembershipInspector(sender)
	if err != nil {
		return cleanup(err)
	}
	coupon, err := engagement.NewCouponInspector(sender)
	if err != nil {
		return cleanup(err)
	}
	audienceClient, err := lineinfra.NewAudienceClientWithHTTPClient(token, transport.Client)
	if err != nil {
		return cleanup(err)
	}
	audienceReader, err := line.NewAudienceReader(audienceClient)
	if err != nil {
		return cleanup(err)
	}
	inspectorAudiences, err := engagement.NewAudienceInspector(audienceReader)
	if err != nil {
		return cleanup(err)
	}
	audienceBlobClient, err := lineinfra.NewAudienceBlobClientWithHTTPClient(token, transport.Client)
	if err != nil {
		return cleanup(err)
	}
	audienceManagerAdapter, err := line.NewAudienceManager(audienceClient, audienceBlobClient)
	if err != nil {
		return cleanup(err)
	}
	audienceManager, err := engagement.NewAudienceOperations(audienceManagerAdapter)
	if err != nil {
		return cleanup(err)
	}
	menuManager, err := engagement.NewMenuManager(sender)
	if err != nil {
		return cleanup(err)
	}
	bulkMessenger, err := messaging.NewBulkSender(sender)
	if err != nil {
		return cleanup(err)
	}
	validator, err := messaging.NewValidationService(sender)
	if err != nil {
		return cleanup(err)
	}
	webhookAdmin, err := channel.NewWebhookConfigurator(sender)
	if err != nil {
		return cleanup(err)
	}
	actions, err := messaging.NewConversationActions(sender)
	if err != nil {
		return cleanup(err)
	}
	accountLink, err := contacts.NewAccountLinker(sender)
	if err != nil {
		return cleanup(err)
	}
	eligibility, err := engagement.NewEligibilityOperations(sender)
	if err != nil {
		return cleanup(err)
	}
	return &LINEOpsRuntime{Actions: actions, AccountLink: accountLink, Eligibility: eligibility, MenuImageReader: blobSender, Messaging: messages, Validation: validator, Profiles: profiles, Channel: inspector, WebhookAdmin: webhookAdmin, Menus: menus, MenuManager: menuManager, RichMenuImages: richMenuImages, Memberships: membership, Coupons: coupon, Contacts: contactsInspector, Content: content, Insights: inspectorInsights, Audiences: inspectorAudiences, AudienceManager: audienceManager, BulkMessaging: bulkMessenger, outbound: transport}, nil
}

// Close satisfies the CLI runtime contract. Releasing idle HTTP connections
// cannot report an error; the managed transport preserves idempotent cleanup.
func (r *LINEOpsRuntime) Close() error {
	if r != nil && r.outbound != nil {
		r.outbound.Close()
	}
	return nil
}
