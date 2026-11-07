package experiencepresentation

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrContextRequired = errors.New("context is required")

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return ErrContextRequired
	}
	return ctx.Err()
}

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

// ProviderDocument preserves the complete provider command surface while the
// richer provider-independent action algebra is introduced incrementally. It
// is owned here, not by a shared ports package; SDK decoding stays in adapters.
type ProviderDocument struct{ Value any }

type RichMenuDefinitionCommands interface {
	CreateRichMenu(context.Context, ProviderDocument) error
	ValidateRichMenu(context.Context, ProviderDocument) error
	DeleteRichMenu(context.Context, string) error
}

type RichMenuAliasCommands interface {
	CreateRichMenuAlias(context.Context, ProviderDocument) error
	UpdateRichMenuAlias(context.Context, string, ProviderDocument) error
	DeleteRichMenuAlias(context.Context, string) error
}

type RecipientRichMenuCommands interface {
	BulkLinkRichMenu(context.Context, ProviderDocument) error
	BulkUnlinkRichMenu(context.Context, ProviderDocument) error
	LinkRichMenuToUser(context.Context, string, string) error
	UnlinkRichMenuFromUser(context.Context, string) error
}

type RichMenuBatchCommands interface {
	RichMenuBatch(context.Context, ProviderDocument) error
	ValidateRichMenuBatch(context.Context, ProviderDocument) error
}

type DefaultRichMenuCommands interface {
	SetDefaultRichMenu(context.Context, string) error
	CancelDefaultRichMenu(context.Context) error
}

// ProviderCommandGateway is the compatibility composition of the narrow
// command capabilities. Consumers should accept the smallest leaf interface.
type ProviderCommandGateway interface {
	RichMenuDefinitionCommands
	RichMenuAliasCommands
	RecipientRichMenuCommands
	RichMenuBatchCommands
	DefaultRichMenuCommands
}

func validID(value string) bool { return value != "" && value == strings.TrimSpace(value) }
