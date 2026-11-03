package cli

import (
	"strings"
	"testing"
)

func TestCouponCommandAndLegacyMembershipAlias(t *testing.T) {
	getenv := func(string) string { return "" }
	for _, args := range [][]string{
		{"memberships", "--kind", "coupons", "--read"},
		{"memberships", "--kind", "coupon", "--coupon-id", "coupon-1", "--read"},
		{"coupons", "--read"},
		{"coupons", "--kind", "coupon", "--coupon-id", "coupon-1", "--read"},
	} {
		err := run(args, getenv)
		if err == nil || !strings.Contains(err.Error(), "access token") {
			t.Fatalf("coupon command did not reach the authorized runtime path: args=%v error=%v", args, err)
		}
	}
}
