package cli

import "testing"

var (
	_ OperatorRuntime            = (*OperatorDomainRuntime)(nil)
	_ AudienceDefinitionsRuntime = (*OperatorDomainRuntime)(nil)
	_ InteractionSurfacesRuntime = (*OperatorDomainRuntime)(nil)
	_ PromotionsRuntime          = (*OperatorDomainRuntime)(nil)
	_ AutomationRequestsRuntime  = (*OperatorDomainRuntime)(nil)
)

type stubAudienceOperator struct{ AudienceOperator }
type stubSurfaceOperator struct{ SurfaceOperator }
type stubPromotionOperator struct{ PromotionOperator }
type stubAutomationOperator struct{ AutomationOperator }

func TestOperatorDomainRuntimeValidatesDependencies(t *testing.T) {
	a, s := &stubAudienceOperator{}, &stubSurfaceOperator{}
	p, w := &stubPromotionOperator{}, &stubAutomationOperator{}
	for name, build := range map[string]func() (*OperatorDomainRuntime, error){
		"audience": func() (*OperatorDomainRuntime, error) {
			return NewOperatorDomainRuntime(nil, s, p, w, func() error { return nil })
		},
		"surface": func() (*OperatorDomainRuntime, error) {
			return NewOperatorDomainRuntime(a, nil, p, w, func() error { return nil })
		},
		"promotion": func() (*OperatorDomainRuntime, error) {
			return NewOperatorDomainRuntime(a, s, nil, w, func() error { return nil })
		},
		"automation": func() (*OperatorDomainRuntime, error) {
			return NewOperatorDomainRuntime(a, s, p, nil, func() error { return nil })
		},
		"close": func() (*OperatorDomainRuntime, error) { return NewOperatorDomainRuntime(a, s, p, w, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			if runtime, err := build(); err == nil || runtime != nil {
				t.Fatalf("got (%v, %v), want nil, error", runtime, err)
			}
		})
	}
}

func TestOperatorDomainRuntimeCloseIsIdempotent(t *testing.T) {
	calls := 0
	runtime, err := NewOperatorDomainRuntime(&stubAudienceOperator{}, &stubSurfaceOperator{}, &stubPromotionOperator{}, &stubAutomationOperator{}, func() error { calls++; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("close calls = %d, want 1", calls)
	}
}
