package calls

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machines"
)

const NotFinished fail.Code = "machines.not-finished"

type CleanupIn struct {
	// value: the ephemeral workspaces to delete, by name; none is every one of this
	// Mac's that may go.
	VMs     []string
	Confirm string
}

var Cleanup = call.Call[Deps, CleanupIn, machines.Cleaned]{
	// invariant: SPEC B5.9: one yes or no deletes every ephemeral workspace asked for,
	// each one the VM delete it would be alone, with its disk, its ports and
	// its shared accounts; a VM that may not go is refused before any goes,
	// and the question comes back with every candidate and what it loses.
	Name: "machines.cleanup", Where: call.InEngine, Applies: machine.Anywhere, Confirm: confirm.Cleanup,
	Errors: []fail.Code{confirm.YesNeeded, NotFinished},
	Run: func(ctx context.Context, d Deps, in CleanupIn, _ call.Progress) (machines.Cleaned, error) {
		if err := d.Machines.CheckMac(); err != nil {
			return machines.Cleaned{}, err
		}
		if err := d.Machines.EngineUp(ctx); err != nil {
			return machines.Cleaned{}, err
		}
		read, err := d.Machines.AllMachinesAsItFills(ctx, true, func(machines.AllMachines) {})
		if err != nil {
			return machines.Cleaned{}, err
		}
		var vms []machines.VM
		for _, h := range read.Macs {
			if h.ThisMac {
				vms = h.VMs
			}
		}
		gone, kept := machines.Cleanup(vms)
		out := machines.Cleaned{Kept: names(kept)}
		pick := names(gone)
		for _, name := range in.VMs {
			if !slices.Contains(pick, name) {
				return out, fail.New(fail.Refused, NotFinished, name, nil)
			}
		}
		if len(in.VMs) > 0 {
			pick = in.VMs
		}
		if out.Candidates, err = candidates(ctx, d, gone, pick); err != nil {
			return out, err
		}
		if len(pick) == 0 {
			return out, nil
		}
		if err := confirm.Require(confirm.For(confirm.Cleanup, d.Kind), strings.Join(pick, ", "), in.Confirm); err != nil {
			return out, err
		}
		for _, name := range pick {
			if _, err := d.Machines.Delete(ctx, name); err != nil {
				return out, err
			}
			out.Deleted = append(out.Deleted, name)
		}
		return out, nil
	},
}

func candidates(ctx context.Context, d Deps, gone []machines.VM, pick []string) ([]machines.Candidate, error) {
	var now time.Time
	if d.Now != nil {
		now = d.Now()
	}
	var out []machines.Candidate
	for _, vm := range gone {
		if !slices.Contains(pick, vm.Name) {
			continue
		}
		c := machines.CandidateOf(vm, now)
		if d.Holdings != nil {
			ports, accounts, err := d.Holdings.Held(ctx, vm.Name)
			if err != nil {
				return nil, err
			}
			c.Ports, c.Accounts = ports, accounts
		}
		out = append(out, c)
	}
	return out, nil
}

func names(vms []machines.VM) []string {
	out := make([]string, 0, len(vms))
	for _, vm := range vms {
		out = append(out, vm.Name)
	}
	return out
}
