package initcmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/page"
	pagesetup "github.com/jterrazz/studio-station/src/pages/setup"
	setupcalls "github.com/jterrazz/studio-station/src/setup/calls"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

const SetupMoved = "apps and settings no longer come with setup: station apps install <app>, station settings apply <setting>"

type Deps struct {
	ThisMachine page.Context
	// value: the plan of the Machinefile named, then its run, as `station
	// apply --machinefile` prints them.
	Apply func(ctx context.Context, machinefile string, io kit.IO) error
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(Leaf(d, env, ""), env)
}

func Leaf(d Deps, env kit.Env, note string) kit.Leaf[Input] {
	// value: init's leaf; a note, when set, is said on stderr before it
	// runs, never with --json or --page, which keep stderr for the script.
	return kit.Leaf[Input]{
		Use:     "init",
		Short:   "Make this machine known to station: its name, the ~/.zshrc block, the XDG folders",
		Example: "  station init\n  station init --name macbook\n  station init --machinefile jterrazz-macbook-pro",
		Group:   kit.StationItself,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Name, "name", "", "the name this Mac takes (asked on a terminal when omitted)")
			f.StringVar(&in.Machinefile, "machinefile", "", "the Machinefile this machine carries, by its name or its file: its plan is printed, then applied")
		},
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			if note != "" && !io.JSON && !io.PageJSON {
				io.Note(note)
			}
			report, err := call.Invoke(ctx, env.Calls, env.Target(), setupcalls.Init, setupcalls.InitIn{Name: in.Name, Interactive: io.Interactive}, nil)
			if len(report.Steps) > 0 {
				if answerErr := io.Page(pagesetup.Outcome(report, d.ThisMachine, in.Machinefile), nil); answerErr != nil {
					return answerErr
				}
			}
			if err != nil || in.Machinefile == "" {
				return err
			}
			return d.Apply(ctx, in.Machinefile, io)
		},
	}
}
