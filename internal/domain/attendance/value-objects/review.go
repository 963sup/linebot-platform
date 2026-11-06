package valueobjects

type ReviewState string
type ReviewProvenance string

const (
	ReviewClear     ReviewState = "clear"
	ReviewRequired  ReviewState = "required"
	ReviewConfirmed ReviewState = "confirmed"
	ReviewCorrected ReviewState = "corrected"
	ReviewVoided    ReviewState = "voided"
)

const (
	ReviewProvenanceHumanReview    ReviewProvenance = "human_review"
	ReviewProvenanceManualOverride ReviewProvenance = "manual_override"
)

func (s ReviewState) Valid() bool {
	switch s {
	case ReviewClear, ReviewRequired, ReviewConfirmed, ReviewCorrected, ReviewVoided:
		return true
	default:
		return false
	}
}

func (p ReviewProvenance) Valid() bool {
	return p == ReviewProvenanceHumanReview || p == ReviewProvenanceManualOverride
}

type PunchKind string

const (
	ClockIn  PunchKind = "clock_in"
	ClockOut PunchKind = "clock_out"
)

type SubmissionOutcome string

const (
	SubmissionRecorded  SubmissionOutcome = "recorded"
	SubmissionDuplicate SubmissionOutcome = "duplicate"
)
