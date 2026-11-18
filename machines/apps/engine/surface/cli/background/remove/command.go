package backgroundremove

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/background"
	backgroundcalls "github.com/jterrazz/studio-station/apps/engine/background/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagebackground "github.com/jterrazz/studio-station/apps/engine/pages/background"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct{ Label string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "remove <label>",
		Short:   "Remove a background task of your own: boot it out and delete its plist; a root one is yours to remove",
		Example: "  station background remove com.example.agent\n  station background remove com.example.agent --yes",
		Args:    cobra.ExactArgs(1),
		Bind:    func(in *Input, args []string) error { in.Label = args[0]; return nil },
		Asks:    true,
		Confirm: func(in *Input) string { return pagebackground.Confirm(in.Label).Question },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			var task background.Task
			err := io.Confirmed(func(answer string) (err error) {
				task, err = call.Invoke(ctx, env.Calls, env.Target(), backgroundcalls.Remove, backgroundcalls.RemoveIn{Label: in.Label, Confirm: answer}, nil)
				return err
			})
			if err != nil {
				return err
			}
			return io.Page(pagebackground.Removed(task, d.ThisMachine), nil)
		},
	}, env)
}
