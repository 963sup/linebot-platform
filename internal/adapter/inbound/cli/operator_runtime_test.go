package cli

import (
	"errors"
	"testing"
)

type stubAudienceOperator struct{ AudienceOperator }

var _ AudienceDefinitionsRuntime = (*AudienceDomainRuntime)(nil)

func TestAudienceDomainRuntime(t *testing.T) {
	if _, err := NewAudienceDomainRuntime(nil, func() error { return nil }); err == nil {
		t.Fatal("missing operator accepted")
	}
	if _, err := NewAudienceDomainRuntime(&stubAudienceOperator{}, nil); err == nil {
		t.Fatal("missing closer accepted")
	}
	calls := 0
	sentinel := errors.New("close")
	r, err := NewAudienceDomainRuntime(&stubAudienceOperator{}, func() error { calls++; return sentinel })
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if !errors.Is(r.Close(), sentinel) {
			t.Fatal("lost close error")
		}
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}

type stubSurfaceOperator struct{ SurfaceOperator }

var _ InteractionSurfacesRuntime = (*SurfaceDomainRuntime)(nil)

func TestSurfaceDomainRuntime(t *testing.T) {
	if _, err := NewSurfaceDomainRuntime(nil, func() error { return nil }); err == nil {
		t.Fatal("missing operator accepted")
	}
	if _, err := NewSurfaceDomainRuntime(&stubSurfaceOperator{}, nil); err == nil {
		t.Fatal("missing closer accepted")
	}
	calls := 0
	sentinel := errors.New("close")
	r, err := NewSurfaceDomainRuntime(&stubSurfaceOperator{}, func() error { calls++; return sentinel })
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if !errors.Is(r.Close(), sentinel) {
			t.Fatal("lost close error")
		}
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}

type stubPromotionOperator struct{ PromotionOperator }

var _ PromotionsRuntime = (*PromotionDomainRuntime)(nil)

func TestPromotionDomainRuntime(t *testing.T) {
	if _, err := NewPromotionDomainRuntime(nil, func() error { return nil }); err == nil {
		t.Fatal("missing operator accepted")
	}
	if _, err := NewPromotionDomainRuntime(&stubPromotionOperator{}, nil); err == nil {
		t.Fatal("missing closer accepted")
	}
	calls := 0
	sentinel := errors.New("close")
	r, err := NewPromotionDomainRuntime(&stubPromotionOperator{}, func() error { calls++; return sentinel })
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if !errors.Is(r.Close(), sentinel) {
			t.Fatal("lost close error")
		}
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}
