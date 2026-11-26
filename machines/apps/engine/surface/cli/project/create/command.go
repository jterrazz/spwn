package projectcreate

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageproject "github.com/jterrazz/studio-station/apps/engine/pages/project"
	projectcalls "github.com/jterrazz/studio-station/apps/engine/project/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	Folder      func() string
}

type Input struct{ Name, Harness string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "create <name>",
		Short:   "Make a project: a new folder of this one, with its spwn.yaml and a starter agent",
		Example: "  station project create site\n  station project create site --harness " + catalog.Harnesses()[len(catalog.Harnesses())-1],
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Harness, "harness", "", "the harness its starter agent runs in, one of "+strings.Join(catalog.Harnesses(), ", ")+" (default "+catalog.Harnesses()[0]+")")
		},
		Bind: func(in *Input, args []string) error { in.Name = args[0]; return nil },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			p, err := call.Invoke(ctx, env.Calls, machine.Ref{}, projectcalls.Create, projectcalls.CreateIn{Folder: d.Folder(), Name: in.Name, Harness: in.Harness}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageproject.Created(p, d.ThisMachine), nil)
		},
	}, env)
}
