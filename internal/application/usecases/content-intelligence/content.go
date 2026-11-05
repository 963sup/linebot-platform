package contentintelligence

import (
	"context"
	"errors"
	"strings"
	"time"
)

const MaxInputBytes = 1 << 20

const (
	AuditFailureGeneration   = "generation_failed"
	AuditFailureInvalidDraft = "invalid_draft"
)

var (
	ErrGenerationFailed      = errors.New("content generation failed")
	ErrGenerationAuditFailed = errors.New("content generation audit failed")
	errInvalidDraft          = errors.New("content generator returned an invalid draft")
)

// safeProcessError preserves a cause for internal classification while keeping
// provider and storage details out of boundary-facing error strings.
type safeProcessError struct {
	message error
	cause   error
}

func (e *safeProcessError) Error() string { return e.message.Error() }

func (e *safeProcessError) Unwrap() []error { return []error{e.message, e.cause} }

type GenerationPolicy struct {
	Purpose        string
	MaxOutputBytes int
}

type GenerationRequest struct {
	ID     string
	Input  string
	Policy GenerationPolicy
}

type ContentDraft struct {
	RequestID string
	Content   string
	Provider  string
	Model     string
	CreatedAt time.Time
}

type AuditOutcome string

const (
	AuditSucceeded AuditOutcome = "succeeded"
	AuditFailed    AuditOutcome = "failed"
)

type AuditEntry struct {
	RequestID  string
	Outcome    AuditOutcome
	Provider   string
	Model      string
	RecordedAt time.Time
	Failure    string
}

// ContentGenerator is implemented by an outbound model-provider adapter.
type ContentGenerator interface {
	GenerateDraft(context.Context, GenerationRequest) (ContentDraft, error)
}

// GenerationAudit records process outcomes without exposing provider SDK types.
type GenerationAudit interface {
	RecordGeneration(context.Context, AuditEntry) error
}

type Clock interface {
	Now() time.Time
}

type Process struct {
	generator ContentGenerator
	audit     GenerationAudit
	clock     Clock
}

func NewProcess(generator ContentGenerator, audit GenerationAudit, clock Clock) (*Process, error) {
	if generator == nil || audit == nil || clock == nil {
		return nil, errors.New("content generator, generation audit, and clock are required")
	}
	return &Process{generator: generator, audit: audit, clock: clock}, nil
}

func (p *Process) GenerateDraft(ctx context.Context, request GenerationRequest) (ContentDraft, error) {
	if err := validateContext(ctx); err != nil {
		return ContentDraft{}, err
	}
	if err := validateRequest(request); err != nil {
		return ContentDraft{}, err
	}
	draft, generateErr := p.generator.GenerateDraft(ctx, request)
	entry := AuditEntry{RequestID: request.ID, Outcome: AuditFailed, RecordedAt: p.clock.Now()}
	if generateErr == nil {
		if err := validateDraft(request.ID, draft); err != nil {
			generateErr = err
		} else {
			entry.Outcome = AuditSucceeded
			entry.Provider = draft.Provider
			entry.Model = draft.Model
		}
	}
	if generateErr != nil {
		entry.Failure = auditFailure(generateErr)
	}
	auditErr := p.audit.RecordGeneration(ctx, entry)
	if generateErr != nil {
		generateErr = &safeProcessError{message: ErrGenerationFailed, cause: generateErr}
	}
	if auditErr != nil {
		auditErr = &safeProcessError{message: ErrGenerationAuditFailed, cause: auditErr}
	}
	if processErr := errors.Join(generateErr, auditErr); processErr != nil {
		return ContentDraft{}, processErr
	}
	return draft, nil
}

func auditFailure(err error) string {
	if errors.Is(err, errInvalidDraft) {
		return AuditFailureInvalidDraft
	}
	return AuditFailureGeneration
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	return ctx.Err()
}

func validateRequest(request GenerationRequest) error {
	if !validText(request.ID) || !validText(request.Policy.Purpose) {
		return errors.New("generation request ID and purpose are required")
	}
	if request.Input == "" || len(request.Input) > MaxInputBytes {
		return errors.New("generation input is required and must be within bounds")
	}
	if request.Policy.MaxOutputBytes <= 0 || request.Policy.MaxOutputBytes > MaxInputBytes {
		return errors.New("generation output limit must be within bounds")
	}
	return nil
}

func validateDraft(requestID string, draft ContentDraft) error {
	if draft.RequestID != requestID || !validText(draft.Provider) || !validText(draft.Model) || draft.Content == "" || draft.CreatedAt.IsZero() {
		return errInvalidDraft
	}
	return nil
}

func validText(value string) bool {
	return value != "" && strings.TrimSpace(value) == value
}
