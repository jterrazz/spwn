package diskmove

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	diskcalls "github.com/jterrazz/studio-station/apps/engine/disk/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagedisk "github.com/jterrazz/studio-station/apps/engine/pages/disk"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	// value: this machine's home, so a folder the shell expanded is sent as
	// ~/…, the same folder on the machine that moves it.
	Home string
}

type Input struct{ Folder string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "move <folder>",
		Short:   "Move a misplaced folder where its app expects it; nothing is lost",
		Example: "  station disk move ~/go",
		Args:    cobra.ExactArgs(1),
		Bind: func(in *Input, args []string) error {
			in.Folder = args[0]
			if rest, ok := strings.CutPrefix(in.Folder, d.Home+"/"); ok && d.Home != "" {
				in.Folder = "~/" + rest
			}
			return nil
		},
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			outcome, err := call.Invoke(ctx, env.Calls, env.Target(), diskcalls.Move, diskcalls.MoveIn{Folder: in.Folder}, nil)
			if err != nil {
				return err
			}
			return io.Page(pagedisk.Moved(outcome, d.ThisMachine), nil)
		},
	}, env)
}
