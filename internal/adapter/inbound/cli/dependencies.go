package cli

import (
	"context"
	"encoding/json"
	"time"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/application/usecases/channel"
	"github.com/963sup/linebot-platform/internal/application/usecases/contacts"
	"github.com/963sup/linebot-platform/internal/application/usecases/engagement"
	"github.com/963sup/linebot-platform/internal/application/usecases/insights"
	"github.com/963sup/linebot-platform/internal/application/usecases/messaging"
	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
	"golang.org/x/oauth2"
)

// PushRuntime is the narrow dependency required by the push command.
// Runtime construction belongs to the composition root, not this adapter.
type PushRuntime interface {
	Push(ctx context.Context, recipient string, messages []outbound.Message, retryKey string) error
	Close() error
}

type ProfileRuntime interface {
	Get(ctx context.Context, userID string) (outbound.UserProfile, error)
	Close() error
}

type ValidationRuntime interface {
	Validate(ctx context.Context, kind string, messages []outbound.Message) error
	Close() error
}

type StatusRuntime interface {
	Inspect(ctx context.Context, state outbound.EventState, limit int, now time.Time) (outbound.EventStatusReport, error)
	Close() error
}

type WorkspaceRuntime interface {
	Execute(context.Context, string, workspace.Command, workspace.Approval) ([]byte, error)
}

type LinkTokenRuntime interface {
	IssueLinkToken(context.Context, string) (outbound.LINEResult, error)
	Close() error
}

type ActionsRuntime interface {
	Execute(context.Context, messaging.ConversationActionCommand) error
	Close() error
}

type EligibilityRuntime interface {
	Execute(context.Context, engagement.EligibilityCommand) (json.RawMessage, error)
	Close() error
}

type AudienceRuntime interface {
	Inspect(context.Context, engagement.AudienceQuery) (json.RawMessage, error)
	Close() error
}

type ContactsRuntime interface {
	Inspect(context.Context, contacts.Inspection) (outbound.LINEResult, error)
	Close() error
}

type CouponsRuntime interface {
	Inspect(context.Context, engagement.CouponQuery) (json.RawMessage, error)
	Close() error
}

type InsightsRuntime interface {
	Inspect(context.Context, insights.Query) (outbound.LINEResult, error)
	Close() error
}

type MembershipsRuntime interface {
	Inspect(context.Context, engagement.MembershipQuery) (json.RawMessage, error)
	Close() error
}

type MenusRuntime interface {
	Inspect(context.Context, engagement.RichMenuQuery) (json.RawMessage, error)
	Close() error
}

type ChannelInspectionRuntime interface {
	Inspect(context.Context, channel.Inspection) (outbound.LINEResult, error)
	Close() error
}

type ContentRuntime interface {
	Inspect(context.Context, contacts.ContentQuery) (outbound.BinaryContent, outbound.LINEResult, error)
	Close() error
}

type MenuImageRuntime interface {
	RichMenuImage(context.Context, string) (outbound.BinaryContent, error)
	Close() error
}

type MenuManagementRuntime interface {
	Execute(context.Context, engagement.MenuCommand) error
	Close() error
}

type MenuImageUploadRuntime interface {
	Upload(context.Context, string, string, []byte) error
	Close() error
}

type WebhookRuntime interface {
	Set(context.Context, string) error
	Test(context.Context, string) (outbound.WebhookEndpointTest, error)
	Close() error
}

type TokenRuntime interface {
	Execute(context.Context, channel.TokenCommand) (outbound.LINEResult, error)
	Close()
}

type AudienceManagementRuntime interface {
	Execute(context.Context, engagement.AudienceCommand) (json.RawMessage, error)
	Close() error
}

type BulkMessagingRuntime interface {
	Broadcast(context.Context, []outbound.Message, string) error
	Multicast(context.Context, []string, []outbound.Message, string) error
	Narrowcast(context.Context, outbound.LINERequest, string) error
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
