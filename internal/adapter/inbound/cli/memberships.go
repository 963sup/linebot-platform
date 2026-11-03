package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/engagement"
	"github.com/963sup/linebot-platform/internal/bootstrap"
)

func runMemberships(args []string, getenv func(string) string, output io.Writer) error {
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
	q := engagement.MembershipQuery{Kind: *kind, UserID: *user, MembershipID: *membershipID, CouponID: *couponID, Start: *start, Limit: *limit}
	if err := engagement.ValidateMembershipQuery(q); err != nil {
		return err
	}
	rt, err := bootstrap.NewLINEOpsRuntime(getenv(bootstrap.KeyChannelToken))
	if err != nil {
		return err
	}
	defer rt.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	result, err := rt.Memberships.Inspect(ctx, q)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, string(result))
	return err
}
