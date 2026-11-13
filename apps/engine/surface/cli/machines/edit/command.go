package machinesedit

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachines "github.com/jterrazz/studio-station/apps/engine/pages/machines"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	// value: the machines known here, for completion.
	Names func() []string
	// value: the sizes the flags say, read with their units as create reads them.
	Resources   func(cpus int, memory, disk string) (machines.Resources, error)
	ThisMachine page.Context
}

type Input struct {
	Name                string
	CPUs                int
	Memory, Disk, Check string
	NoCheck             bool
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "edit <vm>",
		Short:   "Change a VM's CPUs, memory or disk, applied when it next starts; or a server's health check",
		Example: "  station machines edit linux-1 --memory 8G\n  station machines edit linux-1 --disk 100G\n  station machines edit infra --check infra:6443",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.IntVar(&in.CPUs, "cpus", 0, "virtual CPUs")
			f.StringVar(&in.Memory, "memory", "", "memory with its unit, e.g. 8G or 512M")
			f.StringVar(&in.Disk, "disk", "", "disk size with its unit, e.g. 100G; a disk only grows")
			f.StringVar(&in.Check, "check", "", "a server's health check: a URL (https://infra:6443/readyz) or host:port")
			f.BoolVar(&in.NoCheck, "no-check", false, "remove a server's health check")
		},
		Bind: func(in *Input, args []string) error {
			in.Name = args[0]
			return nil
		},
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			r, err := d.Resources(in.CPUs, in.Memory, in.Disk)
			if err != nil {
				return err
			}
			edited, err := call.Invoke(ctx, env.Calls, env.Target(), machinescalls.Edit, machinescalls.EditIn{VM: in.Name, Resources: r, Check: in.Check, NoCheck: in.NoCheck}, nil)
			if err != nil {
				return kit.OnItsMac(ctx, env, in.Name, err)
			}
			return io.Page(pagemachines.Edited(edited, d.ThisMachine), nil)
		},
	}, env)
}
