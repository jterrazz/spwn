package calls

import (
	"cmp"
	"context"
	"errors"
	"strings"

	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machinefile"
)

type UpgradeIn struct {
	// value: the Machinefile, by its name; empty, the one this machine is
	// bound to, else the one named after it.
	Machinefile string
	Interactive bool
}

var Upgrade = call.Call[Deps, UpgradeIn, machinefile.Upgraded]{
	// value: every app the Machinefile lists that is there updated by the
	// method that holds it, the managers first, a version the Machinefile
	// pins kept; then the lock written anew, its moves shown.
	Name: "machinefile.upgrade", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, NeedsTerminal},
	Run: func(ctx context.Context, d Deps, in UpgradeIn, p call.Progress) (machinefile.Upgraded, error) {
		// invariant: an upgrade is the one run that takes a shared source
		// pinned elsewhere since the lock: the lock it leaves holds the
		// commit read now.
		plan, err := judged(ctx, d, in.Machinefile)
		if err != nil {
			return machinefile.Upgraded{}, err
		}
		out := machinefile.Upgraded{Plan: plan}
		if asked := plan.UpgradeNeeds(); !in.Interactive && len(asked) > 0 {
			return out, refusedOffTerminal(asked, "upgrade")
		}
		source, err := sharedSource(d)
		if err != nil {
			return out, err
		}
		lines := plan.Upgrades()
		for i, l := range lines {
			if ctx.Err() != nil {
				for _, rest := range lines[i:] {
					out.Done = append(out.Done, machinefile.Done{Line: rest, State: rest.Seen.State, Version: rest.Seen.Version, Reason: machinefile.NotRun})
				}
				break
			}
			if l.Seen.Condition == machinefile.Absent {
				r := &run{d: d, p: p, in: ApplyIn{Interactive: in.Interactive}, name: plan.Name}
				out.Done = append(out.Done, r.reinstall(ctx, l))
				continue
			}
			out.Done = append(out.Done, update(ctx, d, l, p))
		}
		out.Interrupted = ctx.Err() != nil
		now := machinefile.Relocked(plan, out.Done, source)
		out.Lock = machinefile.Diff(plan.Lock.LockFile, now)
		if !out.Lock.Empty() {
			if err := d.Locks.Write(context.WithoutCancel(ctx), plan.Lock.LockFile, now); err != nil {
				return out, err
			}
		}
		return out, ctx.Err()
	},
}

// invariant: an app the lock holds that is not there is installed by the
// method the lock names, at the newest version the Machinefile allows (the
// lock's own version is what could no longer install), and the lock takes
// what that method reads once it ran.
func (r *run) reinstall(ctx context.Context, l machinefile.Line) machinefile.Done {
	held := cmp.Or(l.Step.Locked, l.Step.Recorded)
	fresh := l
	fresh.Step.Locked = ""
	done := r.install(ctx, fresh, machinefile.Done{Line: l, State: l.Seen.State, Version: l.Seen.Version})
	done.Line = l
	if done.Installed {
		done.Updated = true
		done.Detail = strings.TrimSpace("installed anew, the lock held " + held)
	}
	return done
}

func update(ctx context.Context, d Deps, l machinefile.Line, p call.Progress) machinefile.Done {
	// invariant: an app whose version the Machinefile pins is kept at it;
	// every other is updated by the method that holds it, and the lock
	// takes the version that method lists once it ran.
	done := machinefile.Done{Line: l, State: l.Seen.State, Method: l.Seen.Method, Version: l.Seen.Version}
	if l.Step.Version != "" {
		done.Reason = machinefile.Pinned
		return done
	}
	outcomes, err := call.Invoke(ctx, d.Calls, machine.Ref{}, appscalls.Update, appscalls.UpdateIn{Names: []string{l.Step.Item}}, p)
	switch {
	case ctx.Err() != nil:
		done.State, done.Reason = catalog.StateBroken, machinefile.Interrupted
		return done
	case err != nil:
		var refused *fail.Error
		done.State, done.Detail = catalog.StateBroken, err.Error()
		if errors.As(err, &refused) && refused.Code == appscalls.NotInstalled {
			done.State, done.Detail = catalog.StateAbsent, "not installed: station apply installs it"
		}
		return done
	}
	for _, o := range outcomes {
		if o.Name != l.Step.Item {
			continue
		}
		done.State, done.Detail, done.Log, done.Updated = o.Result, o.Detail, o.Log, true
		if o.Method != "" {
			done.Method, done.Version = string(o.Method), o.Version
		}
	}
	return done
}
