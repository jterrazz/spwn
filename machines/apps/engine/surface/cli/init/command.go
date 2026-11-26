package initcmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagesetup "github.com/jterrazz/studio-station/apps/engine/pages/setup"
	setupcalls "github.com/jterrazz/studio-station/apps/engine/setup/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	// value: the Machinefile named, resolved and read with the state its
	// apply needs, as `station machinefile apply --machinefile` refuses it; init runs it
	// before any step.
	Resolve func(ctx context.Context, machinefile string) error
	// value: the plan of the Machinefile named, then its run, as `station
	// apply --machinefile` prints them.
	Apply func(ctx context.Context, machinefile string, io kit.IO) error
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(Leaf(d, env), env)
}

func Leaf(d Deps, env kit.Env) kit.Leaf[Input] {
	return kit.Leaf[Input]{
		Use:     "init",
		Short:   "Make this machine known to station: its name, the ~/.zshrc block, the XDG folders",
		Example: "  station init\n  station init --name macbook\n  station init --machinefile macbook-pro",
		Group:   kit.Spwn,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Name, "name", "", "the name this Mac takes (asked on a terminal when omitted)")
			f.StringVar(&in.Machinefile, "machinefile", "", "the Machinefile this machine carries, by its name or its file: its plan is printed, then applied")
		},
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			if in.Machinefile != "" {
				if err := d.Resolve(ctx, in.Machinefile); err != nil {
					return err
				}
			}
			plan, err := call.Invoke(ctx, env.Calls, env.Target(), setupcalls.Init, setupcalls.InitIn{Name: in.Name, Plan: true}, nil)
			if err != nil {
				return err
			}
			if len(plan.Moves) > 0 {
				if shown := io.Page(pagesetup.Moves(plan.Moves, d.ThisMachine), nil); shown != nil {
					return shown
				}
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
