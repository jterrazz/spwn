package skill

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// runWithOut executes a cobra.Command's RunE with a fresh output buffer.
// Returns the buffer and any error from RunE.
func runWithOut(t *testing.T, c *cobra.Command, args ...string) (*bytes.Buffer, error) {
	t.Helper()
	out := new(bytes.Buffer)
	cmd := &cobra.Command{Use: "test"}
	cmd.SetOut(out)
	cmd.SetErr(out)
	return out, c.RunE(cmd, args)
}

func setupTempHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	return tmp
}

// ── ls ───────────────────────────────────────────────────────────────────────

func TestSkillLs_EmptyDirectory(t *testing.T) {
	setupTempHome(t)
	out, err := runWithOut(t, lsCmd)
	if err != nil {
		t.Fatalf("ls: %v", err)
	}
	if !strings.Contains(out.String(), "No skills authored") {
		t.Errorf("expected empty-state message, got: %s", out.String())
	}
}

func TestSkillLs_ListsAuthoredSkills(t *testing.T) {
	home := setupTempHome(t)
	dir := filepath.Join(home, "skills")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "paper-reading.md"), []byte("# Paper Reading"), 0o644)
	os.WriteFile(filepath.Join(dir, "refactoring.md"), []byte("# Refactoring"), 0o644)
	// A non-md file should be ignored.
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignore"), 0o644)

	out, err := runWithOut(t, lsCmd)
	if err != nil {
		t.Fatalf("ls: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "paper-reading") {
		t.Errorf("expected paper-reading in output, got: %s", s)
	}
	if !strings.Contains(s, "refactoring") {
		t.Errorf("expected refactoring in output, got: %s", s)
	}
	if strings.Contains(s, "notes") {
		t.Errorf("non-md file should be filtered, got: %s", s)
	}
}

// ── new ──────────────────────────────────────────────────────────────────────

func TestSkillNew_CreatesTemplateFile(t *testing.T) {
	home := setupTempHome(t)
	_, err := runWithOut(t, newCmd, "paper-reading")
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	path := filepath.Join(home, "skills", "paper-reading.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# paper-reading") {
		t.Errorf("template missing title: %s", content)
	}
	if !strings.Contains(content, "When to use") {
		t.Errorf("template missing 'When to use' section: %s", content)
	}
	if !strings.Contains(content, "Steps") {
		t.Errorf("template missing 'Steps' section: %s", content)
	}
}

func TestSkillNew_DuplicateErrors(t *testing.T) {
	setupTempHome(t)
	// First creation succeeds.
	if _, err := runWithOut(t, newCmd, "refactoring"); err != nil {
		t.Fatal(err)
	}
	// Second should fail.
	if _, err := runWithOut(t, newCmd, "refactoring"); err == nil {
		t.Error("expected error creating duplicate skill, got nil")
	}
}

// ── show ─────────────────────────────────────────────────────────────────────

func TestSkillShow_DisplaysContent(t *testing.T) {
	home := setupTempHome(t)
	dir := filepath.Join(home, "skills")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "paper-reading.md"), []byte("# Paper Reading\n\nHow to read papers."), 0o644)

	out, err := runWithOut(t, showCmd, "paper-reading")
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if !strings.Contains(out.String(), "How to read papers.") {
		t.Errorf("show output missing content: %s", out.String())
	}
}

func TestSkillShow_NotFoundErrors(t *testing.T) {
	setupTempHome(t)
	if _, err := runWithOut(t, showCmd, "no-such-skill"); err == nil {
		t.Error("expected error when skill doesn't exist")
	}
}

// ── rm ───────────────────────────────────────────────────────────────────────

func TestSkillRm_RemovesFile(t *testing.T) {
	home := setupTempHome(t)
	dir := filepath.Join(home, "skills")
	os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, "refactoring.md")
	os.WriteFile(path, []byte("# Refactoring"), 0o644)

	if _, err := runWithOut(t, rmCmd, "refactoring"); err != nil {
		t.Fatalf("rm: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("skill file should be gone, got: %v", err)
	}
}

func TestSkillRm_NotFoundErrors(t *testing.T) {
	setupTempHome(t)
	if _, err := runWithOut(t, rmCmd, "ghost"); err == nil {
		t.Error("expected error when removing absent skill")
	}
}

// ── edit ─────────────────────────────────────────────────────────────────────

func TestSkillEdit_NotFoundErrors(t *testing.T) {
	setupTempHome(t)
	if _, err := runWithOut(t, editCmd, "ghost"); err == nil {
		t.Error("expected error when editing absent skill")
	}
}

// ── stubs (install / publish) ────────────────────────────────────────────────

func TestSkillInstall_Stub(t *testing.T) {
	out, err := runWithOut(t, installCmd, "@community/rust-review")
	if err != nil {
		t.Fatalf("install stub: %v", err)
	}
	if !strings.Contains(out.String(), "registry") {
		t.Errorf("stub should mention registry: %s", out.String())
	}
}

func TestSkillPublish_Stub(t *testing.T) {
	out, err := runWithOut(t, publishCmd, "refactoring")
	if err != nil {
		t.Fatalf("publish stub: %v", err)
	}
	if !strings.Contains(out.String(), "registry") {
		t.Errorf("stub should mention registry: %s", out.String())
	}
}
