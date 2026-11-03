package cli

import (
	"bytes"
	"testing"
)

func TestCouponsFailClosed(t *testing.T) {
	var output bytes.Buffer
	getenv := func(string) string { return "" }
	for _, args := range [][]string{
		{"--kind", "coupons"},
		{"--read", "--kind", "coupon"},
		{"--read", "--kind", "unexpected"},
		{"--read", "--kind", "coupons", "--limit", "1001"},
		{"--read", "--kind", "coupons", "--coupon-id", "unexpected"},
		{"--read", "--kind", "coupon", "--coupon-id", " id "},
	} {
		if err := runCoupons(args, getenv, &output); err == nil {
			t.Fatalf("invalid coupon query accepted: %v", args)
		}
	}
	if output.Len() != 0 {
		t.Fatal("invalid coupon query should not expose data")
	}
}
