package tool

import (
	"bytes"
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

// ── ls ───────────────────────────────────────────────────────────────────────

func TestToolLs_ListsBuiltInPacks(t *testing.T) {
	out, err := runWithOut(t, lsCmd)
	if err != nil {
		t.Fatalf("ls: %v", err)
	}
	s := out.String()
	for _, pack := range []string{"@spwn/unix", "@spwn/git", "@spwn/node", "@spwn/python"} {
		if !strings.Contains(s, pack) {
			t.Errorf("expected %s in output, got: %s", pack, s)
		}
	}
}

// ── stubs (show / search / install / rm / publish) ──────────────────────────

func TestToolShow_Stub(t *testing.T) {
	out, err := runWithOut(t, showCmd, "@spwn/python")
	if err != nil {
		t.Fatalf("show stub: %v", err)
	}
	if !strings.Contains(out.String(), "not yet wired") {
		t.Errorf("stub should mention 'not yet wired': %s", out.String())
	}
}

func TestToolSearch_Stub(t *testing.T) {
	out, err := runWithOut(t, searchCmd, "python")
	if err != nil {
		t.Fatalf("search stub: %v", err)
	}
	if !strings.Contains(out.String(), "registry") {
		t.Errorf("stub should mention registry: %s", out.String())
	}
}

func TestToolInstall_Stub(t *testing.T) {
	out, err := runWithOut(t, getCmd, "@spwn/python")
	if err != nil {
		t.Fatalf("install stub: %v", err)
	}
	if !strings.Contains(out.String(), "Built-in packs") {
		t.Errorf("install stub should mention built-in packs: %s", out.String())
	}
}

func TestToolRm_Stub(t *testing.T) {
	out, err := runWithOut(t, rmCmd, "@spwn/python")
	if err != nil {
		t.Fatalf("rm stub: %v", err)
	}
	if !strings.Contains(out.String(), "no tools") {
		t.Errorf("rm stub should mention no tools installed: %s", out.String())
	}
}

func TestToolPublish_Stub(t *testing.T) {
	out, err := runWithOut(t, publishCmd, "./my-tool")
	if err != nil {
		t.Fatalf("publish stub: %v", err)
	}
	if !strings.Contains(out.String(), "registry") {
		t.Errorf("stub should mention registry: %s", out.String())
	}
}
