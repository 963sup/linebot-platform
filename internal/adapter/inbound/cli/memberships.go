package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/membership"
	"github.com/963sup/linebot-platform/internal/application/usecases/promotion"
)

func runMemberships(parent context.Context, args []string, getenv func(string) string, output io.Writer, dependencies ...Dependencies) error {
	deps := dependency(dependencies)
	flags := flag.NewFlagSet("memberships", flag.ContinueOnError)
	kind := flags.String("kind", "", "memberships|subscription|membership-users|coupons|coupon")
	user := flags.String("user", "", "user ID")
	membershipID := flags.Int("membership-id", 0, "membership ID")
	couponID := flags.String("coupon-id", "", "coupon ID")
	start := flags.String("start", "", "pagination cursor")
	limit := flags.Int("limit", 0, "maximum page size")
	read := flags.Bool("read", false, "explicitly authorize regional/account-scoped LINE query")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*read || len(flags.Args()) != 0 {
		return errors.New("memberships requires --read and no extra arguments")
	}
	if getenv == nil || output == nil {
		return errors.New("missing CLI configuration")
	}
	// Keep the original memberships --kind coupon(s) CLI syntax compatible.
	if *kind == "coupons" || *kind == "coupon" {
		return inspectCoupon(parent, *kind, promotion.CouponID(*couponID), promotion.CouponPageRequest{Cursor: *start, Limit: *limit}, getenv, output, deps)
	}
	if deps.MembershipsRuntime == nil {
		return errors.New("memberships runtime is not configured")
	}
	rt, err := deps.MembershipsRuntime(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer rt.Close()
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	page := membership.PageRequest{Token: *start, Limit: *limit}
	var result any
	switch *kind {
	case "memberships":
		result, err = rt.ListMemberships(ctx, page)
	case "subscription":
		result, err = rt.GetSubscription(ctx, membership.RecipientID(*user))
	case "membership-users":
		if *membershipID <= 0 || int64(*membershipID) > int64(2147483647) {
			return errors.New("membership ID must be in range 1-2147483647")
		}
		result, err = rt.ListMembers(ctx, membership.MembershipID(fmt.Sprint(*membershipID)), page)
	default:
		return errors.New("unsupported membership query kind")
	}
	if err != nil {
		return err
	}
	return writeJSON(output, result)
}
