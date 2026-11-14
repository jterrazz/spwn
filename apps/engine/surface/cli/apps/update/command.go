package appsupdate

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageapps "github.com/jterrazz/studio-station/apps/engine/pages/apps"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	Now         func() time.Time
	// value: the keys the catalog's managers install by, in its order.
	Managers []string
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "update [<app...>]",
		Short:   "Update apps; bare, every one with an update available",
		Example: "  station apps update\n  station apps update gh lazygit\n  station apps update --manager " + first(d.Managers) + "\n  station apps update --all-machines",
		Args:    cobra.ArbitraryArgs,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringArrayVar(&in.Manager, "manager", nil, "update every app of a manager: "+strings.Join(d.Managers, ", "))
			f.BoolVar(&in.AllMachines, "all-machines", false, "on every Mac and workspace; servers are skipped")
		},
		Bind: binder(d.Managers),
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			req := appscalls.UpdateIn{Names: in.Names, Managers: in.Manager}
			if in.AllMachines {
				return everyMachine(ctx, env, d, req, io)
			}
			start := d.Now()
			steps := kit.Steps(io)
			outcomes, err := call.Invoke(ctx, env.Calls, env.Target(), appscalls.Update, req, steps)
			steps.Stop()
			if err != nil {
				return err
			}
			if err := io.Page(pageapps.Updated(kit.Worded(outcomes), ran(env, d.Now().Sub(start)), d.ThisMachine), nil); err != nil {
				return err
			}
			return partial(outcomes, "re-run it: station apps update ")
		},
	}, env)
}

func everyMachine(ctx context.Context, env kit.Env, d Deps, req appscalls.UpdateIn, io kit.IO) error {
	// invariant: every Mac and running workspace is updated in turn, This Mac
	// first; a server is skipped and said, a silent Mac is a failure.
	list, err := call.Read(ctx, env.Calls, env.Self, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
	if err != nil {
		return err
	}
	var updates []pageapps.MachineUpdate
	failed := 0
	run := func(name string, target machine.Ref) {
		outcomes, err := call.Invoke(ctx, env.Calls, target, appscalls.Update, req, nil)
		u := pageapps.MachineUpdate{Machine: name, Outcomes: kit.Worded(outcomes)}
		var refused *fail.Error
		switch {
		case errors.As(err, &refused) && refused.Type == fail.NotForThisMachine:
			u.Skipped = "server"
		case err != nil:
			u.Err = kit.Screened(err, "").Message
			failed++
		}
		if n, _ := apps.FailedOf(outcomes); n > 0 {
			failed++
		}
		updates = append(updates, u)
	}
	for _, h := range list.All.Macs {
		if !h.Reachable && !h.ThisMac {
			updates = append(updates, pageapps.MachineUpdate{Machine: h.Name, Err: h.Name + " does not answer"})
			failed++
			continue
		}
		run(h.Name, machine.Ref{Name: h.Name})
		for _, vm := range h.VMs {
			switch {
			case vm.Lifetime.Kind() == machine.Server:
				updates = append(updates, pageapps.MachineUpdate{Machine: vm.Name, Skipped: "server"})
				continue
			case vm.State != machines.Running:
				updates = append(updates, pageapps.MachineUpdate{Machine: vm.Name, Skipped: string(vm.State)})
				continue
			}
			run(vm.Name, machine.Ref{Name: vm.Name, On: h.Name})
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := io.Page(pageapps.EveryMachine(updates, d.ThisMachine), nil); err != nil {
		return err
	}
	if failed > 0 {
		return kit.PartialFailure{Failed: failed, Total: len(updates), Noun: "machines", FirstHint: "station machines"}
	}
	return nil
}

func partial(outcomes []apps.Outcome, repair string) error {
	n, first := apps.FailedOf(outcomes)
	if n == 0 {
		return nil
	}
	return kit.PartialFailure{Failed: n, Total: len(outcomes), Noun: "apps", FirstHint: repair + first}
}

func Via(cmd *cobra.Command) *cobra.Command {
	// value: update as the hidden v2 upgrade takes it: --via is --manager.
	cmd.Flags().SetNormalizeFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
		if name == "via" {
			name = "manager"
		}
		return pflag.NormalizedName(name)
	})
	return cmd
}

func first(list []string) string {
	if len(list) == 0 {
		return ""
	}
	return list[0]
}

func ran(env kit.Env, took time.Duration) pageapps.Run {
	run := pageapps.Run{Took: took}
	if env.Far() {
		run.On = env.Target().Name
	}
	return run
}
