package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/cliproject"
)

// startCmd and stopCmd are reserved for a future world where agents
// run as autonomous daemons - they have their own loop, react to
// events without a human trigger, and have a meaningful "started" vs
// "stopped" state. Today that's not how spwn works: an agent is a
// tool you pick up, not a service you boot. Every invocation is
// user-initiated (`spwn agent <name>`), and between invocations the
// container is just an idle sandbox.
//
// Rather than pretending otherwise, these commands are registered but
// return a friendly "not yet" error so the command space is reserved
// and users who reach for them get a clear explanation.

var startCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start an agent as a background daemon [planned]",
	Long: `Run an agent as a long-lived autonomous process.

This command is reserved for a future release. Today, spwn agents
are invoked interactively ("spwn agent <name>") - they don't run on
their own between invocations.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return notYetImplemented("start", args[0])
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop an agent daemon [planned]",
	Long: `Kill a running autonomous agent process.

This command is reserved for a future release. Today, spwn agents
don't run between invocations - there is nothing to stop.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return notYetImplemented("stop", args[0])
	},
}

func init() {
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(stopCmd)
}

func notYetImplemented(verb, name string) error {
	return fmt.Errorf(
		"agent %s is not yet implemented.\n"+
			"Today's agents are interactive: run \"spwn agent %s\" to open a session.\n"+
			"spwn agent start/stop will land when agents become autonomous daemons",
		verb, name,
	)
}

// findWorldForAgent locates the first spwn.yaml world entry that
// references the named agent. Returns a descriptive error when no
// project is active or the agent is absent from every world.
func findWorldForAgent(agentName string) (string, error) {
	p, err := cliproject.Require()
	if err != nil {
		return "", err
	}
	for name, w := range p.Manifest.Worlds {
		for _, a := range w.Agents {
			if a == agentName {
				return name, nil
			}
		}
	}
	return "", fmt.Errorf("agent %q is not in any world in spwn.yaml", agentName)
}
