package valueobjects

type Status string

const (
	StatusPending Status = "pending"
	StatusSending Status = "sending"
	StatusSent    Status = "sent"
	StatusExpired Status = "expired"
	StatusUnknown Status = "unknown"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusSending, StatusSent, StatusExpired, StatusUnknown:
		return true
	default:
		return false
	}
}

func (s Status) Terminal() bool {
	return s == StatusSent || s == StatusExpired || s == StatusUnknown
}
