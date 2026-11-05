package experiencepresentation

import (
	"context"
	"errors"
	"strings"
	"time"
)

type ProviderSurfaceID string
type AliasID string
type RecipientID string

type ProviderSurface struct {
	ID        ProviderSurfaceID
	Name      string
	Size      string
	Selected  bool
	CreatedAt time.Time
}

type ProviderAlias struct {
	ID        AliasID
	SurfaceID ProviderSurfaceID
}

type BatchProgress struct {
	RequestID string
	Status    string
	Completed int
	Failed    int
}

type ProviderReader interface {
	ListSurfaces(context.Context) ([]ProviderSurface, error)
	GetSurface(context.Context, ProviderSurfaceID) (ProviderSurface, error)
	GetDefaultSurface(context.Context) (ProviderSurface, error)
	ListAliases(context.Context) ([]ProviderAlias, error)
	GetAlias(context.Context, AliasID) (ProviderAlias, error)
	GetRecipientSurface(context.Context, RecipientID) (ProviderSurface, error)
	GetBatchProgress(context.Context, string) (BatchProgress, error)
}

type ProviderQueries struct{ reader ProviderReader }

func NewProviderQueries(reader ProviderReader) (*ProviderQueries, error) {
	if reader == nil {
		return nil, errors.New("presentation provider reader is required")
	}
	return &ProviderQueries{reader: reader}, nil
}

func (q *ProviderQueries) List(ctx context.Context) ([]ProviderSurface, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	return q.reader.ListSurfaces(ctx)
}

func (q *ProviderQueries) Get(ctx context.Context, id ProviderSurfaceID) (ProviderSurface, error) {
	if err := validateContext(ctx); err != nil {
		return ProviderSurface{}, err
	}
	if !validID(string(id)) {
		return ProviderSurface{}, errors.New("surface ID is required")
	}
	return q.reader.GetSurface(ctx, id)
}

func (q *ProviderQueries) Default(ctx context.Context) (ProviderSurface, error) {
	if err := validateContext(ctx); err != nil {
		return ProviderSurface{}, err
	}
	return q.reader.GetDefaultSurface(ctx)
}

func (q *ProviderQueries) ListAliases(ctx context.Context) ([]ProviderAlias, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	return q.reader.ListAliases(ctx)
}

func (q *ProviderQueries) GetAlias(ctx context.Context, id AliasID) (ProviderAlias, error) {
	if err := validateContext(ctx); err != nil {
		return ProviderAlias{}, err
	}
	if !validID(string(id)) {
		return ProviderAlias{}, errors.New("alias ID is required")
	}
	return q.reader.GetAlias(ctx, id)
}

func (q *ProviderQueries) ForRecipient(ctx context.Context, id RecipientID) (ProviderSurface, error) {
	if err := validateContext(ctx); err != nil {
		return ProviderSurface{}, err
	}
	if !validID(string(id)) {
		return ProviderSurface{}, errors.New("recipient ID is required")
	}
	return q.reader.GetRecipientSurface(ctx, id)
}

func (q *ProviderQueries) BatchProgress(ctx context.Context, requestID string) (BatchProgress, error) {
	if err := validateContext(ctx); err != nil {
		return BatchProgress{}, err
	}
	if !validID(requestID) {
		return BatchProgress{}, errors.New("batch request ID is required")
	}
	return q.reader.GetBatchProgress(ctx, requestID)
}

type ProviderRegion struct {
	Bounds          [4]int
	ActionReference string
}

type SurfaceDefinition struct {
	Name        string
	Size        string
	ChatBarText string
	Selected    bool
	Regions     []ProviderRegion
}

type RecipientAssignment struct {
	RecipientID RecipientID
	SurfaceID   ProviderSurfaceID
}

type BatchCommand struct {
	Link   []RecipientAssignment
	Unlink []RecipientID
}

type SurfaceImage struct {
	ContentType string
	Data        []byte
}

// ProviderDocument preserves the complete provider command surface while the
// richer provider-independent action algebra is introduced incrementally. It
// is owned here, not by a shared ports package; SDK decoding stays in adapters.
type ProviderDocument struct{ Value any }

type ProviderCommandGateway interface {
	CreateRichMenu(context.Context, ProviderDocument) error
	ValidateRichMenu(context.Context, ProviderDocument) error
	CreateRichMenuAlias(context.Context, ProviderDocument) error
	UpdateRichMenuAlias(context.Context, string, ProviderDocument) error
	BulkLinkRichMenu(context.Context, ProviderDocument) error
	BulkUnlinkRichMenu(context.Context, ProviderDocument) error
	RichMenuBatch(context.Context, ProviderDocument) error
	ValidateRichMenuBatch(context.Context, ProviderDocument) error
	SetDefaultRichMenu(context.Context, string) error
	CancelDefaultRichMenu(context.Context) error
	LinkRichMenuToUser(context.Context, string, string) error
	UnlinkRichMenuFromUser(context.Context, string) error
	DeleteRichMenu(context.Context, string) error
	DeleteRichMenuAlias(context.Context, string) error
}

type ProviderManager interface {
	CreateSurface(context.Context, SurfaceDefinition) (ProviderSurface, error)
	ValidateSurface(context.Context, SurfaceDefinition) error
	CreateAlias(context.Context, ProviderAlias) error
	UpdateAlias(context.Context, ProviderAlias) error
	DeleteAlias(context.Context, AliasID) error
	SetDefault(context.Context, ProviderSurfaceID) error
	ClearDefault(context.Context) error
	AssignRecipient(context.Context, RecipientAssignment) error
	UnassignRecipient(context.Context, RecipientID) error
	DeleteSurface(context.Context, ProviderSurfaceID) error
	ExecuteBatch(context.Context, BatchCommand) (BatchProgress, error)
	ValidateBatch(context.Context, BatchCommand) error
	UploadImage(context.Context, ProviderSurfaceID, SurfaceImage) error
}

type ProviderOperations struct{ manager ProviderManager }

func NewProviderOperations(manager ProviderManager) (*ProviderOperations, error) {
	if manager == nil {
		return nil, errors.New("presentation provider manager is required")
	}
	return &ProviderOperations{manager: manager}, nil
}

func (o *ProviderOperations) Create(ctx context.Context, definition SurfaceDefinition) (ProviderSurface, error) {
	if err := validateContext(ctx); err != nil {
		return ProviderSurface{}, err
	}
	if err := validateSurface(definition); err != nil {
		return ProviderSurface{}, err
	}
	return o.manager.CreateSurface(ctx, definition)
}

func (o *ProviderOperations) Validate(ctx context.Context, definition SurfaceDefinition) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if err := validateSurface(definition); err != nil {
		return err
	}
	return o.manager.ValidateSurface(ctx, definition)
}

func (o *ProviderOperations) SaveAlias(ctx context.Context, alias ProviderAlias, update bool) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if !validID(string(alias.ID)) || !validID(string(alias.SurfaceID)) {
		return errors.New("alias and surface IDs are required")
	}
	if update {
		return o.manager.UpdateAlias(ctx, alias)
	}
	return o.manager.CreateAlias(ctx, alias)
}

func (o *ProviderOperations) DeleteAlias(ctx context.Context, id AliasID) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if !validID(string(id)) {
		return errors.New("alias ID is required")
	}
	return o.manager.DeleteAlias(ctx, id)
}

func (o *ProviderOperations) SetDefault(ctx context.Context, id ProviderSurfaceID) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if !validID(string(id)) {
		return errors.New("surface ID is required")
	}
	return o.manager.SetDefault(ctx, id)
}

func (o *ProviderOperations) ClearDefault(ctx context.Context) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	return o.manager.ClearDefault(ctx)
}

func (o *ProviderOperations) Assign(ctx context.Context, assignment RecipientAssignment) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if err := validateAssignment(assignment); err != nil {
		return err
	}
	return o.manager.AssignRecipient(ctx, assignment)
}

func (o *ProviderOperations) Unassign(ctx context.Context, id RecipientID) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if !validID(string(id)) {
		return errors.New("recipient ID is required")
	}
	return o.manager.UnassignRecipient(ctx, id)
}

func (o *ProviderOperations) Delete(ctx context.Context, id ProviderSurfaceID) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if !validID(string(id)) {
		return errors.New("surface ID is required")
	}
	return o.manager.DeleteSurface(ctx, id)
}

func (o *ProviderOperations) ExecuteBatch(ctx context.Context, command BatchCommand) (BatchProgress, error) {
	if err := validateContext(ctx); err != nil {
		return BatchProgress{}, err
	}
	if err := validateBatch(command); err != nil {
		return BatchProgress{}, err
	}
	return o.manager.ExecuteBatch(ctx, command)
}

func (o *ProviderOperations) ValidateBatch(ctx context.Context, command BatchCommand) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if err := validateBatch(command); err != nil {
		return err
	}
	return o.manager.ValidateBatch(ctx, command)
}

func (o *ProviderOperations) UploadImage(ctx context.Context, id ProviderSurfaceID, image SurfaceImage) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if !validID(string(id)) || len(image.Data) == 0 || (image.ContentType != "image/png" && image.ContentType != "image/jpeg") {
		return errors.New("surface ID and PNG or JPEG image are required")
	}
	return o.manager.UploadImage(ctx, id, image)
}

func validateSurface(value SurfaceDefinition) error {
	if !validID(value.Name) || !validID(value.Size) || !validID(value.ChatBarText) || len(value.Regions) == 0 {
		return errors.New("surface name, size, chat bar text, and regions are required")
	}
	for _, region := range value.Regions {
		if !validID(region.ActionReference) {
			return errors.New("surface region action is required")
		}
	}
	return nil
}

func validateBatch(command BatchCommand) error {
	if len(command.Link) == 0 && len(command.Unlink) == 0 {
		return errors.New("batch requires assignments")
	}
	for _, assignment := range command.Link {
		if err := validateAssignment(assignment); err != nil {
			return err
		}
	}
	for _, id := range command.Unlink {
		if !validID(string(id)) {
			return errors.New("batch recipient ID is required")
		}
	}
	return nil
}

func validateAssignment(value RecipientAssignment) error {
	if !validID(string(value.RecipientID)) || !validID(string(value.SurfaceID)) {
		return errors.New("recipient and surface IDs are required")
	}
	return nil
}

func validID(value string) bool { return value != "" && value == strings.TrimSpace(value) }
