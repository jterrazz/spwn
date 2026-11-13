package kit

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/words"
)

func Help() *cobra.Command {
	// invariant: help names a command it can find; any other topic is an
	// unknown command, as the same word run without help is.
	return &cobra.Command{
		Use:     "help [command]",
		Short:   "The help of station, or of one command",
		GroupID: StationItself,
		RunE: func(c *cobra.Command, args []string) error {
			target, rest, err := c.Root().Find(args)
			if err != nil || target == nil || len(rest) > 0 || (len(args) > 0 && target == c.Root()) {
				return fail.New(fail.Refused, words.UnknownCommand, strings.Join(args, " "), nil)
			}
			return target.Help()
		},
	}
}
