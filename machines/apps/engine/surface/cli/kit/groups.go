package kit

import "github.com/spf13/cobra"

const (
	Machine = "machine"
	Agent   = "agent"
	OS      = "os"
	Spwn    = "spwn"
)

func Groups() []*cobra.Group {
	// value: the help's four groups, the three spaces then Spwn itself, in
	// their one order (docs/reference/cli.md, "The help"); a group with no
	// command is not drawn.
	return []*cobra.Group{
		{ID: Machine, Title: "Machine"},
		{ID: Agent, Title: "Agent"},
		{ID: OS, Title: "OS"},
		{ID: Spwn, Title: "Spwn"},
	}
}
