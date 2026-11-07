package valueobjects

type PunchKind string

const (
	ClockIn  PunchKind = "clock_in"
	ClockOut PunchKind = "clock_out"
)

func (v PunchKind) Valid() bool { return v == ClockIn || v == ClockOut }

type SubmissionOutcome string

const (
	SubmissionRecorded  SubmissionOutcome = "recorded"
	SubmissionDuplicate SubmissionOutcome = "already_recorded"
)
