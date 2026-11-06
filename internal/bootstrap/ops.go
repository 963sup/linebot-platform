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
	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
	lineinfra "github.com/963sup/linebot-platform/internal/infrastructure/line"
)

// LINEOpsRuntime owns one operator command's LINE capabilities and managed
// transport. Provider clients and transports remain private to bootstrap.
//
// Capabilities whose LINE adapters have not yet implemented their new
// consumer-owned contracts are deliberately absent. Bootstrap must not route
// those operations through legacy global ports merely to preserve wiring.
type LINEOpsRuntime struct {
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
	outbound        *httpclient.Managed
}

func newLINEOpsRuntime() (*LINEOpsRuntime, error) {
	transport, err := httpclient.New(httpclient.Config{Timeout: httpclient.DefaultTimeout}, nil)
	if err != nil {
		return nil, err
	}
	return &LINEOpsRuntime{outbound: transport}, nil
}

func (r *LINEOpsRuntime) fail(err error) (*LINEOpsRuntime, error) {
	_ = r.Close()
	return nil, err
}

func newLINEMessagingOpsRuntime(token string) (*LINEOpsRuntime, error) {
	runtime, err := newLINEOpsRuntime()
	if err != nil {
		return nil, err
	}
	client, err := lineinfra.NewMessagingClientWithHTTPClient(token, runtime.outbound.Client)
	if err != nil {
		return runtime.fail(err)
	}
	sender, err := line.NewSender(client)
	if err != nil {
		return runtime.fail(err)
	}
	delivery, err := messagingdelivery.NewDeliveryService(sender)
	if err != nil {
		return runtime.fail(err)
	}
	validation, err := messagingdelivery.NewValidationService(sender)
	if err != nil {
		return runtime.fail(err)
	}
	profiles, err := contactdirectory.NewQueries(sender)
	if err != nil {
		return runtime.fail(err)
	}
	contacts, err := contactdirectory.NewConversationQueries(sender)
	if err != nil {
		return runtime.fail(err)
	}
	webhookAdmin, err := providerconfiguration.New(sender)
	if err != nil {
		return runtime.fail(err)
	}
	actions, err := conversationoperations.New(sender)
	if err != nil {
		return runtime.fail(err)
	}
	accountLink, err := accountlinktoken.New(sender)
	if err != nil {
		return runtime.fail(err)
	}
	coupons, err := promotion.NewCouponQueries(sender)
	if err != nil {
		return runtime.fail(err)
	}
	memberships, err := membership.NewQueries(sender)
	if err != nil {
		return runtime.fail(err)
	}
	menus, err := experiencepresentation.NewProviderQueries(sender)
	if err != nil {
		return runtime.fail(err)
	}
	runtime.Messaging = delivery
	runtime.BulkMessaging = delivery
	runtime.PhoneDelivery = sender
	runtime.PhoneReports = sender
	runtime.Validation = validation
	runtime.Profiles = profiles
	runtime.Contacts = contacts
	runtime.WebhookAdmin = webhookAdmin
	runtime.Channel = sender
	runtime.DeliveryReports = sender
	runtime.Coupons = coupons
	runtime.CouponCreator = sender
	runtime.Memberships = memberships
	runtime.Menus = menus
	runtime.MenuCommands = sender
	runtime.Actions = actions
	runtime.AccountLink = accountLink
	return runtime, nil
}

func newLINEInsightOpsRuntime(token string) (*LINEOpsRuntime, error) {
	runtime, err := newLINEOpsRuntime()
	if err != nil {
		return nil, err
	}
	insightClient, err := lineinfra.NewInsightClientWithHTTPClient(token, runtime.outbound.Client)
	if err != nil {
		return runtime.fail(err)
	}
	insightReader, err := line.NewInsightReader(insightClient)
	if err != nil {
		return runtime.fail(err)
	}
	reports, err := insightsreporting.NewReports(insightReader)
	if err != nil {
		return runtime.fail(err)
	}
	runtime.Insights = reports
	return runtime, nil
}

func newLINEBlobOpsRuntime(token string) (*LINEOpsRuntime, error) {
	runtime, err := newLINEOpsRuntime()
	if err != nil {
		return nil, err
	}
	blobClient, err := lineinfra.NewMessagingBlobClientWithHTTPClient(token, runtime.outbound.Client)
	if err != nil {
		return runtime.fail(err)
	}
	blobSender, err := line.NewBlobSender(blobClient)
	if err != nil {
		return runtime.fail(err)
	}
	media, err := mediatransfer.New(blobSender)
	if err != nil {
		return runtime.fail(err)
	}
	runtime.Content = media
	runtime.RichMenuImages = media
	runtime.MenuImageReader = media
	return runtime, nil
}

func newLINEAudienceOpsRuntime(token string) (*LINEOpsRuntime, error) {
	runtime, err := newLINEOpsRuntime()
	if err != nil {
		return nil, err
	}
	audienceClient, err := lineinfra.NewAudienceClientWithHTTPClient(token, runtime.outbound.Client)
	if err != nil {
		return runtime.fail(err)
	}
	audienceReader, err := line.NewAudienceReader(audienceClient)
	if err != nil {
		return runtime.fail(err)
	}
	audiences, err := audience.NewProviderQueries(audienceReader)
	if err != nil {
		return runtime.fail(err)
	}
	audienceBlobClient, err := lineinfra.NewAudienceBlobClientWithHTTPClient(token, runtime.outbound.Client)
	if err != nil {
		return runtime.fail(err)
	}
	audienceAdapter, err := line.NewAudienceManager(audienceClient, audienceBlobClient)
	if err != nil {
		return runtime.fail(err)
	}
	audienceManager, err := audience.NewProviderOperations(audienceAdapter)
	if err != nil {
		return runtime.fail(err)
	}
	runtime.Audiences = audiences
	runtime.AudienceManager = audienceManager
	return runtime, nil
}

// Close satisfies the CLI runtime contract. Releasing idle HTTP connections
// cannot report an error; the managed transport preserves idempotent cleanup.
func (r *LINEOpsRuntime) Close() error {
	if r != nil && r.outbound != nil {
		r.outbound.Close()
	}
	return nil
}
