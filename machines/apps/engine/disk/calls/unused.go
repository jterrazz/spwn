package calls

import (
	"context"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/disk"
)

type FreeUnusedIn struct {
	// value: what to free, by the names disk.read's Station.CanFree lists;
	// at least one, each once: the typed answer is them, joined by ", ".
	Names   []string
	Confirm string
}

var FreeUnused = call.Call[Deps, FreeUnusedIn, []disk.Outcome]{
	// invariant: what nothing uses on this Mac goes only by name, after the
	// names typed: the question is asked where the call came from, on the
	// Mac that asked it, and its answer binds the set it named. Each name is
	// read again here, and refused before anything goes when nothing-uses
	// no longer holds it.
	Name: "disk.free-unused", Section: section.Disk, Where: call.AtTarget, Applies: AppliesOnAMac, Confirm: confirm.FreeUnused, Journal: keptFreeUnused,
	Errors: []fail.Code{NamesNeeded, NotFreeable, RecordUnreadable, confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in FreeUnusedIn, _ call.Progress) ([]disk.Outcome, error) {
		var names []string
		for _, name := range in.Names {
			if !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
		if len(names) == 0 {
			return nil, fail.New(fail.Refused, NamesNeeded, "", nil)
		}
		unused, err := d.Disk.Unused(ctx)
		if err != nil {
			return nil, coded(err)
		}
		pick := make([]disk.Freeable, 0, len(names))
		for _, name := range names {
			i := slices.IndexFunc(unused, func(f disk.Freeable) bool { return f.Name == name })
			if i < 0 {
				return nil, fail.New(fail.Refused, NotFreeable, name, nil)
			}
			pick = append(pick, unused[i])
		}
		if err := confirm.Require(confirm.Of(confirm.FreeUnused, d.Kind), strings.Join(names, ", "), in.Confirm); err != nil {
			return nil, err
		}
		return d.Disk.FreeUnused(ctx, pick), nil
	},
}

func AppliesOnAMac(p machine.Profile) machine.Verdict {
	// invariant: what nothing uses is a Mac's: its machines' and images'
	// disks live where the engine runs.
	if !p.Caps.Engine {
		return machine.No(p.Ref.Name + " holds no machine disk: only a Mac makes VMs")
	}
	return machine.Yes
}
