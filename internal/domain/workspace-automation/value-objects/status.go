package valueobjects

// Status is the lifecycle state of an automation request.
type Status string

const (
	StatusRequested Status = "requested"
	StatusApproved  Status = "approved"
	StatusExecuting Status = "executing"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusUnknown   Status = "unknown"
)

func (s Status) Valid() bool {
	switch s {
	case StatusRequested, StatusApproved, StatusExecuting, StatusSucceeded, StatusFailed, StatusUnknown:
		return true
	default:
		return false
	}
}

func (s Status) Terminal() bool {
	return s == StatusSucceeded || s == StatusFailed || s == StatusUnknown
}
