package main

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureSource() source {
	return source{Name: "cc-skills", Repository: "https://github.com/samber/cc-skills.git", Commit: strings.Repeat("a", 40), SkillRoot: "skills"}
}

func TestBootstrapRepository(t *testing.T) {
	want := t.TempDir()
	t.Setenv("BOOTSTRAP_REPO", want)
	got, err := bootstrapRepository()
	if err != nil || got != want {
		t.Fatalf("bootstrapRepository() = %q, %v; want %q", got, err, want)
	}
}

func TestBootstrapRepositoryRejectsMissingAndRelativeValues(t *testing.T) {
	for _, value := range []string{"", "relative/repo"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("BOOTSTRAP_REPO", value)
			if _, err := bootstrapRepository(); err == nil {
				t.Fatalf("BOOTSTRAP_REPO %q was accepted", value)
			}
		})
	}
}

func TestPrepareAgentHomeCreatesFirstRunDirectories(t *testing.T) {
	home := filepath.Join(t.TempDir(), "managed-home")
	agents, err := prepareAgentHome(home)
	if err != nil {
		t.Fatal(err)
	}
	if agents != filepath.Join(home, ".agents") {
		t.Fatalf("agents path = %q", agents)
	}
	for _, path := range []string{home, agents} {
		info, statErr := os.Lstat(path)
		if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("managed directory %q invalid: %v", path, statErr)
		}
	}
}
func archive(t *testing.T, names ...string) []byte {
	t.Helper()
	var b bytes.Buffer
	w := tar.NewWriter(&b)
	for _, name := range names {
		data := []byte("---\nname: example\n---\nExample\n")
		if err := w.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestValidateSources(t *testing.T) {
	if err := validateSources([]source{fixtureSource()}); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*source){func(s *source) { s.Commit = "main" }, func(s *source) { s.Repository = "https://evil.example/repo" }, func(s *source) { s.SkillRoot = "../escape" }, func(s *source) { s.Name = "../escape" }} {
		s := fixtureSource()
		change(&s)
		if validateSources([]source{s}) == nil {
			t.Fatalf("invalid source accepted: %+v", s)
		}
	}
	if validateSources([]source{fixtureSource(), fixtureSource()}) == nil {
		t.Fatal("duplicate accepted")
	}
}
func TestExtractRejectsTraversalAndSymlinks(t *testing.T) {
	for _, name := range []string{"../escape", "/tmp/escape", "skills/../../escape", "skills\\escape"} {
		if err := extractSkills(archive(t, name), t.TempDir(), "skills"); err == nil {
			t.Errorf("accepted %q", name)
		}
	}
	var b bytes.Buffer
	w := tar.NewWriter(&b)
	if err := w.WriteHeader(&tar.Header{Name: "skills/link", Typeflag: tar.TypeSymlink, Linkname: "../../outside"}); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if extractSkills(b.Bytes(), t.TempDir(), "skills") == nil {
		t.Fatal("symlink accepted")
	}
}
func TestInstallIdempotentAndPreservesChanges(t *testing.T) {
	home := t.TempDir()
	s := fixtureSource()
	data := archive(t, "skills/example/SKILL.md")
	if err := installArchive(home, s, data); err != nil {
		t.Fatal(err)
	}
	if err := installArchive(home, s, data); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(home, ".agents", "skills", s.Name, "example", "SKILL.md")
	if err := os.WriteFile(target, []byte("user edits"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := installArchive(home, s, data); err == nil {
		t.Fatal("overwrote modified installation")
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "user edits" {
		t.Fatal("user changes lost")
	}
}
func TestMissingSkillDoesNotInstall(t *testing.T) {
	home := t.TempDir()
	if err := installArchive(home, fixtureSource(), archive(t, "skills/README.md")); err == nil {
		t.Fatal("missing SKILL accepted")
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "cc-skills")); !os.IsNotExist(err) {
		t.Fatal("failed install published")
	}
}
func TestCheckIsReadOnlyAndDoesNotExposeSecrets(t *testing.T) {
	home := filepath.Join(t.TempDir(), "absent-home")
	t.Setenv("DEVELOPERKNOWLEDGE_API_KEY", "private-test-value")
	r := inspect(home, []source{fixtureSource()})
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private-test-value") {
		t.Fatal("secret leaked")
	}
	if r.HostVerified {
		t.Fatal("static report claimed runtime verification")
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatal("check created files")
	}
}
func TestExistingDirectoryAndSymlinkProtected(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, ".agents", "skills", "cc-skills")
	if err := os.MkdirAll(target, 0700); err != nil {
		t.Fatal(err)
	}
	if err := installArchive(home, fixtureSource(), archive(t, "skills/example/SKILL.md")); err == nil {
		t.Fatal("unmanaged target overwritten")
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), target); err != nil {
		t.Skip(err)
	}
	if err := installArchive(home, fixtureSource(), archive(t, "skills/example/SKILL.md")); err == nil {
		t.Fatal("symlink accepted")
	}
}
func TestMarketplaceRecordRequiresPinnedRef(t *testing.T) {
	data := []byte(`[{"name":"google-plugins","source":"google/skills","ref":"abc"}]`)
	found, pinned := marketplaceRecord(data, "google-plugins", "abc")
	if !found || !pinned {
		t.Fatal("pinned record not found")
	}
	_, pinned = marketplaceRecord(data, "google-plugins", "other")
	if pinned {
		t.Fatal("wrong ref accepted")
	}
	found, _ = marketplaceRecord([]byte(`not json`), "google-plugins", "abc")
	if found {
		t.Fatal("invalid JSON accepted")
	}
}

func TestGitArchivePAXMetadata(t *testing.T) {
	var b bytes.Buffer
	w := tar.NewWriter(&b)
	if err := w.WriteHeader(&tar.Header{Name: "pax_global_header", Typeflag: tar.TypeXGlobalHeader, PAXRecords: map[string]string{"comment": "pinned-commit"}}); err != nil {
		t.Fatal(err)
	}
	data := []byte("example")
	if err := w.WriteHeader(&tar.Header{Name: "skills/example/SKILL.md", Typeflag: tar.TypeReg, Size: int64(len(data)), Mode: 0600}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := extractSkills(b.Bytes(), t.TempDir(), "skills"); err != nil {
		t.Fatal(err)
	}
}
