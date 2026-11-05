package valueobjects_test

import (
	"testing"

	valueobjects "github.com/963sup/linebot-platform/internal/domain/experience-presentation/value-objects"
)

func TestPresentationDefensivelyCopiesRegions(t *testing.T) {
	t.Parallel()

	region, err := valueobjects.NewRegion("primary", "open", "")
	if err != nil {
		t.Fatalf("NewRegion(...) error = %v, want nil", err)
	}
	input := []valueobjects.Region{region}
	presentation, err := valueobjects.NewPresentation(input)
	if err != nil {
		t.Fatalf("NewPresentation(...) error = %v, want nil", err)
	}
	input[0] = valueobjects.Region{}
	if got := presentation.Regions()[0]; !got.Valid() {
		t.Errorf("Presentation.Regions()[0] after input mutation = %v, want valid region", got)
	}
	output := presentation.Regions()
	output[0] = valueobjects.Region{}
	if got := presentation.Regions()[0]; !got.Valid() {
		t.Errorf("Presentation.Regions()[0] after output mutation = %v, want valid region", got)
	}
}

func TestNewPresentationRejectsDuplicateRegions(t *testing.T) {
	t.Parallel()

	region, err := valueobjects.NewRegion("primary", "open", "")
	if err != nil {
		t.Fatalf("NewRegion(...) error = %v, want nil", err)
	}
	if _, err := valueobjects.NewPresentation([]valueobjects.Region{region, region}); err == nil {
		t.Error("NewPresentation(duplicate regions) error = nil, want non-nil")
	}
}

func TestNewRegionRequiresIDAndAction(t *testing.T) {
	t.Parallel()

	for _, input := range [][2]string{{"", "open"}, {"primary", ""}} {
		if _, err := valueobjects.NewRegion(input[0], input[1], ""); err == nil {
			t.Errorf("NewRegion(%q, %q, %q) error = nil, want non-nil", input[0], input[1], "")
		}
	}
}
