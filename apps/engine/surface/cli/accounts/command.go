package accountscmd

import (
	"errors"

	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageaccounts "github.com/jterrazz/studio-station/apps/engine/pages/accounts"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct{ AllMachines bool }

func Command(d Deps, env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Command(kit.Leaf[Input]{
		Use:     "accounts",
		Short:   "Who is signed in here, and the accounts shared with VMs: names and scopes, never a token",
		Example: "  station accounts\n  station accounts --on linux-1\n  station accounts --all-machines\n  station accounts --json",
		Group:   kit.OnAMachine,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.BoolVar(&in.AllMachines, "all-machines", false, "every Mac and workspace, each with its own sign-ins and shares")
		},
		Read:  func(*Input) bool { return true },
		Live:  true,
		Pages: true,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			if in.AllMachines {
				return everyMachine(ctx, env, io)
			}
			updates, err := call.Follow(ctx, env.Calls, env.Target(), accountscalls.List, accountscalls.ListIn{VMs: io.CanLive() || io.PageJSON}, nil)
			if err != nil {
				return err
			}
			if io.CanLive() {
				return io.LivePage(kit.Updates(updates, func(r accounts.Report) page.Page {
					return pageaccounts.Page(r, d.ThisMachine)
				}))
			}
			var last accounts.Report
			for u := range updates {
				if u.Err != nil {
					return u.Err
				}
				last = u.Value
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			shared := held(ctx, env, last)
			return io.PageRecords(pageaccounts.Page(last, d.ThisMachine), pageaccounts.List(last, shared), pageaccounts.ListContract(accountscalls.List.Schema(), last, shared))
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}

// value: the shares a reading lists: a Mac's own, or, for a workspace, the
// shares its Mac holds with it, read from that Mac.
func held(ctx context.Context, env kit.Env, r accounts.Report) []accounts.Shared {
	if len(r.Machines) == 0 || r.Machines[0].Kind != accounts.KindVM {
		return r.Shares
	}
	vm := r.Machines[0].Name
	h, ok := kit.Holder(ctx, env, vm)
	if !ok {
		return nil
	}
	return sharesOf(ctx, env, machine.Ref{Name: h.Mac}, vm)
}

func sharesOf(ctx context.Context, env kit.Env, mac machine.Ref, vm string) []accounts.Shared {
	r, err := call.Read(ctx, env.Calls, mac, accountscalls.List, accountscalls.ListIn{}, nil)
	if err != nil {
		return nil
	}
	var out []accounts.Shared
	for _, sh := range r.Shares {
		if sh.VM == vm {
			out = append(out, sh)
		}
	}
	return out
}

func everyMachine(ctx context.Context, env kit.Env, io kit.IO) error {
	// invariant: every Mac then each of its workspaces, This Mac first; a
	// server is named and not listed, a stopped workspace says so, a Mac that
	// does not answer is its own line and the others are filled.
	list, err := call.Read(ctx, env.Calls, env.Self, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
	if err != nil {
		return err
	}
	var blocks []pageaccounts.MachineBlock
	for _, h := range list.All.Macs {
		mac := pageaccounts.MachineBlock{Name: h.Name, Label: page.MacName(h.Name, env.Self.Name), Kind: string(machine.Computer), ThisMac: h.ThisMac}
		if !h.Reachable && !h.ThisMac {
			mac.Mark, mac.Reason, mac.State, mac.Fault = page.Broken, page.Unreachable, "does not answer", pageaccounts.Fault{Type: string(fail.Unreachable)}
			blocks = append(blocks, mac)
			continue
		}
		read, err := call.Read(ctx, env.Calls, machine.Ref{Name: h.Name}, accountscalls.List, accountscalls.ListIn{}, nil)
		if err != nil {
			blocks = append(blocks, faulted(mac, err))
			continue
		}
		mac.Report, mac.Shared = read, read.Shares
		blocks = append(blocks, mac)
		for _, vm := range h.VMs {
			b := pageaccounts.MachineBlock{Name: vm.Name, Label: vm.Name, Kind: string(vm.Lifetime.Kind()), On: h.Name}
			if vm.Lifetime.Kind() == machine.Server {
				b.Note, b.Server = vm.Name+" is a server: not listed", true
				blocks = append(blocks, b)
				continue
			}
			if vm.State != machines.Running {
				b.Mark, b.Reason, b.State = page.Absent, page.Stopped, string(vm.State)
				b.Fault = pageaccounts.Fault{Type: string(fail.Refused), Code: string(accountscalls.VMStopped)}
				b.Note = vm.Name + " is stopped"
				blocks = append(blocks, b)
				continue
			}
			r, err := call.Read(ctx, env.Calls, machine.Ref{Name: vm.Name, On: h.Name}, accountscalls.List, accountscalls.ListIn{}, nil)
			var refused *fail.Error
			if errors.As(err, &refused) && refused.Type == fail.NotForThisMachine {
				b.Note, b.Server = vm.Name+" is a server: not listed", true
				blocks = append(blocks, b)
				continue
			}
			if err != nil {
				blocks = append(blocks, faulted(b, err))
				continue
			}
			var shared []accounts.Shared
			for _, sh := range read.Shares {
				if sh.VM == vm.Name {
					shared = append(shared, sh)
				}
			}
			b.Report, b.Shared = r, shared
			blocks = append(blocks, b)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return io.Page(pageaccounts.EveryMachine(blocks), pageaccounts.EveryMachineContract(accountscalls.List.Schema(), blocks))
}

func faulted(b pageaccounts.MachineBlock, err error) pageaccounts.MachineBlock {
	worded := kit.Translate(err, "")
	b.Note = worded.Message
	b.Fault = pageaccounts.Fault{Type: string(worded.Type), Code: string(worded.Code)}
	if b.Fault.Type == "" {
		b.Fault.Type = string(fail.Failed)
	}
	return b
}

func Retired(verb string, env kit.Env) *cobra.Command {
	// value: secrets allow and disallow, retired: the share list is the
	// owner's, written by hand in the preferences.
	cmd := kit.Command(kit.Leaf[Input]{
		Use:     verb,
		Args:    cobra.ArbitraryArgs,
		RawArgs: true,
		Run: func(context.Context, *Input, kit.IO) error {
			return kit.Refuse("station secrets "+verb+" is retired: the share list is written by hand", "the share list is share_list in ~/.config/station/preferences.json")
		},
	}, env)
	cmd.Hidden = true
	cmd.PersistentPreRunE = func(*cobra.Command, []string) error { return nil }
	return cmd
}
