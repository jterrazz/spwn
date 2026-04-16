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

func TestCLI_ErrorNoUsageDump_AgentShowMissingArg(t *testing.T) {
	_, stderr, err := executeCommand("agent", "show")
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
	for _, section := range []string{"Lifecycle:", "Observe:", "Knowledge:"} {
		assertContains(t, out, section, "world help sections")
	}
}

func TestCLI_AgentHelpGrouped(t *testing.T) {
	out, _, err := executeCommand("agent", "--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, section := range []string{"Lifecycle:", "Compose:", "Conversation:", "Evolution:", "Portability:"} {
		assertContains(t, out, section, "agent help sections")
	}
}

func TestCLI_WorldHelpShowsSpawnFlags(t *testing.T) {
	out, _, err := executeCommand("world", "--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, flag := range []string{"--agent", "--workspace", "--interactive"} {
		assertContains(t, out, flag, "world spawn flags")
	}
}

func TestCLI_AgentHelpShowsComposeFlags(t *testing.T) {
	out, _, err := executeCommand("agent", "--help")
	if err != nil {
		t.Fatal(err)
	}

	// The Compose section should mention the composition flags.
	for _, flag := range []string{"--dep", "--skill"} {
		assertContains(t, out, flag, "agent compose flags")
	}
}

// --- Root help structure ---

func TestCLI_RootHelpSections(t *testing.T) {
	out, _, err := executeCommand("--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, section := range []string{"Quick Start:", "Entities:", "Building blocks:", "Shortcuts:", "Coordination:", "System:"} {
		assertContains(t, out, section, "root help sections")
	}
}

func TestCLI_RootHelpFooter(t *testing.T) {
	out, _, err := executeCommand("--help")
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, out, "spwn <command> --help", "root help footer hint")
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

func TestCLI_AgentCreateHelp(t *testing.T) {
	out, _, err := executeCommand("agent", "create", "--help")
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, out, "create", "agent create help")
	// QA finding #31: the help long description used to lie about
	// a "6-layer Mind" when the scaffolded tree has 5 layers. Pin
	// the corrected wording here.
	assertContains(t, out, "5-layer Mind", "agent create description")
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
