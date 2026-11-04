// Command codexplugins checks or installs explicitly approved, pinned skills.
// It never executes downloaded scripts, provisions credentials, or verifies MCP connectivity.
package main

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const receiptName = ".codex-install.json"
const maxArchiveBytes = 64 << 20

var shaPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)
var approvedSources = map[string]string{
	"cc-skills":                  "https://github.com/samber/cc-skills.git",
	"cc-skills-golang":           "https://github.com/samber/cc-skills-golang.git",
	"google-developer-knowledge": "https://github.com/google/skills.git",
}

type source struct {
	Name       string `json:"name"`
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	SkillRoot  string `json:"skill_root"`
}
type receipt struct {
	Source source `json:"source"`
	Digest string `json:"digest"`
}
type item struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Detail string `json:"detail"`
}
type report struct {
	Sources       []item `json:"sources"`
	Prerequisites []item `json:"prerequisites"`
	HostVerified  bool   `json:"host_verified"`
	Marketplace   string `json:"google_marketplace"`
	NativePlugin  string `json:"google_native_plugin"`
}

func validateSources(sources []source) error {
	if len(sources) == 0 {
		return errors.New("empty source lock")
	}
	seen := map[string]bool{}
	for _, s := range sources {
		if approvedSources[s.Name] != s.Repository || s.Repository == "" || !shaPattern.MatchString(s.Commit) || seen[s.Name] {
			return fmt.Errorf("invalid or duplicate approved source %q", s.Name)
		}
		if !fs.ValidPath(s.SkillRoot) || strings.Contains(s.SkillRoot, "\\") {
			return errors.New("invalid skill path")
		}
		expected := "skills"
		if s.Name == "google-developer-knowledge" {
			expected = "plugins/cloud/google-cloud-developer/skills/retrieving-developer-knowledge"
		}
		if s.SkillRoot != expected {
			return fmt.Errorf("unapproved skill root for %s", s.Name)
		}
		seen[s.Name] = true
	}
	return nil
}

func loadSources(file string) ([]source, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var sources []source
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&sources); err != nil {
		return nil, err
	}
	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New("source lock contains trailing data")
	}
	return sources, validateSources(sources)
}

func digestTree(root string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if p == root {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		if rel == receiptName {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("installed skill contains a symlink")
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("non-regular skill file")
		}
		fmt.Fprintf(h, "%s\x00%d\x00", filepath.ToSlash(rel), info.Size())
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(h, f)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func installationState(home string, s source) (string, string) {
	target := filepath.Join(home, ".agents", "skills", s.Name)
	info, err := os.Lstat(target)
	if os.IsNotExist(err) {
		return "missing", "approved skills are not installed"
	}
	if err != nil {
		return "blocked", "cannot inspect installation"
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "blocked", "target is not a managed real directory"
	}
	data, err := os.ReadFile(filepath.Join(target, receiptName))
	if err != nil {
		return "blocked", "existing directory has no managed receipt; it will not be overwritten"
	}
	var r receipt
	if json.Unmarshal(data, &r) != nil || r.Source != s {
		return "blocked", "source or pinned revision differs; explicit reviewed update required"
	}
	digest, err := digestTree(target)
	if err != nil || digest != r.Digest {
		return "blocked", "installed content changed; it will not be overwritten"
	}
	return "installed", "pinned skill files match receipt; host discovery and execution remain unverified"
}

func inspect(home string, sources []source) report {
	r := report{Marketplace: "not-verified", NativePlugin: "not-installed-by-this-tool: google-cloud-developer includes duplicate MCP and ON_INSTALL authentication"}
	for _, s := range sources {
		state, detail := installationState(home, s)
		r.Sources = append(r.Sources, item{s.Name, state, detail})
	}
	for _, name := range []string{"GITHUB_MCP_TOKEN", "DEVELOPERKNOWLEDGE_API_KEY"} {
		state := "missing"
		if os.Getenv(name) != "" {
			state = "present-unverified"
		}
		r.Prerequisites = append(r.Prerequisites, item{name, state, "only presence checked; value is never reported"})
	}
	for _, name := range []string{"codex", "git"} {
		state := "missing"
		if _, err := exec.LookPath(name); err == nil {
			state = "executable-present"
		}
		r.Prerequisites = append(r.Prerequisites, item{name, state, "presence is not a successful host or MCP check"})
	}
	state := "missing"
	if info, err := os.Stat(filepath.Join(home, ".local", "bin", "gopls")); err == nil && info.Mode().IsRegular() && info.Mode()&0111 != 0 {
		state = "executable-present"
	}
	r.Prerequisites = append(r.Prerequisites, item{"gopls", state, "setup-installed path; MCP handshake not checked"})
	return r
}

// extractSkills rejects unsafe archive members rather than following links or rewriting paths.
func extractSkills(data []byte, dest, root string) error {
	if len(data) > maxArchiveBytes {
		return errors.New("archive exceeds size limit")
	}
	tr := tar.NewReader(bytes.NewReader(data))
	count := 0
	var total int64
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(h.Name, "/")
		if !fs.ValidPath(name) || strings.Contains(name, "\\") || path.Clean(name) != name {
			return errors.New("unsafe archive path")
		}
		if h.Typeflag == tar.TypeXGlobalHeader {
			// git archive emits a PAX global comment containing the commit ID.
			continue
		}
		if h.Typeflag == tar.TypeDir {
			continue
		}
		if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeRegA {
			return errors.New("archive links and special files are forbidden")
		}
		if h.Size < 0 || h.Size > 16<<20 {
			return errors.New("archive file exceeds size limit")
		}
		total += h.Size
		if total > maxArchiveBytes {
			return errors.New("expanded archive exceeds size limit")
		}
		rel := ""
		if name == "LICENSE" {
			rel = "UPSTREAM-LICENSE"
		} else if strings.HasPrefix(name, root+"/") {
			rel = strings.TrimPrefix(name, root+"/")
		} else {
			return errors.New("archive member outside approved skill root")
		}
		if !fs.ValidPath(rel) || rel == receiptName {
			return errors.New("unsafe installation path")
		}
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, copyErr := io.CopyN(f, tr, h.Size)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if filepath.Base(rel) == "SKILL.md" {
			count++
		}
	}
	if count == 0 {
		return errors.New("approved source contains no SKILL.md")
	}
	return nil
}

func realDirectories(paths ...string) error {
	for _, p := range paths {
		if err := os.Mkdir(p, 0700); err != nil && !os.IsExist(err) {
			return err
		}
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing non-directory or symlink: %s", p)
		}
	}
	return nil
}

func installArchive(home string, s source, data []byte) error {
	if err := validateSources([]source{s}); err != nil {
		return err
	}
	state, detail := installationState(home, s)
	if state == "installed" {
		return nil
	}
	if state != "missing" {
		return errors.New(detail)
	}
	agents := filepath.Join(home, ".agents")
	skills := filepath.Join(agents, "skills")
	if err := realDirectories(agents, skills); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(agents, ".codex-install-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if err = extractSkills(data, stage, s.SkillRoot); err != nil {
		return err
	}
	digest, err := digestTree(stage)
	if err != nil {
		return err
	}
	record, err := json.MarshalIndent(receipt{s, digest}, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(stage, receiptName), record, 0600); err != nil {
		return err
	}
	target := filepath.Join(skills, s.Name)
	if _, err = os.Lstat(target); !os.IsNotExist(err) {
		return errors.New("target appeared during installation; refusing overwrite")
	}
	return os.Rename(stage, target)
}

type commandRunner func(string, ...string) ([]byte, error)

func runCommand(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stdout limitedBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s command failed; authentication/network/permissions may be required (output redacted)", name)
	}
	return stdout.Bytes(), nil
}

type limitedBuffer struct{ data bytes.Buffer }

func (b *limitedBuffer) Bytes() []byte { return b.data.Bytes() }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.data.Len()+len(p) > maxArchiveBytes {
		return 0, errors.New("command output exceeds limit")
	}
	return b.data.Write(p)
}

func downloadSkills(s source, run commandRunner) ([]byte, error) {
	temp, err := os.MkdirTemp("", "codex-source-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temp)
	if _, err = run("git", "init", "--bare", temp); err != nil {
		return nil, err
	}
	args := []string{"-c", "protocol.file.allow=never", "-c", "core.hooksPath=/dev/null", "-C", temp, "fetch", "--depth=1", s.Repository, s.Commit}
	if _, err = run("git", args...); err != nil {
		return nil, err
	}
	data, err := run("git", "-C", temp, "rev-parse", "FETCH_HEAD")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(data)) != s.Commit {
		return nil, errors.New("fetched commit differs from approved pin")
	}
	// git archive does not run a checkout, repository hooks, or downloaded setup scripts.
	return run("git", "-C", temp, "archive", "--format=tar", s.Commit, "--", s.SkillRoot, "LICENSE")
}

// marketplaceRecords accepts only recognized listing shapes. An unknown host
// schema is a blocker, never evidence that a marketplace is absent.
func marketplaceRecords(data []byte) ([]map[string]any, error) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, errors.New("invalid marketplace JSON")
	}
	if object, ok := value.(map[string]any); ok {
		var exists bool
		value, exists = object["marketplaces"]
		if !exists {
			return nil, errors.New("unsupported marketplace listing object")
		}
	}
	array, ok := value.([]any)
	if !ok {
		return nil, errors.New("unsupported marketplace listing shape")
	}
	records := make([]map[string]any, 0, len(array))
	seen := map[string]bool{}
	for _, entry := range array {
		record, ok := entry.(map[string]any)
		if !ok {
			return nil, errors.New("unsupported marketplace record")
		}
		name, ok := record["name"].(string)
		if !ok || name == "" || seen[name] {
			return nil, errors.New("missing or duplicate marketplace name")
		}
		seen[name] = true
		records = append(records, record)
	}
	return records, nil
}

func marketplaceRecord(data []byte, name, pin string) (bool, bool) {
	records, err := marketplaceRecords(data)
	if err != nil {
		return false, false
	}
	for _, record := range records {
		if record["name"] != name {
			continue
		}
		url, _ := record["source"].(string)
		ref, _ := record["ref"].(string)
		sha, _ := record["sha"].(string)
		if source, ok := record["source"].(map[string]any); ok {
			url, _ = source["url"].(string)
			if sourceRef, ok := source["ref"].(string); ok {
				ref = sourceRef
			}
			if sourceSHA, ok := source["sha"].(string); ok {
				sha = sourceSHA
			}
		}
		approved := url == "google/skills" || url == "https://github.com/google/skills" || url == "https://github.com/google/skills.git"
		return true, approved && (ref == pin || sha == pin)
	}
	return false, false
}

func registerGoogle(s source, run commandRunner) (string, error) {
	help, err := run("codex", "plugin", "marketplace", "add", "--help")
	if err != nil || !bytes.Contains(help, []byte("--ref")) {
		return "unsupported", errors.New("host lacks pinned marketplace add; use its /plugins browser after reviewing source")
	}
	listing, err := run("codex", "plugin", "marketplace", "list", "--json")
	if err != nil {
		return "unverified", err
	}
	if _, err := marketplaceRecords(listing); err != nil {
		return "unverified", err
	}
	if found, pinned := marketplaceRecord(listing, "google-plugins", s.Commit); found {
		if pinned {
			return "registered-pinned", nil
		}
		return "blocked", errors.New("existing Google marketplace pin cannot be verified; not overwritten")
	}
	if _, err = run("codex", "plugin", "marketplace", "add", "google/skills", "--ref", s.Commit); err != nil {
		return "failed", err
	}
	listing, err = run("codex", "plugin", "marketplace", "list", "--json")
	if err != nil {
		return "unverified", err
	}
	found, pinned := marketplaceRecord(listing, "google-plugins", s.Commit)
	if !found || !pinned {
		return "unverified", errors.New("marketplace added but host listing does not prove the pinned revision")
	}
	return "registered-pinned", nil
}

func run(args []string) int {
	flags := flag.NewFlagSet("codexplugins", flag.ContinueOnError)
	install := flags.Bool("install", false, "install approved pinned skills; never execute their scripts")
	marketplace := flags.Bool("register-google-marketplace", false, "register pinned Google marketplace; requires --install")
	lock := flags.String("lock", ".codex/plugins.lock.json", "reviewed source lock")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	if flags.NArg() != 0 || (*marketplace && !*install) {
		fmt.Fprintln(os.Stderr, "invalid arguments: marketplace changes require --install")
		return 1
	}
	sources, err := loadSources(*lock)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	r := inspect(home, sources)
	exit := 0
	if *install {
		agents := filepath.Join(home, ".agents")
		if err := realDirectories(agents); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		guard := filepath.Join(agents, ".codex-plugin-install.lock")
		if err := os.Mkdir(guard, 0700); err != nil {
			fmt.Fprintln(os.Stderr, "installation lock exists or is inaccessible; no changes made")
			return 2
		}
		defer os.Remove(guard)
		for _, s := range sources {
			state, _ := installationState(home, s)
			if state == "installed" {
				continue
			}
			if state != "missing" {
				exit = 2
				continue
			}
			data, err := downloadSkills(s, runCommand)
			if err == nil {
				err = installArchive(home, s, data)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", s.Name, err)
				exit = 2
			}
		}
		r = inspect(home, sources)
		if *marketplace {
			for _, s := range sources {
				if s.Name == "google-developer-knowledge" {
					state, err := registerGoogle(s, runCommand)
					r.Marketplace = state
					if err != nil {
						fmt.Fprintln(os.Stderr, err)
						exit = 2
					}
				}
			}
		}
	}
	for _, s := range r.Sources {
		if s.State != "installed" {
			exit = 2
		}
	}
	sort.Slice(r.Sources, func(i, j int) bool { return r.Sources[i].Name < r.Sources[j].Name })
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(r); err != nil {
		return 1
	}
	return exit
}
func main() { os.Exit(run(os.Args[1:])) }
