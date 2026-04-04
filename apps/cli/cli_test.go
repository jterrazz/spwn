package cli

import (
	"bytes"
	"strings"
	"testing"
)

// executeCommand runs rootCmd with the given args and captures stdout/stderr.
// Cobra commands maintain state between calls, so we reset args each time.
// --help short-circuits before PersistentPreRunE, avoiding filesystem side effects.
func executeCommand(args ...string) (string, string, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()

	return stdout.String(), stderr.String(), err
}

// assertContains checks that output contains the given substring.
func assertContains(t *testing.T, output, substring, context string) {
	t.Helper()
	if !strings.Contains(output, substring) {
		t.Errorf("%s: output missing %q\n--- output ---\n%s", context, substring, output)
	}
}

// --- Root help ---

func TestCLI_Help(t *testing.T) {
	out, _, err := executeCommand("--help")
	if err != nil {
		t.Fatal(err)
	}

	// Verify all top-level subcommands are listed.
	for _, sub := range []string{"world", "agent", "profile", "msg", "snap", "architect", "dash", "init"} {
		assertContains(t, out, sub, "root help")
	}
}

func TestCLI_HelpContainsDescription(t *testing.T) {
	out, _, err := executeCommand("--help")
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, out, "spwn", "root help description")
	assertContains(t, out, "Quick Start", "root help quick start section")
	assertContains(t, out, "World:", "root help world section")
	assertContains(t, out, "world", "root help world command")
	assertContains(t, out, "agent", "root help agent command")
	assertContains(t, out, "System:", "root help system section")
}

// --- World help ---

func TestCLI_WorldHelp(t *testing.T) {
	out, _, err := executeCommand("world", "--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, sub := range []string{"list", "inspect", "logs", "attach", "destroy"} {
		assertContains(t, out, sub, "world help")
	}
}

// --- Agent help ---

func TestCLI_AgentHelp(t *testing.T) {
	out, _, err := executeCommand("agent", "--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, sub := range []string{"new", "ls", "rm", "talk", "inspect", "dream", "sleep", "fork", "export", "import"} {
		assertContains(t, out, sub, "agent help")
	}
}

func TestCLI_AgentTalkHelp(t *testing.T) {
	out, _, err := executeCommand("agent", "talk", "--help")
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, out, "talk", "agent talk help")
	assertContains(t, out, "agent-name", "agent talk usage")
	assertContains(t, out, "interactive", "agent talk description")
}

func TestCLI_AgentDeleteHelp(t *testing.T) {
	out, _, err := executeCommand("agent", "delete", "--help")
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, out, "delete", "agent rm help")
	assertContains(t, out, "agent-name", "agent rm usage")
}

// --- Architect help ---

func TestCLI_ArchitectHelp(t *testing.T) {
	out, _, err := executeCommand("architect", "--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, sub := range []string{"start", "stop", "status"} {
		assertContains(t, out, sub, "architect help")
	}
}

// --- Agent --npc flag ---

func TestCLI_AgentNPCFlag(t *testing.T) {
	out, _, err := executeCommand("agent", "--help")
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, out, "--npc", "agent help npc flag")
}

// --- Get help ---

func TestCLI_GetHelp(t *testing.T) {
	out, _, err := executeCommand("get", "--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, sub := range []string{"install", "ls", "search", "rm"} {
		assertContains(t, out, sub, "get help")
	}
}

// --- Dash help ---

func TestCLI_DashHelp(t *testing.T) {
	out, _, err := executeCommand("dash", "--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, sub := range []string{"start", "open"} {
		assertContains(t, out, sub, "dash help")
	}
}

// --- Init help ---

func TestCLI_InitHelp(t *testing.T) {
	out, _, err := executeCommand("init", "--help")
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, out, "init", "init help")
	assertContains(t, out, ".spwn", "init help mentions config dir")
}

// --- Unknown command ---

func TestCLI_UnknownCommand(t *testing.T) {
	_, _, err := executeCommand("nonexistent")
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

// --- Global flags ---

func TestCLI_GlobalFlags(t *testing.T) {
	out, _, err := executeCommand("--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, flag := range []string{"--json", "--quiet", "--verbose"} {
		assertContains(t, out, flag, "global flags")
	}
}

func TestCLI_GlobalFlagShortcuts(t *testing.T) {
	out, _, err := executeCommand("--help")
	if err != nil {
		t.Fatal(err)
	}

	// Short flags -q and -v should appear in the help output.
	assertContains(t, out, "-q", "global short flag quiet")
	assertContains(t, out, "-v", "global short flag verbose")
}
