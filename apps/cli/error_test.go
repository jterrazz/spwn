package cli

import (
	"strings"
	"testing"

	"spwn.sh/apps/cli/ui"
)

// --- Error suppression ---

func TestCLI_UnknownCommandNoUsageDump(t *testing.T) {
	_, stderr, err := executeCommand("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown command")
	}

	// SilenceUsage should prevent Cobra from dumping the full usage
	if strings.Contains(stderr, "Available Commands:") {
		t.Error("unknown command should NOT dump full usage")
	}
	if strings.Contains(stderr, "Flags:") && strings.Contains(stderr, "--json") {
		t.Error("unknown command should NOT show all flags")
	}
}

func TestCLI_ErrorNoUsageDump_WorldDestroyMissingArg(t *testing.T) {
	_, stderr, err := executeCommand("world", "destroy")
	if err == nil {
		t.Fatal("expected error for missing argument")
	}

	// Should NOT dump full world help on missing arg
	if strings.Contains(stderr, "Available Commands:") {
		t.Error("missing arg should NOT dump full usage")
	}
}

func TestCLI_ErrorNoUsageDump_AgentInspectMissingArg(t *testing.T) {
	_, stderr, err := executeCommand("agent", "inspect")
	if err == nil {
		t.Fatal("expected error for missing argument")
	}

	if strings.Contains(stderr, "Available Commands:") {
		t.Error("missing arg should NOT dump full usage")
	}
}

// --- DisplayedError ---

func TestDisplayedError_Wraps(t *testing.T) {
	inner := &ui.DisplayedError{Err: errTest}
	if inner.Error() != "test error" {
		t.Errorf("expected 'test error', got %q", inner.Error())
	}
	if inner.Unwrap() != errTest {
		t.Error("Unwrap should return inner error")
	}
}

func TestDisplayedError_DetectedByExecute(t *testing.T) {
	// Verify that Execute() can type-assert on *ui.DisplayedError
	var err error = &ui.DisplayedError{Err: errTest}
	_, ok := err.(*ui.DisplayedError)
	if !ok {
		t.Error("should be detectable as *ui.DisplayedError")
	}
}

var errTest = errorString("test error")

type errorString string

func (e errorString) Error() string { return string(e) }

// --- Help structure tests ---

func TestCLI_WorldHelpGrouped(t *testing.T) {
	out, _, err := executeCommand("world", "--help")
	if err != nil {
		t.Fatal(err)
	}

	// Should show grouped sections
	for _, section := range []string{"Lifecycle:", "Observe:", "Snapshots:"} {
		assertContains(t, out, section, "world help sections")
	}
}

func TestCLI_AgentHelpGrouped(t *testing.T) {
	out, _, err := executeCommand("agent", "--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, section := range []string{"Lifecycle:", "Evolution:", "Portability:", "Spawn Flags:"} {
		assertContains(t, out, section, "agent help sections")
	}
}

func TestCLI_WorldHelpShowsSpawnFlags(t *testing.T) {
	out, _, err := executeCommand("world", "--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, flag := range []string{"--agent", "--workspace", "--interactive", "--no-agent", "--runtime"} {
		assertContains(t, out, flag, "world spawn flags")
	}
}

func TestCLI_AgentHelpShowsSpawnFlags(t *testing.T) {
	out, _, err := executeCommand("agent", "--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, flag := range []string{"--world", "--npc"} {
		assertContains(t, out, flag, "agent spawn flags")
	}
}

// --- Root help structure ---

func TestCLI_RootHelpSections(t *testing.T) {
	out, _, err := executeCommand("--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, section := range []string{"Quick Start:", "Orchestration:", "World:", "Agent:", "Marketplace:", "System:", "Flags:"} {
		assertContains(t, out, section, "root help sections")
	}
}

func TestCLI_RootHelpFooter(t *testing.T) {
	out, _, err := executeCommand("--help")
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, out, "Use \"spwn <command> --help\"", "root help footer hint")
}

// --- Subcommand help fallback ---

func TestCLI_WorldListHelp(t *testing.T) {
	out, _, err := executeCommand("world", "list", "--help")
	if err != nil {
		t.Fatal(err)
	}

	// Should show list-specific help, NOT the grouped world help
	assertContains(t, out, "list", "world list help")
	// Should NOT contain the section headers from grouped help
	if strings.Contains(out, "Snapshots:") {
		t.Error("world list --help should show list help, not parent grouped help")
	}
}

func TestCLI_AgentInitHelp(t *testing.T) {
	out, _, err := executeCommand("agent", "init", "--help")
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, out, "init", "agent init help")
	assertContains(t, out, "6-layer Mind", "agent init description")
}

// --- Bare command shows help ---

func TestCLI_BareWorldShowsHelp(t *testing.T) {
	out, _, err := executeCommand("world")
	if err != nil {
		t.Fatal(err)
	}

	// Bare "world" with no flags should show help
	assertContains(t, out, "Lifecycle:", "bare world shows grouped help")
}

func TestCLI_BareAgentShowsHelp(t *testing.T) {
	out, _, err := executeCommand("agent")
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, out, "Lifecycle:", "bare agent shows grouped help")
}
