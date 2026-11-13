package usecmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachines "github.com/jterrazz/studio-station/apps/engine/pages/machines"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	// value: the machines known here, for completion.
	Names       func() []string
	ThisMachine page.Context
}

type Input struct{ Machine string }

const thisMac = "this-mac"

func Command(d Deps, env kit.Env) *cobra.Command {
	// invariant: a surface preference, no call: it chooses the machine the
	// later commands of this terminal go to, and runs here whatever it is.
	return kit.Command(kit.Leaf[Input]{
		Use:      "use [<machine>|-]",
		Short:    "Choose the machine this terminal's commands run on",
		Example:  "  station use linux-1\n  station use -",
		Group:    kit.ThisTerminal,
		Args:     cobra.MaximumNArgs(1),
		Bind:     bind,
		Complete: func([]string, string) []string { return append([]string{thisMac, "-"}, d.Names()...) },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			return run(ctx, d, env, in, io)
		},
	}, env)
}

func bind(in *Input, args []string) error {
	if len(args) == 1 {
		in.Machine = args[0]
	}
	return nil
}

func run(ctx context.Context, d Deps, env kit.Env, in *Input, io kit.IO) error {
	if env.Use == nil || !env.Use.Block() {
		return kit.Refuse("this terminal has no station block", "use --on "+example(in.Machine)+", or run station init")
	}
	now, _ := env.Use.Read()
	switch in.Machine {
	case "":
		return io.Page(pagemachines.UseCurrent(now.Machine, kit.Clock(now.Since, env.Use.Now()), d.ThisMachine), nil)
	case "-":
		if now.Previous == "" && now.Machine == "" {
			return kit.Refuse("this terminal has used no other machine", "station use <machine>")
		}
		return back(env, d, io, now, now.Previous)
	}
	to := in.Machine
	if to == thisMac || to == env.Self.Name || to == d.ThisMachine.Machine {
		return back(env, d, io, now, "")
	}
	if env.On.Known != nil {
		if known, err := env.On.Known(ctx, to); err == nil && !known {
			return kit.Refuse("no machine named \""+to+"\"", "station machines")
		}
	}
	if err := env.Use.Write(kit.Choice{Machine: to, Previous: now.Machine, Since: env.Use.Now()}); err != nil {
		return err
	}
	return io.Page(pagemachines.UseChosen(to, d.ThisMachine), nil)
}

func back(env kit.Env, d Deps, io kit.IO, now kit.Choice, to string) error {
	if err := env.Use.Write(kit.Choice{Machine: to, Previous: now.Machine, Since: env.Use.Now()}); err != nil {
		return err
	}
	return io.Page(pagemachines.UseBack(to, now.Machine, d.ThisMachine), nil)
}

func example(machine string) string {
	if machine == "" || machine == "-" || machine == thisMac {
		return "<machine>"
	}
	return machine
}
