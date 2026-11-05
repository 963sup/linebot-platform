package audience

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	DefaultPageSize     = 20
	MaxPageSize         = 100
	MaxAudienceFileSize = 20 << 20
)

// ProviderAudienceID is an external audience identifier. It deliberately does
// not replace the provider-independent AudienceID used by the aggregate.
type ProviderAudienceID int64

type PageRequest struct {
	Number int64
	Size   int64
}

type ProviderAudience struct {
	ID          ProviderAudienceID
	Description string
	Status      string
	Shared      bool
	CreatedAt   time.Time
}

type ProviderAudiencePage struct {
	Audiences []ProviderAudience
	Page      int64
	NextPage  int64
}

// ProviderReader is the consumer-owned port for provider audience queries.
type ProviderReader interface {
	GetAudience(context.Context, ProviderAudienceID, bool) (ProviderAudience, error)
	ListAudiences(context.Context, PageRequest, bool) (ProviderAudiencePage, error)
}

type ProviderQueries struct{ reader ProviderReader }

func NewProviderQueries(reader ProviderReader) (*ProviderQueries, error) {
	if reader == nil {
		return nil, errors.New("audience provider reader is required")
	}
	return &ProviderQueries{reader: reader}, nil
}

func (q *ProviderQueries) Get(ctx context.Context, id ProviderAudienceID, shared bool) (ProviderAudience, error) {
	if err := validateContext(ctx); err != nil {
		return ProviderAudience{}, err
	}
	if id <= 0 {
		return ProviderAudience{}, errors.New("provider audience ID must be positive")
	}
	return q.reader.GetAudience(ctx, id, shared)
}

func (q *ProviderQueries) List(ctx context.Context, page PageRequest, shared bool) (ProviderAudiencePage, error) {
	if err := validateContext(ctx); err != nil {
		return ProviderAudiencePage{}, err
	}
	if page.Number < 1 || page.Size < 1 || page.Size > MaxPageSize {
		return ProviderAudiencePage{}, errors.New("audience page must be positive and size must be 1-100")
	}
	return q.reader.ListAudiences(ctx, page, shared)
}

type CreateGroupCommand struct {
	Description string
	Recipients  []string
}

type AppendRecipientsCommand struct {
	ID         ProviderAudienceID
	Recipients []string
}

type CreateInteractionGroupCommand struct {
	Description     string
	SourceRequestID string
}

type AudienceFile struct {
	Data              []byte
	Description       string
	UploadDescription string
	IFAAudience       bool
}

// ProviderManager is intentionally semantic: provider JSON and SDK request
// objects must be constructed by the outbound adapter.
type ProviderManager interface {
	CreateGroup(context.Context, CreateGroupCommand) (ProviderAudience, error)
	AppendRecipients(context.Context, AppendRecipientsCommand) error
	CreateClickGroup(context.Context, CreateInteractionGroupCommand) (ProviderAudience, error)
	CreateImpressionGroup(context.Context, CreateInteractionGroupCommand) (ProviderAudience, error)
	UpdateDescription(context.Context, ProviderAudienceID, string) error
	DeleteGroup(context.Context, ProviderAudienceID) error
	CreateFromFile(context.Context, AudienceFile) (ProviderAudience, error)
	AppendFromFile(context.Context, ProviderAudienceID, AudienceFile) error
}

type ProviderOperations struct{ manager ProviderManager }

func NewProviderOperations(manager ProviderManager) (*ProviderOperations, error) {
	if manager == nil {
		return nil, errors.New("audience provider manager is required")
	}
	return &ProviderOperations{manager: manager}, nil
}

func (o *ProviderOperations) CreateGroup(ctx context.Context, command CreateGroupCommand) (ProviderAudience, error) {
	if err := validateContext(ctx); err != nil {
		return ProviderAudience{}, err
	}
	if !validDescription(command.Description) || !validRecipients(command.Recipients) {
		return ProviderAudience{}, errors.New("audience description and recipients are required")
	}
	return o.manager.CreateGroup(ctx, command)
}

func (o *ProviderOperations) AppendRecipients(ctx context.Context, command AppendRecipientsCommand) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if command.ID <= 0 || !validRecipients(command.Recipients) {
		return errors.New("audience ID and recipients are required")
	}
	return o.manager.AppendRecipients(ctx, command)
}

func (o *ProviderOperations) CreateInteractionGroup(ctx context.Context, command CreateInteractionGroupCommand, impression bool) (ProviderAudience, error) {
	if err := validateContext(ctx); err != nil {
		return ProviderAudience{}, err
	}
	if !validDescription(command.Description) || !validText(command.SourceRequestID) {
		return ProviderAudience{}, errors.New("interaction audience source is required")
	}
	if impression {
		return o.manager.CreateImpressionGroup(ctx, command)
	}
	return o.manager.CreateClickGroup(ctx, command)
}

func (o *ProviderOperations) UpdateDescription(ctx context.Context, id ProviderAudienceID, description string) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if id <= 0 || !validDescription(description) {
		return errors.New("audience ID and description are required")
	}
	return o.manager.UpdateDescription(ctx, id, description)
}

func (o *ProviderOperations) Delete(ctx context.Context, id ProviderAudienceID) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if id <= 0 {
		return errors.New("provider audience ID must be positive")
	}
	return o.manager.DeleteGroup(ctx, id)
}

func (o *ProviderOperations) CreateFromFile(ctx context.Context, file AudienceFile) (ProviderAudience, error) {
	if err := validateContext(ctx); err != nil {
		return ProviderAudience{}, err
	}
	if err := validateAudienceFile(file, true); err != nil {
		return ProviderAudience{}, err
	}
	return o.manager.CreateFromFile(ctx, file)
}

func (o *ProviderOperations) AppendFromFile(ctx context.Context, id ProviderAudienceID, file AudienceFile) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if id <= 0 {
		return errors.New("provider audience ID must be positive")
	}
	if err := validateAudienceFile(file, false); err != nil {
		return err
	}
	return o.manager.AppendFromFile(ctx, id, file)
}

func validateAudienceFile(file AudienceFile, creating bool) error {
	if len(file.Data) == 0 || len(file.Data) > MaxAudienceFileSize || file.UploadDescription != strings.TrimSpace(file.UploadDescription) {
		return errors.New("bounded audience file and valid upload description are required")
	}
	if creating && !validDescription(file.Description) {
		return errors.New("audience file description is required")
	}
	if !creating && (file.Description != "" || file.IFAAudience) {
		return errors.New("append file accepts only upload description")
	}
	return nil
}

func validRecipients(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if !validText(value) {
			return false
		}
	}
	return true
}

func validDescription(value string) bool { return validText(value) }
func validText(value string) bool        { return value != "" && value == strings.TrimSpace(value) }
