package agent

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
)

func init() {
	Cmd.AddCommand(publishCmd)
	Cmd.AddCommand(getCmd)
}

var publishCmd = &cobra.Command{
	Use:    "publish <agent-name>",
	Short:  "Publish an agent to the registry (memory stripped)",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	Long: `Publish an agent to the community registry for others to pull.

Memory (journal, knowledge, sessions) is stripped before publishing:
only the composition (dependencies) and SOUL.md ship.

Not yet implemented: it tracks the registry (planned).`,
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

Not yet implemented: it tracks the registry port (planned).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ref := args[0]
		return notImplemented(fmt.Sprintf("agent get %q", ref),
			"The registry is planned for a future release.")
	},
}

func notImplemented(what, detail string) error {
	fmt.Fprintf(os.Stderr, "\n  %s %s: not yet implemented\n", ui.Red("✗"), what)
	if detail != "" {
		fmt.Fprintf(os.Stderr, "  %s\n", ui.Faint(detail))
	}
	fmt.Fprintln(os.Stderr)
	return &notImplementedError{what: what}
}

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
