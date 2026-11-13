package machinescmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachines "github.com/jterrazz/studio-station/apps/engine/pages/machines"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type SSHDeps struct {
	// value: the machines known here, for completion.
	Names       func() []string
	ThisMachine page.Context
}

type SSHInput struct{ Name string }

func SSHCommand(d SSHDeps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[SSHInput]{
		Use:     kit.MachinesSSHUse,
		Short:   "Print the ssh command that reaches a machine",
		Example: "  station machines ssh infra\n  $(station machines ssh infra) uptime",
		Args:    cobra.ExactArgs(1),
		Bind: func(in *SSHInput, args []string) error {
			in.Name = args[0]
			return nil
		},
		Read:     func(*SSHInput) bool { return true },
		Pages:    true,
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *SSHInput, io kit.IO) error {
			target, err := call.Invoke(ctx, env.Calls, env.Target(), machinescalls.SSH, machinescalls.SSHIn{Name: in.Name}, nil)
			if err != nil {
				return err
			}
			if io.JSON || io.PageJSON {
				return io.Page(pagemachines.SSH(in.Name, target, d.ThisMachine), pagemachines.SSHContract(in.Name, target))
			}
			return io.Raw([]byte(target.Command + "\n"))
		},
	}, env)
}
