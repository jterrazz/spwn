package cli

import (
	"strings"
	"testing"
)

func TestArchitectMode_BlockedCommands(t *testing.T) {
	t.Setenv("SPWN_ARCHITECT_MODE", "1")

	// Note: --help bypasses PersistentPreRunE, so we test with bare commands.
	// "auth" (bare) runs its RunE.
	_, _, err := executeCommand("auth")
	if err == nil {
		t.Error("expected \"auth\" to be blocked in Architect mode")
	} else if !strings.Contains(err.Error(), "not available in Architect mode") {
		t.Errorf("expected Architect mode error for \"auth\", got: %s", err)
	}
}

func TestArchitectMode_AllowedCommands(t *testing.T) {
	t.Setenv("SPWN_ARCHITECT_MODE", "1")

	for _, cmd := range []string{"world", "agent", "architect", "install"} {
		_, _, err := executeCommand(cmd, "--help")
		if err != nil {
			t.Errorf("expected %q to be allowed in Architect mode, got: %s", cmd, err)
		}
	}
}

func TestArchitectMode_RootHelpAllowed(t *testing.T) {
	t.Setenv("SPWN_ARCHITECT_MODE", "1")

	out, _, err := executeCommand("--help")
	if err != nil {
		t.Fatalf("root --help should work in Architect mode: %s", err)
	}
	if !strings.Contains(out, "spwn") {
		t.Error("root help should contain 'spwn'")
	}
}

func TestArchitectMode_NotActive(t *testing.T) {
	t.Setenv("SPWN_ARCHITECT_MODE", "")

	// auth --help should work when Architect mode is off
	_, _, err := executeCommand("auth", "--help")
	if err != nil {
		t.Errorf("auth should work when Architect mode is off: %s", err)
	}
}
