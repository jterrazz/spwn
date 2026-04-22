package agent

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
)

// ── agent publish / get ────────────────────────────────────────────────
//
// Lifecycle stubs for the planned registry. Composition itself lives
// on the root `spwn install` / `spwn uninstall` verbs (with an
// optional --agent <name> to scope to a single agent); the legacy
// `agent add` / `agent remove` composition commands were retired when
// install picked up agent scoping.

func init() {
	Cmd.AddCommand(publishCmd)
	Cmd.AddCommand(getCmd)
}

// Hidden: true keeps these reachable programmatically but strips
// them from `spwn agent --help`. Flip to false when the registry
// ships.
var publishCmd = &cobra.Command{
	Use:    "publish <agent-name>",
	Short:  "Publish an agent to the registry (memory stripped)",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	Long: `Publish an agent to the community registry for others to pull.

Memory (journal, knowledge, sessions) is stripped before publishing —
only the composition (dependencies) and SOUL.md ship.

Not yet implemented — tracks the registry (planned).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		return notImplemented(fmt.Sprintf("agent publish %q", name),
			"The registry is planned for a future release.")
	},
}

var getCmd = &cobra.Command{
	Use:    "get <agent-ref>",
	Short:  "Install a shared agent from the registry",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	Long: `Install a shared agent from the community registry into
./spwn/agents/<name>/.

The installed agent starts with a fresh memory but inherits the
full composition from its published form.

Not yet implemented — tracks the registry port (planned).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ref := args[0]
		return notImplemented(fmt.Sprintf("agent get %q", ref),
			"The registry is planned for a future release.")
	},
}

// notImplemented writes a structured "not yet implemented" message to
// stderr and returns a DisplayedError that carries exit code 2 —
// the dedicated "feature unavailable" code used across the CLI so
// scripts can distinguish a missing feature from a runtime failure.
func notImplemented(what, detail string) error {
	fmt.Fprintf(os.Stderr, "\n  %s %s: not yet implemented\n", ui.Red("✗"), what)
	if detail != "" {
		fmt.Fprintf(os.Stderr, "  %s\n", ui.Faint(detail))
	}
	fmt.Fprintln(os.Stderr)
	return &notImplementedError{what: what}
}

// notImplementedError carries the exit-2 ("feature unavailable")
// signal back to the process entry point while keeping the
// already-rendered banner untouched (it embeds DisplayedError so
// root.Execute skips its generic "Error:" line).
type notImplementedError struct {
	what string
}

func (e *notImplementedError) Error() string {
	return fmt.Sprintf("%s: not yet implemented", e.what)
}

// ExitCode returns 2 so cmd/spwn/main.go forwards the signal to
// os.Exit. The CLI reserves exit code 2 for "planned but not yet
// implemented" features; exit 1 stays for runtime failures.
func (e *notImplementedError) ExitCode() int { return 2 }

// IsSpwnExitCoder marks this as spwn's own ExitCoder — satisfies the
// marker on cli.ExitCoder so Execute() distinguishes it from
// stdlib's os/exec.*ExitError (which also has ExitCode()).
func (e *notImplementedError) IsSpwnExitCoder() {}
