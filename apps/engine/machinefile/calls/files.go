package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type FilesIn struct{}

// value: every Machinefile This Mac reads, each with what it makes, its
// text, this Mac's overlay of it and the machines that follow it.

type Files struct {
	Files []machinefile.FileView
}

var ListFiles = call.Call[Deps, FilesIn, Files]{
	// value: the Machinefiles of every source This Mac reads, the owner's,
	// the shared one and station's, in name order. It changes nothing.
	Name: "machinefile.files", Where: call.InProcess, Applies: machine.Anywhere,
	Errors: []fail.Code{Invalid, NewerSchema, Unavailable, CatalogInvalid, CatalogNewer},
	Run: func(ctx context.Context, d Deps, _ FilesIn, _ call.Progress) (Files, error) {
		lib, err := library(ctx, d)
		if err != nil {
			return Files{}, Worded(err)
		}
		files := machinefile.FilesOf(lib)
		follow := followers(ctx, d)
		for i := range files {
			files[i].Followers = follow[files[i].Name]
			if files[i].Followers == nil {
				files[i].Followers = []machinefile.Follower{}
			}
		}
		return Files{Files: files}, nil
	},
}

// invariant: a Mac follows the Machinefile it is bound to, as its own
// engine says it; a VM the one it was made from, as its record keeps it.
// A Mac that does not answer is left out, never guessed bound.
func followers(ctx context.Context, d Deps) map[string][]machinefile.Follower {
	out := map[string][]machinefile.Follower{}
	if bound, err := d.Prefs.Bound(); err == nil && bound != "" {
		out[bound] = append(out[bound], machinefile.Follower{Machine: d.Name, Mac: d.Name, ThisMac: true})
	}
	if d.AllMachines == nil || d.Far == nil {
		return out
	}
	all, err := d.AllMachines(ctx)
	if err != nil {
		return out
	}
	for _, mac := range all.Macs {
		if !mac.ThisMac && mac.Reachable {
			if n, err := call.Invoke(ctx, d.Far, machine.Ref{Name: mac.Name}, LastNight, NightlyIn{}, nil); err == nil && n.Bound != "" {
				out[n.Bound] = append(out[n.Bound], machinefile.Follower{Machine: mac.Name, Mac: mac.Name})
			}
		}
		for _, vm := range mac.VMs {
			if vm.Machinefile != "" {
				out[vm.Machinefile] = append(out[vm.Machinefile], follower(mac, vm))
			}
		}
	}
	return out
}

func follower(mac machines.MacView, vm machines.VM) machinefile.Follower {
	return machinefile.Follower{Machine: vm.Name, Mac: mac.Name, ThisMac: mac.ThisMac}
}
