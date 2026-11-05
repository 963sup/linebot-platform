package valueobjects

import (
	"errors"
	"strings"
)

// Action names the business operation requested from workspace automation.
type Action struct{ name string }

func NewAction(name string) (Action, error) {
	if name == "" || name != strings.TrimSpace(name) || strings.ContainsAny(name, "\r\n\t") {
		return Action{}, errors.New("invalid automation action")
	}
	return Action{name: name}, nil
}

func (a Action) Name() string { return a.name }
func (a Action) Valid() bool  { return a.name != "" }
