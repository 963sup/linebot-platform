package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
)

const stagingProjectEnv = "LINEBOT_STAGING_PROJECT"

// runEvents only reads staging Firestore. It cannot enqueue work or call LINE.
func runEvents(args []string, getenv func(string) string) error {
	if getenv == nil {
		return errors.New("environment reader is required")
	}
	flags := flag.NewFlagSet("events", flag.ContinueOnError)
	project := flags.String("project", "", "explicit staging GCP project")
	state := flags.String("state", "unknown", "pending|sending|sent|expired|unknown")
	limit := flags.Int("limit", 10, "metadata samples (0–100)")
	inspect := flags.Bool("inspect", false, "print bounded read-only sample")
	check := flags.Bool("check", false, "monitor counts and exit nonzero on alert")
	pendingAge := flags.Duration("max-pending-age", 2*time.Minute, "maximum pending age before alert")
	expiredMax := flags.Int64("max-expired", -1, "alert if expired count exceeds this value; -1 disables")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || *inspect == *check {
		return errors.New("choose exactly one of --inspect or --check; no extra arguments")
	}
	selected := messagingdelivery.EventState(*state)
	if !selected.Valid() || *limit < 0 || *limit > 100 || *pendingAge <= 0 || *expiredMax < -1 {
		return errors.New("invalid outbox state or alert thresholds")
	}
	allowed := strings.TrimSpace(getenv(stagingProjectEnv))
	if *project == "" || allowed == "" || *project != allowed {
		return errors.New("project must match nonempty LINEBOT_STAGING_PROJECT")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if statusRuntimeFactory == nil {
		return errors.New("status runtime is not configured")
	}
	runtime, err := statusRuntimeFactory(ctx, *project)
	if err != nil {
		return err
	}
	defer runtime.Close()
	samples := *limit
	if *check {
		samples = 0 // periodic monitoring never prints correlatable keys
	}
	report, err := runtime.Inspect(ctx, selected, samples, time.Now().UTC())
	if err != nil {
		return err
	}
	type wireSample struct {
		KeyHint   string                       `json:"key_hint"`
		State     messagingdelivery.EventState `json:"state"`
		UpdatedAt time.Time                    `json:"updated_at"`
	}
	type wireReport struct {
		GeneratedAt             time.Time                              `json:"generated_at"`
		Counts                  map[messagingdelivery.EventState]int64 `json:"counts"`
		OldestPendingAgeSeconds int64                                  `json:"oldest_pending_age_seconds"`
		Samples                 []wireSample                           `json:"samples,omitempty"`
	}
	wire := wireReport{
		GeneratedAt:             report.GeneratedAt,
		Counts:                  report.Counts,
		OldestPendingAgeSeconds: report.OldestPendingAgeSeconds,
		Samples:                 make([]wireSample, 0, len(report.Samples)),
	}
	for _, sample := range report.Samples {
		wire.Samples = append(wire.Samples, wireSample{
			KeyHint: sample.KeyHint, State: sample.State, UpdatedAt: sample.UpdatedAt,
		})
	}
	if err := json.NewEncoder(os.Stdout).Encode(wire); err != nil {
		return fmt.Errorf("write outbox status: %w", err)
	}
	if *check && (report.Counts[messagingdelivery.EventUnknown] > 0 ||
		report.OldestPendingAgeSeconds > int64(pendingAge.Seconds()) ||
		(*expiredMax >= 0 && report.Counts[messagingdelivery.EventExpired] > *expiredMax)) {
		return errors.New("outbox alert threshold exceeded: operator review required")
	}
	return nil
}
