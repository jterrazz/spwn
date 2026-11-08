package machinescleanup

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	model "github.com/jterrazz/studio-station/src/machines"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/page"
	pagemachines "github.com/jterrazz/studio-station/src/pages/machines"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	// value: the machines known here, for completion.
	Names       func() []string
	ThisMachine page.Context
}

type Input struct {
	VMs []string
	// value: what the call said would go, when it asked.
	asked model.Cleaned
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "cleanup [<vm>...]",
		Short:   "Delete the throwaways whose agent finished, or that are stopped",
		Example: "  station machines cleanup --yes\n  station machines cleanup agent-7f --yes --on mac-mini",
		Args:    cobra.ArbitraryArgs,
		Bind: func(in *Input, args []string) error {
			in.VMs = args
			return nil
		},
		Asks: true,
		Ask: func(in *Input) (bool, error) {
			return env.Prompt.Confirm(pagemachines.CleanupConfirm(in.asked.Candidates))
		},
		Refuse: func(in *Input) error {
			return kit.Refuse(pagemachines.CleanupRefused(in.asked.Candidates).Text, "re-run with --yes")
		},
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			var out model.Cleaned
			err := io.Confirmed(func(answer string) error {
				var err error
				out, err = call.Invoke(ctx, env.Calls, env.Target(), machinescalls.Cleanup, machinescalls.CleanupIn{VMs: in.VMs, Confirm: answer}, nil)
				if _, asks := confirm.Answer(err); asks {
					in.asked = out
				}
				return err
			})
			if err != nil {
				return err
			}
			mac := d.ThisMachine.Machine
			if env.Far() {
				mac = env.Target().Name
			}
			return io.Page(pagemachines.Cleaned(out, mac, d.ThisMachine), nil)
		},
	}, env)
}
