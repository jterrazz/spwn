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

type Input struct{ Name, Harness, From string }

func Command(d Deps, env kit.Env) *cobra.Command {
	examples := catalog.Examples()
	return kit.Command(kit.Leaf[Input]{
		Use:     "create <name>",
		Short:   "Make a project: a new folder of this one, with its spwn.yaml and a starter agent, or the files of an example",
		Example: "  station project create site\n  station project create site --harness " + catalog.Harnesses()[len(catalog.Harnesses())-1] + "\n  station project create --from " + examples[0] + "\n  station project create mine --from " + examples[len(examples)-1],
		Args: func(c *cobra.Command, a []string) error {
			if c.Flags().Changed("from") {
				return cobra.MaximumNArgs(1)(c, a)
			}
			return cobra.ExactArgs(1)(c, a)
		},
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Harness, "harness", "", "the harness its starter agent runs in, one of "+strings.Join(catalog.Harnesses(), ", ")+" (default "+catalog.Harnesses()[0]+")")
			f.StringVar(&in.From, "from", "", "make the project from an example of the catalog, one of "+strings.Join(examples, ", ")+"; the name defaults to the example's")
		},
		Bind: func(in *Input, args []string) error {
			if len(args) > 0 {
				in.Name = args[0]
			}
			return nil
		},
		Near: catalog.Examples,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			p, err := call.Invoke(ctx, env.Calls, machine.Ref{}, projectcalls.Create, projectcalls.CreateIn{Folder: d.Folder(), Name: in.Name, Harness: in.Harness, From: in.From}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageproject.Created(p, d.ThisMachine), nil)
		},
	}, env)
}
