package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func validHandoff(phase string) handoff {
	h := handoff{
		Version: "1", HandoffID: "task-1", Phase: phase, Task: "verify handoff", Owner: "orchestrator",
		Role: "reviewer", RoleMode: "explicit", RolePath: ".codex/agents/reviewer.toml", RoleSHA256: strings.Repeat("d", 64), Workspace: "/workspace/repo",
		Head: strings.Repeat("a", 40), ScopedFingerprint: strings.Repeat("b", 64),
		Scope: []string{"cmd/codexhandoff"}, Acceptance: []string{"tests pass"},
		Skills:       []skillEvidence{{Path: ".agents/skills/example/SKILL.md", SHA256: strings.Repeat("c", 64), Status: "selected-parent-read"}},
		Capabilities: []capability{{Name: "repomix", Required: true, State: "call-verified", SelectedReason: "cross-file evidence", HostNative: "exposed", LocalSmoke: "local-smoke", Authentication: "not-verified", Authorization: "authorized", EvidenceTimeUTC: time.Now().UTC().Format(time.RFC3339)}},
	}
	if phase == "ready" {
		h.Skills[0].Status = "child-read-verified"
	}
	if phase == "result" {
		h.State, h.NextAction = "COMPLETED", "review"
	}
	return h
}

func TestValidatePhases(t *testing.T) {
	for _, phase := range []string{"prepare", "ready", "start", "result"} {
		if err := validateHandoff(validHandoff(phase), phase); err != nil {
			t.Errorf("%s: %v", phase, err)
		}
	}
}

func TestRejectsUnavailableRequiredCapability(t *testing.T) {
	h := validHandoff("ready")
	h.Capabilities[0].State = "unavailable"
	if err := validateHandoff(h, "ready"); err == nil {
		t.Fatal("expected required capability rejection")
	}
}

func TestTransitionBindsPriorAndCapabilities(t *testing.T) {
	prepare := validHandoff("prepare")
	ready := validHandoff("ready")
	digest, err := documentHash(prepare)
	if err != nil {
		t.Fatal(err)
	}
	ready.PreviousSHA256 = digest
	if err := validateTransition(prepare, ready); err != nil {
		t.Fatal(err)
	}
	ready.Capabilities = nil
	if err := validateTransition(prepare, ready); err == nil {
		t.Fatal("expected dropped capability rejection")
	}
	ready = validHandoff("ready")
	ready.PreviousSHA256 = strings.Repeat("f", 64)
	if err := validateTransition(prepare, ready); err == nil {
		t.Fatal("expected forged prior hash rejection")
	}
}

func TestRejectsUnprovenSkillAndResult(t *testing.T) {
	h := validHandoff("ready")
	h.Skills[0].Status = "selected-parent-read"
	if err := validateHandoff(h, "ready"); err == nil {
		t.Fatal("expected READY skill evidence rejection")
	}
	h = validHandoff("result")
	h.State = "PROPOSED"
	if err := validateHandoff(h, "result"); err == nil {
		t.Fatal("expected result state rejection")
	}
}

func TestStrictJSONAndPhase(t *testing.T) {
	h := validHandoff("prepare")
	b, _ := json.Marshal(h)
	var out, stderr bytes.Buffer
	if code := validateCommand([]string{"--phase", "prepare"}, bytes.NewReader(b), &out, &stderr); code != 0 {
		t.Fatalf("validate failed: %s", stderr.String())
	}
	b = append(b[:len(b)-1], []byte(",\"unknown\":true}")...)
	if code := validateCommand([]string{"--phase", "prepare"}, bytes.NewReader(b), &out, &stderr); code == 0 {
		t.Fatal("expected unknown field rejection")
	}
}

func TestCleanPathsRejectsEscape(t *testing.T) {
	if _, err := cleanPaths(t.TempDir(), []string{"../outside"}); err == nil {
		t.Fatal("expected path escape rejection")
	}
}

func TestScopedFingerprintTracksStagedAndUntrackedContents(t *testing.T) {
	root := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	runGit("init")
	runGit("config", "user.email", "test@example.invalid")
	runGit("config", "user.name", "test")
	tracked := filepath.Join(root, "tracked.txt")
	untracked := filepath.Join(root, "untracked.txt")
	if err := os.WriteFile(tracked, []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "tracked.txt")
	runGit("commit", "-m", "base")
	if err := os.WriteFile(tracked, []byte("staged-one"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "tracked.txt")
	first, err := scopedFingerprint(root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tracked, []byte("staged-two"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "tracked.txt")
	second, err := scopedFingerprint(root, []string{"."})
	if err != nil || first == second {
		t.Fatalf("staged content collision: %v", err)
	}
	if err := os.WriteFile(untracked, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	third, err := scopedFingerprint(root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(untracked, []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	fourth, err := scopedFingerprint(root, []string{"."})
	if err != nil || third == fourth {
		t.Fatalf("untracked content collision: %v", err)
	}
}

func TestCleanPathsRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := cleanPaths(root, []string{"escape"}); err == nil {
		t.Fatal("expected symlink escape rejection")
	}
	if _, err := cleanPaths(root, []string{"escape/new.go"}); err == nil {
		t.Fatal("expected nonexistent child below symlink rejection")
	}
}
