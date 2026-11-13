package kit

import "github.com/spf13/cobra"

const (
	Machines        = "machines"
	OnAMachine      = "on-a-machine"
	ImagesAndEngine = "images-and-engine"
	ThisTerminal    = "this-terminal"
	StationItself   = "station-itself"
)

func Groups() []*cobra.Group {
	// value: the help map's groups, in their one order (SPEC §4.1).
	return []*cobra.Group{
		{ID: Machines, Title: "Machines"},
		{ID: OnAMachine, Title: "On a machine"},
		{ID: ImagesAndEngine, Title: "A Mac's images and engine"},
		{ID: ThisTerminal, Title: "This terminal"},
		{ID: StationItself, Title: "Station itself"},
	}
}
