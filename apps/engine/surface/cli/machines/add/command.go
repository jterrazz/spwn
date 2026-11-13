package machinesadd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachines "github.com/jterrazz/studio-station/apps/engine/pages/machines"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct{ Name, User, Key string }

func Command(d Deps, env kit.Env) *cobra.Command {
	cmd := kit.Command(kit.Leaf[Input]{
		Use:     "add <mac>",
		Short:   "Add another Mac by its " + catalog.TailnetProduct + " name, once its ssh and its station answer",
		Example: "  station machines add jterrazz-mac-mini --user jterrazz.agent --key ~/.ssh/id_ed25519",
		Args:    cobra.MaximumNArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Name, "name", "", "the Mac's name, as station v3.0 took it")
			f.StringVar(&in.User, "user", "", "the login on that Mac, when it differs from yours")
			f.StringVar(&in.Key, "key", "", "the ssh key file (default ~/.ssh/id_ed25519)")
		},
		Bind: func(in *Input, args []string) error {
			if len(args) == 1 {
				in.Name = args[0]
			}
			if in.Name == "" {
				return kit.Refuse("say which Mac to add", "station machines add <mac>")
			}
			return nil
		},
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			added, err := call.Invoke(ctx, env.Calls, env.Target(), machinescalls.Add, machinescalls.AddIn{Name: in.Name, User: in.User, Key: in.Key}, nil)
			if e := new(fail.Error); errors.As(err, &e) && e.Code == call.Unreachable {
				// invariant: a Mac that did not answer was not added: the
				// hint is the list, never the page of a Mac it does not hold.
				worded := kit.Translate(err, "machines add")
				worded.Hint = "station machines"
				return worded
			}
			if err != nil {
				return err
			}
			return io.Page(pagemachines.Added(added, d.ThisMachine), nil)
		},
	}, env)
	_ = cmd.Flags().MarkHidden("name")
	return cmd
}
