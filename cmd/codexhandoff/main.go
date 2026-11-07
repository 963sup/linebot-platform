// Command codexhandoff creates and validates repository-local agent handoff evidence.
// It does not spawn agents or claim that host capabilities are available.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
)

var phases = map[string]bool{"prepare": true, "ready": true, "start": true, "result": true}
var resultStates = map[string]bool{"COMPLETED": true, "PARTIAL": true, "BLOCKED": true}
var roleNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
var capabilityStates = map[string]bool{"not-required": true, "not-verified": true, "exposed": true, "call-verified": true, "authenticated": true, "authorized": true, "unavailable": true, "authentication-blocked": true, "authorization-blocked": true, "local-smoke": true, "cli-fallback": true}

type skillEvidence struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Status string `json:"status"`
}

type capability struct {
	Name            string `json:"name"`
	Required        bool   `json:"required"`
	State           string `json:"state"`
	SelectedReason  string `json:"selected_reason,omitempty"`
	HostNative      string `json:"host_native"`
	LocalSmoke      string `json:"local_smoke"`
	Authentication  string `json:"authentication"`
	Authorization   string `json:"authorization"`
	Fallback        string `json:"fallback,omitempty"`
	EvidenceTimeUTC string `json:"evidence_time_utc"`
}

type artifact struct {
	Type              string `json:"type"`
	SHA256            string `json:"sha256"`
	SourceFingerprint string `json:"source_fingerprint"`
	Freshness         string `json:"freshness"`
}

type handoff struct {
	Version           string          `json:"version"`
	HandoffID         string          `json:"handoff_id"`
	Phase             string          `json:"phase"`
	Task              string          `json:"task"`
	Owner             string          `json:"owner"`
	Role              string          `json:"role"`
	RoleMode          string          `json:"role_mode"`
	RolePath          string          `json:"role_path"`
	RoleSHA256        string          `json:"role_sha256"`
	Workspace         string          `json:"workspace"`
	Head              string          `json:"head"`
	ScopedFingerprint string          `json:"scoped_fingerprint"`
	Scope             []string        `json:"scope"`
	Ownership         []string        `json:"ownership"`
	Acceptance        []string        `json:"acceptance"`
	Skills            []skillEvidence `json:"skills"`
	Capabilities      []capability    `json:"capabilities"`
	Artifacts         []artifact      `json:"artifacts"`
	Inputs            []string        `json:"inputs,omitempty"`
	Files             []string        `json:"files,omitempty"`
	ContractChanges   []string        `json:"contract_changes,omitempty"`
	Evidence          []string        `json:"evidence,omitempty"`
	TestsRun          []string        `json:"tests_run,omitempty"`
	Findings          []string        `json:"findings,omitempty"`
	Risks             []string        `json:"risks,omitempty"`
	NextAction        string          `json:"next_action,omitempty"`
	State             string          `json:"state,omitempty"`
	PreviousSHA256    string          `json:"previous_sha256,omitempty"`
}

type listFlag []string

func (f *listFlag) String() string { return strings.Join(*f, ",") }
func (f *listFlag) Set(v string) error {
	for _, item := range strings.Split(v, ",") {
		if item = strings.TrimSpace(item); item != "" {
			*f = append(*f, item)
		}
	}
	return nil
}

func git(root string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func gitRaw(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

func fileHash(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func scopedFingerprint(root string, scope []string) (string, error) {
	args := []string{"status", "--porcelain=v1", "--untracked-files=all", "--"}
	args = append(args, scope...)
	status, err := git(root, args...)
	if err != nil {
		return "", err
	}
	diffArgs := []string{"diff", "HEAD", "--binary", "--"}
	diffArgs = append(diffArgs, scope...)
	diff, err := git(root, diffArgs...)
	if err != nil {
		return "", err
	}
	otherArgs := []string{"ls-files", "-z", "--others", "--exclude-standard", "--"}
	otherArgs = append(otherArgs, scope...)
	others, err := gitRaw(root, otherArgs...)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00", status, diff)
	for _, raw := range strings.Split(string(others), "\x00") {
		rel := raw
		if rel == "" {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(rel))
		info, statErr := os.Lstat(path)
		if statErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("untracked scope entry must be a regular file: %s", rel)
		}
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			return "", readErr
		}
		fmt.Fprintf(h, "%s\x00%d\x00", rel, len(contents))
		h.Write(contents)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func cleanPaths(root string, values []string) ([]string, error) {
	seen := map[string]bool{}
	clean := make([]string, 0, len(values))
	for _, value := range values {
		value = filepath.ToSlash(filepath.Clean(value))
		if value == "." || filepath.IsAbs(value) || value == ".." || strings.HasPrefix(value, "../") {
			return nil, fmt.Errorf("path must be repository-relative: %q", value)
		}
		if !seen[value] {
			seen[value] = true
			clean = append(clean, value)
		}
	}
	sort.Strings(clean)
	for _, value := range clean {
		candidate := filepath.Join(root, filepath.FromSlash(value))
		for current := candidate; current != root; current = filepath.Dir(current) {
			info, lstatErr := os.Lstat(current)
			if lstatErr == nil && info.Mode()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("path contains symlink component: %q", value)
			}
			if lstatErr != nil && !os.IsNotExist(lstatErr) {
				return nil, lstatErr
			}
		}
		resolved, err := filepath.EvalSymlinks(candidate)
		if err == nil {
			rel, relErr := filepath.Rel(root, resolved)
			if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return nil, fmt.Errorf("path escapes repository through symlink: %q", value)
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return clean, nil
}

func snapshot(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var scope, ownership, acceptance, skills, capabilityFlags listFlag
	flags.Var(&scope, "scope", "comma-separated repository-relative scope; repeatable")
	flags.Var(&ownership, "ownership", "comma-separated write ownership; repeatable")
	flags.Var(&acceptance, "acceptance", "acceptance criterion; repeatable")
	flags.Var(&skills, "skill", "repository-relative SKILL.md path; repeatable")
	flags.Var(&capabilityFlags, "capability", "name:required|optional:reason; repeatable")
	id := flags.String("id", "", "handoff ID")
	task := flags.String("task", "", "task summary")
	owner := flags.String("owner", "orchestrator", "handoff owner")
	role := flags.String("role", "", "registered role")
	roleMode := flags.String("role-mode", "explicit", "explicit or native-verified")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return 2
	}
	root, err := git(".", "rev-parse", "--show-toplevel")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	root, _ = filepath.EvalSymlinks(root)
	scope, err = cleanPaths(root, scope)
	if err != nil || len(scope) == 0 {
		fmt.Fprintln(stderr, "at least one valid --scope is required")
		return 2
	}
	ownership, err = cleanPaths(root, ownership)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	skills, err = cleanPaths(root, skills)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	head, err := git(root, "rev-parse", "HEAD")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fingerprint, err := scopedFingerprint(root, scope)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	rolePath := filepath.ToSlash(filepath.Join(".codex", "agents", *role+".toml"))
	if !roleNamePattern.MatchString(*role) {
		fmt.Fprintln(stderr, "role must be a registered lowercase role name")
		return 2
	}
	if _, err = cleanPaths(root, []string{rolePath}); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	roleDigest, err := fileHash(filepath.Join(root, filepath.FromSlash(rolePath)))
	if err != nil {
		fmt.Fprintf(stderr, "role %s: %v\n", *role, err)
		return 2
	}
	h := handoff{Version: "1", HandoffID: *id, Phase: "prepare", Task: *task, Owner: *owner, Role: *role, RoleMode: *roleMode, RolePath: rolePath, RoleSHA256: roleDigest, Workspace: root, Head: head, ScopedFingerprint: fingerprint, Scope: scope, Ownership: ownership, Acceptance: acceptance, Skills: []skillEvidence{}, Capabilities: []capability{}, Artifacts: []artifact{}}
	for _, path := range skills {
		digest, hashErr := fileHash(filepath.Join(root, filepath.FromSlash(path)))
		if hashErr != nil {
			fmt.Fprintf(stderr, "skill %s: %v\n", path, hashErr)
			return 2
		}
		h.Skills = append(h.Skills, skillEvidence{Path: path, SHA256: digest, Status: "selected-parent-read"})
	}
	for _, declaration := range capabilityFlags {
		parts := strings.SplitN(declaration, ":", 3)
		if len(parts) != 3 || (parts[1] != "required" && parts[1] != "optional") || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[2]) == "" {
			fmt.Fprintln(stderr, "capability must be name:required|optional:reason")
			return 2
		}
		h.Capabilities = append(h.Capabilities, capability{Name: parts[0], Required: parts[1] == "required", State: "not-verified", SelectedReason: parts[2], HostNative: "not-verified", LocalSmoke: "not-verified", Authentication: "not-verified", Authorization: "not-verified", EvidenceTimeUTC: time.Now().UTC().Format(time.RFC3339)})
	}
	if err := validateHandoff(h, "prepare"); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(h); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func decode(r io.Reader) (handoff, error) {
	var h handoff
	d := json.NewDecoder(bufio.NewReader(r))
	d.DisallowUnknownFields()
	if err := d.Decode(&h); err != nil {
		return h, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return h, errors.New("handoff contains trailing JSON")
	}
	return h, nil
}

func validateHandoff(h handoff, expected string) error {
	if h.Version != "1" || !phases[h.Phase] || (expected != "" && h.Phase != expected) {
		return errors.New("invalid handoff version or phase")
	}
	if strings.TrimSpace(h.HandoffID) == "" || strings.TrimSpace(h.Task) == "" || strings.TrimSpace(h.Owner) == "" || strings.TrimSpace(h.Role) == "" {
		return errors.New("handoff_id, task, owner and role are required")
	}
	if h.RoleMode != "explicit" && h.RoleMode != "native-verified" {
		return errors.New("role_mode must be explicit or native-verified")
	}
	if h.RolePath != filepath.ToSlash(filepath.Join(".codex", "agents", h.Role+".toml")) || len(h.RoleSHA256) != 64 {
		return errors.New("role_path and role_sha256 must identify the selected role guidance")
	}
	if !filepath.IsAbs(h.Workspace) || len(h.Head) != 40 || len(h.ScopedFingerprint) != 64 || len(h.Scope) == 0 || len(h.Acceptance) == 0 {
		return errors.New("workspace, HEAD, scoped fingerprint, scope and acceptance are required")
	}
	for _, s := range h.Skills {
		if s.Path == "" || len(s.SHA256) != 64 || s.Status == "" {
			return errors.New("each skill requires path, SHA-256 and status")
		}
	}
	for _, c := range h.Capabilities {
		if c.Name == "" || c.SelectedReason == "" || !capabilityStates[c.State] || !capabilityStates[c.HostNative] || !capabilityStates[c.LocalSmoke] || !capabilityStates[c.Authentication] || !capabilityStates[c.Authorization] || c.EvidenceTimeUTC == "" {
			return errors.New("each capability requires explicit exposure, smoke, auth and evidence states")
		}
		if _, err := time.Parse(time.RFC3339, c.EvidenceTimeUTC); err != nil {
			return errors.New("capability evidence_time_utc must be RFC3339")
		}
	}
	if h.Phase == "ready" {
		for _, s := range h.Skills {
			if s.Status != "child-read-verified" {
				return errors.New("READY requires child-read-verified skill evidence")
			}
		}
		for _, c := range h.Capabilities {
			if c.Required && (c.State == "not-required" || c.State == "not-verified" || c.State == "unavailable" || strings.HasSuffix(c.State, "-blocked")) {
				return fmt.Errorf("READY required capability %s is not usable", c.Name)
			}
		}
	}
	if h.Phase == "result" {
		if !resultStates[h.State] || h.NextAction == "" {
			return errors.New("RESULT requires START binding, COMPLETED, PARTIAL or BLOCKED state and next_action")
		}
	}
	return nil
}

func documentHash(h handoff) (string, error) {
	b, err := json.Marshal(h)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func sameCapabilities(a, b []capability) bool {
	if len(a) != len(b) {
		return false
	}
	want := map[string]bool{}
	for _, c := range a {
		if _, duplicate := want[c.Name]; duplicate {
			return false
		}
		want[c.Name] = c.Required
	}
	for _, c := range b {
		required, ok := want[c.Name]
		if !ok || required != c.Required {
			return false
		}
		delete(want, c.Name)
	}
	return len(want) == 0
}

func sameSkills(a, b []skillEvidence) bool {
	if len(a) != len(b) {
		return false
	}
	want := map[string]string{}
	for _, skill := range a {
		if _, duplicate := want[skill.Path]; duplicate {
			return false
		}
		want[skill.Path] = skill.SHA256
	}
	for _, skill := range b {
		digest, ok := want[skill.Path]
		if !ok || digest != skill.SHA256 {
			return false
		}
		delete(want, skill.Path)
	}
	return len(want) == 0
}

func validateTransition(previous, current handoff) error {
	next := map[string]string{"prepare": "ready", "ready": "start", "start": "result"}
	if next[previous.Phase] != current.Phase {
		return errors.New("invalid handoff phase transition")
	}
	digest, err := documentHash(previous)
	if err != nil || current.PreviousSHA256 != digest {
		return errors.New("previous_sha256 does not bind the supplied prior handoff")
	}
	if previous.HandoffID != current.HandoffID || previous.Role != current.Role || previous.RolePath != current.RolePath || previous.RoleSHA256 != current.RoleSHA256 || previous.Workspace != current.Workspace || previous.Head != current.Head || (current.Phase != "result" && previous.ScopedFingerprint != current.ScopedFingerprint) {
		return errors.New("handoff identity changed across transition")
	}
	if !slices.Equal(previous.Scope, current.Scope) || !slices.Equal(previous.Ownership, current.Ownership) {
		return errors.New("scope or ownership changed without a new PREPARE")
	}
	if !sameCapabilities(previous.Capabilities, current.Capabilities) {
		return errors.New("capability declarations changed across transition")
	}
	if !sameSkills(previous.Skills, current.Skills) || !slices.Equal(previous.Acceptance, current.Acceptance) {
		return errors.New("skill evidence or acceptance criteria changed across transition")
	}
	return nil
}

func validateCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	phase := flags.String("phase", "", "expected prepare, ready, start or result phase")
	previousPath := flags.String("previous", "", "prior phase JSON for transition validation")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || !phases[*phase] {
		return 2
	}
	h, err := decode(stdin)
	if err == nil {
		err = validateHandoff(h, *phase)
	}
	if err == nil && *phase != "prepare" {
		if *previousPath == "" {
			err = errors.New("non-PREPARE validation requires --previous")
		} else {
			file, openErr := os.Open(*previousPath)
			if openErr != nil {
				err = openErr
			} else {
				previous, decodeErr := decode(file)
				closeErr := file.Close()
				if decodeErr != nil {
					err = decodeErr
				} else if closeErr != nil {
					err = closeErr
				} else if validateErr := validateHandoff(previous, ""); validateErr != nil {
					err = validateErr
				} else {
					err = validateTransition(previous, h)
				}
			}
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "handoff %s %s valid\n", h.HandoffID, h.Phase)
	return 0
}

func render(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		return 2
	}
	h, err := decode(stdin)
	if err == nil {
		err = validateHandoff(h, "prepare")
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "# Handoff %s\n\nRole: %s (%s)\nTask: %s\nWorkspace: %s\nHEAD: %s\nScoped fingerprint: %s\n\nRead the explicitly supplied role/workflow and every selected Skill. Before doing task work, return a READY handoff with observed identity, child-read-verified Skill hashes, and the child session capability matrix. Do not infer native role or MCP availability from this prompt.\n", h.HandoffID, h.Role, h.RoleMode, h.Task, h.Workspace, h.Head, h.ScopedFingerprint)
	return 0
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: codexhandoff <snapshot|validate|render>")
		return 2
	}
	switch args[0] {
	case "snapshot":
		return snapshot(args[1:], stdout, stderr)
	case "validate":
		return validateCommand(args[1:], stdin, stdout, stderr)
	case "render":
		return render(args[1:], stdin, stdout, stderr)
	default:
		fmt.Fprintln(stderr, "unknown command")
		return 2
	}
}

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
