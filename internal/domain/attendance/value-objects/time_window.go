package valueobjects

import (
	"errors"
	"time"
)

type TimeWindow struct{ opensAt, closesAt time.Time }

func NewTimeWindow(opensAt, closesAt time.Time) (TimeWindow, error) {
	if opensAt.IsZero() || closesAt.Before(opensAt) {
		return TimeWindow{}, errors.New("invalid time window")
	}
	return TimeWindow{opensAt: opensAt.UTC(), closesAt: closesAt.UTC()}, nil
}
func (w TimeWindow) OpensAt() time.Time  { return w.opensAt }
func (w TimeWindow) ClosesAt() time.Time { return w.closesAt }
func (w TimeWindow) Contains(at time.Time) bool {
	return !at.IsZero() && !at.Before(w.opensAt) && !at.After(w.closesAt)
}
