package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	operatorCommandTimeout = 20 * time.Second
	operatorFileLimit      = 256 << 10
)

type operatorFlags struct {
	action  *string
	id      *string
	project *string
	read    *bool
	apply   *bool
	confirm *string
}

type operatorPlan struct {
	Resource string         `json:"resource"`
	Action   string         `json:"action"`
	ID       string         `json:"id"`
	DryRun   bool           `json:"dry_run"`
	Input    map[string]any `json:"input,omitempty"`
}

func bindOperatorFlags(fs *flag.FlagSet) operatorFlags {
	return operatorFlags{
		action:  fs.String("action", "", "operator action"),
		id:      fs.String("id", "", "aggregate identifier"),
		project: fs.String("project", "", "explicit staging GCP project"),
		read:    fs.Bool("read", false, "explicit permission for a read-only get"),
		apply:   fs.Bool("apply", false, "apply a persistent or external mutation"),
		confirm: fs.String("confirm", "", "must exactly match --action when applying"),
	}
}

func (f operatorFlags) validate(args []string) error {
	if len(args) != 0 {
		return errors.New("unexpected positional arguments")
	}
	if strings.TrimSpace(*f.action) == "" || strings.TrimSpace(*f.id) == "" {
		return errors.New("--action and --id are required")
	}
	return nil
}

func (f operatorFlags) authorizeGet(getenv func(string) string) error {
	if !*f.read || *f.apply || *f.confirm != "" {
		return errors.New("get requires --read and rejects --apply/--confirm")
	}
	return validateOperatorProject(*f.project, getenv)
}

func (f operatorFlags) authorizeMutation(out io.Writer, resource string, input map[string]any) (bool, error) {
	if *f.read {
		return false, errors.New("mutation rejects --read")
	}
	if !*f.apply {
		return false, writeOperatorJSON(out, operatorPlan{
			Resource: resource, Action: *f.action, ID: *f.id, DryRun: true, Input: input,
		})
	}
	if *f.confirm != *f.action {
		return false, fmt.Errorf("--confirm must exactly equal %q", *f.action)
	}
	return true, nil
}

func validateOperatorProject(project string, getenv func(string) string) error {
	if getenv == nil {
		return errors.New("environment reader is required")
	}
	allowed := strings.TrimSpace(getenv(stagingProjectEnv))
	if project == "" || allowed == "" || project != allowed {
		return errors.New("project must match nonempty LINEBOT_STAGING_PROJECT")
	}
	return nil
}

func writeOperatorJSON(out io.Writer, value any) error {
	if out == nil {
		return errors.New("output writer is required")
	}
	if err := json.NewEncoder(out).Encode(value); err != nil {
		return fmt.Errorf("write operator summary: %w", err)
	}
	return nil
}

func hashOperatorValue(value string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
}

func parseOperatorTime(name, value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, fmt.Errorf("--%s is required", name)
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("--%s must be RFC3339", name)
	}
	return parsed, nil
}

func readOperatorPrivateJSON(path string, target any) error {
	data, err := readOperatorPrivateFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("operator JSON file is invalid")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("operator JSON file has trailing data")
	}
	return nil
}

func readOperatorPrivateFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return nil, errors.New("operator input file must be regular and private (0600)")
	}
	return readPrivateBounded(path, operatorFileLimit)
}

func runAudienceDefinitions(args []string, getenv func(string) string, out io.Writer, deps Dependencies) error {
	fs := flag.NewFlagSet("audience-definitions", flag.ContinueOnError)
	common := bindOperatorFlags(fs)
	source := fs.String("source", "", "criteria source")
	definitionFile := fs.String("definition-file", "", "private criteria definition file (0600, max 256 KiB)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := common.validate(fs.Args()); err != nil {
		return err
	}
	action := *common.action
	if action == "get" {
		if err := common.authorizeGet(getenv); err != nil {
			return err
		}
		return withAudienceRuntime(common, deps, func(ctx context.Context, runtime AudienceDefinitionsRuntime) error {
			value, err := runtime.GetAudienceDefinition(ctx, *common.id)
			if err != nil {
				return err
			}
			return writeOperatorJSON(out, value)
		})
	}
	allowed := map[string]bool{"define": true, "revise": true, "activate": true, "archive": true, "sync": true}
	if !allowed[action] {
		return errors.New("audience-definitions --action must be get|define|revise|activate|archive|sync")
	}
	var definition string
	input := map[string]any{}
	if action == "define" || action == "revise" {
		if strings.TrimSpace(*source) == "" || *definitionFile == "" {
			return errors.New("define/revise require --source and --definition-file")
		}
		data, err := readOperatorPrivateFile(*definitionFile)
		if err != nil {
			return err
		}
		definition = string(data)
		input = map[string]any{"source": *source, "definition_sha256": hashOperatorValue(definition), "definition_bytes": len(data)}
	}
	authorized, err := common.authorizeMutation(out, "audience-definitions", input)
	if err != nil || !authorized {
		return err
	}
	if err := validateOperatorProject(*common.project, getenv); err != nil {
		return err
	}
	return withAudienceRuntime(common, deps, func(ctx context.Context, runtime AudienceDefinitionsRuntime) error {
		var value any
		var err error
		switch action {
		case "define":
			value, err = runtime.DefineAudience(ctx, *common.id, *source, definition)
		case "revise":
			value, err = runtime.ReviseAudience(ctx, *common.id, *source, definition)
		case "activate":
			value, err = runtime.ActivateAudience(ctx, *common.id)
		case "archive":
			value, err = runtime.ArchiveAudience(ctx, *common.id)
		case "sync":
			value, err = runtime.SyncAudience(ctx, *common.id)
		}
		if err != nil {
			return err
		}
		return writeOperatorJSON(out, value)
	})
}

func withAudienceRuntime(flags operatorFlags, deps Dependencies, operation func(context.Context, AudienceDefinitionsRuntime) error) (err error) {
	if deps.AudienceDefinitions == nil {
		return errors.New("audience definitions runtime is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), operatorCommandTimeout)
	defer cancel()
	runtime, err := deps.AudienceDefinitions(ctx, *flags.project)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, runtime.Close()) }()
	return operation(ctx, runtime)
}

func runInteractionSurfaces(args []string, getenv func(string) string, out io.Writer, deps Dependencies) error {
	fs := flag.NewFlagSet("interaction-surfaces", flag.ContinueOnError)
	common := bindOperatorFlags(fs)
	regionsFile := fs.String("regions-file", "", "private JSON array of regions (0600, max 256 KiB)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := common.validate(fs.Args()); err != nil {
		return err
	}
	action := *common.action
	if action == "get" {
		if err := common.authorizeGet(getenv); err != nil {
			return err
		}
		return withSurfaceRuntime(common, deps, func(ctx context.Context, runtime InteractionSurfacesRuntime) error {
			value, err := runtime.GetSurface(ctx, *common.id)
			if err != nil {
				return err
			}
			return writeOperatorJSON(out, value)
		})
	}
	allowed := map[string]bool{"define": true, "revise": true, "publish": true, "retire": true, "deploy": true}
	if !allowed[action] {
		return errors.New("interaction-surfaces --action must be get|define|revise|publish|retire|deploy")
	}
	var regions []InteractionRegion
	input := map[string]any{}
	if action == "define" || action == "revise" {
		if err := readOperatorPrivateJSON(*regionsFile, &regions); err != nil {
			return err
		}
		if len(regions) == 0 {
			return errors.New("regions file must contain at least one region")
		}
		input["region_count"] = len(regions)
	}
	authorized, err := common.authorizeMutation(out, "interaction-surfaces", input)
	if err != nil || !authorized {
		return err
	}
	if err := validateOperatorProject(*common.project, getenv); err != nil {
		return err
	}
	return withSurfaceRuntime(common, deps, func(ctx context.Context, runtime InteractionSurfacesRuntime) error {
		var value any
		var err error
		switch action {
		case "define":
			value, err = runtime.DefineSurface(ctx, *common.id, regions)
		case "revise":
			value, err = runtime.ReviseSurface(ctx, *common.id, regions)
		case "publish":
			value, err = runtime.PublishSurface(ctx, *common.id)
		case "retire":
			value, err = runtime.RetireSurface(ctx, *common.id)
		case "deploy":
			value, err = runtime.DeploySurface(ctx, *common.id)
		}
		if err != nil {
			return err
		}
		return writeOperatorJSON(out, value)
	})
}

func withSurfaceRuntime(flags operatorFlags, deps Dependencies, operation func(context.Context, InteractionSurfacesRuntime) error) (err error) {
	if deps.InteractionSurfaces == nil {
		return errors.New("interaction surfaces runtime is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), operatorCommandTimeout)
	defer cancel()
	runtime, err := deps.InteractionSurfaces(ctx, *flags.project)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, runtime.Close()) }()
	return operation(ctx, runtime)
}

func runPromotions(args []string, getenv func(string) string, out io.Writer, deps Dependencies) error {
	fs := flag.NewFlagSet("promotions", flag.ContinueOnError)
	common := bindOperatorFlags(fs)
	offer := fs.String("offer", "", "offer reference")
	policy := fs.String("policy", "", "eligibility policy reference")
	startsAtRaw := fs.String("starts-at", "", "promotion start (RFC3339)")
	endsAtRaw := fs.String("ends-at", "", "promotion end (RFC3339)")
	atRaw := fs.String("at", "", "action time (RFC3339)")
	recipient := fs.String("recipient", "", "recipient reference")
	eligibleRaw := fs.String("eligible", "", "eligibility decision true|false")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := common.validate(fs.Args()); err != nil {
		return err
	}
	action := *common.action
	if action == "get" {
		if err := common.authorizeGet(getenv); err != nil {
			return err
		}
		return withPromotionRuntime(common, deps, func(ctx context.Context, runtime PromotionsRuntime) error {
			value, err := runtime.GetPromotion(ctx, *common.id)
			if err != nil {
				return err
			}
			return writeOperatorJSON(out, value)
		})
	}
	allowed := map[string]bool{"define": true, "activate": true, "close": true, "delivery-intent": true}
	if !allowed[action] {
		return errors.New("promotions --action must be get|define|activate|close|delivery-intent")
	}
	var startsAt, endsAt, at time.Time
	var eligible bool
	input := map[string]any{}
	var err error
	switch action {
	case "define":
		if strings.TrimSpace(*offer) == "" || strings.TrimSpace(*policy) == "" {
			return errors.New("define requires --offer and --policy")
		}
		startsAt, err = parseOperatorTime("starts-at", *startsAtRaw)
		if err == nil {
			endsAt, err = parseOperatorTime("ends-at", *endsAtRaw)
		}
		input = map[string]any{"offer_sha256": hashOperatorValue(*offer), "policy_sha256": hashOperatorValue(*policy), "starts_at": startsAt, "ends_at": endsAt}
	case "activate":
		at, err = parseOperatorTime("at", *atRaw)
		input["at"] = at
	case "delivery-intent":
		if strings.TrimSpace(*recipient) == "" || strings.TrimSpace(*policy) == "" || *eligibleRaw == "" {
			return errors.New("delivery-intent requires --recipient, --policy, --eligible and --at")
		}
		eligible, err = strconv.ParseBool(*eligibleRaw)
		if err == nil {
			at, err = parseOperatorTime("at", *atRaw)
		}
		input = map[string]any{"recipient_sha256": hashOperatorValue(*recipient), "policy_sha256": hashOperatorValue(*policy), "eligible": eligible, "at": at}
	}
	if err != nil {
		return err
	}
	authorized, err := common.authorizeMutation(out, "promotions", input)
	if err != nil || !authorized {
		return err
	}
	if err := validateOperatorProject(*common.project, getenv); err != nil {
		return err
	}
	return withPromotionRuntime(common, deps, func(ctx context.Context, runtime PromotionsRuntime) error {
		var value any
		var err error
		switch action {
		case "define":
			value, err = runtime.DefinePromotion(ctx, *common.id, *offer, *policy, startsAt, endsAt)
		case "activate":
			value, err = runtime.ActivatePromotion(ctx, *common.id, at)
		case "close":
			value, err = runtime.ClosePromotion(ctx, *common.id)
		case "delivery-intent":
			value, err = runtime.CreateDeliveryIntent(ctx, *common.id, *recipient, *policy, eligible, at)
		}
		if err != nil {
			return err
		}
		return writeOperatorJSON(out, value)
	})
}

func withPromotionRuntime(flags operatorFlags, deps Dependencies, operation func(context.Context, PromotionsRuntime) error) (err error) {
	if deps.Promotions == nil {
		return errors.New("promotions runtime is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), operatorCommandTimeout)
	defer cancel()
	runtime, err := deps.Promotions(ctx, *flags.project)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, runtime.Close()) }()
	return operation(ctx, runtime)
}

func runAutomationRequests(args []string, getenv func(string) string, out io.Writer, deps Dependencies) error {
	fs := flag.NewFlagSet("automation-requests", flag.ContinueOnError)
	common := bindOperatorFlags(fs)
	actionName := fs.String("automation-action", "", "provider-independent automation action")
	scopesRaw := fs.String("scopes", "", "comma-separated required OAuth scopes")
	approvedBy := fs.String("approved-by", "", "approval principal")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := common.validate(fs.Args()); err != nil {
		return err
	}
	action := *common.action
	if action == "get" {
		if err := common.authorizeGet(getenv); err != nil {
			return err
		}
		return withAutomationRuntime(common, deps, func(ctx context.Context, runtime AutomationRequestsRuntime) error {
			value, err := runtime.GetAutomationRequest(ctx, *common.id)
			if err != nil {
				return err
			}
			return writeOperatorJSON(out, value)
		})
	}
	allowed := map[string]bool{"request": true, "approve": true, "execute": true}
	if !allowed[action] {
		return errors.New("automation-requests --action must be get|request|approve|execute")
	}
	var scopes []string
	input := map[string]any{}
	switch action {
	case "request":
		if strings.TrimSpace(*actionName) == "" || strings.TrimSpace(*scopesRaw) == "" {
			return errors.New("request requires --automation-action and --scopes")
		}
		for _, scope := range strings.Split(*scopesRaw, ",") {
			scope = strings.TrimSpace(scope)
			if scope == "" {
				return errors.New("--scopes must contain nonempty comma-separated values")
			}
			scopes = append(scopes, scope)
		}
		sort.Strings(scopes)
		input = map[string]any{"automation_action": *actionName, "scope_count": len(scopes)}
	case "approve":
		if strings.TrimSpace(*approvedBy) == "" {
			return errors.New("approve requires --approved-by")
		}
		input["approved_by_sha256"] = hashOperatorValue(*approvedBy)
	}
	authorized, err := common.authorizeMutation(out, "automation-requests", input)
	if err != nil || !authorized {
		return err
	}
	if err := validateOperatorProject(*common.project, getenv); err != nil {
		return err
	}
	return withAutomationRuntime(common, deps, func(ctx context.Context, runtime AutomationRequestsRuntime) error {
		var value AutomationRequestSummary
		var err error
		switch action {
		case "request":
			value, err = runtime.RequestAutomation(ctx, *common.id, *actionName, scopes)
		case "approve":
			value, err = runtime.ApproveAutomation(ctx, *common.id, *approvedBy)
		case "execute":
			value, err = runtime.ExecuteAutomation(ctx, *common.id)
		}
		if err != nil {
			return err
		}
		return writeOperatorJSON(out, value)
	})
}

func withAutomationRuntime(flags operatorFlags, deps Dependencies, operation func(context.Context, AutomationRequestsRuntime) error) (err error) {
	if deps.AutomationRequests == nil {
		return errors.New("automation requests runtime is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), operatorCommandTimeout)
	defer cancel()
	runtime, err := deps.AutomationRequests(ctx, *flags.project)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, runtime.Close()) }()
	return operation(ctx, runtime)
}
