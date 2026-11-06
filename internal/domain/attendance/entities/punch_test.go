package entities

import (
	"errors"
	"testing"
	"time"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

func TestPunchCorrectionRetainsAuditSnapshotsAndEvidence(t *testing.T) {
	punch, recordedAt, initialLocation := reviewPunchFixture(t)
	if err := punch.Confirm(recordedAt.Add(time.Minute), "reviewer", "verified anomaly"); err != nil {
		t.Fatal(err)
	}
	correctedLocation := locationAssessment(t, initialLocation.EvidenceRef(), 5, 5, false, false)
	correctedAt := recordedAt.Add(30 * time.Minute)
	if err := punch.Correct(correctedAt, correctedLocation, false, valueobjects.ReviewProvenanceManualOverride, recordedAt.Add(2*time.Minute), "reviewer", "manual correction"); err != nil {
		t.Fatal(err)
	}

	history := punch.ReviewHistory()
	if len(history) != 2 {
		t.Fatalf("history length=%d, want 2", len(history))
	}
	confirmation, correction := history[0], history[1]
	if confirmation.Provenance() != valueobjects.ReviewProvenanceHumanReview || !confirmation.Before().Equal(confirmation.After()) {
		t.Fatal("confirmation did not retain an unchanged human-review snapshot")
	}
	if correction.Provenance() != valueobjects.ReviewProvenanceManualOverride {
		t.Fatalf("correction provenance=%q", correction.Provenance())
	}
	if !correction.Before().EffectiveAt().Equal(recordedAt) || !correction.Before().Location().Equal(initialLocation) || !correction.Before().TimeAnomaly() {
		t.Fatal("correction before snapshot does not match the prior punch state")
	}
	if !correction.After().EffectiveAt().Equal(correctedAt) || !correction.After().Location().Equal(correctedLocation) || correction.After().TimeAnomaly() {
		t.Fatal("correction after snapshot does not match the corrected punch state")
	}
	if punch.Location().EvidenceRef() != initialLocation.EvidenceRef() {
		t.Fatal("correction changed the evidence reference")
	}
}

func TestPunchCorrectionRequiresManualOverrideAndSameEvidence(t *testing.T) {
	for _, tc := range []struct {
		name       string
		provenance valueobjects.ReviewProvenance
		evidence   string
	}{
		{name: "non-manual provenance", provenance: valueobjects.ReviewProvenanceHumanReview, evidence: "evidence-1"},
		{name: "different evidence", provenance: valueobjects.ReviewProvenanceManualOverride, evidence: "evidence-2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			punch, recordedAt, initialLocation := reviewPunchFixture(t)
			ref, _ := valueobjects.NewEvidenceRef(tc.evidence)
			location := locationAssessment(t, ref, 5, 5, false, false)
			if err := punch.Correct(recordedAt.Add(time.Minute), location, false, tc.provenance, recordedAt.Add(2*time.Minute), "reviewer", "correction"); !errors.Is(err, ErrInvalidReview) {
				t.Fatalf("Correct error=%v, want ErrInvalidReview", err)
			}
			if !punch.EffectiveAt().Equal(recordedAt) || !punch.Location().Equal(initialLocation) || len(punch.ReviewHistory()) != 0 {
				t.Fatal("rejected correction mutated punch")
			}
		})
	}
}

func TestRestorePunchValidatesAuditHistory(t *testing.T) {
	punch, recordedAt, initialLocation := reviewPunchFixture(t)
	if err := punch.Confirm(recordedAt.Add(time.Minute), "reviewer", "verified anomaly"); err != nil {
		t.Fatal(err)
	}
	correctedLocation := locationAssessment(t, initialLocation.EvidenceRef(), 5, 5, false, false)
	if err := punch.Correct(recordedAt.Add(30*time.Minute), correctedLocation, false, valueobjects.ReviewProvenanceManualOverride, recordedAt.Add(2*time.Minute), "reviewer", "manual correction"); err != nil {
		t.Fatal(err)
	}
	if err := punch.Void(recordedAt.Add(3*time.Minute), "reviewer", "duplicate record"); err != nil {
		t.Fatal(err)
	}

	validHistory := punch.ReviewHistory()
	if _, err := RestorePunch(punch.ID(), punch.Kind(), punch.SubmissionID(), punch.RecordedAt(), punch.EffectiveAt(), punch.Location(), punch.TimeAnomaly(), punch.ReviewState(), validHistory); err != nil {
		t.Fatalf("valid history rejected: %v", err)
	}

	invalidOrder := append([]ReviewAction(nil), validHistory...)
	invalidOrder[1].at = invalidOrder[0].At().Add(-time.Second)

	illegalTransition := append([]ReviewAction(nil), validHistory...)
	unchanged := illegalTransition[1].Before()
	illegalTransition[1] = reviewAction(t, valueobjects.ReviewConfirmed, valueobjects.ReviewProvenanceHumanReview, illegalTransition[1].At(), "confirm twice", unchanged, unchanged)

	brokenChain := append([]ReviewAction(nil), validHistory...)
	brokenBefore := reviewSnapshot(t, recordedAt.Add(time.Second), initialLocation, true)
	brokenChain[1] = reviewAction(t, valueobjects.ReviewCorrected, valueobjects.ReviewProvenanceManualOverride, brokenChain[1].At(), "broken chain", brokenBefore, brokenChain[1].After())

	changedEvidence := append([]ReviewAction(nil), validHistory...)
	otherRef, _ := valueobjects.NewEvidenceRef("evidence-2")
	otherLocation := locationAssessment(t, otherRef, 5, 5, false, false)
	changedEvidence[1].after.location = otherLocation

	tests := []struct {
		name        string
		effectiveAt time.Time
		location    valueobjects.LocationAssessment
		anomaly     bool
		state       valueobjects.ReviewState
		history     []ReviewAction
	}{
		{name: "history order", effectiveAt: punch.EffectiveAt(), location: punch.Location(), anomaly: punch.TimeAnomaly(), state: punch.ReviewState(), history: invalidOrder},
		{name: "illegal transition", effectiveAt: punch.EffectiveAt(), location: punch.Location(), anomaly: punch.TimeAnomaly(), state: punch.ReviewState(), history: illegalTransition},
		{name: "broken before-after chain", effectiveAt: punch.EffectiveAt(), location: punch.Location(), anomaly: punch.TimeAnomaly(), state: punch.ReviewState(), history: brokenChain},
		{name: "changed evidence reference", effectiveAt: punch.EffectiveAt(), location: punch.Location(), anomaly: punch.TimeAnomaly(), state: punch.ReviewState(), history: changedEvidence},
		{name: "final state mismatch", effectiveAt: punch.EffectiveAt(), location: punch.Location(), anomaly: punch.TimeAnomaly(), state: valueobjects.ReviewCorrected, history: validHistory},
		{name: "final snapshot mismatch", effectiveAt: recordedAt, location: initialLocation, anomaly: true, state: punch.ReviewState(), history: validHistory},
		{name: "missing correction history", effectiveAt: recordedAt.Add(time.Minute), location: correctedLocation, anomaly: false, state: valueobjects.ReviewClear},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := RestorePunch(punch.ID(), punch.Kind(), punch.SubmissionID(), recordedAt, tc.effectiveAt, tc.location, tc.anomaly, tc.state, tc.history); !errors.Is(err, ErrInvalidEntity) {
				t.Fatalf("RestorePunch error=%v, want ErrInvalidEntity", err)
			}
		})
	}
}

func TestRestorePunchDerivesInitialReviewStateFromAnomalies(t *testing.T) {
	punch, recordedAt, initialLocation := reviewPunchFixture(t)
	if _, err := RestorePunch(punch.ID(), punch.Kind(), punch.SubmissionID(), recordedAt, recordedAt, initialLocation, true, valueobjects.ReviewRequired, nil); err != nil {
		t.Fatalf("valid initial anomaly rejected: %v", err)
	}
	clearLocation := locationAssessment(t, initialLocation.EvidenceRef(), 5, 5, false, false)
	clearSnapshot := reviewSnapshot(t, recordedAt, clearLocation, false)
	confirmation := reviewAction(t, valueobjects.ReviewConfirmed, valueobjects.ReviewProvenanceHumanReview, recordedAt.Add(time.Minute), "invalid confirmation", clearSnapshot, clearSnapshot)

	if _, err := RestorePunch(punch.ID(), punch.Kind(), punch.SubmissionID(), recordedAt, recordedAt, clearLocation, false, valueobjects.ReviewConfirmed, []ReviewAction{confirmation}); !errors.Is(err, ErrInvalidEntity) {
		t.Fatalf("RestorePunch error=%v, want ErrInvalidEntity", err)
	}
}

func reviewPunchFixture(t *testing.T) (Punch, time.Time, valueobjects.LocationAssessment) {
	t.Helper()
	punchID, _ := valueobjects.NewPunchID("punch-1")
	submissionID, _ := valueobjects.NewSubmissionID("submission-1")
	ref, _ := valueobjects.NewEvidenceRef("evidence-1")
	location := locationAssessment(t, ref, 200, 100, true, true)
	recordedAt := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	punch, err := NewPunch(punchID, valueobjects.ClockIn, submissionID, recordedAt, location, true)
	if err != nil {
		t.Fatal(err)
	}
	return punch, recordedAt, location
}

func locationAssessment(t *testing.T, ref valueobjects.EvidenceRef, distance, accuracy float64, outside, lowAccuracy bool) valueobjects.LocationAssessment {
	t.Helper()
	location, err := valueobjects.NewLocationAssessment(distance, accuracy, outside, lowAccuracy, ref, "location-v1")
	if err != nil {
		t.Fatal(err)
	}
	return location
}

func reviewSnapshot(t *testing.T, effectiveAt time.Time, location valueobjects.LocationAssessment, timeAnomaly bool) ReviewSnapshot {
	t.Helper()
	snapshot, err := NewReviewSnapshot(effectiveAt, location, timeAnomaly)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func reviewAction(t *testing.T, state valueobjects.ReviewState, provenance valueobjects.ReviewProvenance, at time.Time, reason string, before, after ReviewSnapshot) ReviewAction {
	t.Helper()
	action, err := NewReviewAction(state, provenance, at, "reviewer", reason, before, after)
	if err != nil {
		t.Fatal(err)
	}
	return action
}
