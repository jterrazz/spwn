package tool

import (
	"bytes"
	"errors"
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
	_, err := runWithOut(t, showCmd, "@spwn/python")
	// show is deliberately unimplemented — it must signal
	// feature-unavailable (exit 2) rather than succeed silently.
	if err == nil {
		t.Fatal("show stub should return a not-implemented error")
	}
	var coder interface{ ExitCode() int }
	if !errorAs(err, &coder) || coder.ExitCode() != 2 {
		t.Errorf("expected ExitCode()==2, got err=%v", err)
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
	_, err := runWithOut(t, getCmd, "@spwn/python")
	// get is deliberately unimplemented — expect feature-unavailable.
	if err == nil {
		t.Fatal("get stub should return a not-implemented error")
	}
	var coder interface{ ExitCode() int }
	if !errorAs(err, &coder) || coder.ExitCode() != 2 {
		t.Errorf("expected ExitCode()==2, got err=%v", err)
	}
}

// errorAs is a minimal errors.As bridge to avoid pulling the stdlib
// import for a single call site. It walks Unwrap chains and assigns
// the first match to target.
func errorAs(err error, target interface{}) bool {
	return errors.As(err, target)
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
