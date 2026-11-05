package valueobjects

import (
	"errors"
	"strings"
)

// Output is the final, provider-neutral result description of an execution.
type Output struct{ detail string }

func NewOutput(detail string) (Output, error) {
	if detail == "" || detail != strings.TrimSpace(detail) {
		return Output{}, errors.New("invalid automation output")
	}
	return Output{detail: detail}, nil
}

func (o Output) Detail() string { return o.detail }
func (o Output) Valid() bool    { return o.detail != "" }
