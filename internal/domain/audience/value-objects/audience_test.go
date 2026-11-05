package valueobjects_test

import (
	"testing"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/audience/value-objects"
)

func TestNewAudienceID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "trimmed", raw: " audience-1 ", want: "audience-1"},
		{name: "empty", raw: "  ", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := valueobjects.NewAudienceID(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewAudienceID(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
			if got.String() != tt.want {
				t.Errorf("NewAudienceID(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestNewSelectionCriteriaRequiresSourceAndDefinition(t *testing.T) {
	t.Parallel()

	for _, input := range [][2]string{{"", "x"}, {"orders", ""}} {
		if _, err := valueobjects.NewSelectionCriteria(input[0], input[1]); err == nil {
			t.Errorf("NewSelectionCriteria(%q, %q) error = nil, want non-nil", input[0], input[1])
		}
	}
}
