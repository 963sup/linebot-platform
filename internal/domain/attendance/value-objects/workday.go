package valueobjects

import (
	"errors"
	"time"
)

const workdayLayout = "2006-01-02"

// Workday is a calendar day in the configured business timezone.
type Workday string

func NewWorkday(value string) (Workday, error) {
	parsed, err := time.Parse(workdayLayout, value)
	if err != nil || parsed.Format(workdayLayout) != value {
		return "", errors.New("invalid workday")
	}
	return Workday(value), nil
}

func WorkdayAt(at time.Time, location *time.Location) (Workday, error) {
	if at.IsZero() || location == nil {
		return "", errors.New("workday time and location are required")
	}
	return NewWorkday(at.In(location).Format(workdayLayout))
}

func (v Workday) String() string { return string(v) }
func (v Workday) Valid() bool    { _, err := NewWorkday(string(v)); return err == nil }
