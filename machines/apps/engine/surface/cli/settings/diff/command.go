package settingsdiff

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagesettings "github.com/jterrazz/studio-station/apps/engine/pages/settings"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	Names       func() []string
	ThisMachine page.Context
}

type Input struct{ Name string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:      "diff <setting>",
		Short:    "What station would write against what is on this machine",
		Example:  "  station settings diff starship-dotfiles",
		Args:     cobra.ExactArgs(1),
		Bind:     func(in *Input, args []string) error { in.Name = args[0]; return nil },
		Complete: func([]string, string) []string { return d.Names() },
		Pages:    true,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			out, err := call.Invoke(ctx, env.Calls, env.Target(), settingscalls.Diff, settingscalls.DiffIn{Name: in.Name}, nil)
			if err != nil {
				return err
			}
			if out.Diff != "" && io.PageJSON {
				return io.Page(pagesettings.Diff(in.Name, out.Diff, d.ThisMachine), nil)
			}
			if out.Diff != "" {
				return io.Raw([]byte(out.Diff))
			}
			return io.Page(pagesettings.Values(out.Row, out.Before, out.After, d.ThisMachine), nil)
		},
	}, env)
}
