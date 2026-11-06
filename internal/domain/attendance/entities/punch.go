package entities

import (
	"errors"
	"strings"
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

var ErrInvalidReview = errors.New("invalid attendance review action")

// ReviewSnapshot is the immutable business state on one side of a review action.
type ReviewSnapshot struct {
	effectiveAt time.Time
	location    valueobjects.LocationAssessment
	timeAnomaly bool
}

func NewReviewSnapshot(effectiveAt time.Time, location valueobjects.LocationAssessment, timeAnomaly bool) (ReviewSnapshot, error) {
	if effectiveAt.IsZero() || !location.Valid() {
		return ReviewSnapshot{}, ErrInvalidReview
	}
	return ReviewSnapshot{effectiveAt: effectiveAt.UTC(), location: location, timeAnomaly: timeAnomaly}, nil
}

func (s ReviewSnapshot) EffectiveAt() time.Time                    { return s.effectiveAt }
func (s ReviewSnapshot) Location() valueobjects.LocationAssessment { return s.location }
func (s ReviewSnapshot) TimeAnomaly() bool                         { return s.timeAnomaly }
func (s ReviewSnapshot) Equal(other ReviewSnapshot) bool {
	return s.effectiveAt.Equal(other.effectiveAt) && s.location.Equal(other.location) && s.timeAnomaly == other.timeAnomaly
}

type ReviewAction struct {
	state      valueobjects.ReviewState
	provenance valueobjects.ReviewProvenance
	at         time.Time
	reviewer   string
	reason     string
	before     ReviewSnapshot
	after      ReviewSnapshot
}

func NewReviewAction(state valueobjects.ReviewState, provenance valueobjects.ReviewProvenance, at time.Time, reviewer, reason string, before, after ReviewSnapshot) (ReviewAction, error) {
	if (state != valueobjects.ReviewConfirmed && state != valueobjects.ReviewCorrected && state != valueobjects.ReviewVoided) || !provenance.Valid() || at.IsZero() || strings.TrimSpace(reviewer) == "" || strings.TrimSpace(reason) == "" || !before.valid() || !after.valid() {
		return ReviewAction{}, ErrInvalidReview
	}
	if state == valueobjects.ReviewCorrected {
		if provenance != valueobjects.ReviewProvenanceManualOverride || before.Location().EvidenceRef() != after.Location().EvidenceRef() {
			return ReviewAction{}, ErrInvalidReview
		}
	} else if provenance != valueobjects.ReviewProvenanceHumanReview || !before.Equal(after) {
		return ReviewAction{}, ErrInvalidReview
	}
	return ReviewAction{state: state, provenance: provenance, at: at.UTC(), reviewer: reviewer, reason: reason, before: before, after: after}, nil
}

func (a ReviewAction) State() valueobjects.ReviewState           { return a.state }
func (a ReviewAction) Provenance() valueobjects.ReviewProvenance { return a.provenance }
func (a ReviewAction) At() time.Time                             { return a.at }
func (a ReviewAction) Reviewer() string                          { return a.reviewer }
func (a ReviewAction) Reason() string                            { return a.reason }
func (a ReviewAction) Before() ReviewSnapshot                    { return a.before }
func (a ReviewAction) After() ReviewSnapshot                     { return a.after }

type Punch struct {
	id           valueobjects.PunchID
	kind         valueobjects.PunchKind
	submissionID valueobjects.SubmissionID
	recordedAt   time.Time
	effectiveAt  time.Time
	location     valueobjects.LocationAssessment
	timeAnomaly  bool
	reviewState  valueobjects.ReviewState
	history      []ReviewAction
}

func NewPunch(id valueobjects.PunchID, kind valueobjects.PunchKind, submissionID valueobjects.SubmissionID, serverTime time.Time, location valueobjects.LocationAssessment, timeAnomaly bool) (Punch, error) {
	if !id.Valid() || (kind != valueobjects.ClockIn && kind != valueobjects.ClockOut) || !submissionID.Valid() || serverTime.IsZero() || !location.Valid() {
		return Punch{}, ErrInvalidEntity
	}
	state := initialReviewState(location, timeAnomaly)
	serverTime = serverTime.UTC()
	return Punch{id: id, kind: kind, submissionID: submissionID, recordedAt: serverTime, effectiveAt: serverTime, location: location, timeAnomaly: timeAnomaly, reviewState: state}, nil
}

func RestorePunch(id valueobjects.PunchID, kind valueobjects.PunchKind, submissionID valueobjects.SubmissionID, recordedAt, effectiveAt time.Time, location valueobjects.LocationAssessment, timeAnomaly bool, state valueobjects.ReviewState, history []ReviewAction) (Punch, error) {
	if !id.Valid() || (kind != valueobjects.ClockIn && kind != valueobjects.ClockOut) || !submissionID.Valid() || recordedAt.IsZero() || effectiveAt.IsZero() || !location.Valid() || !state.Valid() {
		return Punch{}, ErrInvalidEntity
	}
	recordedAt, effectiveAt = recordedAt.UTC(), effectiveAt.UTC()
	finalSnapshot, err := NewReviewSnapshot(effectiveAt, location, timeAnomaly)
	if err != nil || !validRestoredHistory(recordedAt, state, finalSnapshot, history) {
		return Punch{}, ErrInvalidEntity
	}
	return Punch{id: id, kind: kind, submissionID: submissionID, recordedAt: recordedAt, effectiveAt: effectiveAt, location: location, timeAnomaly: timeAnomaly, reviewState: state, history: append([]ReviewAction(nil), history...)}, nil
}

func (p Punch) ID() valueobjects.PunchID                  { return p.id }
func (p Punch) Kind() valueobjects.PunchKind              { return p.kind }
func (p Punch) SubmissionID() valueobjects.SubmissionID   { return p.submissionID }
func (p Punch) RecordedAt() time.Time                     { return p.recordedAt }
func (p Punch) EffectiveAt() time.Time                    { return p.effectiveAt }
func (p Punch) Location() valueobjects.LocationAssessment { return p.location }
func (p Punch) TimeAnomaly() bool                         { return p.timeAnomaly }
func (p Punch) ReviewState() valueobjects.ReviewState     { return p.reviewState }
func (p Punch) ReviewHistory() []ReviewAction             { return append([]ReviewAction(nil), p.history...) }

func (p *Punch) Confirm(reviewedAt time.Time, reviewer, reason string) error {
	if p == nil {
		return ErrInvalidReview
	}
	return p.appendReview(valueobjects.ReviewConfirmed, valueobjects.ReviewProvenanceHumanReview, reviewedAt, reviewer, reason, p.snapshot())
}

func (p *Punch) Correct(effectiveAt time.Time, location valueobjects.LocationAssessment, timeAnomaly bool, provenance valueobjects.ReviewProvenance, reviewedAt time.Time, reviewer, reason string) error {
	if p == nil || provenance != valueobjects.ReviewProvenanceManualOverride || effectiveAt.IsZero() || !location.Valid() || location.EvidenceRef() != p.location.EvidenceRef() {
		return ErrInvalidReview
	}
	after, err := NewReviewSnapshot(effectiveAt, location, timeAnomaly)
	if err != nil {
		return err
	}
	return p.appendReview(valueobjects.ReviewCorrected, provenance, reviewedAt, reviewer, reason, after)
}

func (p *Punch) Void(reviewedAt time.Time, reviewer, reason string) error {
	if p == nil {
		return ErrInvalidReview
	}
	return p.appendReview(valueobjects.ReviewVoided, valueobjects.ReviewProvenanceHumanReview, reviewedAt, reviewer, reason, p.snapshot())
}

func (p *Punch) appendReview(state valueobjects.ReviewState, provenance valueobjects.ReviewProvenance, at time.Time, reviewer, reason string, after ReviewSnapshot) error {
	if p == nil || !reviewTransitionAllowed(p.reviewState, state) || at.Before(p.recordedAt) || len(p.history) > 0 && at.Before(p.history[len(p.history)-1].At()) {
		return ErrInvalidReview
	}
	action, err := NewReviewAction(state, provenance, at, reviewer, reason, p.snapshot(), after)
	if err != nil {
		return err
	}
	p.reviewState = state
	p.effectiveAt, p.location, p.timeAnomaly = after.EffectiveAt(), after.Location(), after.TimeAnomaly()
	p.history = append(p.history, action)
	return nil
}

func (p Punch) snapshot() ReviewSnapshot {
	return ReviewSnapshot{effectiveAt: p.effectiveAt, location: p.location, timeAnomaly: p.timeAnomaly}
}

func (s ReviewSnapshot) valid() bool {
	return !s.effectiveAt.IsZero() && s.location.Valid()
}

func initialReviewState(location valueobjects.LocationAssessment, timeAnomaly bool) valueobjects.ReviewState {
	if timeAnomaly || location.RequiresReview() {
		return valueobjects.ReviewRequired
	}
	return valueobjects.ReviewClear
}

func reviewTransitionAllowed(from, to valueobjects.ReviewState) bool {
	switch to {
	case valueobjects.ReviewConfirmed:
		return from == valueobjects.ReviewRequired
	case valueobjects.ReviewCorrected, valueobjects.ReviewVoided:
		return from.Valid() && from != valueobjects.ReviewVoided
	default:
		return false
	}
}

func validRestoredHistory(recordedAt time.Time, finalState valueobjects.ReviewState, finalSnapshot ReviewSnapshot, history []ReviewAction) bool {
	currentSnapshot := finalSnapshot
	if len(history) > 0 {
		currentSnapshot = history[0].Before()
	}
	if !currentSnapshot.EffectiveAt().Equal(recordedAt) {
		return false
	}
	currentState := initialReviewState(currentSnapshot.Location(), currentSnapshot.TimeAnomaly())
	var previousAt time.Time
	for i, action := range history {
		validated, err := NewReviewAction(action.State(), action.Provenance(), action.At(), action.Reviewer(), action.Reason(), action.Before(), action.After())
		if err != nil || validated != action || action.At().Before(recordedAt) || i > 0 && action.At().Before(previousAt) || !action.Before().Equal(currentSnapshot) || !reviewTransitionAllowed(currentState, action.State()) {
			return false
		}
		currentState, currentSnapshot, previousAt = action.State(), action.After(), action.At()
	}
	return currentState == finalState && currentSnapshot.Equal(finalSnapshot)
}
