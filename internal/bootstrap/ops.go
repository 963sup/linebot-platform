package bootstrap

import (
	"github.com/963sup/linebot-platform/internal/adapter/outbound/line"
	accountlinktoken "github.com/963sup/linebot-platform/internal/application/usecases/account-link-token"
	"github.com/963sup/linebot-platform/internal/application/usecases/audience"
	contactdirectory "github.com/963sup/linebot-platform/internal/application/usecases/contact-directory"
	conversationoperations "github.com/963sup/linebot-platform/internal/application/usecases/conversation-operations"
	experiencepresentation "github.com/963sup/linebot-platform/internal/application/usecases/experience-presentation"
	insightsreporting "github.com/963sup/linebot-platform/internal/application/usecases/insights-reporting"
	mediatransfer "github.com/963sup/linebot-platform/internal/application/usecases/media-transfer"
	"github.com/963sup/linebot-platform/internal/application/usecases/membership"
	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/963sup/linebot-platform/internal/application/usecases/promotion"
	providerconfiguration "github.com/963sup/linebot-platform/internal/application/usecases/provider-configuration"
	requestvalidation "github.com/963sup/linebot-platform/internal/application/usecases/request-validation"
	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
	lineinfra "github.com/963sup/linebot-platform/internal/infrastructure/line"
)

// LINEOpsRuntime is the composition root for operator-only LINE capabilities.
// Every exported capability is an Application service backed by a LINE
// adapter; provider clients and transports remain private to bootstrap.
//
// Capabilities whose LINE adapters have not yet implemented their new
// consumer-owned contracts are deliberately absent. Bootstrap must not route
// those operations through legacy global ports merely to preserve wiring.
type LINEOpsRuntime struct {
	Messaging       *messagingdelivery.DeliveryService
	BulkMessaging   *messagingdelivery.DeliveryService
	PhoneDelivery   messagingdelivery.PhoneDeliveryGateway
	PhoneReports    insightsreporting.PhoneDeliveryReader
	Validation      *requestvalidation.Service
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
	delivery, err := messagingdelivery.NewDeliveryService(sender)
	if err != nil {
		return cleanup(err)
	}
	validation, err := requestvalidation.New(sender)
	if err != nil {
		return cleanup(err)
	}
	profiles, err := contactdirectory.NewQueries(sender)
	if err != nil {
		return cleanup(err)
	}
	contacts, err := contactdirectory.NewConversationQueries(sender)
	if err != nil {
		return cleanup(err)
	}
	webhookAdmin, err := providerconfiguration.New(sender)
	if err != nil {
		return cleanup(err)
	}
	actions, err := conversationoperations.New(sender)
	if err != nil {
		return cleanup(err)
	}
	accountLink, err := accountlinktoken.New(sender)
	if err != nil {
		return cleanup(err)
	}
	coupons, err := promotion.NewCouponQueries(sender)
	if err != nil {
		return cleanup(err)
	}
	memberships, err := membership.NewQueries(sender)
	if err != nil {
		return cleanup(err)
	}
	menus, err := experiencepresentation.NewProviderQueries(sender)
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
	reports, err := insightsreporting.NewReports(insightReader)
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
	media, err := mediatransfer.New(blobSender)
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
	audiences, err := audience.NewProviderQueries(audienceReader)
	if err != nil {
		return cleanup(err)
	}
	audienceBlobClient, err := lineinfra.NewAudienceBlobClientWithHTTPClient(token, transport.Client)
	if err != nil {
		return cleanup(err)
	}
	audienceAdapter, err := line.NewAudienceManager(audienceClient, audienceBlobClient)
	if err != nil {
		return cleanup(err)
	}
	audienceManager, err := audience.NewProviderOperations(audienceAdapter)
	if err != nil {
		return cleanup(err)
	}

	return &LINEOpsRuntime{
		Messaging: delivery, BulkMessaging: delivery, PhoneDelivery: sender, PhoneReports: sender, Validation: validation,
		Profiles: profiles, Contacts: contacts, WebhookAdmin: webhookAdmin,
		Channel: sender, DeliveryReports: sender,
		Audiences: audiences, AudienceManager: audienceManager,
		Coupons: coupons, CouponCreator: sender,
		Memberships: memberships, Menus: menus, MenuCommands: sender, Insights: reports,
		Content: media, RichMenuImages: media, MenuImageReader: media,
		Actions: actions, AccountLink: accountLink, outbound: transport,
	}, nil
}

// Close satisfies the CLI runtime contract. Releasing idle HTTP connections
// cannot report an error; the managed transport preserves idempotent cleanup.
func (r *LINEOpsRuntime) Close() error {
	if r != nil && r.outbound != nil {
		r.outbound.Close()
	}
	return nil
}
