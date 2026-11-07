package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/promotion"
)

// runCoupons offers a separate command without breaking the legacy membership alias.
func runCoupons(parent context.Context, args []string, getenv func(string) string, output io.Writer, dependencies ...Dependencies) error {
	deps := dependency(dependencies)
	flags := flag.NewFlagSet("coupons", flag.ContinueOnError)
	kind := flags.String("kind", "coupons", "coupons|coupon")
	id := flags.String("coupon-id", "", "LINE coupon ID for --kind coupon")
	start := flags.String("start", "", "pagination cursor")
	limit := flags.Int("limit", 0, "maximum page size")
	read := flags.Bool("read", false, "authorize an eligible channel coupon query")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*read || len(flags.Args()) != 0 {
		return errors.New("coupons requires --read and no extra arguments")
	}
	return inspectCoupon(parent, *kind, promotion.CouponID(*id), promotion.CouponPageRequest{Cursor: *start, Limit: *limit}, getenv, output, deps)
}

func inspectCoupon(parent context.Context, kind string, id promotion.CouponID, page promotion.CouponPageRequest, getenv func(string) string, output io.Writer, deps Dependencies) (err error) {
	if getenv == nil || output == nil {
		return errors.New("missing CLI configuration")
	}
	if getenv(messagingAPIChannelAccessTokenEnv) == "" {
		return errors.New("access token is required")
	}
	if deps.CouponsRuntime == nil {
		return errors.New("coupons runtime is not configured")
	}
	rt, err := deps.CouponsRuntime(getenv(messagingAPIChannelAccessTokenEnv))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, rt.Close()) }()
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	var result any
	switch kind {
	case "coupons":
		if id != "" {
			return errors.New("coupon ID is not valid for coupon list")
		}
		result, err = rt.List(ctx, page)
	case "coupon":
		result, err = rt.Get(ctx, id)
	default:
		return errors.New("unsupported coupon query kind")
	}
	if err != nil {
		return err
	}
	return writeJSON(output, result)
}

func writeJSON(output io.Writer, value any) error { return json.NewEncoder(output).Encode(value) }
