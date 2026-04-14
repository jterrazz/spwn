package profile

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

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

func TestProfileLs_EmptyDirectory(t *testing.T) {
	setupTempHome(t)
	out, err := runWithOut(t, lsCmd)
	if err != nil {
		t.Fatalf("ls: %v", err)
	}
	if !strings.Contains(out.String(), "No profiles authored") {
		t.Errorf("expected empty-state message, got: %s", out.String())
	}
}

func TestProfileLs_ListsAuthoredProfiles(t *testing.T) {
	home := setupTempHome(t)
	dir := filepath.Join(home, "profiles")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "researcher.md"), []byte("# Researcher"), 0o644)
	os.WriteFile(filepath.Join(dir, "the-one.md"), []byte("# The One"), 0o644)

	out, err := runWithOut(t, lsCmd)
	if err != nil {
		t.Fatalf("ls: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "researcher") {
		t.Errorf("expected researcher in output, got: %s", s)
	}
	if !strings.Contains(s, "the-one") {
		t.Errorf("expected the-one in output, got: %s", s)
	}
}

// ── new ──────────────────────────────────────────────────────────────────────

func TestProfileNew_CreatesTemplate(t *testing.T) {
	home := setupTempHome(t)
	_, err := runWithOut(t, newCmd, "researcher")
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	path := filepath.Join(home, "profiles", "researcher.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "name: researcher") {
		t.Errorf("template missing frontmatter name: %s", content)
	}
	if !strings.Contains(content, "# researcher") {
		t.Errorf("template missing title: %s", content)
	}
	if !strings.Contains(content, "## Style") {
		t.Errorf("template missing Style section: %s", content)
	}
}

func TestProfileNew_DuplicateErrors(t *testing.T) {
	setupTempHome(t)
	if _, err := runWithOut(t, newCmd, "researcher"); err != nil {
		t.Fatal(err)
	}
	if _, err := runWithOut(t, newCmd, "researcher"); err == nil {
		t.Error("expected error on duplicate profile, got nil")
	}
}

// ── show ─────────────────────────────────────────────────────────────────────

func TestProfileShow_DisplaysContent(t *testing.T) {
	home := setupTempHome(t)
	dir := filepath.Join(home, "profiles")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "researcher.md"), []byte("# Researcher\n\nCurious and methodical."), 0o644)

	out, err := runWithOut(t, showCmd, "researcher")
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if !strings.Contains(out.String(), "Curious and methodical") {
		t.Errorf("show output missing content: %s", out.String())
	}
}

func TestProfileShow_NotFoundErrors(t *testing.T) {
	setupTempHome(t)
	if _, err := runWithOut(t, showCmd, "ghost"); err == nil {
		t.Error("expected error when profile doesn't exist")
	}
}

// ── rm ───────────────────────────────────────────────────────────────────────

func TestProfileRm_RemovesFile(t *testing.T) {
	home := setupTempHome(t)
	dir := filepath.Join(home, "profiles")
	os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, "researcher.md")
	os.WriteFile(path, []byte("# Researcher"), 0o644)

	if _, err := runWithOut(t, rmCmd, "researcher"); err != nil {
		t.Fatalf("rm: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("profile file should be gone, got: %v", err)
	}
}

func TestProfileRm_NotFoundErrors(t *testing.T) {
	setupTempHome(t)
	if _, err := runWithOut(t, rmCmd, "ghost"); err == nil {
		t.Error("expected error when removing absent profile")
	}
}

// ── edit ─────────────────────────────────────────────────────────────────────

func TestProfileEdit_NotFoundErrors(t *testing.T) {
	setupTempHome(t)
	if _, err := runWithOut(t, editCmd, "ghost"); err == nil {
		t.Error("expected error when editing absent profile")
	}
}

// ── stubs ────────────────────────────────────────────────────────────────────

func TestProfileInstall_Stub(t *testing.T) {
	out, err := runWithOut(t, getCmd, "@community/pragmatic-dev")
	if err != nil {
		t.Fatalf("install stub: %v", err)
	}
	if !strings.Contains(out.String(), "registry") {
		t.Errorf("stub should mention registry: %s", out.String())
	}
}

func TestProfilePublish_Stub(t *testing.T) {
	out, err := runWithOut(t, publishCmd, "researcher")
	if err != nil {
		t.Fatalf("publish stub: %v", err)
	}
	if !strings.Contains(out.String(), "registry") {
		t.Errorf("stub should mention registry: %s", out.String())
	}
}
